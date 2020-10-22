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
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev2 "github.com/envoyproxy/go-control-plane/pkg/resource/v2"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
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

func TestAdminServer_EDSDumpHandlerV2(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamEdsResponseChannel := make(chan *v2.DiscoveryResponse)
	upstreamEdsResponseChannelV3 := make(chan *discoveryv3.DiscoveryResponse)
	client := upstream.NewMockEDS(
		ctx,
		upstream.CallOptions{Timeout: time.Second},
		nil,
		upstreamEdsResponseChannelV3,
		upstreamEdsResponseChannel,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, stats.NewMockScope("mock_orchestrator"))

	respChannel, cancelWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcp.Request{
		TypeUrl: resourcev2.EndpointType,
		Node: &corev2.Node{
			Id:      "test-1",
			Cluster: "test-prod1",
		},
	}))
	respChannelv3, cancelWatchv3 := orchestrator.CreateWatch(transport.NewRequestV3(&gcpv3.Request{
		TypeUrl: resourcev3.EndpointType,
		Node: &corev3.Node{
			Id:      "test-2",
			Cluster: "test-prod2",
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
	endpointv3 := &endpointv3.ClusterLoadAssignment{
		ClusterName: "test-prod2",
		Endpoints: []*endpointv3.LocalityLbEndpoints{
			{
				LbEndpoints: []*endpointv3.LbEndpoint{
					{
						HostIdentifier: getEndpointV3("0.0.0.2"),
					},
					{
						HostIdentifier: getEndpointV3("0.0.0.3"),
					},
				},
			},
		},
	}

	endpointAny, _ := ptypes.MarshalAny(endpoint)
	upstreamEdsResponseChannel <- &v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     resourcev2.EndpointType,
		Resources: []*any.Any{
			endpointAny,
		},
	}
	endpointAnyV3, _ := ptypes.MarshalAny(endpointv3)
	upstreamEdsResponseChannelV3 <- &discoveryv3.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     resourcev3.EndpointType,
		Resources: []*any.Any{
			endpointAnyV3,
		},
	}

	<-respChannel.GetChannel().V2
	<-respChannelv3.GetChannel().V3

	rr := getResponse(t, "eds", &orchestrator)
	assert.Equal(t, http.StatusOK, rr.Code)
	verifyEdsLen(t, rr, 2)

	rr = getResponse(t, "edsv3", &orchestrator)
	assert.Equal(t, http.StatusOK, rr.Code)
	verifyEdsLen(t, rr, 2)

	cancelWatch()
	cancelWatchv3()
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

	rr := getResponse(t, "eds", &orchestrator)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

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

	verifyKeyLen(t, 0, &orchestrator)

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

	verifyKeyLen(t, 2, &orchestrator)
	cancelWatch()
	cancelWatch2()
}

func verifyEdsLen(t *testing.T, rr *httptest.ResponseRecorder, len int) {
	eds := &marshallable.EDS{}
	err := json.Unmarshal(rr.Body.Bytes(), eds)
	assert.NoError(t, err)
	assert.Len(t, eds.Endpoints, len)
}

func getResponse(t *testing.T, key string, o *orchestrator.Orchestrator) *httptest.ResponseRecorder {
	req, err := http.NewRequest("GET", "/eds/"+key, nil)
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

func getEndpointV3(address string) *endpointv3.LbEndpoint_Endpoint {
	return &endpointv3.LbEndpoint_Endpoint{
		Endpoint: &endpointv3.Endpoint{
			Address: &corev3.Address{
				Address: &corev3.Address_SocketAddress{
					SocketAddress: &corev3.SocketAddress{
						Address: address,
					},
				},
			},
		},
	}
}

func verifyKeyLen(t *testing.T, len int, o *orchestrator.Orchestrator) {
	req, err := http.NewRequest("GET", "/keys", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	handler := keyDumpHandler(o)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	keys := &marshallable.Key{}
	err = json.Unmarshal(rr.Body.Bytes(), keys)
	assert.NoError(t, err)
	assert.Len(t, keys.Names, len)
}
