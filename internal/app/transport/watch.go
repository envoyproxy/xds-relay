package transport

import (
	"fmt"

	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

// Watch interface abstracts v2 and v3 watches
type Watch interface {
	Close()
	GetChannelV2() chan gcp.Response
	GetChannelV3() chan gcpv3.Response
	Send(s interface{}) (bool, error)
}

var _ Watch = &WatchV2{}

// WatchV2 is the transport object that takes care of send responses to the xds clients
type WatchV2 struct {
	Req *gcp.Request
	out chan gcp.Response
}

// NewWatchV2 creates a new watch object
func NewWatchV2(req *gcp.Request) Watch {
	return &WatchV2{
		Req: req,
		out: make(chan gcp.Response, 1),
	}
}

// Close closes the communication with the xds client
func (w *WatchV2) Close() {
	close(w.out)
}

// GetChannelV2 gets the v2 channel used for communication with the xds client
func (w *WatchV2) GetChannelV2() chan gcp.Response {
	return w.out
}

// GetChannelV3 gets the v3 channel used for communication with the xds client
func (w *WatchV2) GetChannelV3() chan gcpv3.Response {
	return nil
}

// Send sends the xds response over wire
func (w *WatchV2) Send(s interface{}) (bool, error) {
	resp, ok := s.(gcp.Response)
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
