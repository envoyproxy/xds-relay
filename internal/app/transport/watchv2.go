package transport

import (
	"fmt"
	"sync"

	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

var _ Watch = &watchV2{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV2 struct {
	out       chan<- gcpv2.Response
	tombstone bool
	mu        sync.Mutex
}

// NewWatchV2 creates a new watch object
func NewWatchV2(resp chan<- gcpv2.Response) Watch {
	return &watchV2{
		out: resp,
	}
}

func (w *watchV2) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tombstone = true
	w.out = nil
}

// Send sends the xds response over wire
func (w *watchV2) Send(s Response) error {
	w.mu.Lock()
	if w.tombstone {
		return nil
	}
	w.mu.Unlock()

	var response gcpv2.Response
	if s != nil {
		response = &gcpv2.PassthroughResponse{DiscoveryResponse: s.Get().V2, Request: s.GetRequest().V2}
	}
	select {
	case w.out <- response:
		w.mu.Lock()
		w.tombstone = true
		w.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("channel is blocked")
	}
}

func (w *watchV2) IsClosed() bool {
	return w.tombstone
}
