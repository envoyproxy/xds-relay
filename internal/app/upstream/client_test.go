package upstream_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	mock "github.com/envoyproxy/xds-relay/test/mocks/upstream"
	"github.com/stretchr/testify/assert"
)

type CallOptions = upstream.CallOptions

func TestOpenStreamShouldReturnErrorForInvalidTypeUrl(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	client := mock.NewClient(
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
	_, ok := err.(*upstream.UnsupportedResourceError)
	assert.True(t, ok)
	assert.Nil(t, respCh)

	cancel()
}

func TestOpenStreamShouldResturnErrorOnStreamCreationFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	responseChan := make(chan *v2.DiscoveryResponse)
	client := mock.NewClient(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		fmt.Errorf("error"),
		responseChan,
		func(m interface{}) error { return nil })

	typeURLs := []string{
		upstream.ListenerTypeURL,
		upstream.ClusterTypeURL,
		upstream.RouteTypeURL,
		upstream.EndpointTypeURL,
	}
	for _, typeURL := range typeURLs {
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
	client := mock.NewClient(
		ctx,
		CallOptions{Timeout: time.Nanosecond},
		nil,
		responseChan,
		func(m interface{}) error { return nil })

	respCh, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: upstream.ListenerTypeURL,
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
	client := mock.NewClient(ctx, CallOptions{Timeout: time.Nanosecond}, nil, responseChan, func(m interface{}) error {
		message = m.(*v2.DiscoveryRequest)
		wait <- true
		return nil
	})

	node := &core.Node{}
	_, _ = client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: upstream.ListenerTypeURL,
		Node:    node,
	})
	<-wait
	assert.NotNil(t, message)
	assert.Equal(t, message.GetNode(), node)
	assert.Equal(t, message.TypeUrl, upstream.ListenerTypeURL)
	cancel()
}

func TestOpenStreamShouldSendErrorIfSendFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	responseChan := make(chan *v2.DiscoveryResponse)
	sendError := fmt.Errorf("")
	client := mock.NewClient(ctx, CallOptions{Timeout: time.Second}, nil, responseChan, func(m interface{}) error {
		return sendError
	})

	resp, _ := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: upstream.ListenerTypeURL,
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
	client := mock.NewClient(ctx, CallOptions{Timeout: time.Second}, nil, responseChan, func(m interface{}) error {
		responseChan <- response
		return nil
	})

	resp, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: upstream.ListenerTypeURL,
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
	client := mock.NewClient(ctx, CallOptions{Timeout: time.Second}, nil, responseChan, func(m interface{}) error {
		message := m.(*v2.DiscoveryRequest)

		assert.Equal(t, message.GetVersionInfo(), lastAppliedVersion)
		assert.Equal(t, message.GetResponseNonce(), lastAppliedVersion)

		response := &v2.DiscoveryResponse{
			VersionInfo: strconv.Itoa(index),
			Nonce:       strconv.Itoa(index),
			TypeUrl:     upstream.ListenerTypeURL,
		}
		lastAppliedVersion = strconv.Itoa(index)
		index++
		responseChan <- response
		return nil
	})

	resp, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: upstream.ListenerTypeURL,
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
	client := mock.NewClient(ctx, CallOptions{Timeout: time.Nanosecond}, nil, responseChan, func(m interface{}) error {
		<-ctx.Done()
		return nil
	})

	resp, err := client.OpenStream(ctx, &v2.DiscoveryRequest{
		TypeUrl: upstream.ListenerTypeURL,
		Node:    &core.Node{},
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	val := <-resp
	assert.Equal(t, val.Err.Error(), "context deadline exceeded")
	assert.Nil(t, val.Response)
	cancel()
}
