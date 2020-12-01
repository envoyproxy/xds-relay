package transport

import (
	"fmt"

	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

var _ Watch = &watchV3{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV3 struct {
	out chan<- gcpv3.Response
}

// NewWatchV3 creates a new watch object
func NewWatchV3(resp chan<- gcpv3.Response) Watch {
	return &watchV3{
		out: resp,
	}
}

// Send sends the xds response over wire
func (w *watchV3) Send(s Response) error {
	var response gcpv3.Response
	if s != nil {
		response = &gcpv3.PassthroughResponse{DiscoveryResponse: s.Get().V3, Request: s.GetRequest().V3}
	}
	select {
	case w.out <- response:
		return nil
	default:
		return fmt.Errorf("channel is blocked")
	}
}
