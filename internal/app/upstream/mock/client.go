package upstream

import (
	"context"
	"fmt"
	"io"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// NewClient creates a mock implementation for testing
func NewClient(
	ctx context.Context,
	callOptions upstream.CallOptions,
	errorOnCreate error,
	receiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) upstream.Client {
	return upstream.NewMockClient(
		ctx,
		createMockLdsClient(errorOnCreate, receiveChan, sendCb),
		createMockRdsClient(errorOnCreate, receiveChan, sendCb),
		createMockEdsClient(errorOnCreate, receiveChan, sendCb),
		createMockCdsClient(errorOnCreate, receiveChan, sendCb),
		callOptions,
	)
}

func createMockLdsClient(
	errorOnCreate error,
	receiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) v2.ListenerDiscoveryServiceClient {
	return &mockClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockCdsClient(
	errorOnCreate error,
	receiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) v2.ClusterDiscoveryServiceClient {
	return &mockClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockRdsClient(
	errorOnCreate error,
	receiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) v2.RouteDiscoveryServiceClient {
	return &mockClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockEdsClient(
	errorOnCreate error,
	receiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) v2.EndpointDiscoveryServiceClient {
	return &mockClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

type mockClient struct {
	errorOnStreamCreate error
	receiveChan         chan *v2.DiscoveryResponse
	sendCb              func(m interface{}) error
}

type mockGrpcStream struct {
	ctx         context.Context
	receiveChan chan *v2.DiscoveryResponse
	sendCb      func(m interface{}) error
}

func (stream *mockGrpcStream) SendMsg(m interface{}) error {
	return stream.sendCb(m)
}

func (stream *mockGrpcStream) RecvMsg(m interface{}) error {
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

func (stream *mockGrpcStream) Send(*v2.DiscoveryRequest) error {
	return fmt.Errorf("Not implemented")
}

func (stream *mockGrpcStream) Recv() (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (stream *mockGrpcStream) Header() (metadata.MD, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (stream *mockGrpcStream) Trailer() metadata.MD {
	return nil
}

func (stream *mockGrpcStream) CloseSend() error {
	return nil
}

func (stream *mockGrpcStream) Context() context.Context {
	return stream.ctx
}

func (c *mockClient) StreamListeners(
	ctx context.Context,
	opts ...grpc.CallOption) (v2.ListenerDiscoveryService_StreamListenersClient, error) {
	if c.errorOnStreamCreate != nil {
		return nil, c.errorOnStreamCreate
	}
	return &mockGrpcStream{ctx: ctx, receiveChan: c.receiveChan, sendCb: c.sendCb}, nil
}

func (c *mockClient) StreamClusters(
	ctx context.Context,
	opts ...grpc.CallOption) (v2.ClusterDiscoveryService_StreamClustersClient, error) {
	if c.errorOnStreamCreate != nil {
		return nil, c.errorOnStreamCreate
	}
	return &mockGrpcStream{ctx: ctx, receiveChan: c.receiveChan, sendCb: c.sendCb}, nil
}

func (c *mockClient) StreamRoutes(
	ctx context.Context,
	opts ...grpc.CallOption) (v2.RouteDiscoveryService_StreamRoutesClient, error) {
	if c.errorOnStreamCreate != nil {
		return nil, c.errorOnStreamCreate
	}
	return &mockGrpcStream{ctx: ctx, receiveChan: c.receiveChan, sendCb: c.sendCb}, nil
}

func (c *mockClient) StreamEndpoints(
	ctx context.Context,
	opts ...grpc.CallOption) (v2.EndpointDiscoveryService_StreamEndpointsClient, error) {
	if c.errorOnStreamCreate != nil {
		return nil, c.errorOnStreamCreate
	}
	return &mockGrpcStream{ctx: ctx, receiveChan: c.receiveChan, sendCb: c.sendCb}, nil
}

func (c *mockClient) DeltaListeners(
	ctx context.Context,
	opts ...grpc.CallOption) (v2.ListenerDiscoveryService_DeltaListenersClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClient) DeltaClusters(
	ctx context.Context,
	opts ...grpc.CallOption) (v2.ClusterDiscoveryService_DeltaClustersClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClient) DeltaRoutes(
	ctx context.Context,
	opts ...grpc.CallOption) (v2.RouteDiscoveryService_DeltaRoutesClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClient) DeltaEndpoints(
	ctx context.Context,
	opts ...grpc.CallOption) (v2.EndpointDiscoveryService_DeltaEndpointsClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClient) FetchListeners(
	ctx context.Context,
	in *v2.DiscoveryRequest,
	opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClient) FetchClusters(
	ctx context.Context,
	in *v2.DiscoveryRequest,
	opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClient) FetchRoutes(
	ctx context.Context,
	in *v2.DiscoveryRequest,
	opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *mockClient) FetchEndpoints(
	ctx context.Context,
	in *v2.DiscoveryRequest,
	opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}
