// Package cache provides a public interface and implementation for an in-memory cache that keeps the most recent
// response from the control plane per aggregated key.
package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
	"github.com/uber-go/tally"

	"github.com/envoyproxy/xds-relay/internal/app/metrics"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
)

type Cache interface {
	// Fetch returns the cached resource if it exists.
	Fetch(key string) (*Resource, error)

	// SetResponse sets the cache response and returns the list of requests.
	SetResponse(key string, resp transport.Response) (*RequestsStore, error)

	// AddRequest adds the request to the cache.
	AddRequest(key string, req transport.Request) error

	// DeleteRequest removes the given request from any cache entries it's present in.
	DeleteRequest(key string, req transport.Request) error

	// GetReadOnlyCache returns a copy of the cache that only exposes read-only methods in its interface.
	GetReadOnlyCache() ReadOnlyCache
}

type ReadOnlyCache interface {
	// Fetch returns the cached resource if it exists.
	FetchReadOnly(key string) (Resource, error)
}

type cache struct {
	cacheMu sync.RWMutex
	cache   lru.Cache
	ttl     time.Duration

	logger log.Logger
	scope  tally.Scope
}

type RequestsStore struct {
	m *sync.Map
}

func NewRequestsStore() *RequestsStore {
	return &RequestsStore{
		m: &sync.Map{},
	}
}

func (m *RequestsStore) Set(r transport.Request) {
	_, _ = m.m.LoadOrStore(r, struct{}{})
}

func (m *RequestsStore) get(r transport.Request) bool {
	_, ok := m.m.Load(r)
	return ok
}

func (m *RequestsStore) Delete(r transport.Request) {
	m.m.Delete((r))
}

func (m *RequestsStore) ForEach(f func(key transport.Request)) {
	m.m.Range(func(key, value interface{}) bool {
		watch, ok := key.(transport.Request)
		if !ok {
			panic(key)
		}
		f(watch)
		return true
	})
}

type Resource struct {
	Resp           transport.Response
	Requests       *RequestsStore
	ExpirationTime time.Time
}

// OnEvictFunc is a callback function for each eviction. Receives the key and cache value when called.
type OnEvictFunc func(key string, value Resource)

func NewCache(maxEntries int, onEvicted OnEvictFunc, ttl time.Duration,
	logger log.Logger, scope tally.Scope) (Cache, error) {
	if ttl < 0 {
		return nil, fmt.Errorf("ttl must be nonnegative but was set to %v", ttl)
	}
	return &cache{
		cache: lru.Cache{
			// Max number of cache entries before an item is evicted. Zero means no limit.
			MaxEntries: maxEntries,
			// OnEvict is called for each eviction.
			OnEvicted: func(cacheKey lru.Key, cacheValue interface{}) {
				key, ok := cacheKey.(string)
				if !ok {
					panic(fmt.Sprintf("Unable to cast key %v to string upon eviction", cacheKey))
				}
				value, ok := cacheValue.(Resource)
				if !ok {
					panic(fmt.Sprintf("Unable to cast value %v to resource upon eviction", cacheValue))
				}
				onEvicted(key, value)
			},
		},
		// Duration before which an item is evicted for expiring. Zero means no expiration time.
		ttl:    ttl,
		logger: logger.Named("cache"),
		scope:  scope.SubScope(metrics.ScopeCache),
	}, nil
}

func (c *cache) GetReadOnlyCache() ReadOnlyCache {
	return c
}

func (c *cache) FetchReadOnly(key string) (Resource, error) {
	resource, err := c.Fetch(key)
	if resource == nil {
		return Resource{}, err
	}
	return *resource, err
}

func (c *cache) Fetch(key string) (*Resource, error) {
	c.cacheMu.RLock()
	metrics.CacheFetchSubscope(c.scope, key).Counter(metrics.CacheFetchAttempt).Inc(1)
	value, found := c.cache.Get(key)
	c.cacheMu.RUnlock()
	if !found {
		metrics.CacheFetchSubscope(c.scope, key).Counter(metrics.CacheFetchMiss).Inc(1)
		return nil, fmt.Errorf("no value found for key: %s", key)
	}
	resource, ok := value.(Resource)
	if !ok {
		metrics.CacheFetchSubscope(c.scope, key).Counter(metrics.CacheFetchError).Inc(1)
		return nil, fmt.Errorf("unable to cast cache value to type resource for key: %s", key)
	}
	// Lazy eviction based on TTL occurs here. Fetch does not increase the lifespan of the key.
	if resource.isExpired(time.Now()) {
		c.cacheMu.Lock()
		defer c.cacheMu.Unlock()
		value, found = c.cache.Get(key)
		if !found {
			// The entry was already evicted.
			return nil, nil
		}
		resource, ok = value.(Resource)
		if !ok {
			metrics.CacheFetchSubscope(c.scope, key).Counter(metrics.CacheFetchError).Inc(1)
			return nil, fmt.Errorf("unable to cast cache value to type resource for key: %s", key)
		}
		// This second check for expiration is required in case a recent SetResponse call was made to the same key
		// from another goroutine, extending the deadline for eviction. Without it, a key that was recently refreshed
		// may be prematurely removed by the goroutine calling Fetch.
		if resource.isExpired(time.Now()) {
			metrics.CacheFetchSubscope(c.scope, key).Counter(metrics.CacheFetchExpired).Inc(1)
			c.cache.Remove(key)
			return nil, nil
		}
	}
	if resource.Resp != nil {
		c.logger.With(
			"aggregated_key", key,
			"response_version", resource.Resp.GetPayloadVersion(),
			"response_type", resource.Resp.GetTypeURL(),
		).Debug(context.Background(), "fetch")
	}

	return &resource, nil
}

