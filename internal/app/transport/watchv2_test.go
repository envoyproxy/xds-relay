package transport

import (
	"sync"
	"testing"

	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	cachev2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/stretchr/testify/assert"
)

var discoveryResponsev2 = &discoveryv2.DiscoveryResponse{}
var discoveryRequestv2 = &gcp.Request{}

func TestSendSuccessfulV2(t *testing.T) {
	var wg sync.WaitGroup
	ch := make(chan gcp.Response, 1)
	wg.Add(2)
	w := NewWatchV2(ch)
	expected := &cachev2.PassthroughResponse{DiscoveryResponse: discoveryResponsev2, Request: discoveryRequestv2}

	go func() {
		got, more := <-ch
		assert.True(t, more)
		assert.Equal(t, expected, got)
		wg.Done()
	}()

	go func() {
		err := w.Send(NewResponseV2(discoveryRequestv2, discoveryResponsev2))
		assert.NoError(t, err)
		wg.Done()
	}()
	wg.Wait()
}

func TestSendErrorWhenBlockedV2(t *testing.T) {
	ch := make(chan gcp.Response)
	w := NewWatchV2(ch)
	err := w.Send(NewResponseV2(discoveryRequestv2, discoveryResponsev2))
	assert.Error(t, err)
}

func TestSendAllowsOneResponseV2(t *testing.T) {
	ch := make(chan gcp.Response, 1)
	w := NewWatchV2(ch)
	err := w.Send(NewResponseV2(discoveryRequestv2, discoveryResponsev2))
	assert.NoError(t, err)

	<-ch
	err = w.Send(NewResponseV2(discoveryRequestv2, discoveryResponsev2))
	assert.NoError(t, err)
	select {
	case <-ch:
		assert.Fail(t, "Response is not expected")
	default:
	}
}

func TestSendAllowsNilResponseV2(t *testing.T) {
	ch := make(chan gcp.Response, 1)
	w := NewWatchV2(ch)
	err := w.Send(nil)
	assert.NoError(t, err)

	resp := <-ch
	assert.Nil(t, resp)
}
