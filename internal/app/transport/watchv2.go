package transport

import (
	"fmt"
	"sync"

	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/uber-go/tally"

	"github.com/envoyproxy/xds-relay/internal/app/metrics"
)

var _ Watch = &watchV2{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV2 struct {
	out    chan gcpv2.Response
	mu     sync.RWMutex
	closed bool

	scope tally.Scope
}

// newWatchV2 creates a new watch object
func newWatchV2(scope tally.Scope) Watch {
	return &watchV2{
		out:   make(chan gcpv2.Response, 1),
		scope: scope,
	}
}

// Close closes the communication with the xds client
func (w *watchV2) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	close(w.out)
}

// GetChannelV2 gets the v2 channel used for communication with the xds client
func (w *watchV2) GetChannel() *ChannelVersion {
	return &ChannelVersion{V2: w.out}
}

// Send sends the xds response over wire
func (w *watchV2) Send(s Response) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.closed {
		return nil
	}

	// Drop the older response currently in the transport queue, so that we
	// always send the latest response. Normally, we should hit the default
	// case (channel is empty), but there are times when the response fanout
	// is slower than receiving of a new upstream response.
	select {
	case <-w.out:
		w.scope.Counter(metrics.OrchestratorWatchDequeued).Inc(1)
	default:
	}

	select {
	case w.out <- &gcpv2.PassthroughResponse{DiscoveryResponse: s.Get().V2, Request: s.GetRequest().V2}:
		return nil
	default:
		// sanity check, should never happen because of the dequeue above.
		return fmt.Errorf("channel is blocked")
	}
}
