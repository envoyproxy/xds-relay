package upstream

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestOpenStreamShouldReturnErrorForInvalidTypeUrl(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	client := NewMockClient(ctx, CallOptions{Timeout: time.Nanosecond}, nil, responseChan, func(m interface{}) error { return nil })

	respCh, err := client.OpenStream(ctx, nil)
	assert.NotNil(t, err)
	assert.Nil(t, respCh)

	respCh, err = client.OpenStream(ctx, &v2.DiscoveryRequest{})
	assert.NotNil(t, err)
	assert.Nil(t, respCh)

	cancel()
}

func TestOpenStreamShouldResturnErrorOnStreamCreationFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	client := NewMockClient(ctx, CallOptions{Timeout: time.Nanosecond}, fmt.Errorf("error"), responseChan, func(m interface{}) error { return nil })

	for _, typeURL := range []string{listenerTypeURL, clusterTypeURL, routeTypeURL, endpointTypeURL} {
		t.Run(typeURL, func(t *testing.T) {
			respCh, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
				TypeUrl: typeURL,
				Node:    &envoy_api_v2_core.Node{},
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
	client := NewMockClient(ctx, CallOptions{Timeout: time.Nanosecond}, nil, responseChan, func(m interface{}) error { return nil })

	respCh, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: listenerTypeURL,
		Node:    &envoy_api_v2_core.Node{},
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
	client := NewMockClient(ctx, CallOptions{Timeout: time.Nanosecond}, nil, responseChan, func(m interface{}) error {
		message = m.(*v2.DiscoveryRequest)
		wait <- true
		return nil
	})

	node := &envoy_api_v2_core.Node{}
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
	client := NewMockClient(ctx, CallOptions{Timeout: time.Second}, nil, responseChan, func(m interface{}) error {
		return sendError
	})

	resp, _ := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: listenerTypeURL,
		Node:    &envoy_api_v2_core.Node{},
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
	client := NewMockClient(ctx, CallOptions{Timeout: time.Second}, nil, responseChan, func(m interface{}) error {
		responseChan <- response
		return nil
	})

	resp, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: listenerTypeURL,
		Node:    &envoy_api_v2_core.Node{},
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
	client := NewMockClient(ctx, CallOptions{Timeout: time.Second}, nil, responseChan, func(m interface{}) error {
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
		Node:    &envoy_api_v2_core.Node{},
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
	client := NewMockClient(ctx, CallOptions{Timeout: time.Nanosecond}, nil, responseChan, func(m interface{}) error {
		for {
			select {
			case _, ok := <-responseChan:
				if !ok {
					return nil
				}
			}
		}
	})

	resp, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: listenerTypeURL,
		Node:    &envoy_api_v2_core.Node{},
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	val := <-resp
	assert.Equal(t, val.Err.Error(), "rpc error: code = DeadlineExceeded desc = timeout")
	assert.Nil(t, val.Response)
	cancel()
}

func NewMockClient(ctx context.Context, callOptions CallOptions, errorOnCreate error, receiveChan chan *v2.DiscoveryResponse, sendCb func(m interface{}) error) Client {
	return &client{
		ldsClient:   createMockLdsClient(errorOnCreate, receiveChan, sendCb),
		rdsClient:   createMockRdsClient(errorOnCreate, receiveChan, sendCb),
		edsClient:   createMockEdsClient(errorOnCreate, receiveChan, sendCb),
		cdsClient:   createMockCdsClient(errorOnCreate, receiveChan, sendCb),
		callOptions: callOptions,
	}
}

func createMockLdsClient(errorOnCreate error, receiveChan chan *v2.DiscoveryResponse, sendCb func(m interface{}) error) v2.ListenerDiscoveryServiceClient {
	return &mockLdsClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockCdsClient(errorOnCreate error, receiveChan chan *v2.DiscoveryResponse, sendCb func(m interface{}) error) v2.ClusterDiscoveryServiceClient {
	return &mockLdsClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockRdsClient(errorOnCreate error, receiveChan chan *v2.DiscoveryResponse, sendCb func(m interface{}) error) v2.RouteDiscoveryServiceClient {
	return &mockLdsClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

func createMockEdsClient(errorOnCreate error, receiveChan chan *v2.DiscoveryResponse, sendCb func(m interface{}) error) v2.EndpointDiscoveryServiceClient {
	return &mockLdsClient{errorOnStreamCreate: errorOnCreate, receiveChan: receiveChan, sendCb: sendCb}
}

type mockLdsClient struct {
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

func (ldsClient *mockLdsClient) StreamListeners(ctx context.Context, opts ...grpc.CallOption) (v2.ListenerDiscoveryService_StreamListenersClient, error) {
	if ldsClient.errorOnStreamCreate != nil {
		return nil, ldsClient.errorOnStreamCreate
	}
	return &mockGrpcStream{ctx: ctx, receiveChan: ldsClient.receiveChan, sendCb: ldsClient.sendCb}, nil
}

func (ldsClient *mockLdsClient) StreamClusters(ctx context.Context, opts ...grpc.CallOption) (v2.ClusterDiscoveryService_StreamClustersClient, error) {
	if ldsClient.errorOnStreamCreate != nil {
		return nil, ldsClient.errorOnStreamCreate
	}
	return &mockGrpcStream{ctx: ctx, receiveChan: ldsClient.receiveChan, sendCb: ldsClient.sendCb}, nil
}

func (ldsClient *mockLdsClient) StreamRoutes(ctx context.Context, opts ...grpc.CallOption) (v2.RouteDiscoveryService_StreamRoutesClient, error) {
	if ldsClient.errorOnStreamCreate != nil {
		return nil, ldsClient.errorOnStreamCreate
	}
	return &mockGrpcStream{ctx: ctx, receiveChan: ldsClient.receiveChan, sendCb: ldsClient.sendCb}, nil
}

func (ldsClient *mockLdsClient) StreamEndpoints(ctx context.Context, opts ...grpc.CallOption) (v2.EndpointDiscoveryService_StreamEndpointsClient, error) {
	if ldsClient.errorOnStreamCreate != nil {
		return nil, ldsClient.errorOnStreamCreate
	}
	return &mockGrpcStream{ctx: ctx, receiveChan: ldsClient.receiveChan, sendCb: ldsClient.sendCb}, nil
}

func (ldsClient *mockLdsClient) DeltaListeners(ctx context.Context, opts ...grpc.CallOption) (v2.ListenerDiscoveryService_DeltaListenersClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (ldsClient *mockLdsClient) DeltaClusters(ctx context.Context, opts ...grpc.CallOption) (v2.ClusterDiscoveryService_DeltaClustersClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (ldsClient *mockLdsClient) DeltaRoutes(ctx context.Context, opts ...grpc.CallOption) (v2.RouteDiscoveryService_DeltaRoutesClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (ldsClient *mockLdsClient) DeltaEndpoints(ctx context.Context, opts ...grpc.CallOption) (v2.EndpointDiscoveryService_DeltaEndpointsClient, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (ldsClient *mockLdsClient) FetchListeners(ctx context.Context, in *v2.DiscoveryRequest, opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (ldsClient *mockLdsClient) FetchClusters(ctx context.Context, in *v2.DiscoveryRequest, opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (ldsClient *mockLdsClient) FetchRoutes(ctx context.Context, in *v2.DiscoveryRequest, opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (ldsClient *mockLdsClient) FetchEndpoints(ctx context.Context, in *v2.DiscoveryRequest, opts ...grpc.CallOption) (*v2.DiscoveryResponse, error) {
	return nil, fmt.Errorf("Not implemented")
}
