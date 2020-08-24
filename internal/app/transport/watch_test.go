package transport

import (
	"sync"
	"testing"

	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetChannel(t *testing.T) {
	w := NewWatchV2(&gcp.Request{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_, more := <-w.GetCh().(chan gcp.Response)
		assert.False(t, more)
		wg.Done()
	}()

	w.Close()
	wg.Wait()
}

func TestSendSuccessful(t *testing.T) {
	w := NewWatchV2(&gcp.Request{})
	resp := gcp.RawResponse{}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		got, more := <-w.GetCh().(chan gcp.Response)
		assert.True(t, more)
		assert.Equal(t, resp, got)
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
	w := NewWatchV2(&gcp.Request{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		ok, err := w.Send("")
		assert.False(t, ok)
		assert.NotNil(t, err)
		wg.Done()
	}()
	wg.Wait()
}

func TestSendFalseWhenBlocked(t *testing.T) {
	w := NewWatchV2(&gcp.Request{})
	var wg sync.WaitGroup
	resp := gcp.RawResponse{}
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

func sendWithCloseChannelOnFailure(t *testing.T, w Watch, wg *sync.WaitGroup, r interface{}) {
	ok, err := w.Send(r)
	assert.Nil(t, err)
	if !ok {
		w.Close()
	}
	wg.Done()
}
