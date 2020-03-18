package multiplexer

import (
	"context"
	"fmt"

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
	conn      *grpc.ClientConn
	stream    grpc.ClientStream
	cdsClient v2.ClusterDiscoveryServiceClient
	ldsClient v2.ListenerDiscoveryServiceClient
	typeURL   string
}

// NewMux ...
func NewMux(ctx context.Context, conn *grpc.ClientConn, typeURL string) (Multiplexer, error) {
	switch typeURL {
	case clusterTypeURL:
		client := v2.NewClusterDiscoveryServiceClient(conn)
		return &multiplexer{
			conn:      conn,
			cdsClient: client,
		}
	case listenerTypeURL:
		client := v2.NewListenerDiscoveryServiceClient(conn)
		return &multiplexer{
			conn:      conn,
			ldsClient: client,
		}
	}
	return nil, fmt.Errorf("incorrect typeUrl")
}

func (m *multiplexer) QueueDiscoveryRequest(
	ctx context.Context,
	requestChan chan v2.DiscoveryRequest,
	responseChan chan *v2.DiscoveryResponse) {
	var stream grpc.ClientStream
	switch m.typeURL {
	case clusterTypeURL:
		stream, err := m.cdsClient.StreamClusters(ctx)
	case listenerTypeURL:
		stream, err := m.ldsClient.StreamListeners(ctx)
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
