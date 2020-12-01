package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/envoyproxy/xds-relay/internal/app/cache"
	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"github.com/envoyproxy/xds-relay/internal/pkg/stats"
	"github.com/envoyproxy/xds-relay/internal/pkg/util/testutils"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
	"go.uber.org/goleak"
	"google.golang.org/genproto/googleapis/rpc/status"
)

type mockSimpleUpstreamClient struct {
	responseChan <-chan transport.Response
}

func (m mockSimpleUpstreamClient) OpenStream(req transport.Request, key string) (<-chan transport.Response, func()) {
	return m.responseChan, func() {}
}

type mockMultiStreamUpstreamClient struct {
	ldsResponseChan <-chan transport.Response
	cdsResponseChan <-chan transport.Response

	t      *testing.T
	mapper mapper.Mapper
}

func (m mockMultiStreamUpstreamClient) OpenStream(
	req transport.Request, key string,
) (<-chan transport.Response, func()) {
	aggregatedKey, err := m.mapper.GetKey(req)
	assert.NoError(m.t, err)

	if aggregatedKey == "lds" {
		return m.ldsResponseChan, func() {}
	} else if aggregatedKey == "cds" {
		return m.cdsResponseChan, func() {}
	}

	m.t.Errorf("Unsupported aggregated key, %s", aggregatedKey)
	return nil, func() {}
}

func newMockOrchestrator(t *testing.T, mockScope tally.Scope, mapper mapper.Mapper,
	upstreamClient upstream.Client) *orchestrator {
	orchestrator := &orchestrator{
		logger:                log.MockLogger,
		scope:                 mockScope,
		mapper:                mapper,
		upstreamClient:        upstreamClient,
		downstreamResponseMap: newDownstreamResponseMap(),
		upstreamResponseMap:   newUpstreamResponseMap(),
	}

	cache, err := cache.NewCache(1000, orchestrator.onCacheEvicted, 10*time.Second, log.MockLogger, mockScope)
	assert.NoError(t, err)
	orchestrator.cache = cache

	return orchestrator
}

func assertEqualResponse(t *testing.T, got gcp.Response, expected *v2.DiscoveryResponse, req *gcp.Request) {
	gotDiscoveryResponse, err := got.GetDiscoveryResponse()
	assert.NoError(t, err)
	assert.Equal(t, expected, gotDiscoveryResponse)
	assert.Equal(t, req, got.GetRequest())
}

func TestMain(m *testing.M) {
	defer goleak.VerifyTestMain(m)
}

func TestNew(t *testing.T) {
	// Trivial test to ensure orchestrator instantiates.
	ctx, cancel := context.WithCancel(context.Background())
	upstreamClient := upstream.NewMock(
		context.Background(),
		upstream.CallOptions{},
		nil,
		nil,
		nil,
		nil,
		nil,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)

	config := aggregationv1.KeyerConfiguration{
		Fragments: []*aggregationv1.KeyerConfiguration_Fragment{
			{
				Rules: []*aggregationv1.KeyerConfiguration_Fragment_Rule{},
			},
		},
	}
	requestMapper := mapper.New(&config, stats.NewMockScope(""))

	cacheConfig := bootstrapv1.Cache{
		Ttl: &duration.Duration{
			Seconds: 10,
		},
		MaxEntries: 10,
	}

	orchestrator := New(ctx, log.MockLogger, tally.NewTestScope("prefix",
		make(map[string]string)), requestMapper, upstreamClient, &cacheConfig)
	cancel()
	assert.NotNil(t, orchestrator)
}

