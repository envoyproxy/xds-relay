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
	"context"
	"sync"
	"time"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"k8s.io/client-go/util/workqueue"
)

// downstreamResponseMap is a map of downstream xDS client requests to response
// channels.
type downstreamResponseMap struct {
	mu      sync.RWMutex
	watches map[string][]transport.Watch
	wq      workqueue.RateLimitingInterface
}

func newDownstreamResponseMap(ctx context.Context) *downstreamResponseMap {
	d := &downstreamResponseMap{
		watches: make(map[string][]transport.Watch),
		wq: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(time.Second, time.Second),
			"watchermanager"),
	}

	go cleanup(d)
	go func() {
		<-ctx.Done()
		d.wq.ShutDown()
	}()
	return d
}

func cleanup(d *downstreamResponseMap) {
	for {
		item, shutdown := d.wq.Get()
		if shutdown {
			return
		}
		aggregatedKey := item.(string)
		if watches, ok := d.watches[aggregatedKey]; ok {
			cpy := make([]transport.Watch, 0)
			d.mu.Lock()
			for _, w := range watches {
				if !w.IsClosed() {
					cpy = append(cpy, w)
				}
			}
			d.watches[aggregatedKey] = cpy
			d.mu.Unlock()
		}

		d.wq.Done(item)
	}
}

// createWatch initializes a new channel for a request if it doesn't already
// exist.
func (d *downstreamResponseMap) createWatch(req transport.Request, aggregatedKey string) transport.Watch {
	d.mu.Lock()
	defer d.mu.Unlock()
	w := req.CreateWatch()
	if _, ok := d.watches[aggregatedKey]; !ok {
		d.watches[aggregatedKey] = make([]transport.Watch, 0)
	}

	d.watches[aggregatedKey] = append(d.watches[aggregatedKey], w)
	return w
}

// get retrieves the channel where responses are set for the specified request.
func (d *downstreamResponseMap) get(aggregatedKey string) ([]transport.Watch, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	channel, ok := d.watches[aggregatedKey]
	return channel, ok
}

// delete removes the response channel and request entry from the map and
// closes the corresponding channel.
func (d *downstreamResponseMap) delete(w transport.Watch, aggregatedKey string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if watch, ok := d.watches[aggregatedKey]; ok {
		// wait for all writes to the responseChannel to complete before closing.
		for _, w := range watch {
			w.Close()
		}
		d.wq.AddRateLimited(aggregatedKey)
	}
}

// deleteAll removes all response channels and request entries from the map and
// closes the corresponding channels.
func (d *downstreamResponseMap) deleteAll(aggregatedKey string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, watch := range d.watches[aggregatedKey] {
		watch.Close()
	}
	delete(d.watches, aggregatedKey)
}

// getAggregatedKeys returns a list of aggregated keys for all requests in the downstream response map.
func (d *downstreamResponseMap) getAggregatedKeys(m *mapper.Mapper) (map[string]bool, error) {
	return nil, nil
}
