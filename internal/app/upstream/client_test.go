package upstream

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestOpenStreamShouldReturnErrorForInvalidTypeUrl(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	client := GetMockClient(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		nil,
		responseChan,
		func(m interface{}) error { return nil })

	respCh, err := client.OpenStream(ctx, nil)
	assert.NotNil(t, err)
	assert.Nil(t, respCh)

	respCh, err = client.OpenStream(ctx, &v2.DiscoveryRequest{})
	assert.NotNil(t, err)
	_, ok := err.(*UnsupportedResourceError)
	assert.True(t, ok)
	assert.Nil(t, respCh)

	cancel()
}

func TestOpenStreamShouldResturnErrorOnStreamCreationFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	client := GetMockClient(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		fmt.Errorf("error"),
		responseChan,
		func(m interface{}) error { return nil })

	for _, typeURL := range []string{listenerTypeURL, clusterTypeURL, routeTypeURL, endpointTypeURL} {
		t.Run(typeURL, func(t *testing.T) {
			respCh, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
				TypeUrl: typeURL,
				Node:    &core.Node{},
			})
			assert.Nil(t, respCh)
			assert.NotNil(t, err)
		})
	}

	cancel()
}

func TestOpenStreamShouldReturnNonEmptyResponseChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	client := GetMockClient(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		nil,
		responseChan,
		func(m interface{}) error { return nil })

	respCh, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: listenerTypeURL,
		Node:    &core.Node{},
	})
	assert.NotNil(t, respCh)
	assert.Nil(t, err)
	cancel()
}

func TestOpenStreamShouldSendTheFirstRequestToOriginServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var message *v2.DiscoveryRequest
	responseChan := make(chan *v2.DiscoveryResponse)
	wait := make(chan bool)
	client := GetMockClient(ctx, CallOptions{Timeout: time.Nanosecond}, nil, responseChan, func(m interface{}) error {
		message = m.(*v2.DiscoveryRequest)
		wait <- true
		return nil
	})

	node := &core.Node{}
	_, _ = client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: listenerTypeURL,
		Node:    node,
	})
	<-wait
	assert.NotNil(t, message)
	assert.Equal(t, message.GetNode(), node)
	assert.Equal(t, message.TypeUrl, listenerTypeURL)
	cancel()
}

func TestOpenStreamShouldSendErrorIfSendFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	responseChan := make(chan *v2.DiscoveryResponse)
	sendError := fmt.Errorf("")
	client := GetMockClient(ctx, CallOptions{Timeout: time.Second}, nil, responseChan, func(m interface{}) error {
		return sendError
	})

	resp, _ := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: listenerTypeURL,
		Node:    &core.Node{},
	})
	val := <-resp
	assert.Equal(t, val.Err, sendError)
	assert.Nil(t, val.Response)
	cancel()
}

func TestOpenStreamShouldSendTheResponseOnTheChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	responseChan := make(chan *v2.DiscoveryResponse)
	response := &v2.DiscoveryResponse{}
	client := GetMockClient(ctx, CallOptions{Timeout: time.Second}, nil, responseChan, func(m interface{}) error {
		responseChan <- response
		return nil
	})

	resp, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: listenerTypeURL,
		Node:    &core.Node{},
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	val := <-resp
	assert.Equal(t, val.Response, response)
	assert.Nil(t, val.Err)
	cancel()
}

func TestOpenStreamShouldSendTheNextRequestWithUpdatedVersionAndNonce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	responseChan := make(chan *v2.DiscoveryResponse)
	lastAppliedVersion := ""
	index := 0
	client := GetMockClient(ctx, CallOptions{Timeout: time.Second}, nil, responseChan, func(m interface{}) error {
		message := m.(*v2.DiscoveryRequest)

		assert.Equal(t, message.GetVersionInfo(), lastAppliedVersion)
		assert.Equal(t, message.GetResponseNonce(), lastAppliedVersion)

		response := &v2.DiscoveryResponse{
			VersionInfo: strconv.Itoa(index),
			Nonce:       strconv.Itoa(index),
			TypeUrl:     listenerTypeURL,
		}
		lastAppliedVersion = strconv.Itoa(index)
		index++
		responseChan <- response
		return nil
	})

	resp, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: listenerTypeURL,
		Node:    &core.Node{},
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	for i := 0; i < 5; i++ {
		val := <-resp
		assert.Equal(t, val.Response.GetVersionInfo(), strconv.Itoa(i))
		assert.Equal(t, val.Response.GetNonce(), strconv.Itoa(i))
		assert.Nil(t, val.Err)
	}

	cancel()
}

func TestOpenStreamShouldSendErrorWhenSendMsgBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	responseChan := make(chan *v2.DiscoveryResponse)
	client := GetMockClient(ctx, CallOptions{Timeout: time.Nanosecond}, nil, responseChan, func(m interface{}) error {
		<-ctx.Done()
		return nil
	})

	resp, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: listenerTypeURL,
		Node:    &core.Node{},
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	val := <-resp
	assert.Equal(t, val.Err.Error(), "rpc error: code = DeadlineExceeded desc = timeout")
	assert.Nil(t, val.Response)
	cancel()
}

func GetMockClient(
	ctx context.Context,
	callOptions CallOptions,
	errorOnCreate error,
	receiveChan chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) Client {
	return NewMockClient(
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
