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

func TestSendFalseWhenBlockedV2(t *testing.T) {
	ch := make(chan gcp.Response, 1)
	defer close(ch)
	w := NewWatchV2(ch)
	err := w.Send(NewResponseV2(discoveryRequestv2, discoveryResponsev2))
	assert.NoError(t, err)
	err = w.Send(NewResponseV2(discoveryRequestv2, discoveryResponsev2))
	assert.NotNil(t, err)
}

func TestCloseSendsNilV2(t *testing.T) {
	ch := make(chan gcp.Response, 1)
	defer close(ch)
	NewWatchV2(ch).Close()
	resp, ok := <-ch
	assert.True(t, ok)
	assert.Nil(t, resp)
}

func TestSendAfterCloseV2(t *testing.T) {
	ch := make(chan gcp.Response, 1)
	defer close(ch)
	w := NewWatchV2(ch)
	err := w.Send(NewResponseV2(discoveryRequestv2, discoveryResponsev2))
	assert.NoError(t, err)
	<-ch
	w.Close()
	select {
	case r := <-ch:
		assert.Nil(t, r)
	default:
	}
	err = w.Send(NewResponseV2(discoveryRequestv2, discoveryResponsev2))
	assert.NoError(t, err)

	select {
	case <-ch:
		assert.Fail(t, "should not send more after close")
	default:
	}
}
