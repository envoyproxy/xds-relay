package transport

import (
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

var _ Watch = &watchV3{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV3 struct {
	out chan gcpv3.Response
}

// newWatchV2 creates a new watch object
func newWatchV3() Watch {
	return &watchV3{
		out: make(chan gcpv3.Response, 1),
	}
}

// Close closes the communication with the xds client
func (w *watchV3) Close() {
	close(w.out)
}

// GetChannelV2 gets the v2 channel used for communication with the xds client
func (w *watchV3) GetChannel() *ChannelVersion {
	return &ChannelVersion{V3: w.out}
}

// Send sends the xds response over wire
func (w *watchV3) Send(s Response) bool {
	select {
	case w.out <- gcpv3.PassthroughResponse{DiscoveryResponse: s.Get().V3, Request: *s.GetRequest().V3}:
		return true
	default:
		return false
	}
}
