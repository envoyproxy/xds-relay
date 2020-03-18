package multiplexer

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"google.golang.org/grpc"
)

// Multiplexer handles the connections to the upstream control plane
type Multiplexer interface {
	// QueueRequest starts a stream with the upstream control plane
	QueueRequest(context.Context, chan *v2.DiscoveryRequest, chan *v2.DiscoveryResponse)
}

type multiplexer struct {
	conn *grpc.ClientConn
}

// NewMux creates an instance based on the typeUrl of the resource
// A new instance of Multiplexer is recommended per xds type
// Eg. For eds requests of different services, create an instance each.
func NewMux(ctx context.Context, conn *grpc.ClientConn, typeURL string) (Multiplexer, error) {
	return &multiplexer{
		conn: conn,
	}, nil
}

func (m *multiplexer) QueueRequest(
	ctx context.Context,
	requestChan chan *v2.DiscoveryRequest,
	responseChan chan *v2.DiscoveryResponse) {

}
