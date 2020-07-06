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

// delete removes the response channel and request entry from the map.
// Note: We don't close the response channel prior to deletion because there
// can be separate go routines that are still attempting to write to the
// channel. We rely on garbage collection to clean up and close outstanding
// response channels once the go routines finish writing to them.
func (d *downstreamResponseMap) delete(req *gcp.Request) chan gcp.Response {
	d.mu.Lock()
	defer d.mu.Unlock()
	if channel, ok := d.responseChannels[req]; ok {
		delete(d.responseChannels, req)
		return channel
	}
	return nil
}

// deleteAll removes all response channels and request entries from the map.
// Note: We don't close the response channel prior to deletion because there
// can be separate go routines that are still attempting to write to the
// channel. We rely on garbage collection to clean up and close outstanding
// response channels once the go routines finish writing to them.
func (d *downstreamResponseMap) deleteAll(watchers map[*gcp.Request]bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for watch := range watchers {
		if d.responseChannels[watch] != nil {
			delete(d.responseChannels, watch)
		}
	}
}

func (d *downstreamResponseMap) keys(m *mapper.Mapper) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var keys []string
	for request := range d.responseChannels {
		key, err := mapper.Mapper.GetKey(*m, *request)
		if err != nil {
			// TODO(lisalu): Log warning.
		} else {
			keys = append(keys, key)
		}
	}
	return keys
}
