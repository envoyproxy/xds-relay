package cache

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/dgraph-io/ristretto"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type Cache interface {
	// Exists returns true if the key exists in the cache, false otherwise.
	Exists(key string) bool

	// Fetch returns the cached response if it exists.
	Fetch(key string) (*envoy_api_v2.DiscoveryResponse, error)

	// SetResponse sets the cache response and returns the list of requests.
	SetResponse(key string, resp envoy_api_v2.DiscoveryResponse) ([]*envoy_api_v2.DiscoveryRequest, error)

	// AddRequest adds the request to the cache and returns whether a stream is open.
	AddRequest(key string, req envoy_api_v2.DiscoveryRequest) (bool, error)
}

type cache struct {
	cacheMu       sync.RWMutex
	cache         *ristretto.Cache
	expireSeconds time.Duration
}

type resource struct {
	resp       *envoy_api_v2.DiscoveryResponse
	requests   []*envoy_api_v2.DiscoveryRequest
	streamOpen bool
}

// Callback function for each eviction. Receives the key hash, conflict hash, cache value, and cost when called.
type onEvictFunc func(key uint64, conflict uint64, value interface{}, cost int64)

func NewCache(numCounters int64, cacheSizeBytes int64, expireSeconds int, onEvict onEvictFunc) (Cache, error) {
	// Config values are set as recommended in the ristretto documentation: https://github.com/dgraph-io/ristretto#Config.
	config := ristretto.Config{
		// NumCounters sets the number of counters/keys to keep for tracking access frequency.
		// The value is recommended to be higher than the max cache capacity,
		// and 10x the number of unique items in the cache when full (note that this is different from MaxCost).
		NumCounters: numCounters,
		// MaxCost denotes the max size in bytes and determines how eviction decisions are made.
		// For consistency, the cost passed into SetWithTTL represents the size of the value in bytes.
		MaxCost: cacheSizeBytes,
		// BufferItems is the size of the Get buffers. The recommended value is 64.
		BufferItems: 64,
		// OnEvict is called for each eviction and closes the stream if a key is removed (e.g. expiry due to TTL).
		OnEvict: onEvict,
	}
	newCache, err := ristretto.NewCache(&config)
	if err != nil {
		return nil, err
	}
	return &cache{
		cache:         newCache,
		expireSeconds: time.Duration(expireSeconds) * time.Second,
	}, nil
}

func (c *cache) Exists(key string) bool {
	_, found := c.cache.Get(key)
	return found
}

func (c *cache) Fetch(key string) (*envoy_api_v2.DiscoveryResponse, error) {
	value, found := c.cache.Get(key)
	if !found {
		return nil, fmt.Errorf("No value found for key: %s", key)
	}
	resource, ok := value.(resource)
	if !ok {
		return nil, fmt.Errorf("Unable to cast cache value to type resource for key: %s", key)
	}
	return resource.resp, nil
}

func (c *cache) SetResponse(key string, resp envoy_api_v2.DiscoveryResponse) ([]*envoy_api_v2.DiscoveryRequest, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	value, found := c.cache.Get(key)
	if !found {
		// If no value exists for the key, instantiate a new one.
		resource := resource{
			resp: &resp,
		}
		cost := unsafe.Sizeof(resource)
		set := c.cache.SetWithTTL(key, resource, int64(cost), c.expireSeconds)
		if !set {
			return nil, fmt.Errorf("Unable to set value for key: %s", key)
		}
		return nil, nil
	}
	resource, ok := value.(resource)
	if !ok {
		return nil, fmt.Errorf("Unable to cast cache value to type resource for key: %s", key)
	}
	resource.resp = &resp
	cost := unsafe.Sizeof(resource)
	set := c.cache.SetWithTTL(key, resource, int64(cost), c.expireSeconds)
	// TODO: Add logic that allows for notifying of watches.
	if !set {
		return resource.requests, fmt.Errorf("Unable to set value for key: %s", key)
	}
	return resource.requests, nil
}

func (c *cache) AddRequest(key string, req envoy_api_v2.DiscoveryRequest) (bool, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	value, found := c.cache.Get(key)
	if !found {
		// If no value exists for the key, instantiate a new one.
		resource := resource{
			requests:   []*envoy_api_v2.DiscoveryRequest{&req},
			streamOpen: true,
		}
		cost := unsafe.Sizeof(resource)
		set := c.cache.SetWithTTL(key, resource, int64(cost), c.expireSeconds)
		if !set {
			return false, fmt.Errorf("Unable to set value for key: %s", key)
		}
		return true, nil
	}
	resource, ok := value.(resource)
	if !ok {
		return false, fmt.Errorf("Unable to cast cache value to type resource for key: %s", key)
	}
	resource.requests = append(resource.requests, &req)
	// TODO: Add logic to guarantee that a stream has been opened.
	resource.streamOpen = true
	cost := unsafe.Sizeof(resource)
	set := c.cache.SetWithTTL(key, resource, int64(cost), c.expireSeconds)
	if !set {
		return false, fmt.Errorf("Unable to set value for key: %s", key)
	}
	return true, nil
}
