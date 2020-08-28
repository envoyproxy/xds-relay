package upstream_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v2"
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
