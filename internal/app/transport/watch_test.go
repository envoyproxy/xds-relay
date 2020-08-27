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
		ok, err := w.Send(resp)
		assert.True(t, ok)
		assert.Nil(t, err)
		wg.Done()
	}()
	wg.Wait()
}

func TestSendCastFailure(t *testing.T) {
	w := newWatchV2()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		ok, err := w.Send(&mockResponse{})
		assert.False(t, ok)
		assert.NotNil(t, err)
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
	go sendWithCloseChannelOnFailure(t, w, &wg, resp)
	go sendWithCloseChannelOnFailure(t, w, &wg, resp)

	wg.Wait()
}

func sendWithCloseChannelOnFailure(t *testing.T, w Watch, wg *sync.WaitGroup, r Response) {
	ok, err := w.Send(r)
	assert.Nil(t, err)
	if !ok {
		w.Close()
	}
	wg.Done()
}

var _ Response = &mockResponse{}

type mockResponse struct {
}

func (r *mockResponse) GetPayloadVersion() string {
	return ""
}

func (r *mockResponse) GetTypeURL() string {
	return ""
}

func (r *mockResponse) GetNonce() string {
	return ""
}

func (r *mockResponse) GetRequest() *RequestVersion {
	return &RequestVersion{}
}

func (r *mockResponse) Get() *ResponseVersion {
	return &ResponseVersion{}
}
