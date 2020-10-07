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
)

type CallOptions = upstream.CallOptions

func TestOpenStreamShouldReturnErrorForInvalidTypeUrl(t *testing.T) {
	client := createMockClient()

	respCh, _ := client.OpenStream(transport.NewRequestV2(&v2.DiscoveryRequest{}))
	_, ok := <-respCh
	assert.False(t, ok)
}

func TestOpenStreamShouldReturnErrorForInvalidTypeUrlV3(t *testing.T) {
	client := createMockClientV3()

	respCh, _ := client.OpenStream(transport.NewRequestV3(&discoveryv3.DiscoveryRequest{}))
	_, ok := <-respCh
	assert.False(t, ok)
}

func TestOpenStreamShouldNotReturnErrorOnStreamCreationFailure(t *testing.T) {
	scope := stats.NewMockScope("mock")
	client := createMockClientWithError(scope)

	typeURLs := map[string][]string{
		resource.ListenerType: {"mock.lds.stream_failure+", "mock.lds.stream_opened+"},
		resource.ClusterType:  {"mock.cds.stream_failure+", "mock.cds.stream_opened+"},
		resource.RouteType:    {"mock.rds.stream_failure+", "mock.rds.stream_opened+"},
		resource.EndpointType: {"mock.eds.stream_failure+", "mock.eds.stream_opened+"},
	}
	for url, stats := range typeURLs {
		t.Run(url, func(t *testing.T) {
			respCh, _ := client.OpenStream(
				transport.NewRequestV2(&v2.DiscoveryRequest{
					TypeUrl: url,
					Node:    &core.Node{},
				}))
			assert.NotNil(t, respCh)
			for {
				if v, ok := scope.Snapshot().Counters()[stats[0]]; ok {
					assert.Equal(t, int64(1), v.Value())
					break
				}
			}
			for {
				if v, ok := scope.Snapshot().Counters()[stats[1]]; ok {
					assert.Equal(t, int64(1), v.Value())
					break
				}
			}
		})
	}
}

func TestOpenStreamShouldNotReturnErrorOnStreamCreationFailureV3(t *testing.T) {
	scope := stats.NewMockScope("mock")
	client := createMockClientWithErrorV3(scope)

	typeURLs := map[string][]string{
		resourcev3.ListenerType: {"mock.lds.stream_failure+", "mock.lds.stream_opened+"},
		resourcev3.ClusterType:  {"mock.cds.stream_failure+", "mock.cds.stream_opened+"},
		resourcev3.RouteType:    {"mock.rds.stream_failure+", "mock.rds.stream_opened+"},
		resourcev3.EndpointType: {"mock.eds.stream_failure+", "mock.eds.stream_opened+"},
	}
	for url, stats := range typeURLs {
		t.Run(url, func(t *testing.T) {
			respCh, _ := client.OpenStream(
				transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
					TypeUrl: url,
					Node:    &corev3.Node{},
				}))
			assert.NotNil(t, respCh)
			for {
				if v, ok := scope.Snapshot().Counters()[stats[0]]; ok {
					assert.Equal(t, int64(1), v.Value())
					break
				}
			}
			for {
				if v, ok := scope.Snapshot().Counters()[stats[1]]; ok {
					assert.Equal(t, int64(1), v.Value())
					break
				}
			}
		})
	}
}

func TestOpenStreamShouldReturnNonEmptyResponseChannel(t *testing.T) {
	client := createMockClient()

	respCh, done := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	assert.NotNil(t, respCh)
	done()
}

func TestOpenStreamShouldReturnNonEmptyResponseChannelV3(t *testing.T) {
	client := createMockClientV3()

	respCh, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	assert.NotNil(t, respCh)
	done()
}

