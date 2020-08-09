package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/envoyproxy/xds-relay/internal/pkg/stats"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/envoyproxy/xds-relay/internal/app/cache"
	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"github.com/envoyproxy/xds-relay/internal/pkg/util/testutils"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
)

type mockSimpleUpstreamClient struct {
	responseChan <-chan *v2.DiscoveryResponse
}

func (m mockSimpleUpstreamClient) OpenStream(req v2.DiscoveryRequest) (<-chan *v2.DiscoveryResponse, func(), error) {
	return m.responseChan, func() {}, nil
}

type mockMultiStreamUpstreamClient struct {
	ldsResponseChan <-chan *v2.DiscoveryResponse
	cdsResponseChan <-chan *v2.DiscoveryResponse

	t      *testing.T
	mapper mapper.Mapper
}

func (m mockMultiStreamUpstreamClient) OpenStream(
	req v2.DiscoveryRequest,
) (<-chan *v2.DiscoveryResponse, func(), error) {
	aggregatedKey, err := m.mapper.GetKey(req)
	assert.NoError(m.t, err)

	if aggregatedKey == "lds" {
		return m.ldsResponseChan, func() {}, nil
	} else if aggregatedKey == "cds" {
		return m.cdsResponseChan, func() {}, nil
	}

	m.t.Errorf("Unsupported aggregated key, %s", aggregatedKey)
	return nil, func() {}, nil
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

func assertEqualResponse(t *testing.T, got gcp.Response, expected v2.DiscoveryResponse, req gcp.Request) {
	gotDiscoveryResponse, err := got.GetDiscoveryResponse()
	assert.NoError(t, err)
	assert.Equal(t, expected, *gotDiscoveryResponse)
	assert.Equal(t, req, *got.GetRequest())
}

func TestNew(t *testing.T) {
	// Trivial test to ensure orchestrator instantiates.
	upstreamClient := upstream.NewMock(
		context.Background(),
		upstream.CallOptions{},
		nil,
		nil,
		nil,
		nil,
		nil,
		func(m interface{}) error { return nil },
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

	orchestrator := New(context.Background(), log.MockLogger, tally.NewTestScope("prefix",
		make(map[string]string)), requestMapper, upstreamClient, &cacheConfig)
	assert.NotNil(t, orchestrator)
}

func TestGoldenPath(t *testing.T) {
	upstreamResponseChannel := make(chan *v2.DiscoveryResponse)
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

	req := gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
	}
	aggregatedKey, err := mapper.GetKey(req)
	assert.NoError(t, err)

	respChannel, cancelWatch := orchestrator.CreateWatch(req)
	countersSnapshot := mockScope.Snapshot().Counters()
	assert.EqualValues(
		t, 1, countersSnapshot[fmt.Sprintf("mock_orchestrator.watch.created+key=%v", aggregatedKey)].Value())
	assert.NotNil(t, respChannel)
	assert.Equal(t, 1, len(orchestrator.downstreamResponseMap.responseChannels))
	testutils.AssertSyncMapLen(t, 1, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Equal(t, "lds", key.(string))
		return true
	})

	resp := v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		Resources: []*any.Any{
			{
				Value: []byte("lds resource"),
			},
		},
	}
	upstreamResponseChannel <- &resp

	gotResponse := <-respChannel
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
	assert.Equal(t, 0, len(orchestrator.downstreamResponseMap.responseChannels))
}

