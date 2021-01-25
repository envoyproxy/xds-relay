package upstream

import (
	"context"
	"fmt"
	"io"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockClientV3 struct {
	errorOnStreamCreate []error
	receiveChan         chan *v2.DiscoveryResponse
	sendCb              func(m interface{}) error
}

type mockGrpcStreamV3 struct {
	ctx         context.Context
	receiveChan chan *v2.DiscoveryResponse
	sendCb      func(m interface{}) error
}

func (stream *mockGrpcStreamV3) SendMsg(m interface{}) error {
	select {
	case <-stream.ctx.Done():
		return nil
	default:
		return stream.sendCb(m)
	}
}

func (stream *mockGrpcStreamV3) RecvMsg(m interface{}) error {
	for {
		select {
		// https://github.com/grpc/grpc-go/issues/1894#issuecomment-370487012
		case <-stream.ctx.Done():
			return io.EOF
		case resp := <-stream.receiveChan:
			message := m.(*v2.DiscoveryResponse)
			message.VersionInfo = resp.GetVersionInfo()
			message.Nonce = resp.GetNonce()
			message.TypeUrl = resp.GetTypeUrl()
			message.Resources = resp.GetResources()
			return nil
		}
	}
}

func (stream *mockGrpcStreamV3) Send(*v2.DiscoveryRequest) error {
	return fmt.Errorf("Not implemented")
}

func (stream *mockGrpcStreamV3) Recv() (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (stream *mockGrpcStreamV3) Header() (metadata.MD, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (stream *mockGrpcStreamV3) Trailer() metadata.MD {
	return nil
}

func (stream *mockGrpcStreamV3) CloseSend() error {
	return nil
}

func (stream *mockGrpcStreamV3) Context() context.Context {
	return stream.ctx
}

func (c *mockClientV3) StreamListeners(
	ctx context.Context,
	opts ...grpc.CallOption) (listenerservice.ListenerDiscoveryService_StreamListenersClient, error) {
	if c.errorOnStreamCreate != nil && len(c.errorOnStreamCreate) != 0 {
		e := c.errorOnStreamCreate[0]
		c.errorOnStreamCreate = c.errorOnStreamCreate[1:]
		return nil, e
	}
	return &mockGrpcStreamV3{ctx: ctx, receiveChan: c.receiveChan, sendCb: c.sendCb}, nil
}

func (c *mockClientV3) StreamClusters(
	ctx context.Context,
	opts ...grpc.CallOption) (clusterservice.ClusterDiscoveryService_StreamClustersClient, error) {
	if c.errorOnStreamCreate != nil && len(c.errorOnStreamCreate) != 0 {
		e := c.errorOnStreamCreate[0]
		c.errorOnStreamCreate = c.errorOnStreamCreate[1:]
		return nil, e
	}
	return &mockGrpcStreamV3{ctx: ctx, receiveChan: c.receiveChan, sendCb: c.sendCb}, nil
}

func (c *mockClientV3) StreamRoutes(
	ctx context.Context,
	opts ...grpc.CallOption) (routeservice.RouteDiscoveryService_StreamRoutesClient, error) {
	if c.errorOnStreamCreate != nil && len(c.errorOnStreamCreate) != 0 {
		e := c.errorOnStreamCreate[0]
		c.errorOnStreamCreate = c.errorOnStreamCreate[1:]
		return nil, e
	}
	return &mockGrpcStreamV3{ctx: ctx, receiveChan: c.receiveChan, sendCb: c.sendCb}, nil
}

func (c *mockClientV3) StreamEndpoints(
	ctx context.Context,
	opts ...grpc.CallOption) (endpointservice.EndpointDiscoveryService_StreamEndpointsClient, error) {
	if c.errorOnStreamCreate != nil && len(c.errorOnStreamCreate) != 0 {
		e := c.errorOnStreamCreate[0]
		c.errorOnStreamCreate = c.errorOnStreamCreate[1:]
		return nil, e
	}
	return &mockGrpcStreamV3{ctx: ctx, receiveChan: c.receiveChan, sendCb: c.sendCb}, nil
}

func (c *mockClientV3) DeltaListeners(
	ctx context.Context,
	opts ...grpc.CallOption) (listenerservice.ListenerDiscoveryService_DeltaListenersClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClientV3) DeltaClusters(
	ctx context.Context,
	opts ...grpc.CallOption) (clusterservice.ClusterDiscoveryService_DeltaClustersClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClientV3) DeltaRoutes(
	ctx context.Context,
	opts ...grpc.CallOption) (routeservice.RouteDiscoveryService_DeltaRoutesClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClientV3) DeltaEndpoints(
	ctx context.Context,
	opts ...grpc.CallOption) (endpointservice.EndpointDiscoveryService_DeltaEndpointsClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClientV3) FetchListeners(
	ctx context.Context,
	in *v2.DiscoveryRequest,
	opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClientV3) FetchClusters(
	ctx context.Context,
	in *v2.DiscoveryRequest,
	opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClientV3) FetchRoutes(
	ctx context.Context,
	in *v2.DiscoveryRequest,
	opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClientV3) FetchEndpoints(
	ctx context.Context,
	in *v2.DiscoveryRequest,
	opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}