func TestGoldenPath(t *testing.T) {
	upstreamResponseChannel := make(chan transport.Response)
	mapper := mapper.NewMock(t)
	mockScope := stats.NewMockScope("mock_orchestrator")
	orchestrator := newMockOrchestrator(
		t,
		mockScope,
		mapper,
		mockSimpleUpstreamClient{
			responseChan: upstreamResponseChannel,
		},
	)
	assert.NotNil(t, orchestrator)

	req := &gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
	}
	ch := make(chan gcp.Response, 1)
	r := transport.NewRequestV2(req)
	aggregatedKey, err := mapper.GetKey(r)
	assert.NoError(t, err)

	cancelWatch := orchestrator.CreateWatch(r, transport.NewWatchV2(ch))
	countersSnapshot := mockScope.Snapshot().Counters()
	assert.EqualValues(
		t, 1, countersSnapshot[fmt.Sprintf("mock_orchestrator.watch.created+key=%v", aggregatedKey)].Value())
	assert.Equal(t, 1, len(orchestrator.downstreamResponseMap.watches))
	testutils.AssertSyncMapLen(t, 1, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Equal(t, "lds", key.(string))
		return true
	})

	resp := &v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		Resources: []*any.Any{
			{
				Value: []byte("lds resource"),
			},
		},
	}
	upstreamResponseChannel <- transport.NewResponseV2(req, resp)

	gotResponse := <-ch
	assertEqualResponse(t, gotResponse, resp, req)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	orchestrator.shutdown(ctx)
	testutils.AssertSyncMapLen(t, 0, orchestrator.upstreamResponseMap.internal)

	cancelWatch()

	countersSnapshot = mockScope.Snapshot().Counters()
	assert.EqualValues(
		t, 1, countersSnapshot[fmt.Sprintf("mock_orchestrator.watch.fanout+key=%v", aggregatedKey)].Value())
	assert.EqualValues(
		t, 1, countersSnapshot[fmt.Sprintf("mock_orchestrator.watch.canceled+key=%v", aggregatedKey)].Value())
	assert.Equal(t, 0, len(orchestrator.downstreamResponseMap.watches))
}

func TestUnaggregatedKey(t *testing.T) {
	upstreamResponseChannel := make(chan transport.Response)
	mapper := mapper.NewMock(t)
	mockScope := stats.NewMockScope("mock_orchestrator")
	orchestrator := newMockOrchestrator(
		t,
		mockScope,
		mapper,
		mockSimpleUpstreamClient{
			responseChan: upstreamResponseChannel,
		},
	)
	assert.NotNil(t, orchestrator)

	ch := make(chan gcp.Response, 1)
	req := transport.NewRequestV2(
		&gcp.Request{
			TypeUrl: "type.googleapis.com/envoy.api.v2.UnsupportedType",
		},
	)
	// assert this request will not map to an aggregated key.
	_, err := mapper.GetKey(req)
	assert.Error(t, err)

	_ = orchestrator.CreateWatch(req, transport.NewWatchV2(ch))
	assert.Equal(t, 0, len(orchestrator.downstreamResponseMap.watches))
	r, more := <-ch
	assert.Nil(t, r)
	assert.True(t, more)
}

func TestCachedResponse(t *testing.T) {
	upstreamResponseChannel := make(chan transport.Response)
	mapper := mapper.NewMock(t)
	mockScope := stats.NewMockScope("prefix")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	orchestrator := newMockOrchestrator(
		t,
		mockScope,
		mapper,
		mockSimpleUpstreamClient{
			responseChan: upstreamResponseChannel,
		},
	)
	assert.NotNil(t, orchestrator)

	// Test scenario with different request and response versions.
	// Version is different, so we expect a response.
	req := &gcp.Request{
		VersionInfo: "0",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
	}

	mockResponse := &v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		Resources: []*any.Any{
			{
				Value: []byte("lds resource"),
			},
		},
	}

	ch := make(chan gcp.Response, 1)
	r := transport.NewRequestV2(req)
	cancelWatch1 := orchestrator.CreateWatch(r, transport.NewWatchV2(ch))
	assert.Len(t, orchestrator.downstreamResponseMap.watches, 1)
	testutils.AssertSyncMapLen(t, 1, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Equal(t, "lds", key.(string))
		return true
	})

	upstreamResponseChannel <- transport.NewResponseV2(req, mockResponse)
	gotResponse := <-ch
	assertEqualResponse(t, gotResponse, mockResponse, req)

	ch2 := make(chan gcp.Response, 1)
	cancelWatch2 := orchestrator.CreateWatch(transport.NewRequestV2(req), transport.NewWatchV2(ch2))
	assert.Nil(t, cancelWatch2)
	assert.Len(t, orchestrator.downstreamResponseMap.watches, 1)
	testutils.AssertSyncMapLen(t, 1, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Equal(t, "lds", key.(string))
		return true
	})

	gotResponse = <-ch2
	assertEqualResponse(t, gotResponse, mockResponse, req)
	testutils.AssertSyncMapLen(t, 1, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Contains(t, "lds", key.(string))
		return true
	})

	cancelWatch1()
	assert.Len(t, orchestrator.downstreamResponseMap.watches, 0)

	cancel()
	orchestrator.shutdown(ctx)
	testutils.AssertSyncMapLen(t, 0, orchestrator.upstreamResponseMap.internal)
}

