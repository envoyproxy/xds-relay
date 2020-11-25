package transport

import (
	"fmt"

	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

var _ Watch = &watchV2{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV2 struct {
	out       chan<- gcpv2.Response
	tombstone bool
}

// NewWatchV2 creates a new watch object
func NewWatchV2(resp chan<- gcpv2.Response) Watch {
	return &watchV2{
		out: resp,
	}
}

func (w *watchV2) Close() {
	w.tombstone = true
}

// Send sends the xds response over wire
func (w *watchV2) Send(s Response) error {
	if w.tombstone {
		return nil
	}

	select {
	case w.out <- &gcpv2.PassthroughResponse{DiscoveryResponse: s.Get().V2, Request: s.GetRequest().V2}:
		return nil
	default:
		return fmt.Errorf("channel is blocked")
	}
}