func (c *cache) SetResponse(key string, response transport.Response) (*RequestsStore, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	metrics.CacheSetSubscope(c.scope, key).Counter(metrics.CacheSetAttempt).Inc(1)
	value, found := c.cache.Get(key)
	if !found {
		resource := Resource{
			Resp:           response,
			ExpirationTime: c.getExpirationTime(time.Now()),
			Requests:       NewRequestsStore(),
		}
		c.cache.Add(key, resource)
		metrics.CacheSetSubscope(c.scope, key).Counter(metrics.CacheSetSuccess).Inc(1)
		c.logger.With(
			"aggregated_key", key,
			"response_type", response.GetTypeURL(),
		).Debug(context.Background(), "set response")
		return resource.Requests, nil
	}
	resource, ok := value.(Resource)
	if !ok {
		metrics.CacheSetSubscope(c.scope, key).Counter(metrics.CacheSetError).Inc(1)
		return nil, fmt.Errorf("unable to cast cache value to type resource for key: %s", key)
	}
	resource.Resp = response
	resource.ExpirationTime = c.getExpirationTime(time.Now())
	c.cache.Add(key, resource)
	c.logger.With(
		"aggregated_key", key, "response_type", response.GetTypeURL(),
	).Debug(context.Background(), "set response")
	metrics.CacheSetSubscope(c.scope, key).Counter(metrics.CacheSetSuccess).Inc(1)
	return resource.Requests, nil
}

func (c *cache) AddRequest(key string, req transport.Request) error {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	metrics.CacheAddRequestSubscope(c.scope, key).Counter(metrics.CacheAddAttempt).Inc(1)
	value, found := c.cache.Get(key)
	if !found {
		requests := NewRequestsStore()
		requests.Set(req)
		resource := Resource{
			Requests:       requests,
			ExpirationTime: c.getExpirationTime(time.Now()),
		}
		c.cache.Add(key, resource)
		c.logger.With(
			"aggregated_key", key,
			"node_id", req.GetNodeID(),
			"request_type", req.GetTypeURL(),
		).Debug(context.Background(), "request added")
		metrics.CacheAddRequestSubscope(c.scope, key).Counter(metrics.CacheAddSuccess).Inc(1)
		return nil
	}
	resource, ok := value.(Resource)
	if !ok {
		metrics.CacheAddRequestSubscope(c.scope, key).Counter(metrics.CacheAddError).Inc(1)
		return fmt.Errorf("unable to cast cache value to type resource for key: %s", key)
	}

	resource.Requests.Set(req)
	c.cache.Add(key, resource)
	c.logger.With(
		"aggregated_key", key,
		"node_id", req.GetNodeID(),
		"request_type", req.GetTypeURL(),
	).Debug(context.Background(), "request added")
	metrics.CacheAddRequestSubscope(c.scope, key).Counter(metrics.CacheAddSuccess).Inc(1)
	return nil
}

func (c *cache) DeleteRequest(key string, req transport.Request) error {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	metrics.CacheAddRequestSubscope(c.scope, key).Counter(metrics.CacheDeleteAttempt).Inc(1)
	value, found := c.cache.Get(key)
	if !found {
		return nil
	}
	resource, ok := value.(Resource)
	if !ok {
		metrics.CacheAddRequestSubscope(c.scope, key).Counter(metrics.CacheDeleteError).Inc(1)
		return fmt.Errorf("unable to cast cache value to type resource for key: %s", key)
	}
	resource.Requests.Delete(req)
	c.cache.Add(key, resource)
	c.logger.With(
		"aggregated_key", key,
		"node_id", req.GetNodeID(),
		"request_type", req.GetTypeURL(),
	).Debug(context.Background(), "request deleted")
	metrics.CacheAddRequestSubscope(c.scope, key).Counter(metrics.CacheDeleteSuccess).Inc(1)
	return nil
}

func (r *Resource) isExpired(currentTime time.Time) bool {
	if r.ExpirationTime.IsZero() {
		return false
	}
	return r.ExpirationTime.Before(currentTime)
}

func (c *cache) getExpirationTime(currentTime time.Time) time.Time {
	if c.ttl > 0 {
		return currentTime.Add(c.ttl)
	}
	return time.Time{}
}
