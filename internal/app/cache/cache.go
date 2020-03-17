package cache

import (
	"fmt"
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

	// SetResponse sets the cache response and returns the list of open watches.
	SetResponse(key string, resp envoy_api_v2.DiscoveryResponse) ([]*envoy_api_v2.DiscoveryRequest, error)

	// AddWatch adds the watch to the cache and returns whether a stream is open.
	AddWatch(key string, req envoy_api_v2.DiscoveryRequest) (bool, error)
}

type cache struct {
	cache         *ristretto.Cache
	expireSeconds time.Duration
}

type resource struct {
	resp       *envoy_api_v2.DiscoveryResponse
	watches    []*envoy_api_v2.DiscoveryRequest
	streamOpen bool
}

func newCache(cacheSizeBytes int64, expireSeconds int) (*cache, error) {
	config := ristretto.Config{
		NumCounters: 10 * cacheSizeBytes,
		MaxCost:     cacheSizeBytes,
		BufferItems: 64,
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
	resource.watches = nil
	cost := unsafe.Sizeof(resource)
	set := c.cache.SetWithTTL(key, resource, int64(cost), c.expireSeconds)
	if !set {
		return nil, fmt.Errorf("Unable to set value for key: %s", key)
	}
	return nil, nil
}

func (c *cache) AddWatch(key string, req envoy_api_v2.DiscoveryRequest) (bool, error) {
	value, found := c.cache.Get(key)
	if !found {
		// If no value exists for the key, instantiate a new one.
		resource := resource{
			watches:    []*envoy_api_v2.DiscoveryRequest{&req},
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
	resource.watches = append(resource.watches, &req)
	resource.streamOpen = true
	cost := unsafe.Sizeof(resource)
	set := c.cache.SetWithTTL(key, resource, int64(cost), c.expireSeconds)
	if !set {
		return false, fmt.Errorf("Unable to set value for key: %s", key)
	}
	return true, nil
}
