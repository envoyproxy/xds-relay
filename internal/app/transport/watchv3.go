package transport

import (
	"fmt"
	"sync"

	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/xds-relay/internal/app/metrics"
	"github.com/uber-go/tally"
)

var _ Watch = &watchV3{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV3 struct {
	out    chan gcpv3.Response
	mu     sync.RWMutex
	closed bool

	scope tally.Scope
}

// newWatchV2 creates a new watch object
func NewWatchV3(out chan gcpv3.Response, scope tally.Scope) Watch {
	return &watchV3{
		out:   out,
		scope: scope,
	}
}

// Close closes the communication with the xds client
func (w *watchV3) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	close(w.out)
}

// Send sends the xds response over wire
func (w *watchV3) Send(s Response) error {
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
	case w.out <- &gcpv3.PassthroughResponse{DiscoveryResponse: s.Get().V3, Request: s.GetRequest().V3}:
		return nil
	default:
		// sanity check, should never happen because of the dequeue above.
		return fmt.Errorf("channel is blocked")
	}
}
