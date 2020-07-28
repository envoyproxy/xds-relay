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

	"github.com/envoyproxy/xds-relay/internal/app/mapper"

	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

// downstreamResponseMap is a map of downstream xDS client requests to response
// channels.
type downstreamResponseMap struct {
	mu               sync.RWMutex
	responseChannels map[*gcp.Request]chan gcp.Response
}

func newDownstreamResponseMap() downstreamResponseMap {
	return downstreamResponseMap{
		responseChannels: make(map[*gcp.Request]chan gcp.Response),
	}
}

// createChannel initializes a new channel for a request if it doesn't already
// exist.
func (d *downstreamResponseMap) createChannel(req *gcp.Request) chan gcp.Response {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.responseChannels[req]; !ok {
		d.responseChannels[req] = make(chan gcp.Response, 1)
	}
	return d.responseChannels[req]
}

// get retrieves the channel where responses are set for the specified request.
func (d *downstreamResponseMap) get(req *gcp.Request) (chan gcp.Response, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	channel, ok := d.responseChannels[req]
	return channel, ok
}

// delete removes the response channel and request entry from the map and
// closes the corresponding channel.
func (d *downstreamResponseMap) delete(req *gcp.Request) chan gcp.Response {
	d.mu.Lock()
	defer d.mu.Unlock()
	if channel, ok := d.responseChannels[req]; ok {
		close(channel)
		delete(d.responseChannels, req)
		return channel
	}
	return nil
}

// deleteAll removes all response channels and request entries from the map and
// closes the corresponding channels.
func (d *downstreamResponseMap) deleteAll(watchers map[*gcp.Request]bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for watch := range watchers {
		if channel, ok := d.responseChannels[watch]; ok {
			close(channel)
			delete(d.responseChannels, watch)
		}
	}
}

// getAggregatedKeys returns a list of aggregated keys for all requests in the downstream response map.
func (d *downstreamResponseMap) getAggregatedKeys(m *mapper.Mapper) (map[string]bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	// Since multiple requests can map to the same cache key, we use a map to ensure unique entries.
	keys := make(map[string]bool)
	for request := range d.responseChannels {
		key, err := mapper.Mapper.GetKey(*m, *request)
		if err != nil {
			return nil, err
		}
		keys[key] = true
	}
	return keys, nil
}
