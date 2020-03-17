package multiplexer

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"google.golang.org/grpc"
)

const (
	clusterTypeURL  = "type.googleapis.com/envoy.api.v2.Cluster"
	listenerTypeURL = "type.googleapis.com/envoy.api.v2.Listener"
)

// Multiplexer layer
type Multiplexer interface {
	QueueDiscoveryRequest(context.Context, chan v2.DiscoveryRequest, chan *v2.DiscoveryResponse)
}

type multiplexer struct {
	conn   *grpc.ClientConn
	stream grpc.ClientStream
}

// NewMux ...
func NewMux(ctx context.Context, conn *grpc.ClientConn, typeURL string) Multiplexer {
	return &multiplexer{
		conn: conn,
	}
}

func (m *multiplexer) QueueDiscoveryRequest(
	ctx context.Context,
	requestChan chan v2.DiscoveryRequest,
	responseChan chan *v2.DiscoveryResponse) {
	typeURL := req.GetTypeUrl()
	var stream grpc.ClientStream
	switch typeURL {
	case clusterTypeURL:
		stream, err := v2.NewClusterDiscoveryServiceClient(conn).StreamClusters(ctx)
	case listenerTypeURL:
		stream, err := v2.NewListenerDiscoveryServiceClient(conn).StreamListeners(ctx)
	}

	go func() {
		for {
			select {
			case ctx.Done():
				return
			case req := <-requestChan:
				if err := stream.SendMsg(&req); err != nil {
					stream.CloseSend()
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case ctx.Done():
				return
			default:
				resp, err := stream.Recv()
			}
			if err != nil {
				return
			}
			responseChan <- resp
		}
	}()
}

func getStream() {

}
