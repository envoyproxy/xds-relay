package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type Cache interface {
	// Fetch returns the cached response if it exists.
	Fetch(key string) (*v2.DiscoveryResponse, error)

	// SetResponse sets the cache response and returns the list of requests.
	SetResponse(key string, resp v2.DiscoveryResponse) ([]*v2.DiscoveryRequest, error)

	// AddRequest adds the request to the cache and returns whether a stream is open.
	AddRequest(key string, req v2.DiscoveryRequest) (bool, error)
}

type cache struct {
	cacheMu sync.RWMutex
	cache   lru.Cache
	ttl     time.Duration
}

type resource struct {
	resp       *v2.DiscoveryResponse
	requests   []*v2.DiscoveryRequest
	streamOpen bool
}

// Callback function for each eviction. Receives the key and cache value when called.
type onEvictFunc func(key lru.Key, value interface{})

func NewCache(maxEntries int, onEvicted onEvictFunc, ttl time.Duration) Cache {
	return &cache{
		cache: lru.Cache{
			// Max number of cache entries before an item is evicted. Zero means no limit.
			MaxEntries: maxEntries,
			// OnEvict is called for each eviction and closes the stream if a key is removed
			// (e.g. expiry due to TTL, too many entries).
			OnEvicted: onEvicted,
		},
		// Duration before which an item is evicted for expiring. Zero means no expiration time.
		ttl: ttl,
	}
}

func (c *cache) Fetch(key string) (*v2.DiscoveryResponse, error) {
	value, found := c.cache.Get(key)
	if !found {
		return nil, fmt.Errorf("no value found for key: %s", key)
	}
	resource, ok := value.(resource)
	if !ok {
		return nil, fmt.Errorf("unable to cast cache value to type resource for key: %s", key)
	}
	return resource.resp, nil
}

func (c *cache) SetResponse(key string, resp v2.DiscoveryResponse) ([]*v2.DiscoveryRequest, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	value, found := c.cache.Get(key)
	if !found {
		resource := resource{
			resp: &resp,
		}
		c.cache.Add(key, resource)
		return nil, nil
	}
	resource, ok := value.(resource)
	if !ok {
		return nil, fmt.Errorf("unable to cast cache value to type resource for key: %s", key)
	}
	resource.resp = &resp
	c.cache.Add(key, resource)
	// TODO: Add logic that allows for notifying of watches.
	return resource.requests, nil
}

func (c *cache) AddRequest(key string, req v2.DiscoveryRequest) (bool, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	value, found := c.cache.Get(key)
	if !found {
		// TODO: Add logic to guarantee that a stream has been opened.
		resource := resource{
			requests:   []*v2.DiscoveryRequest{&req},
			streamOpen: true,
		}
		c.cache.Add(key, resource)
		return true, nil
	}
	resource, ok := value.(resource)
	if !ok {
		return false, fmt.Errorf("unable to cast cache value to type resource for key: %s", key)
	}
	resource.requests = append(resource.requests, &req)
	// TODO: Add logic to guarantee that a stream has been opened.
	resource.streamOpen = true
	c.cache.Add(key, resource)
	return true, nil
}
