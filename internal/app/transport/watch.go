package transport

import (
	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// ChannelVersion wraps v2 and v3 response channels
type ChannelVersion struct {
	V2 chan gcpv2.Response
	V3 chan gcpv3.Response
}

// Watch interface abstracts v2 and v3 watches
type Watch interface {
	Close()
	GetChannel() *ChannelVersion
	Send(Response) (bool, error)
}

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
func (w *watchV2) Send(s Response) (bool, error) {
	select {
	case w.out <- gcpv2.PassthroughResponse{DiscoveryResponse: s.Get().V2, Request: *s.GetRequest().V2}:
		return true, nil
	default:
		return false, nil
	}
}

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
func (w *watchV3) Send(s Response) (bool, error) {
	select {
	case w.out <- gcpv3.PassthroughResponse{DiscoveryResponse: s.Get().V3, Request: *s.GetRequest().V3}:
		return true, nil
	default:
		return false, nil
	}
}