func TestMultipleWatchersAndUpstreams(t *testing.T) {
	upstreamResponseChannelLDS := make(chan transport.Response)
	upstreamResponseChannelCDS := make(chan transport.Response)
	mapper := mapper.NewMock(t)
	mockScope := stats.NewMockScope("prefix")
	orchestrator := newMockOrchestrator(
		t,
		mockScope,
		mapper,
		mockMultiStreamUpstreamClient{
			ldsResponseChan: upstreamResponseChannelLDS,
			cdsResponseChan: upstreamResponseChannelCDS,
			mapper:          mapper,
			t:               t,
		},
	)
	assert.NotNil(t, orchestrator)

	req1 := &gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
		Node: &v2_core.Node{
			Id: "req1",
		},
	}
	req2 := &gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
		Node: &v2_core.Node{
			Id: "req2",
		},
	}
	req3 := &gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Cluster",
		Node: &v2_core.Node{
			Id: "req3",
		},
	}

	respChannel1 := make(chan gcp.Response, 1)
	cancelWatch1 := orchestrator.CreateWatch(transport.NewRequestV2(req1), transport.NewWatchV2(respChannel1))
	assert.NotNil(t, respChannel1)

	respChannel2 := make(chan gcp.Response, 1)
	cancelWatch2 := orchestrator.CreateWatch(transport.NewRequestV2(req2), transport.NewWatchV2(respChannel2))
	assert.NotNil(t, respChannel2)

	respChannel3 := make(chan gcp.Response, 1)
	cancelWatch3 := orchestrator.CreateWatch(transport.NewRequestV2(req3), transport.NewWatchV2(respChannel3))
	assert.NotNil(t, respChannel3)

	upstreamResponseLDS := &v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		Resources: []*any.Any{
			{
				Value: []byte("lds resource"),
			},
		},
	}
	upstreamResponseCDS := &v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Cluster",
		Resources: []*any.Any{
			{
				Value: []byte("cds resource"),
			},
		},
	}

	upstreamResponseChannelLDS <- transport.NewResponseV2(req1, upstreamResponseLDS)
	upstreamResponseChannelCDS <- transport.NewResponseV2(req3, upstreamResponseCDS)

	gotResponseFromChannel1 := <-respChannel1
	gotResponseFromChannel2 := <-respChannel2
	gotResponseFromChannel3 := <-respChannel3

	assert.Equal(t, 3, len(orchestrator.downstreamResponseMap.watches))
	testutils.AssertSyncMapLen(t, 2, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Contains(t, []string{"lds", "cds"}, key.(string))
		return true
	})

	assertEqualResponse(t, gotResponseFromChannel1, upstreamResponseLDS, req1)
	assertEqualResponse(t, gotResponseFromChannel2, upstreamResponseLDS, req1)
	assertEqualResponse(t, gotResponseFromChannel3, upstreamResponseCDS, req3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	orchestrator.shutdown(ctx)
	testutils.AssertSyncMapLen(t, 0, orchestrator.upstreamResponseMap.internal)

	cancelWatch1()
	cancelWatch2()
	cancelWatch3()
	assert.Equal(t, 0, len(orchestrator.downstreamResponseMap.watches))
}

