// Package orchestrator is responsible for instrumenting inbound xDS client
// requests to the correct aggregated key, forwarding a representative request
// to the upstream origin server, and managing the lifecycle of downstream and
// upstream connections and associates streams. It implements
// go-control-plane's Cache interface in order to receive xDS-based requests,
// send responses, and handle gRPC streams.
//
// This file manages the bookkeeping of downstream clients by tracking inbound
// requests to their corresponding response channels. The contents of this file
// are intended to only be used within the orchestrator module and should not
// be exported.
package orchestrator

import (
	"sync"

	"github.com/envoyproxy/xds-relay/internal/app/cache"
	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/uber-go/tally"
)

// downstreamResponseMap is a map of downstream xDS client requests to response
// channels.
type downstreamResponseMap struct {
	mu      sync.RWMutex
	watches map[transport.Request]transport.Watch
}

func newDownstreamResponseMap() downstreamResponseMap {
	return downstreamResponseMap{
		watches: make(map[transport.Request]transport.Watch),
	}
}

// createWatch initializes a new channel for a request if it doesn't already
// exist.
func (d *downstreamResponseMap) createWatch(req transport.Request, w transport.Watch, scope tally.Scope) transport.Watch {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.watches[req]; !ok {
		d.watches[req] = w
	}
	return d.watches[req]
}

// get retrieves the channel where responses are set for the specified request.
func (d *downstreamResponseMap) get(req transport.Request) (transport.Watch, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	channel, ok := d.watches[req]
	return channel, ok
}

// delete removes the response channel and request entry from the map and
// closes the corresponding channel.
func (d *downstreamResponseMap) delete(req transport.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if watch, ok := d.watches[req]; ok {
		// wait for all writes to the responseChannel to complete before closing.
		watch.Close()
		delete(d.watches, req)
	}
}

// deleteAll removes all response channels and request entries from the map and
// closes the corresponding channels.
func (d *downstreamResponseMap) deleteAll(watchers *cache.RequestsStore) {
	d.mu.Lock()
	defer d.mu.Unlock()
	watchers.ForEach(func(watch transport.Request) {
		if w, ok := d.watches[watch]; ok {
			// wait for all writes to the responseChannel to complete before closing.
			w.Close()
			delete(d.watches, watch)
		}
	})
}

// getAggregatedKeys returns a list of aggregated keys for all requests in the downstream response map.
func (d *downstreamResponseMap) getAggregatedKeys(m *mapper.Mapper) (map[string]bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	// Since multiple requests can map to the same cache key, we use a map to ensure unique entries.
	keys := make(map[string]bool)
	for request := range d.watches {
		key, err := mapper.Mapper.GetKey(*m, request)
		if err != nil {
			return nil, err
		}
		keys[key] = true
	}
	return keys, nil
}
