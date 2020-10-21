package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/stats"
	"github.com/envoyproxy/xds-relay/pkg/marshallable"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
)

func TestAdminServer_KeyDumpHandler(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamLdsResponseChannel := make(chan *v2.DiscoveryResponse)
	upstreamCdsResponseChannel := make(chan *v2.DiscoveryResponse)
	client := upstream.NewMock(
		ctx,
		upstream.CallOptions{Timeout: time.Second},
		nil,
		upstreamLdsResponseChannel,
		nil,
		nil,
		upstreamCdsResponseChannel,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, stats.NewMockScope("mock_orchestrator"))

	respChannel, cancelWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
		Node: &corev2.Node{
			Id:      "test-1",
			Cluster: "test-prod",
		},
	}))

	respChannel2, cancelWatch2 := orchestrator.CreateWatch(transport.NewRequestV2(&gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Cluster",
		Node: &corev2.Node{
			Id:      "test-1",
			Cluster: "test-prod",
		},
	}))

	upstreamLdsResponseChannel <- &v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		Resources:   []*any.Any{},
	}
	upstreamCdsResponseChannel <- &v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Cluster",
		Resources:   []*any.Any{},
	}
	<-respChannel.GetChannel().V2
	<-respChannel2.GetChannel().V2

	req, err := http.NewRequest("GET", "/keys", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := keyDumpHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	keys := &marshallable.Key{}
	err = json.Unmarshal([]byte(rr.Body.String()), keys)
	assert.NoError(t, err)
	assert.Len(t, keys.Names, 2)
	cancelWatch()
	cancelWatch2()
}
