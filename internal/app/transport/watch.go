package transport

import (
	"fmt"

	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

// Watch interface abstracts v2 and v3 watches
type Watch interface {
	Close()
	GetChannel() interface{}
	Send(s interface{}) (bool, error)
}

var _ Watch = &watchV2{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type watchV2 struct {
	Req *gcpv2.Request
	out chan gcpv2.Response
}

// NewWatchV2 creates a new watch object
func NewWatchV2(req *gcpv2.Request) Watch {
	return &watchV2{
		Req: req,
		out: make(chan gcpv2.Response, 1),
	}
}

// Close closes the communication with the xds client
func (w *watchV2) Close() {
	close(w.out)
}

// GetChannelV2 gets the v2 channel used for communication with the xds client
func (w *watchV2) GetChannel() interface{} {
	return w.out
}

// Send sends the xds response over wire
func (w *watchV2) Send(s interface{}) (bool, error) {
	resp, ok := s.(gcpv2.Response)
	if !ok {
		return false, fmt.Errorf("payload %s could not be casted to DiscoveryResponse", s)
	}
	select {
	case w.out <- resp:
		return true, nil
	default:
		return false, nil
	}
}
