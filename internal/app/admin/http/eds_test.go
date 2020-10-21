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
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/stats"
	"github.com/envoyproxy/xds-relay/pkg/marshallable"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
)

func TestAdminServer_EDSDumpHandler(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamEdsResponseChannel := make(chan *v2.DiscoveryResponse)
	client := upstream.NewMock(
		ctx,
		upstream.CallOptions{Timeout: time.Second},
		nil,
		nil,
		nil,
		upstreamEdsResponseChannel,
		nil,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, stats.NewMockScope("mock_orchestrator"))

	respChannel, cancelWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment",
		Node: &corev2.Node{
			Id:      "test-1",
			Cluster: "test-prod1",
		},
	}))

	endpoint := &v2.ClusterLoadAssignment{
		ClusterName: "test-prod1",
		Endpoints: []*endpoint.LocalityLbEndpoints{
			{
				LbEndpoints: []*endpoint.LbEndpoint{
					{
						HostIdentifier: getEndpoint("0.0.0.0"),
					},
					{
						HostIdentifier: getEndpoint("0.0.0.1"),
					},
				},
			},
		},
	}
	endpointAny, _ := ptypes.MarshalAny(endpoint)
	upstreamEdsResponseChannel <- &v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment",
		Resources: []*any.Any{
			endpointAny,
		},
	}

	got := <-respChannel.GetChannel().V2
	r, _ := got.GetDiscoveryResponse()
	assert.Len(t, r.Resources, 1)
	rr := getResponse(t, &orchestrator)
	assert.Equal(t, http.StatusOK, rr.Code)
	verifyEdsLen(t, rr, 2)
	cancelWatch()
}

func TestAdminServer_EDSDumpHandler404(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamEdsResponseChannel := make(chan *v2.DiscoveryResponse)
	client := upstream.NewMock(
		ctx,
		upstream.CallOptions{Timeout: time.Second},
		nil,
		nil,
		nil,
		upstreamEdsResponseChannel,
		nil,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, stats.NewMockScope("mock_orchestrator"))

	rr := getResponse(t, &orchestrator)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func verifyEdsLen(t *testing.T, rr *httptest.ResponseRecorder, len int) {
	eds := &marshallable.EDS{}
	err := json.Unmarshal(rr.Body.Bytes(), eds)
	assert.NoError(t, err)
	assert.Len(t, eds.Endpoints, len)
}

func getResponse(t *testing.T, o *orchestrator.Orchestrator) *httptest.ResponseRecorder {
	req, err := http.NewRequest("GET", "/eds/eds", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	handler := edsDumpHandler(o)

	handler.ServeHTTP(rr, req)
	return rr
}

func getEndpoint(address string) *endpoint.LbEndpoint_Endpoint {
	return &endpoint.LbEndpoint_Endpoint{
		Endpoint: &endpoint.Endpoint{
			Address: &corev2.Address{
				Address: &corev2.Address_SocketAddress{
					SocketAddress: &corev2.SocketAddress{
						Address: address,
					},
				},
			},
		},
	}
}