func TestUpstreamFailure(t *testing.T) {
	upstreamResponseChannel := make(chan transport.Response)
	mapper := mapper.NewMock(t)
	mockScope := stats.NewMockScope("mock_orchestrator")
	ctx, cancel := context.WithCancel(context.Background())
	orchestrator := newMockOrchestrator(
		t,
		mockScope,
		mapper,
		mockSimpleUpstreamClient{
			responseChan: upstreamResponseChannel,
		},
	)
	assert.NotNil(t, orchestrator)

	req := gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
	}
	aggregatedKey, err := mapper.GetKey(transport.NewRequestV2(&req))
	assert.NoError(t, err)

	respChannel := make(chan gcp.Response, 1)
	cancelWatch := orchestrator.CreateWatch(transport.NewRequestV2(&req), transport.NewWatchV2(respChannel))

	// close upstream channel. This happens when upstream client receives an error
	close(upstreamResponseChannel)

	r, more := <-respChannel
	assert.Nil(t, r)
	assert.True(t, more)

	cancel()
	orchestrator.shutdown(ctx)
	testutils.AssertSyncMapLen(t, 0, orchestrator.upstreamResponseMap.internal)

	cancelWatch()

	countersSnapshot := mockScope.Snapshot().Counters()
	assert.EqualValues(
		t, 1, countersSnapshot[fmt.Sprintf("mock_orchestrator.watch.errors.upstream+key=%v", aggregatedKey)].Value())
	assert.EqualValues(
		t, 1, countersSnapshot[fmt.Sprintf("mock_orchestrator.cache_evict.calls+key=%v", aggregatedKey)].Value())
	assert.EqualValues(
		t, 1, countersSnapshot[fmt.Sprintf("mock_orchestrator.cache_evict.requests_evicted+key=%v", aggregatedKey)].Value())
}

func TestNACKRequest(t *testing.T) {
	upstreamResponseChannel := make(chan transport.Response)
	mapper := mapper.NewMock(t)
	mockScope := stats.NewMockScope("mock_orchestrator")
	orchestrator := newMockOrchestrator(
		t,
		mockScope,
		mapper,
		mockSimpleUpstreamClient{
			responseChan: upstreamResponseChannel,
		},
	)
	assert.NotNil(t, orchestrator)

	// Test scenario of client sending NACK request
	req := gcp.Request{
		VersionInfo: "0",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		ErrorDetail: &status.Status{
			Message: "test_error",
		},
	}

	aggregatedKey, err := mapper.GetKey(transport.NewRequestV2(&req))
	assert.NoError(t, err)
	mockResponse := v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		Resources: []*any.Any{
			{
				Value: []byte("lds resource"),
			},
		},
	}
	watchers, err := orchestrator.cache.SetResponse(aggregatedKey, transport.NewResponseV2(&req, &mockResponse))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(watchers))

	respChannel := make(chan gcp.Response, 1)
	cancelWatch := orchestrator.CreateWatch(transport.NewRequestV2(&req), transport.NewWatchV2(respChannel))
	assert.NotNil(t, respChannel)
	assert.Equal(t, 1, len(orchestrator.downstreamResponseMap.watches))
	testutils.AssertSyncMapLen(t, 1, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Equal(t, "lds", key.(string))
		return true
	})

	// Verify stat increments counter on NACK requests
	countersSnapshot := mockScope.Snapshot().Counters()
	assert.EqualValues(
		t, 1, countersSnapshot[fmt.Sprintf("mock_orchestrator.watch.created_nack+key=%v", aggregatedKey)].Value())

	// Verify that the presence of NACK in request doesn't send an immediate response
	select {
	case <-respChannel:
		assert.Fail(t, "Nack request should block until an update is available")
	default:
	}

	// Verify that an upstream update causes a response
	mockResponse.VersionInfo = "2"
	upstreamResponseChannel <- transport.NewResponseV2(&mockRequest, &mockResponse)

	gotResponse := <-respChannel
	version, err := gotResponse.GetVersion()
	assert.NoError(t, err)
	assert.Equal(t, "2", version)

	// If we pass this point, it's safe to assume the respChannel is empty,
	// otherwise the test would block and not complete.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	orchestrator.shutdown(ctx)
	testutils.AssertSyncMapLen(t, 0, orchestrator.upstreamResponseMap.internal)

	assert.Equal(t, 1, len(orchestrator.downstreamResponseMap.watches))
	cancelWatch()
	assert.Equal(t, 0, len(orchestrator.downstreamResponseMap.watches))
}
