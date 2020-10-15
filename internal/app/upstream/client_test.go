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
	"google.golang.org/genproto/googleapis/rpc/status"
)

type CallOptions = upstream.CallOptions

func TestMain(m *testing.M) {
	defer goleak.VerifyTestMain(m)
}

func TestOpenStreamShouldReturnErrorForInvalidTypeUrl(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := createMockClient(ctx)

	respCh, done := client.OpenStream(transport.NewRequestV2(&v2.DiscoveryRequest{}))
	defer done()
	_, ok := <-respCh
	assert.False(t, ok)
}

func TestOpenStreamShouldReturnErrorForInvalidTypeUrlV3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := createMockClientV3(ctx)

	respCh, done := client.OpenStream(transport.NewRequestV3(&discoveryv3.DiscoveryRequest{}))
	defer done()
	_, ok := <-respCh
	assert.False(t, ok)
}

func TestOpenStreamShouldRetryOnStreamCreationFailure(t *testing.T) {
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
			assert.NotNil(t, respCh)
			for {
				if v, ok := scope.Snapshot().Counters()[stats[0]]; ok && v.Value() == 1 {
					break
				}
			}
			for {
				if v, ok := scope.Snapshot().Counters()[stats[1]]; ok && v.Value() != 0 {
					break
				}
			}
			done()
			blockUntilClean(respCh, func() {})
		})
	}
}

func TestOpenStreamShouldRetryOnStreamCreationFailureV3(t *testing.T) {
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
			assert.NotNil(t, respCh)
			for {
				if v, ok := scope.Snapshot().Counters()[stats[0]]; ok && v.Value() == 1 {
					break
				}
			}
			for {
				if v, ok := scope.Snapshot().Counters()[stats[1]]; ok && v.Value() != 0 {
					break
				}
			}
			done()
			blockUntilClean(respCh, func() {})
		})
	}
}

func TestOpenStreamShouldReturnNonEmptyResponseChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := createMockClient(ctx)

	respCh, done := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	assert.NotNil(t, respCh)

	done()
	cancel()
	blockUntilClean(respCh, func() {})
}

func TestOpenStreamShouldReturnNonEmptyResponseChannelV3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := createMockClientV3(ctx)

	respCh, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	assert.NotNil(t, respCh)

	done()
	cancel()
	blockUntilClean(respCh, func() {})
}

func TestOpenStreamShouldSendTheFirstRequestToOriginServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
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
	resp, done := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    node,
		}))
	<-wait
	assert.NotNil(t, message)
	assert.Equal(t, message.GetNode(), node)
	assert.Equal(t, message.GetTypeUrl(), resource.ListenerType)

	done()
	cancel()
	blockUntilClean(resp, func() {})
}

func TestOpenStreamShouldSendTheFirstRequestToOriginServerV3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
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
	resp, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    node,
		}))
	<-wait
	assert.NotNil(t, message)
	assert.Equal(t, message.GetNode(), node)
	assert.Equal(t, message.GetTypeUrl(), resourcev3.ListenerType)

	done()
	cancel()
	blockUntilClean(resp, func() {})
}

func TestOpenStreamShouldClearNackFromRequestInTheFirstRequestToOriginServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
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
	resp, done := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl:     resource.ListenerType,
			Node:        node,
			ErrorDetail: &status.Status{Message: "message", Code: 1},
		}))
	<-wait
	assert.NotNil(t, message)
	assert.Equal(t, message.GetNode(), node)
	assert.Equal(t, message.GetTypeUrl(), resource.ListenerType)
	assert.Nil(t, message.GetErrorDetail())

	done()
	cancel()
	blockUntilClean(resp, func() {})
}

func TestOpenStreamShouldClearNackFromRequestInTheFirstRequestToOriginServerV3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
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
	resp, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl:     resourcev3.ListenerType,
			Node:        node,
			ErrorDetail: &status.Status{Message: "message", Code: 1},
		}))
	<-wait
	assert.NotNil(t, message)
	assert.Equal(t, message.GetNode(), node)
	assert.Equal(t, message.GetTypeUrl(), resourcev3.ListenerType)
	assert.Nil(t, message.GetErrorDetail())

	done()
	cancel()
	blockUntilClean(resp, func() {})
}

func TestOpenStreamShouldRetryIfSendFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	sendError := fmt.Errorf("")
	errResp := true
	response := &v2.DiscoveryResponse{}
	client := createMockClientWithResponse(ctx, time.Second, responseChan, func(m interface{}) error {
		if errResp {
			errResp = false
			return sendError
		}
		select {
		case <-ctx.Done():
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

	done()
	cancel()
	blockUntilClean(resp, func() {
		close(responseChan)
	})
}

func TestOpenStreamShouldRetryIfSendFailsV3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	sendError := fmt.Errorf("")
	errResp := true
	response := &discoveryv3.DiscoveryResponse{}
	client := createMockClientWithResponseV3(ctx, time.Second, responseChan, func(m interface{}) error {
		if errResp {
			errResp = false
			return sendError
		}
		select {
		case <-ctx.Done():
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
	_, more := <-resp
	assert.True(t, more)

	done()
	cancel()
	blockUntilClean(resp, func() {
		close(responseChan)
	})
}

func TestOpenStreamShouldSendTheResponseOnTheChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	response := &v2.DiscoveryResponse{}
	client := createMockClientWithResponse(ctx, time.Second, responseChan, func(m interface{}) error {
		select {
		case <-ctx.Done():
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
	assert.NotNil(t, resp)
	val := <-resp
	assert.Equal(t, val.Get().V2, response)

	done()
	cancel()
	blockUntilClean(resp, func() {
		close(responseChan)
	})
}

func TestOpenStreamShouldSendTheResponseOnTheChannelV3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	response := &discoveryv3.DiscoveryResponse{}
	client := createMockClientWithResponseV3(ctx, time.Second, responseChan, func(m interface{}) error {
		select {
		case <-ctx.Done():
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
	assert.NotNil(t, resp)
	val := <-resp
	assert.Equal(t, val.Get().V3, response)

	done()
	cancel()
	blockUntilClean(resp, func() {
		close(responseChan)
	})
}

func TestOpenStreamShouldSendTheNextRequestWithUpdatedVersionAndNonce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	lastAppliedVersion := ""
	index := 0
	client := createMockClientWithResponse(ctx, time.Second, responseChan, func(m interface{}) error {
		select {
		case <-ctx.Done():
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
		select {
		case responseChan <- response:
		case <-ctx.Done():
			return nil
		}

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

	done()
	cancel()
	blockUntilClean(resp, func() {
		close(responseChan)
	})
}

func TestOpenStreamShouldSendTheNextRequestWithUpdatedVersionAndNonceV3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	lastAppliedVersion := ""
	index := 0
	client := createMockClientWithResponseV3(ctx, time.Second, responseChan, func(m interface{}) error {
		select {
		case <-ctx.Done():
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
		select {
		case responseChan <- response:
		case <-ctx.Done():
			return nil
		}
		return nil
	}, stats.NewMockScope("mock"))

	resp, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	assert.NotNil(t, resp)
	for i := 0; i < 5; i++ {
		val := <-resp
		assert.Equal(t, val.GetPayloadVersion(), strconv.Itoa(i))
		assert.Equal(t, val.GetNonce(), strconv.Itoa(i))
	}

	done()
	cancel()
	blockUntilClean(resp, func() {
		close(responseChan)
	})
}

func TestOpenStreamShouldRetryWhenSendMsgBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	first := true
	response2 := &v2.DiscoveryResponse{VersionInfo: "2"}
	client := createMockClientWithResponse(ctx, time.Nanosecond, responseChan, func(m interface{}) error {
		if first {
			first = false
			<-ctx.Done()
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			responseChan <- response2
			return nil
		}
	}, stats.NewMockScope("mock"))

	respCh, done := client.OpenStream(transport.NewRequestV2(&v2.DiscoveryRequest{
		TypeUrl: resource.ListenerType,
		Node:    &core.Node{},
	}))
	resp, ok := <-respCh
	assert.True(t, ok)
	assert.Equal(t, resp.Get().V2.VersionInfo, response2.VersionInfo)

	done()
	cancel()
	blockUntilClean(respCh, func() {})
}

func TestOpenStreamShouldRetryWhenSendMsgBlocksV3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	first := true
	response2 := &discoveryv3.DiscoveryResponse{VersionInfo: "2"}
	client := createMockClientWithResponseV3(ctx, time.Nanosecond, responseChan, func(m interface{}) error {
		if first {
			first = false
			<-ctx.Done()
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			responseChan <- response2
			return nil
		}
	}, stats.NewMockScope("mock"))

	respCh, done := client.OpenStream(transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
		TypeUrl: resourcev3.ListenerType,
		Node:    &corev3.Node{},
	}))
	resp, ok := <-respCh
	assert.True(t, ok)
	assert.Equal(t, response2.VersionInfo, resp.Get().V3.VersionInfo)

	done()
	cancel()
	blockUntilClean(respCh, func() {})
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

func blockUntilClean(resp <-chan transport.Response, tearDown func()) {
	for range resp {
	}

	tearDown()
}
