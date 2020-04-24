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
		d.responseChannels[req] = make(chan gcp.Response)
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

// delete closes the response channel and removes it from the map.
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

// deleteAll closes all the response channels for the provided requests and
// removes them from the map.
func (d *downstreamResponseMap) deleteAll(watches map[*gcp.Request]bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for watch := range watches {
		if d.responseChannels[watch] != nil {
			close(d.responseChannels[watch])
			delete(d.responseChannels, watch)
		}
	}
}
