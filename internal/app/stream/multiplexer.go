package multiplexer

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// Multiplexer layer
type Multiplexer interface {
	QueueDiscoveryRequest(context.Context, chan *v2.DiscoveryRequest, chan *v2.DiscoveryResponse)
}
