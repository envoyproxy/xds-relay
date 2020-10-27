package transport

import (
	"sync"
	"testing"

	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/stretchr/testify/assert"
)

var discoveryResponsev3 = &discoveryv3.DiscoveryResponse{}
var discoveryRequestv3 = &gcpv3.Request{}

func TestSendSuccessfulV3(t *testing.T) {
	var wg sync.WaitGroup
	ch := make(chan gcpv3.Response, 1)
	wg.Add(2)
	w := newWatchV3(ch)
	expected := &cachev3.PassthroughResponse{DiscoveryResponse: discoveryResponsev3, Request: discoveryRequestv3}

	go func() {
		more := false
		got, more := <-ch
		assert.True(t, more)
		assert.Equal(t, expected, got)
		wg.Done()
	}()

	go func() {
		ok := w.Send(NewResponseV3(discoveryRequestv3, discoveryResponsev3))
		assert.True(t, ok)
		wg.Done()
	}()
	wg.Wait()
}

func TestSendFalseWhenBlockedV3(t *testing.T) {
	ch := make(chan gcpv3.Response, 1)
	defer close(ch)
	w := newWatchV3(ch)
	ok := w.Send(NewResponseV3(discoveryRequestv3, discoveryResponsev3))
	assert.True(t, ok)
	ok = w.Send(NewResponseV3(discoveryRequestv3, discoveryResponsev3))
	assert.False(t, ok)
}

func TestCloseSendsNilV3(t *testing.T) {
	ch := make(chan gcpv3.Response, 1)
	defer close(ch)
	newWatchV3(ch).Close()
	resp, ok := <-ch
	assert.True(t, ok)
	assert.Nil(t, resp)
}
