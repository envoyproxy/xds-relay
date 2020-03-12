package cache

import (
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

type Cache interface {
	// Exists returns true if the key exists in the cache, false otherwise.
	Exists(key string) bool

	// Fetch returns the cached response if it exists.
	Fetch(key string) (*envoy_service_discovery_v3.DiscoveryResponse, error)

	// IsStreamOpen returns false if no stream is opened against the upstream control plane for the given key.
	IsStreamOpen(key string) (bool, error)

	// SetStreamOpen sets whether there is a stream open against the upstream control plane for the given key.
	SetStreamOpen(key string, open bool) error

	// SetResponse sets the cache response and returns the list of open watches.
	SetResponse(key string, resp envoy_service_discovery_v3.DiscoveryResponse) ([]*envoy_service_discovery_v3.DiscoveryRequest, error)

	// ClearWatches discards the existing watches.
	ClearWatches(key string) error

	// AddWatch adds the watch to the cache and returns whether a stream is open.
	AddWatch(key string, req envoy_service_discovery_v3.DiscoveryRequest) (bool, error)

	// IsWatchPresent returns whether the given request is watching for the resource corresponding to the given key.
	IsWatchPresent(key string, req envoy_service_discovery_v3.DiscoveryRequest) (bool, error)
}
