package orchestrator

import (
	"sync"

	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

// downstreamResponseMap is a map of downstream xDS client requests to response
// channels.
type downstreamResponseMap struct {
	mu              sync.RWMutex
	responseChannel map[*gcp.Request]chan gcp.Response
}

// createChannel initializes a new channel for a request if it doesn't already
// exist.
func (d *downstreamResponseMap) createChannel(req *gcp.Request) chan gcp.Response {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.responseChannel[req]; !ok {
		d.responseChannel[req] = make(chan gcp.Response)
	}
	return d.responseChannel[req]
}

// get retrieves the channel where responses are set for the specified request.
func (d *downstreamResponseMap) get(req *gcp.Request) (chan gcp.Response, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	channel, ok := d.responseChannel[req]
	return channel, ok
}

// delete closes the response channel and removes it from the map.
func (d *downstreamResponseMap) delete(req *gcp.Request) chan gcp.Response {
	d.mu.Lock()
	defer d.mu.Unlock()
	if channel, ok := d.responseChannel[req]; ok {
		close(channel)
		delete(d.responseChannel, req)
		return channel
	}
	return nil
}

// deleteAll closes all the response channels for the provided requests and
// removes them from the map.
func (d *downstreamResponseMap) deleteAll(req []*gcp.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, watch := range req {
		if d.responseChannel[watch] != nil {
			close(d.responseChannel[watch])
			delete(d.responseChannel, watch)
		}
	}
}
