package transport

import (
	"fmt"
	"sync"

	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

var _ Watch = &watchV2{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV2 struct {
	out  chan<- gcpv2.Response
	mu   sync.Mutex
	done bool
}

// NewWatchV2 creates a new watch object
func NewWatchV2(resp chan<- gcpv2.Response) Watch {
	return &watchV2{
		out: resp,
	}
}

// Send sends the xds response over wire
func (w *watchV2) Send(s Response) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.done {
		return nil
	}
	var response gcpv2.Response
	if s != nil {
		response = &gcpv2.PassthroughResponse{DiscoveryResponse: s.Get().V2, Request: s.GetRequest().V2}
	} else {
		close(w.out)
		return nil
	}

	select {
	case w.out <- response:
		w.done = true
		return nil
	default:
		close(w.out)
		return fmt.Errorf("channel is blocked")
	}
}
