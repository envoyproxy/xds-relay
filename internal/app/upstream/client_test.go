package upstream_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v2"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/stats"
	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
	"go.uber.org/goleak"
)

type CallOptions = upstream.CallOptions

func TestOpenStreamShouldReturnErrorForInvalidTypeUrl(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := createMockClient(ctx)

	respCh, done := client.OpenStream(transport.NewRequestV2(&v2.DiscoveryRequest{}))
	defer done()
	_, ok := <-respCh
	assert.False(t, ok)
}

func TestOpenStreamShouldReturnErrorForInvalidTypeUrlV3(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := createMockClientV3(ctx)

	respCh, done := client.OpenStream(transport.NewRequestV3(&discoveryv3.DiscoveryRequest{}))
	defer done()
	_, ok := <-respCh
	assert.False(t, ok)
}

func TestOpenStreamShouldRetryOnStreamCreationFailure(t *testing.T) {
	defer goleak.VerifyNone(t)
	scope := stats.NewMockScope("mock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := createMockClientWithError(ctx, scope)

	typeURLs := map[string][]string{
		resource.ListenerType: {"mock.lds.stream_failure+", "mock.lds.stream_opened+"},
		resource.ClusterType:  {"mock.cds.stream_failure+", "mock.cds.stream_opened+"},
		resource.RouteType:    {"mock.rds.stream_failure+", "mock.rds.stream_opened+"},
		resource.EndpointType: {"mock.eds.stream_failure+", "mock.eds.stream_opened+"},
	}
	for url, stats := range typeURLs {
		t.Run(url, func(t *testing.T) {
			respCh, done := client.OpenStream(
				transport.NewRequestV2(&v2.DiscoveryRequest{
					TypeUrl: url,
					Node:    &core.Node{},
				}))
			defer done()
			assert.NotNil(t, respCh)
			for {
				if v, ok := scope.Snapshot().Counters()[stats[0]]; ok {
					assert.Equal(t, int64(1), v.Value())
					break
				}
			}
			for {
				if v, ok := scope.Snapshot().Counters()[stats[1]]; ok {
					assert.NotEqual(t, int64(0), v.Value())
					break
				}
			}
		})
	}
}

func TestOpenStreamShouldNotRetryOnStreamCreationFailureV3(t *testing.T) {
	defer goleak.VerifyNone(t)
	scope := stats.NewMockScope("mock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := createMockClientWithErrorV3(ctx, scope)

	typeURLs := map[string][]string{
		resourcev3.ListenerType: {"mock.lds.stream_failure+", "mock.lds.stream_opened+"},
		resourcev3.ClusterType:  {"mock.cds.stream_failure+", "mock.cds.stream_opened+"},
		resourcev3.RouteType:    {"mock.rds.stream_failure+", "mock.rds.stream_opened+"},
		resourcev3.EndpointType: {"mock.eds.stream_failure+", "mock.eds.stream_opened+"},
	}
	for url, stats := range typeURLs {
		t.Run(url, func(t *testing.T) {
			respCh, done := client.OpenStream(
				transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
					TypeUrl: url,
					Node:    &corev3.Node{},
				}))
			defer done()
			assert.NotNil(t, respCh)
			for {
				if v, ok := scope.Snapshot().Counters()[stats[0]]; ok {
					assert.Equal(t, int64(1), v.Value())
					break
				}
			}
			for {
				if v, ok := scope.Snapshot().Counters()[stats[1]]; ok {
					assert.NotEqual(t, int64(0), v.Value())
					break
				}
			}
		})
	}
}

func TestOpenStreamShouldReturnNonEmptyResponseChannel(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := createMockClient(ctx)

	respCh, done := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	defer done()
	assert.NotNil(t, respCh)
}

func TestOpenStreamShouldReturnNonEmptyResponseChannelV3(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := createMockClientV3(ctx)

	respCh, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	defer done()
	assert.NotNil(t, respCh)
}

