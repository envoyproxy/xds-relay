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
	"github.com/stretchr/testify/assert"
)

type CallOptions = upstream.CallOptions

func TestOpenStreamShouldReturnErrorForInvalidTypeUrl(t *testing.T) {
	client := createMockClient()

	respCh, _, err := client.OpenStream(transport.NewRequestV2(&v2.DiscoveryRequest{}))
	assert.NotNil(t, err)
	_, ok := err.(*upstream.UnsupportedResourceError)
	assert.True(t, ok)
	assert.Nil(t, respCh)
}

func TestOpenStreamShouldReturnErrorForInvalidTypeUrlV3(t *testing.T) {
	client := createMockClientV3()

	respCh, _, err := client.OpenStream(transport.NewRequestV3(&discoveryv3.DiscoveryRequest{}))
	assert.NotNil(t, err)
	_, ok := err.(*upstream.UnsupportedResourceError)
	assert.True(t, ok)
	assert.Nil(t, respCh)
}

func TestOpenStreamShouldReturnErrorOnStreamCreationFailure(t *testing.T) {
	client := createMockClientWithError()

	typeURLs := []string{
		resource.ListenerType,
		resource.ClusterType,
		resource.RouteType,
		resource.EndpointType,
	}
	for _, typeURL := range typeURLs {
		t.Run(typeURL, func(t *testing.T) {
			respCh, _, err := client.OpenStream(
				transport.NewRequestV2(&v2.DiscoveryRequest{
					TypeUrl: typeURL,
					Node:    &core.Node{},
				}))
			assert.Nil(t, respCh)
			assert.NotNil(t, err)
		})
	}
}

func TestOpenStreamShouldReturnErrorOnStreamCreationFailureV3(t *testing.T) {
	client := createMockClientWithErrorV3()

	typeURLs := []string{
		resourcev3.ListenerType,
		resourcev3.ClusterType,
		resourcev3.RouteType,
		resourcev3.EndpointType,
	}
	for _, typeURL := range typeURLs {
		t.Run(typeURL, func(t *testing.T) {
			respCh, _, err := client.OpenStream(
				transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
					TypeUrl: typeURL,
					Node:    &corev3.Node{},
				}))
			assert.Nil(t, respCh)
			assert.NotNil(t, err)
		})
	}
}

func TestOpenStreamShouldReturnNonEmptyResponseChannel(t *testing.T) {
	client := createMockClient()

	respCh, done, err := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	assert.NotNil(t, respCh)
	assert.Nil(t, err)
	done()
}

func TestOpenStreamShouldReturnNonEmptyResponseChannelV3(t *testing.T) {
	client := createMockClient()

	respCh, done, err := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &corev3.Node{},
		}))
	assert.NotNil(t, respCh)
	assert.Nil(t, err)
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
	)

	node := &core.Node{}
	_, done, _ := client.OpenStream(
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
	)

	node := &corev3.Node{}
	_, done, _ := client.OpenStream(
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

func TestOpenStreamShouldSendErrorIfSendFails(t *testing.T) {
	responseChan := make(chan *v2.DiscoveryResponse)
	sendError := fmt.Errorf("")
	client := createMockClientWithResponse(time.Second, responseChan, func(m interface{}) error {
		return sendError
	})

	resp, done, _ := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	_, more := <-resp
	assert.False(t, more)
	done()
}

func TestOpenStreamShouldSendErrorIfSendFailsV3(t *testing.T) {
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	sendError := fmt.Errorf("")
	client := createMockClientWithResponseV3(time.Second, responseChan, func(m interface{}) error {
		return sendError
	})

	resp, done, _ := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	_, more := <-resp
	assert.False(t, more)
	done()
}

func TestOpenStreamShouldSendTheResponseOnTheChannel(t *testing.T) {
	responseChan := make(chan *v2.DiscoveryResponse)
	response := &v2.DiscoveryResponse{}
	client := createMockClientWithResponse(time.Second, responseChan, func(m interface{}) error {
		responseChan <- response
		return nil
	})

	resp, done, err := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	assert.Nil(t, err)
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
	})

	resp, done, err := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	assert.Nil(t, err)
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
	})

	resp, done, err := client.OpenStream(
		transport.NewRequestV2(&v2.DiscoveryRequest{
			TypeUrl: resource.ListenerType,
			Node:    &core.Node{},
		}))
	assert.Nil(t, err)
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
	})

	resp, done, err := client.OpenStream(
		transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
			TypeUrl: resourcev3.ListenerType,
			Node:    &corev3.Node{},
		}))
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	for i := 0; i < 5; i++ {
		val := <-resp
		assert.Equal(t, val.GetPayloadVersion(), strconv.Itoa(i))
		assert.Equal(t, val.GetNonce(), strconv.Itoa(i))
	}

	done()
}

func TestOpenStreamShouldSendErrorWhenSendMsgBlocks(t *testing.T) {
	responseChan := make(chan *v2.DiscoveryResponse)
	blockedCtx, cancel := context.WithCancel(context.Background())
	client := createMockClientWithResponse(time.Nanosecond, responseChan, func(m interface{}) error {
		// TODO: When stats are available, strengthen the test
		// https://github.com/envoyproxy/xds-relay/issues/61
		<-blockedCtx.Done()
		return nil
	})

	resp, done, err := client.OpenStream(transport.NewRequestV2(&v2.DiscoveryRequest{
		TypeUrl: resource.ListenerType,
		Node:    &core.Node{},
	}))
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	_, more := <-resp
	assert.False(t, more)

	done()
	cancel()
}

func TestOpenStreamShouldSendErrorWhenSendMsgBlocksV3(t *testing.T) {
	responseChan := make(chan *discoveryv3.DiscoveryResponse)
	blockedCtx, cancel := context.WithCancel(context.Background())
	client := createMockClientWithResponseV3(time.Nanosecond, responseChan, func(m interface{}) error {
		// TODO: When stats are available, strengthen the test
		// https://github.com/envoyproxy/xds-relay/issues/61
		<-blockedCtx.Done()
		return nil
	})

	resp, done, err := client.OpenStream(transport.NewRequestV3(&discoveryv3.DiscoveryRequest{
		TypeUrl: resourcev3.ListenerType,
		Node:    &corev3.Node{},
	}))
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	_, more := <-resp
	assert.False(t, more)

	done()
	cancel()
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
		func(m interface{}) error { return nil })
}

func createMockClientWithError() upstream.Client {
	return upstream.NewMock(
		context.Background(),
		CallOptions{Timeout: time.Nanosecond},
		fmt.Errorf("error"),
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		make(chan *v2.DiscoveryResponse),
		func(m interface{}) error { return nil })
}

func createMockClientWithResponse(
	t time.Duration,
	r chan *v2.DiscoveryResponse,
	sendCb func(m interface{}) error) upstream.Client {
	return upstream.NewMock(context.Background(), CallOptions{Timeout: t}, nil, r, r, r, r, sendCb)
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
		func(m interface{}) error { return nil })
}

func createMockClientWithErrorV3() upstream.Client {
	return upstream.NewMockV3(
		context.Background(),
		CallOptions{Timeout: time.Nanosecond},
		fmt.Errorf("error"),
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		make(chan *discoveryv3.DiscoveryResponse),
		func(m interface{}) error { return nil })
}

func createMockClientWithResponseV3(
	t time.Duration,
	r chan *discoveryv3.DiscoveryResponse,
	sendCb func(m interface{}) error) upstream.Client {
	return upstream.NewMockV3(context.Background(), CallOptions{Timeout: t}, nil, r, r, r, r, sendCb)
}
