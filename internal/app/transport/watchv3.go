package transport

import (
	"fmt"
	"sync"

	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

var _ Watch = &watchV3{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV3 struct {
	out       chan<- gcpv3.Response
	tombstone bool
	mu        sync.Mutex
}

// NewWatchV3 creates a new watch object
func NewWatchV3(resp chan<- gcpv3.Response) Watch {
	return &watchV3{
		out: resp,
	}
}

func (w *watchV3) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tombstone = true
}

// Send sends the xds response over wire
func (w *watchV3) Send(s Response) error {
	w.mu.Lock()
	if w.tombstone {
		return nil
	}
	w.mu.Unlock()

	select {
	case w.out <- &gcpv3.PassthroughResponse{DiscoveryResponse: s.Get().V3, Request: s.GetRequest().V3}:
		return nil
	default:
		return fmt.Errorf("channel is blocked")
	}
}