func TestOpenStreamShouldSendTheFirstRequestToOriginServer(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var message *v2.DiscoveryRequest
	responseChan := make(chan *v2.DiscoveryResponse)
	wait := make(chan bool)
	first := true
	client := upstream.NewMock(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		nil,
		responseChan,
		responseChan,
		responseChan,
		responseChan,
		func(m interface{}) error {
			message = m.(*v2.DiscoveryRequest)
			if first {
				close(wait)
				first = false
			}
			return nil
		},
		stats.NewMockScope("mock"),
	)

	node := &core.Node{}
	_, done := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    node,
		}))
	defer done()
	<-wait
	assert.NotNil(t, message)
	assert.Equal(t, message.GetNode(), node)
	assert.Equal(t, message.GetTypeUrl(), resource.ListenerType)
}

func TestOpenStreamShouldSendTheFirstRequestToOriginServerV3(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var message *discoveryv3.DiscoveryRequest
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	wait := make(chan bool)
	first := true
	client := upstream.NewMockV3(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		nil,
		responseChan,
		responseChan,
		responseChan,
		responseChan,
		func(m interface{}) error {
			message = m.(*discoveryv3.DiscoveryRequest)
			if first {
				close(wait)
				first = false
			}
			return nil
		},
		stats.NewMockScope("mock"),
	)

	node := &corev3.Node{}
	_, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    node,
		}))
	defer done()
	<-wait
	assert.NotNil(t, message)
	assert.Equal(t, message.GetNode(), node)
	assert.Equal(t, message.GetTypeUrl(), resourcev3.ListenerType)
}

func TestOpenStreamShouldRetryIfSendFails(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responseChan := make(chan *v2.DiscoveryResponse)
	sendError := fmt.Errorf("")
	errResp := true
	response := &v2.DiscoveryResponse{}
	exit := make(chan struct{})
	defer close(exit)
	client := createMockClientWithResponse(ctx, time.Second, responseChan, func(m interface{}) error {
		if errResp {
			errResp = false
			return sendError
		}
		select {
		case <-exit:
			close(responseChan)
			return nil
		default:
			responseChan <- response
			return nil
		}
	}, stats.NewMockScope("mock"))

	resp, done := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	defer done()
	_, more := <-resp
	assert.True(t, more)
}

func TestOpenStreamShouldRetryIfSendFailsV3(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	sendError := fmt.Errorf("")
	errResp := true
	response := &discoveryv3.DiscoveryResponse{}
	exit := make(chan struct{})
	defer close(exit)
	client := createMockClientWithResponseV3(ctx, time.Second, responseChan, func(m interface{}) error {
		if errResp {
			errResp = false
			return sendError
		}
		select {
		case <-exit:
			close(responseChan)
			return nil
		default:
			responseChan <- response
			return nil
		}
	}, stats.NewMockScope("mock"))

	resp, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	defer done()
	_, more := <-resp
	assert.True(t, more)
}

func TestOpenStreamShouldSendTheResponseOnTheChannel(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responseChan := make(chan *v2.DiscoveryResponse)
	response := &v2.DiscoveryResponse{}
	exit := make(chan struct{})
	defer close(exit)
	client := createMockClientWithResponse(ctx, time.Second, responseChan, func(m interface{}) error {
		select {
		case <-exit:
			close(responseChan)
			return nil
		default:
			responseChan <- response
			return nil
		}
	}, stats.NewMockScope("mock"))

	resp, done := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	defer done()
	assert.NotNil(t, resp)
	val := <-resp
	assert.Equal(t, val.Get().V2, response)
}

func TestOpenStreamShouldSendTheResponseOnTheChannelV3(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	response := &discoveryv3.DiscoveryResponse{}
	exit := make(chan struct{})
	defer close(exit)
	client := createMockClientWithResponseV3(ctx, time.Second, responseChan, func(m interface{}) error {
		select {
		case <-exit:
			close(responseChan)
			return nil
		default:
			responseChan <- response
			return nil
		}
	}, stats.NewMockScope("mock"))

	resp, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	defer done()
	assert.NotNil(t, resp)
	val := <-resp
	assert.Equal(t, val.Get().V3, response)
}