func TestCachedResponse(t *testing.T) {
	upstreamResponseChannel := make(chan *v2.DiscoveryResponse)
	mapper := mapper.NewMock(t)
	mockScope := stats.NewMockScope("prefix")
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
	req := gcp.Request{
		VersionInfo: "0",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
	}

	aggregatedKey, err := mapper.GetKey(req)
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
	watchers, err := orchestrator.cache.SetResponse(aggregatedKey, mockResponse)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(watchers))

	respChannel, cancelWatch := orchestrator.CreateWatch(req)
	assert.NotNil(t, respChannel)
	assert.Equal(t, 1, len(orchestrator.downstreamResponseMap.responseChannels))
	testutils.AssertSyncMapLen(t, 1, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Equal(t, "lds", key.(string))
		return true
	})

	gotResponse := <-respChannel
	assertEqualResponse(t, gotResponse, mockResponse, req)

	// Attempt pushing a more recent response from upstream.
	resp := v2.DiscoveryResponse{
		VersionInfo: "2",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		Resources: []*any.Any{
			{
				Value: []byte("some other lds resource"),
			},
		},
	}

	upstreamResponseChannel <- &resp
	gotResponse = <-respChannel
	assertEqualResponse(t, gotResponse, resp, req)
	testutils.AssertSyncMapLen(t, 1, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Contains(t, "lds", key.(string))
		return true
	})

	// Test scenario with same request and response version.
	// We expect a watch to be open but no response.
	req2 := gcp.Request{
		VersionInfo: "2",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
	}

	respChannel2, cancelWatch2 := orchestrator.CreateWatch(req2)
	assert.NotNil(t, respChannel2)
	assert.Equal(t, 2, len(orchestrator.downstreamResponseMap.responseChannels))
	testutils.AssertSyncMapLen(t, 1, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Contains(t, "lds", key.(string))
		return true
	})

	// If we pass this point, it's safe to assume the respChannel2 is empty,
	// otherwise the test would block and not complete.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	orchestrator.shutdown(ctx)
	testutils.AssertSyncMapLen(t, 0, orchestrator.upstreamResponseMap.internal)

	cancelWatch()
	assert.Equal(t, 1, len(orchestrator.downstreamResponseMap.responseChannels))
	cancelWatch2()
	assert.Equal(t, 0, len(orchestrator.downstreamResponseMap.responseChannels))
}

func TestMultipleWatchersAndUpstreams(t *testing.T) {
	upstreamResponseChannelLDS := make(chan *v2.DiscoveryResponse)
	upstreamResponseChannelCDS := make(chan *v2.DiscoveryResponse)
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

	req1 := gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
		Node: &v2_core.Node{
			Id: "req1",
		},
	}
	req2 := gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
		Node: &v2_core.Node{
			Id: "req2",
		},
	}
	req3 := gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Cluster",
		Node: &v2_core.Node{
			Id: "req3",
		},
	}

	respChannel1, cancelWatch1 := orchestrator.CreateWatch(req1)
	assert.NotNil(t, respChannel1)
	respChannel2, cancelWatch2 := orchestrator.CreateWatch(req2)
	assert.NotNil(t, respChannel2)
	respChannel3, cancelWatch3 := orchestrator.CreateWatch(req3)
	assert.NotNil(t, respChannel3)

	upstreamResponseLDS := v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		Resources: []*any.Any{
			{
				Value: []byte("lds resource"),
			},
		},
	}
	upstreamResponseCDS := v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Cluster",
		Resources: []*any.Any{
			{
				Value: []byte("cds resource"),
			},
		},
	}

	upstreamResponseChannelLDS <- &upstreamResponseLDS
	upstreamResponseChannelCDS <- &upstreamResponseCDS

	gotResponseFromChannel1 := <-respChannel1
	gotResponseFromChannel2 := <-respChannel2
	gotResponseFromChannel3 := <-respChannel3

	assert.Equal(t, 3, len(orchestrator.downstreamResponseMap.responseChannels))
	testutils.AssertSyncMapLen(t, 2, orchestrator.upstreamResponseMap.internal)
	orchestrator.upstreamResponseMap.internal.Range(func(key, val interface{}) bool {
		assert.Contains(t, []string{"lds", "cds"}, key.(string))
		return true
	})

	assertEqualResponse(t, gotResponseFromChannel1, upstreamResponseLDS, req1)
	assertEqualResponse(t, gotResponseFromChannel2, upstreamResponseLDS, req2)
	assertEqualResponse(t, gotResponseFromChannel3, upstreamResponseCDS, req3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	orchestrator.shutdown(ctx)
	testutils.AssertSyncMapLen(t, 0, orchestrator.upstreamResponseMap.internal)

	cancelWatch1()
	cancelWatch2()
	cancelWatch3()
	assert.Equal(t, 0, len(orchestrator.downstreamResponseMap.responseChannels))
}
