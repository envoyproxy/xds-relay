package transport

import (
	"sync"
	"testing"

	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetChannel(t *testing.T) {
	w := newWatchV2()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_, more := <-w.GetChannel().V2
		assert.False(t, more)
		wg.Done()
	}()

	w.Close()
	wg.Wait()
}

func TestSendSuccessful(t *testing.T) {
	w := newWatchV2()
	discoveryResponse := &discoveryv2.DiscoveryResponse{}
	discoveryRequest := &gcp.Request{}
	resp := NewResponseV2(discoveryRequest, discoveryResponse)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		got, more := <-w.GetChannel().V2
		assert.True(t, more)
		assert.Equal(t, cache.PassthroughResponse{DiscoveryResponse: discoveryResponse, Request: *discoveryRequest}, got)
		wg.Done()
	}()
	go func() {
		ok := w.Send(resp)
		assert.True(t, ok)
		wg.Done()
	}()
	wg.Wait()
}

func TestSendFalseWhenBlocked(t *testing.T) {
	w := newWatchV2()
	var wg sync.WaitGroup
	resp := NewResponseV2(&discoveryv2.DiscoveryRequest{}, &discoveryv2.DiscoveryResponse{})
	wg.Add(2)
	// We perform 2 sends with no receive on w.Out .
	// One of the send gets blocked because of no recipient.
	// The second send goes goes to default case due to channel full.
	// The second send closes the channel when blocked.
	// The closed channel terminates the blocked send to exit the test case.
	go sendWithCloseChannelOnFailure(w, &wg, resp)
	go sendWithCloseChannelOnFailure(w, &wg, resp)

	wg.Wait()
}

func sendWithCloseChannelOnFailure(w Watch, wg *sync.WaitGroup, r Response) {
	ok := w.Send(r)
	if !ok {
		w.Close()
	}
	wg.Done()
}