func TestOpenStreamShouldSendTheNextRequestWithUpdatedVersionAndNonce(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responseChan := make(chan *v2.DiscoveryResponse)
	lastAppliedVersion := ""
	index := 0
	exit := make(chan struct{})
	defer close(exit)
	client := createMockClientWithResponse(ctx, time.Second, responseChan, func(m interface{}) error {
		select {
		case <-exit:
			close(responseChan)
			return nil
		default:
		}
		message := m.(*v2.DiscoveryRequest)

		assert.Equal(t, message.GetVersionInfo(), lastAppliedVersion)
		assert.Equal(t, message.GetResponseNonce(), lastAppliedVersion)

		response := &v2.DiscoveryResponse{
			VersionInfo: strconv.Itoa(index),
			Nonce:       strconv.Itoa(index),
			TypeUrl:     resource.ListenerType,
		}
		lastAppliedVersion = strconv.Itoa(index)
		index++
		responseChan <- response
		return nil
	}, stats.NewMockScope("mock"))

	resp, done := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	defer done()
	assert.NotNil(t, resp)
	for i := 0; i < 5; i++ {
		val := <-resp
		assert.Equal(t, val.GetPayloadVersion(), strconv.Itoa(i))
		assert.Equal(t, val.GetNonce(), strconv.Itoa(i))
	}
}

func TestOpenStreamShouldSendTheNextRequestWithUpdatedVersionAndNonceV3(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	lastAppliedVersion := ""
	index := 0
	exit := make(chan struct{})
	defer close(exit)
	client := createMockClientWithResponseV3(ctx, time.Second, responseChan, func(m interface{}) error {
		select {
		case <-exit:
			close(responseChan)
			return nil
		default:
		}
		message := m.(*discoveryv3.DiscoveryRequest)

		assert.Equal(t, message.GetVersionInfo(), lastAppliedVersion)
		assert.Equal(t, message.GetResponseNonce(), lastAppliedVersion)

		response := &discoveryv3.DiscoveryResponse{
			VersionInfo: strconv.Itoa(index),
			Nonce:       strconv.Itoa(index),
			TypeUrl:     resource.ListenerType,
		}
		lastAppliedVersion = strconv.Itoa(index)
		index++
		responseChan <- response
		return nil
	}, stats.NewMockScope("mock"))

	resp, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	defer done()
	assert.NotNil(t, resp)
	for i := 0; i < 5; i++ {
		val := <-resp
		assert.Equal(t, val.GetPayloadVersion(), strconv.Itoa(i))
		assert.Equal(t, val.GetNonce(), strconv.Itoa(i))
	}
}

func createMockClient(ctx context.Context) upstream.Client {
	return upstream.NewMock(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		nil,
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"))
}

func createMockClientWithError(ctx context.Context, scope tally.Scope) upstream.Client {
	return upstream.NewMock(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		[]error{fmt.Errorf("error")},
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		func(m interface{}) error { return nil },
		scope)
}

func createMockClientWithResponse(
	ctx context.Context,
	t time.Duration,
	r chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error,
	scope tally.Scope) upstream.Client {
	return upstream.NewMock(ctx, CallOptions{Timeout: t}, nil, r, r, r, r, sendCb, scope)
}

func createMockClientV3(ctx context.Context) upstream.Client {
	return upstream.NewMockV3(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		nil,
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"))
}

func createMockClientWithErrorV3(ctx context.Context, scope tally.Scope) upstream.Client {
	return upstream.NewMockV3(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		[]error{fmt.Errorf("error")},
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		func(m interface{}) error { return nil },
		scope)
}

func createMockClientWithResponseV3(
	ctx context.Context,
	t time.Duration,
	r chan *discoveryv3.DiscoveryResponse,
	sendCb func(m interface{}) error,
	scope tally.Scope) upstream.Client {
	return upstream.NewMockV3(ctx, CallOptions{Timeout: t}, nil, r, r, r, r, sendCb, scope)
}
