package upstream

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"google.golang.org/grpc"
)

// NewClient creates a grpc connection with an upstream origin server.
// Each xds relay server should create a single such upstream connection.
// grpc will handle the actual number of underlying tcp connections.
//
// The method does not block until the underlying connection is up.
// Returns immediately and connecting the server happens in background
// TODO: Security
// A connection is created with no transport security https://grpc.io/docs/guides/auth/
//
func NewUpstream(ctx context.Context, url string) (XdsClient, error) {
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	ldsClient := v2.NewListenerDiscoveryServiceClient(conn)
	rdsClient := v2.NewRouteDiscoveryServiceClient(conn)
	edsClient := v2.NewEndpointDiscoveryServiceClient(conn)
	cdsClient := v2.NewClusterDiscoveryServiceClient(conn)

	return &xdsClient{
		ldsClient: ldsClient,
		rdsClient: rdsClient,
		edsClient: edsClient,
		cdsClient: cdsClient,
	}

	go shutDown(ctx, conn)
}

func shutDown(ctx context.Context, conn *grpc.ClientConn) {
	for {
		select {
		case <- ctx.Done():
			conn.Close()
		}
	}
}

type XdsClient interface {
	Request(context.Context, *v2.DiscoveryRequest, chan *Response, typeURL string) error
}

type xdsClient struct {
	ldsClient v2.ListenerDiscoveryServiceClient
	rdsClient v2.RouteDiscoveryServiceClient
	edsClient v2.EndpointDiscoveryServiceClient
	cdsClient v2.ClusterDiscoveryServiceClient
}

func (m *xdsClient) Request(context.Context, request chan *v2.DiscoveryRequest, response chan *Response, typeURL string) error {
	var stream grpc.Stream
	switch typeURL{
	case "type.googleapis.com/envoy.api.v2.Listener":
		stream, err := m.ldsClient.StreamListeners(ctx)
	case "type.googleapis.com/envoy.api.v2.Cluster":
		stream, err := m.cdsClient.StreamClusters(ctx)
	case "type.googleapis.com/envoy.api.v2.RouteConfiguration":
		stream, err := m.rdsClient.StreamRoutes(ctx)
	case "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment":
		stream, err := m.edsClient.StreamEndpoints(ctx)
	}
	
	go Send(ctx, request, stream)
	go Recv(ctx, response, stream)
	return nil
}


func Send(ctx context.Context, request chan *v2.DiscoveryRequest, stream grpc.Stream) {
	for {
		select{
			req := <- request
			err := stream.SendMsg(&req)
			if err != nil {
				
			}
		}
	}
}

func Recv(ctx context.Context, response chan *v2.DiscoveryRequest, stream grpc.Stream) {
	for {
		select {
			resp, err := listener.Recv()
			response <- resp	
		}
	}
}
