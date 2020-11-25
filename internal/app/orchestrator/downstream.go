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
	"github.com/envoyproxy/xds-relay/internal/app/transport"
)

// downstreamResponseMap is a map of downstream xDS client requests to response
// channels.
type downstreamResponseMap struct {
	mu      sync.RWMutex
	watches map[string][]transport.Watch
}

func newDownstreamResponseMap() downstreamResponseMap {
	return downstreamResponseMap{
		watches: make(map[string][]transport.Watch),
	}
}

// createWatch initializes a new channel for a request if it doesn't already
// exist.
func (d *downstreamResponseMap) addWatch(aggregatedKey string, w transport.Watch) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.watches[aggregatedKey]; !ok {
		d.watches[aggregatedKey] = []transport.Watch{w}
		return
	}
	d.watches[aggregatedKey] = append(d.watches[aggregatedKey], w)
}

// get retrieves the channel where responses are set for the specified request.
func (d *downstreamResponseMap) getSnapshot(aggregatedKey string) ([]transport.Watch, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	watches, ok := d.watches[aggregatedKey]
	if ok {
		d.watches[aggregatedKey] = []transport.Watch{}
	}

	return watches, ok
}

// get retrieves the channel where responses are set for the specified request.
func (d *downstreamResponseMap) get(aggregatedKey string) []transport.Watch {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.watches[aggregatedKey]
}

// delete removes the response channel and request entry from the map and
// closes the corresponding channel.
func (d *downstreamResponseMap) delete(w transport.Watch) {
	w.Close()
}

// deleteAll removes all response channels and request entries from the map and
// closes the corresponding channels.
func (d *downstreamResponseMap) deleteAll(aggregatedKey string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.watches, aggregatedKey)
}

// getAggregatedKeys returns a list of aggregated keys for all requests in the downstream response map.
func (d *downstreamResponseMap) getAggregatedKeys(m *mapper.Mapper) (map[string]bool, error) {
	return nil, nil
}
