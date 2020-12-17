package transport

import (
	"fmt"
	"sync"

	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
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
func newWatchV3(scope tally.Scope) Watch {
	return &watchV3{
		out:   make(chan gcpv3.Response, 1),
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

// GetChannelV2 gets the v2 channel used for communication with the xds client
func (w *watchV3) GetChannel() *ChannelVersion {
	return &ChannelVersion{V3: w.out}
}

// Send sends the xds response over wire
func (w *watchV3) Send(s Response) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.closed {
		return nil
	}

	select {
	case w.out <- &gcpv3.PassthroughResponse{DiscoveryResponse: s.Get().V3, Request: s.GetRequest().V3}:
		return nil
	default:
		w.closed = true
		close(w.out)
		// sanity check, should never happen because of the dequeue above.
		return fmt.Errorf("channel is blocked")
	}
}
