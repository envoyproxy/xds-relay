package cache

import (
	"fmt"
	"sync"
	"time"

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
	cache   map[string]*resource
	ttl     time.Duration
}

type resource struct {
	resp       *v2.DiscoveryResponse
	requests   []*v2.DiscoveryRequest
	streamOpen bool
}

func NewCache(ttl time.Duration) Cache {
	return &cache{
		cache: make(map[string]*resource),
		ttl:   ttl,
	}
}

func (c *cache) Fetch(key string) (*v2.DiscoveryResponse, error) {
	value, found := c.cache[key]
	if !found {
		return nil, fmt.Errorf("no value found for key: %s", key)
	}
	return value.resp, nil
}

func (c *cache) SetResponse(key string, resp v2.DiscoveryResponse) ([]*v2.DiscoveryRequest, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	value, found := c.cache[key]
	if !found {
		c.cache[key] = &resource{
			resp: &resp,
		}
		return nil, nil
	}
	value.resp = &resp
	c.cache[key] = value
	// TODO: Add logic that allows for notifying of watches.
	return value.requests, nil
}

func (c *cache) AddRequest(key string, req v2.DiscoveryRequest) (bool, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	value, found := c.cache[key]
	if !found {
		// TODO: Add logic to guarantee that a stream has been opened.
		c.cache[key] = &resource{
			requests:   []*v2.DiscoveryRequest{&req},
			streamOpen: true,
		}
		return true, nil
	}
	value.requests = append(value.requests, &req)
	// TODO: Add logic to guarantee that a stream has been opened.
	value.streamOpen = true
	c.cache[key] = value
	return true, nil
}
