package orchestrator

import (
	"sync"

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
func (d *downstreamResponseMap) deleteAll(watches map[*gcp.Request]bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for watch := range watches {
		if d.responseChannels[watch] != nil {
			delete(d.responseChannels, watch)
		}
	}
}
