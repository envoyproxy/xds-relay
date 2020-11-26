// Package orchestrator is responsible for instrumenting inbound xDS client
// requests to the correct aggregated key, forwarding a representative request
// to the upstream origin server, and managing the lifecycle of downstream and
// upstream connections and associates streams. It implements
// go-control-plane's Cache interface in order to receive xDS-based requests,
// send responses, and handle gRPC streams.
//
// This file manages the bookkeeping of downstream clients by tracking inbound
// requests to their corresponding response channels. The contents of this file
// are intended to only be used within the orchestrator module and should not
// be exported.
package orchestrator

import (
	"testing"

	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/stretchr/testify/assert"
)

var (
	mockRequest = gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
	}
)

func Test_downstreamResponseMap_addWatch(t *testing.T) {
	responseMap := newDownstreamResponseMap()
	assert.Equal(t, 0, len(responseMap.watches))
	responseMap.addWatch("key", transport.NewWatchV2(make(chan<- gcp.Response, 1)))
	assert.Equal(t, 1, len(responseMap.watches))
}

func Test_downstreamResponseMap_get(t *testing.T) {
	responseMap := newDownstreamResponseMap()
	responseMap.addWatch("key", transport.NewWatchV2(make(chan<- gcp.Response, 1)))
	assert.Equal(t, 1, len(responseMap.watches))
	assert.Len(t, responseMap.get("key"), 1)
}

func Test_downstreamResponseMap_getSnapshot(t *testing.T) {
	responseMap := newDownstreamResponseMap()
	responseMap.addWatch("key", transport.NewWatchV2(make(chan<- gcp.Response, 1)))
	assert.Equal(t, 1, len(responseMap.watches))

	s, ok := responseMap.getSnapshot("key")
	assert.Len(t, s, 1)
	assert.True(t, ok)

	s, ok = responseMap.getSnapshot("key")
	assert.Len(t, s, 0)
	assert.True(t, ok)

	s, ok = responseMap.getSnapshot("nonexistent")
	assert.Len(t, s, 0)
	assert.False(t, ok)
}

func Test_downstreamResponseMap_delete(t *testing.T) {
	responseMap := newDownstreamResponseMap()
	chan1 := make(chan gcp.Response, 1)
	chan2 := make(chan gcp.Response, 1)
	watch1 := transport.NewWatchV2(chan1)
	watch2 := transport.NewWatchV2(chan2)
	responseMap.addWatch("key1", watch1)
	responseMap.addWatch("key2", watch2)
	assert.Equal(t, 2, len(responseMap.watches))
	assert.Len(t, responseMap.get("key1"), 1)
	assert.Len(t, responseMap.get("key2"), 1)

	responseMap.delete(watch1)
	responseMap.delete(watch2)

	err := watch1.Send(nil)
	assert.NoError(t, err)
	err = watch2.Send(nil)
	assert.NoError(t, err)

	select {
	case <-chan1:
		assert.Fail(t, "watch should be noop after delete")
	default:
	}

	select {
	case <-chan2:
		assert.Fail(t, "watch should be noop after delete")
	default:
	}
}

func Test_downstreamResponseMap_deleteAll(t *testing.T) {
	responseMap := newDownstreamResponseMap()
	responseMap.addWatch("key1", transport.NewWatchV2(make(chan<- gcp.Response, 1)))
	responseMap.addWatch("key2", transport.NewWatchV2(make(chan<- gcp.Response, 1)))
	responseMap.addWatch("key3", transport.NewWatchV2(make(chan<- gcp.Response, 1)))
	assert.Equal(t, 3, len(responseMap.watches))
	responseMap.deleteAll("key1")
	responseMap.deleteAll("key2")
	assert.Len(t, responseMap.watches, 1)
	assert.Len(t, responseMap.get("key3"), 1)
}