func TestOpenStreamShouldSendTheFirstRequestToOriginServer(t *testing.T) {
	var message *v2.DiscoveryRequest
	responseChan := make(chan *v2.DiscoveryResponse)
	wait := make(chan bool)
	client := upstream.NewMock(
		context.Background(),
		CallOptions{Timeout: time.Nanosecond},
		nil,
		responseChan,
		responseChan,
		responseChan,
		responseChan,
		func(m interface{}) error {
			message = m.(*v2.DiscoveryRequest)
			wait <- true
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
	<-wait
	assert.NotNil(t, message)
	assert.Equal(t, message.GetNode(), node)
	assert.Equal(t, message.GetTypeUrl(), resource.ListenerType)
	done()
}

func TestOpenStreamShouldSendTheFirstRequestToOriginServerV3(t *testing.T) {
	var message *discoveryv3.DiscoveryRequest
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	wait := make(chan bool)
	client := upstream.NewMockV3(
		context.Background(),
		CallOptions{Timeout: time.Nanosecond},
		nil,
		responseChan,
		responseChan,
		responseChan,
		responseChan,
		func(m interface{}) error {
			message = m.(*discoveryv3.DiscoveryRequest)
			wait <- true
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
	<-wait
	assert.NotNil(t, message)
	assert.Equal(t, message.GetNode(), node)
	assert.Equal(t, message.GetTypeUrl(), resourcev3.ListenerType)
	done()
}

func TestOpenStreamShouldRetryIfSendFails(t *testing.T) {
	responseChan := make(chan *v2.DiscoveryResponse)
	sendError := fmt.Errorf("")
	first := true
	response := &v2.DiscoveryResponse{}
	client := createMockClientWithResponse(time.Second, responseChan, func(m interface{}) error {
		if first {
			first = !first
			return sendError
		}
		responseChan <- response
		return nil
	}, stats.NewMockScope("mock"))

	resp, done := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	_, more := <-resp
	assert.True(t, more)
	done()
}

func TestOpenStreamShouldRetryIfSendFailsV3(t *testing.T) {
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	sendError := fmt.Errorf("")
	first := true
	response := &discoveryv3.DiscoveryResponse{}
	client := createMockClientWithResponseV3(time.Second, responseChan, func(m interface{}) error {
		if first {
			first = !first
			return sendError
		}
		responseChan <- response
		return nil
	}, stats.NewMockScope("mock"))

	resp, done := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	_, more := <-resp
	assert.True(t, more)
	done()
}

func TestOpenStreamShouldSendTheResponseOnTheChannel(t *testing.T) {
	responseChan := make(chan *v2.DiscoveryResponse)
	response := &v2.DiscoveryResponse{}
	client := createMockClientWithResponse(time.Second, responseChan, func(m interface{}) error {
		responseChan <- response
		return nil
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
}

func TestOpenStreamShouldSendTheResponseOnTheChannelV3(t *testing.T) {
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	response := &discoveryv3.DiscoveryResponse{}
	client := createMockClientWithResponseV3(time.Second, responseChan, func(m interface{}) error {
		responseChan <- response
		return nil
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
}

func TestOpenStreamShouldSendTheNextRequestWithUpdatedVersionAndNonce(t *testing.T) {
	responseChan := make(chan *v2.DiscoveryResponse)
	lastAppliedVersion := ""
	index := 0
	client := createMockClientWithResponse(time.Second, responseChan, func(m interface{}) error {
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
	assert.NotNil(t, resp)
	for i := 0; i < 5; i++ {
		val := <-resp
		assert.Equal(t, val.GetPayloadVersion(), strconv.Itoa(i))
		assert.Equal(t, val.GetNonce(), strconv.Itoa(i))
	}

	done()
}

func TestOpenStreamShouldSendTheNextRequestWithUpdatedVersionAndNonceV3(t *testing.T) {
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	lastAppliedVersion := ""
	index := 0
	client := createMockClientWithResponseV3(time.Second, responseChan, func(m interface{}) error {
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
	assert.NotNil(t, resp)
	for i := 0; i < 5; i++ {
		val := <-resp
		assert.Equal(t, val.GetPayloadVersion(), strconv.Itoa(i))
		assert.Equal(t, val.GetNonce(), strconv.Itoa(i))
	}

	done()
}

func TestOpenStreamShouldRetryWhenSendMsgBlocks(t *testing.T) {
	responseChan := make(chan *v2.DiscoveryResponse)
	blockedCtx, cancel := context.WithCancel(context.Background())
	first := true
	response1 := &v2.DiscoveryResponse{VersionInfo: "1"}
	response2 := &v2.DiscoveryResponse{VersionInfo: "2"}
	client := createMockClientWithResponse(time.Nanosecond, responseChan, func(m interface{}) error {
		if first {
			first = !first
			<-blockedCtx.Done()
			responseChan <- response1
			return nil
		}
		responseChan <- response2
		return nil
	}, stats.NewMockScope("mock"))

	respCh, done := client.OpenStream(transport.NewRequestV2(&v2.DiscoveryRequest{
		TypeUrl: resource.ListenerType,
		Node:    &core.Node{},
	}))
	resp, ok := <-respCh
	assert.True(t, ok)
	assert.Equal(t, resp.Get().V2.VersionInfo, response2.VersionInfo)

	cancel()
	select {
	case <-respCh:
		assert.Fail(t, "Channel should not contain any response")
	default:
	}

	done()
}

func TestOpenStreamShouldSendErrorWhenSendMsgBlocksV3(t *testing.T) {
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	blockedCtx, cancel := context.WithCancel(context.Background())
	first := true
	response1 := &discoveryv3.DiscoveryResponse{VersionInfo: "1"}
	response2 := &discoveryv3.DiscoveryResponse{VersionInfo: "2"}
	client := createMockClientWithResponseV3(time.Nanosecond, responseChan, func(m interface{}) error {
		if first {
			first = !first
			<-blockedCtx.Done()
			responseChan <- response1
			return nil
		}
		responseChan <- response2
		return nil
	}, stats.NewMockScope("mock"))

	respCh, done := client.OpenStream(transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
		TypeUrl: resourcev3.ListenerType,
		Node:    &corev3.Node{},
	}))
	resp, ok := <-respCh
	assert.True(t, ok)
	assert.Equal(t, resp.Get().V3.VersionInfo, response2.VersionInfo)

	cancel()
	select {
	case v := <-respCh:
		assert.Fail(t, "Channel should not contain any response %s", v.Get().V3.VersionInfo)
	default:
	}

	done()
}

func createMockClient() upstream.Client {
	return upstream.NewMock(
		context.Background(),
		CallOptions{Timeout: time.Nanosecond},
		nil,
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"))
}

func createMockClientWithError(scope tally.Scope) upstream.Client {
	return upstream.NewMock(
		context.Background(),
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
	t time.Duration,
	r chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error,
	scope tally.Scope) upstream.Client {
	return upstream.NewMock(context.Background(), CallOptions{Timeout: t}, nil, r, r, r, r, sendCb, scope)
}

func createMockClientV3() upstream.Client {
	return upstream.NewMockV3(
		context.Background(),
		CallOptions{Timeout: time.Nanosecond},
		nil,
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"))
}

func createMockClientWithErrorV3(scope tally.Scope) upstream.Client {
	return upstream.NewMockV3(
		context.Background(),
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
	t time.Duration,
	r chan *discoveryv3.DiscoveryResponse,
	sendCb func(m interface{}) error,
	scope tally.Scope) upstream.Client {
	return upstream.NewMockV3(context.Background(), CallOptions{Timeout: t}, nil, r, r, r, r, sendCb, scope)
}
