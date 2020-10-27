package transport

import (
	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

var _ Watch = &watchV2{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV2 struct {
	out chan gcpv2.Response
}

// newWatchV2 creates a new watch object
func newWatchV2() Watch {
	return &watchV2{
		out: make(chan gcpv2.Response, 1),
	}
}

// Close closes the communication with the xds client
func (w *watchV2) Close() {
	close(w.out)
}

// GetChannelV2 gets the v2 channel used for communication with the xds client
func (w *watchV2) GetChannel() *ChannelVersion {
	return &ChannelVersion{V2: w.out}
}

// Send sends the xds response over wire
func (w *watchV2) Send(s Response) bool {
	select {
	case w.out <- &gcpv2.PassthroughResponse{DiscoveryResponse: s.Get().V2, Request: s.GetRequest().V2}:
		return true
	default:
		return false
	}
}
