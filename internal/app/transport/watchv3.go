package transport

import (
	"fmt"
	"sync"

	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

var _ Watch = &watchV3{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV3 struct {
	out    chan<- gcpv3.Response
	mu     sync.RWMutex
	closed bool
}

// NewWatchV3 creates a new watch object
func NewWatchV3(resp chan<- gcpv3.Response) Watch {
	return &watchV3{
		out: resp,
	}
}

// Close closes the communication with the xds client
func (w *watchV3) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	w.out <- nil
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
		return fmt.Errorf("channel is blocked")
	}
}
