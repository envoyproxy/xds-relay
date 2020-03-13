package cache

import (
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type Cache interface {
	// Exists returns true if the key exists in the cache, false otherwise.
	Exists(key string) bool

	// Fetch returns the cached response if it exists.
	Fetch(key string) (*envoy_api_v2.DiscoveryResponse, error)

	// SetResponse sets the cache response and returns the list of open watches.
	SetResponse(key string, resp envoy_api_v2.DiscoveryResponse) ([]*envoy_api_v2.DiscoveryRequest, error)

	// ClearWatches discards the existing watches.
	ClearWatches(key string) error

	// AddWatch adds the watch to the cache and returns whether a stream is open.
	AddWatch(key string, req envoy_api_v2.DiscoveryRequest) (bool, error)

	// IsWatchPresent returns whether the given request is watching for the resource corresponding to the given key.
	IsWatchPresent(key string, req envoy_api_v2.DiscoveryRequest) (bool, error)
}
