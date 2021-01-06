package handler

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
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
	"github.com/uber-go/tally"
)

func TestAdminServer_VersionHandler(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamResponseChannel := make(chan *discoveryv3.DiscoveryResponse)
	mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
	client := upstream.NewMockV3(
		ctx,
		upstream.CallOptions{SendTimeout: time.Second},
		nil,
		nil,
		nil,
		nil,
		upstreamResponseChannel,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)

	respChannel, cancelWatch := orchestrator.CreateWatch(transport.NewRequestV3(&gcpv3.Request{
		TypeUrl: resourcev3.ClusterType,
		Node: &envoy_config_core_v3.Node{
			Id:      "test-1",
			Cluster: "test-prod",
		},
		ResourceNames: []string{"res"},
	}))

	clusterAny, _ := ptypes.MarshalAny(&clusterv3.Cluster{})

	upstreamResponseChannel <- &discoveryv3.DiscoveryResponse{
		VersionInfo: "123",
		TypeUrl:     resourcev3.ClusterType,
		Resources: []*any.Any{
			clusterAny,
		},
	}
	<-respChannel.GetChannel().V3

	req, err := http.NewRequest("GET", "/cache/version/test_cdsv3", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	handler := versionHandler(&orchestrator)
	handler.ServeHTTP(rr, req)

	version := &marshallable.Version{}
	err = json.Unmarshal(rr.Body.Bytes(), version)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "123", version.Version)
	cancelWatch()
}

func TestAdminServer_VersionHandler404(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamCdsResponseChannel := make(chan *v2.DiscoveryResponse)
	client := upstream.NewMock(
		ctx,
		upstream.CallOptions{SendTimeout: time.Second},
		nil,
		nil,
		nil,
		nil,
		upstreamCdsResponseChannel,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, stats.NewMockScope("mock_orchestrator"))

	req, err := http.NewRequest("GET", "/cache/version/miss", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	handler := versionHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestAdminServer_EDSDumpHandler(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamEdsResponseChannel := make(chan *v2.DiscoveryResponse)
	upstreamEdsResponseChannelV3 := make(chan *discoveryv3.DiscoveryResponse)
	client := upstream.NewMockEDS(
		ctx,
		upstream.CallOptions{SendTimeout: time.Second},
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
		ResourceNames: []string{"res"},
	}))
	respChannelv3, cancelWatchv3 := orchestrator.CreateWatch(transport.NewRequestV3(&gcpv3.Request{
		TypeUrl: resourcev3.EndpointType,
		Node: &corev3.Node{
			Id:      "test-2",
			Cluster: "test-prod2",
		},
		ResourceNames: []string{"res"},
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
		upstream.CallOptions{SendTimeout: time.Second},
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
		upstream.CallOptions{SendTimeout: time.Second},
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

// Cache Dump Handler tests
func TestAdminServer_CacheDumpHandler(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamResponseChannel := make(chan *v2.DiscoveryResponse)
	mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
	client := upstream.NewMock(
		ctx,
		upstream.CallOptions{SendTimeout: time.Second},
		nil,
		upstreamResponseChannel,
		nil,
		nil,
		nil,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)
	assert.NotNil(t, orchestrator)

	reqNode := corev2.Node{
		Id:      "test-1",
		Cluster: "test-prod",
	}
	gcpReq := gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
		Node:    &reqNode,
	}
	respChannel, cancelWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcpReq))
	assert.NotNil(t, respChannel)

	listener := &v2.Listener{
		Name: "lds resource",
	}
	listenerAny, err := ptypes.MarshalAny(listener)
	assert.NoError(t, err)
	resp := v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		Resources: []*any.Any{
			listenerAny,
		},
	}
	upstreamResponseChannel <- &resp
	gotResponse := <-respChannel.GetChannel().V2
	gotDiscoveryResponse, err := gotResponse.GetDiscoveryResponse()
	assert.NoError(t, err)
	assert.Equal(t, &resp, gotDiscoveryResponse)

	req, err := http.NewRequest("GET", cacheURL+"/test_lds", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := cacheDumpHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var actualResponse map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	assert.NoError(t, err)

	filecontentsLds, err := ioutil.ReadFile("testdata/entire_cachev2_lds.json")
	assert.NoError(t, err)
	var expectedLdsResponse map[string]interface{}
	err = json.Unmarshal(filecontentsLds, &expectedLdsResponse)
	assert.NoError(t, err)

	actualCacheResponse := actualResponse["Cache"].([]interface{})
	assert.Equal(t, len(actualCacheResponse), 1)

	assert.Equal(t, expectedLdsResponse["Key"], actualCacheResponse[0].(map[string]interface{})["Key"])
	assert.Equal(t, expectedLdsResponse["Resp"], actualCacheResponse[0].(map[string]interface{})["Resp"])
	cancelWatch()
}

func TestAdminServer_CacheDumpHandler_NotFound(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamResponseChannel := make(chan *v2.DiscoveryResponse)
	mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
	client := upstream.NewMock(
		ctx,
		upstream.CallOptions{SendTimeout: time.Second},
		nil,
		nil,
		nil,
		nil,
		upstreamResponseChannel,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)
	assert.NotNil(t, orchestrator)

	req, err := http.NewRequest("GET", cacheURL+"/cds", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := cacheDumpHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "", rr.Body.String())
}

func testAdminServerCacheDumpHelper(t *testing.T, isVerbose bool, urls []string) {
	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			ctx := context.Background()
			mapper := mapper.NewMock(t)
			upstreamResponseChannelLDS := make(chan *v2.DiscoveryResponse)
			upstreamResponseChannelCDS := make(chan *v2.DiscoveryResponse)
			mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
			client := upstream.NewMock(
				ctx,
				upstream.CallOptions{SendTimeout: time.Second},
				nil,
				upstreamResponseChannelLDS,
				nil,
				nil,
				upstreamResponseChannelCDS,
				func(m interface{}) error { return nil },
				stats.NewMockScope("mock"),
			)
			orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)
			assert.NotNil(t, orchestrator)

			req1Node := corev2.Node{
				Id:      "test-1",
				Cluster: "test-prod",
			}
			gcpReq1 := gcp.Request{
				TypeUrl: resourcev2.ListenerType,
				Node:    &req1Node,
			}
			ldsRespChannel, cancelLDSWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcpReq1))
			assert.NotNil(t, ldsRespChannel)

			req2Node := corev2.Node{
				Id:      "test-2",
				Cluster: "test-prod",
			}
			gcpReq2 := gcp.Request{
				TypeUrl: resourcev2.ClusterType,
				Node:    &req2Node,
			}
			cdsRespChannel, cancelCDSWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcpReq2))
			assert.NotNil(t, cdsRespChannel)

			listener := &v2.Listener{
				Name: "lds resource",
			}
			listenerAny, err := ptypes.MarshalAny(listener)
			assert.NoError(t, err)
			resp := v2.DiscoveryResponse{
				VersionInfo: "1",
				TypeUrl:     resourcev2.ListenerType,
				Resources: []*any.Any{
					listenerAny,
				},
			}
			upstreamResponseChannelLDS <- &resp
			gotResponse := <-ldsRespChannel.GetChannel().V2
			gotDiscoveryResponse, err := gotResponse.GetDiscoveryResponse()
			assert.NoError(t, err)
			assert.Equal(t, &resp, gotDiscoveryResponse)

			cluster := &v2.Cluster{
				Name: "cds resource",
			}
			clusterAny, err := ptypes.MarshalAny(cluster)
			assert.NoError(t, err)
			resp = v2.DiscoveryResponse{
				VersionInfo: "2",
				TypeUrl:     resourcev2.ClusterType,
				Resources: []*any.Any{
					clusterAny,
				},
			}
			upstreamResponseChannelCDS <- &resp
			gotResponse = <-cdsRespChannel.GetChannel().V2
			gotDiscoveryResponse, err = gotResponse.GetDiscoveryResponse()
			assert.NoError(t, err)
			assert.Equal(t, &resp, gotDiscoveryResponse)

			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := cacheDumpHandler(&orchestrator)

			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)

			verifyCacheOutput(t, rr, isVerbose, "testdata/entire_cachev2_cds.json", "testdata/entire_cachev2_lds.json")
			cancelLDSWatch()
			cancelCDSWatch()
		})
	}
}

func TestAdminServer_CacheDumpHandler_EntireCache(t *testing.T) {
	testAdminServerCacheDumpHelper(t, false, []string{cacheURL, cacheURL + "/", cacheURL + "/*"})
}

func TestAdminServer_CacheDumpHandler_WildcardSuffix(t *testing.T) {
	testAdminServerCacheDumpHelper(t, false, []string{cacheURL + "/t*", cacheURL + "/tes*", cacheURL + "/test*"})
}

func TestAdminServer_CacheDumpHandler_WildcardSuffix_NotFound(t *testing.T) {
	wildcardKeys := []string{"/b*", "/tesa*", "/t*est*"}
	for _, key := range wildcardKeys {
		t.Run(key, func(t *testing.T) {
			url := cacheURL + key
			ctx := context.Background()
			mapper := mapper.NewMock(t)
			upstreamResponseChannelLDS := make(chan *v2.DiscoveryResponse)
			upstreamResponseChannelCDS := make(chan *v2.DiscoveryResponse)
			mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
			client := upstream.NewMock(
				ctx,
				upstream.CallOptions{SendTimeout: time.Second},
				nil,
				upstreamResponseChannelLDS,
				nil,
				nil,
				upstreamResponseChannelCDS,
				func(m interface{}) error { return nil },
				stats.NewMockScope("mock"),
			)
			orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)
			assert.NotNil(t, orchestrator)

			req1Node := corev2.Node{
				Id:      "test-1",
				Cluster: "test-prod",
			}
			gcpReq1 := gcp.Request{
				TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
				Node:    &req1Node,
			}
			ldsRespChannel, cancelLDSWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcpReq1))
			assert.NotNil(t, ldsRespChannel)

			req2Node := corev2.Node{
				Id:      "test-2",
				Cluster: "test-prod",
			}
			gcpReq2 := gcp.Request{
				TypeUrl: "type.googleapis.com/envoy.api.v2.Cluster",
				Node:    &req2Node,
			}
			cdsRespChannel, cancelCDSWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcpReq2))
			assert.NotNil(t, cdsRespChannel)

			listener := &v2.Listener{
				Name: "lds resource",
			}
			listenerAny, err := ptypes.MarshalAny(listener)
			assert.NoError(t, err)
			resp := v2.DiscoveryResponse{
				VersionInfo: "1",
				TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
				Resources: []*any.Any{
					listenerAny,
				},
			}
			upstreamResponseChannelLDS <- &resp
			gotResponse := <-ldsRespChannel.GetChannel().V2
			gotDiscoveryResponse, err := gotResponse.GetDiscoveryResponse()
			assert.NoError(t, err)
			assert.Equal(t, &resp, gotDiscoveryResponse)

			cluster := &v2.Cluster{
				Name: "cds resource",
			}
			clusterAny, err := ptypes.MarshalAny(cluster)
			assert.NoError(t, err)
			resp = v2.DiscoveryResponse{
				VersionInfo: "2",
				TypeUrl:     "type.googleapis.com/envoy.api.v2.Cluster",
				Resources: []*any.Any{
					clusterAny,
				},
			}
			upstreamResponseChannelCDS <- &resp
			gotResponse = <-cdsRespChannel.GetChannel().V2
			gotDiscoveryResponse, err = gotResponse.GetDiscoveryResponse()
			assert.NoError(t, err)
			assert.Equal(t, &resp, gotDiscoveryResponse)

			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := cacheDumpHandler(&orchestrator)

			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "", rr.Body.String())

			cancelLDSWatch()
			cancelCDSWatch()
		})
	}
}

// V3 Cache Dump Handler tests
func testAdminServerCacheDumpHandlerV3(t *testing.T, isVerbose bool, urls []string) {
	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			ctx := context.Background()
			mapper := mapper.NewMock(t)
			upstreamResponseChannelLDS := make(chan *discoveryv3.DiscoveryResponse)
			upstreamResponseChannelCDS := make(chan *discoveryv3.DiscoveryResponse)
			mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
			client := upstream.NewMockV3(
				ctx,
				upstream.CallOptions{SendTimeout: time.Second},
				nil,
				upstreamResponseChannelLDS,
				nil,
				nil,
				upstreamResponseChannelCDS,
				func(m interface{}) error { return nil },
				stats.NewMockScope("mock"),
			)
			orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)
			assert.NotNil(t, orchestrator)

			req1Node := envoy_config_core_v3.Node{
				Id:      "test-1",
				Cluster: "test-prod",
			}
			gcpReq1 := gcpv3.Request{
				TypeUrl: resourcev3.ListenerType,
				Node:    &req1Node,
			}
			ldsRespChannel, cancelLDSWatch := orchestrator.CreateWatch(transport.NewRequestV3(&gcpReq1))
			assert.NotNil(t, ldsRespChannel)

			req2Node := envoy_config_core_v3.Node{
				Id:      "test-2",
				Cluster: "test-prod",
			}
			gcpReq2 := gcpv3.Request{
				TypeUrl: resourcev3.ClusterType,
				Node:    &req2Node,
			}
			cdsRespChannel, cancelCDSWatch := orchestrator.CreateWatch(transport.NewRequestV3(&gcpReq2))
			assert.NotNil(t, cdsRespChannel)

			listener := &v2.Listener{
				Name: "lds resource",
			}
			listenerAny, err := ptypes.MarshalAny(listener)
			assert.NoError(t, err)
			resp := discoveryv3.DiscoveryResponse{
				VersionInfo: "1",
				TypeUrl:     resourcev3.ListenerType,
				Resources: []*any.Any{
					listenerAny,
				},
			}
			upstreamResponseChannelLDS <- &resp
			gotResponse := <-ldsRespChannel.GetChannel().V3
			gotDiscoveryResponse, err := gotResponse.GetDiscoveryResponse()
			assert.NoError(t, err)
			assert.Equal(t, &resp, gotDiscoveryResponse)

			cluster := &v2.Cluster{
				Name: "cds resource",
			}
			clusterAny, err := ptypes.MarshalAny(cluster)
			assert.NoError(t, err)
			resp = discoveryv3.DiscoveryResponse{
				VersionInfo: "2",
				TypeUrl:     resourcev3.ClusterType,
				Resources: []*any.Any{
					clusterAny,
				},
			}
			upstreamResponseChannelCDS <- &resp
			gotResponse = <-cdsRespChannel.GetChannel().V3
			gotDiscoveryResponse, err = gotResponse.GetDiscoveryResponse()
			assert.NoError(t, err)
			assert.Equal(t, &resp, gotDiscoveryResponse)

			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := cacheDumpHandler(&orchestrator)

			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
			verifyCacheOutput(t, rr, isVerbose, "testdata/entire_cachev3_cds.json", "testdata/entire_cachev3_lds.json")

			cancelLDSWatch()
			cancelCDSWatch()
		})
	}
}

func TestAdminServer_CacheDumpHandler_EntireCacheV3(t *testing.T) {
	testAdminServerCacheDumpHandlerV3(t, false, []string{cacheURL, cacheURL + "/", cacheURL + "/*"})
}

func TestAdminServer_CacheDumpHandler_WildcardSuffixV3(t *testing.T) {
	testAdminServerCacheDumpHandlerV3(t, false, []string{cacheURL + "/t*", cacheURL + "/tes*", cacheURL + "/test*"})
}

func TestAdminServer_CacheDumpHandler_VerboseCache(t *testing.T) {
	testAdminServerCacheDumpHelper(t, true, []string{cacheURL + "/test*?verbose=true"})
}

func TestAdminServer_CacheDumpHandler_SuccinctCache(t *testing.T) {
	testAdminServerCacheDumpHelper(t, false, []string{cacheURL + "/*?verbose=false"})
}

// Clear Cache Handler tests
func TestAdminServer_ClearCacheHandler(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamResponseChannel := make(chan *v2.DiscoveryResponse)
	mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
	client := upstream.NewMock(
		ctx,
		upstream.CallOptions{SendTimeout: time.Second},
		nil,
		upstreamResponseChannel,
		nil,
		nil,
		nil,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)
	assert.NotNil(t, orchestrator)

	reqNode := corev2.Node{
		Id:      "test-1",
		Cluster: "test-prod",
	}
	gcpReq := gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
		Node:    &reqNode,
	}
	ldsRespChannel, cancelWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcpReq))
	assert.NotNil(t, ldsRespChannel)

	listener := &v2.Listener{
		Name: "lds resource",
	}
	listenerAny, err := ptypes.MarshalAny(listener)
	assert.NoError(t, err)
	resp := v2.DiscoveryResponse{
		VersionInfo: "1",
		TypeUrl:     "type.googleapis.com/envoy.api.v2.Listener",
		Resources: []*any.Any{
			listenerAny,
		},
	}
	upstreamResponseChannel <- &resp
	gotResponse := <-ldsRespChannel.GetChannel().V2
	gotDiscoveryResponse, err := gotResponse.GetDiscoveryResponse()
	assert.NoError(t, err)
	assert.Equal(t, &resp, gotDiscoveryResponse)

	// Assert cache has one entry before clearing
	cacheKeys, err := orchestrator.GetDownstreamAggregatedKeys()
	assert.NoError(t, err)
	assert.Equal(t, len(cacheKeys), 1)

	req, err := http.NewRequest("POST", clearURL+"/test_lds", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := clearCacheHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Assert cache has zero entries after clearing
	cacheKeys, err = orchestrator.GetDownstreamAggregatedKeys()
	assert.NoError(t, err)
	assert.Equal(t, len(cacheKeys), 0)

	cancelWatch()
}

func TestAdminServer_ClearCacheHandler_NotFound(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamResponseChannel := make(chan *v2.DiscoveryResponse)
	mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
	client := upstream.NewMock(
		ctx,
		upstream.CallOptions{SendTimeout: time.Second},
		nil,
		nil,
		nil,
		nil,
		upstreamResponseChannel,
		func(m interface{}) error { return nil },
		stats.NewMockScope("mock"),
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)
	assert.NotNil(t, orchestrator)

	req, err := http.NewRequest("POST", clearURL+"/cds", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := clearCacheHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t,
		"{\n  \"Message\": \"unable to delete entry for nonexistent key: cds\"\n}",
		rr.Body.String())
}

func testAdminServerClearCacheHelper(t *testing.T, urls []string, expectedCacheCount int) {
	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			ctx := context.Background()
			mapper := mapper.NewMock(t)
			upstreamResponseChannelLDS := make(chan *v2.DiscoveryResponse)
			upstreamResponseChannelCDS := make(chan *v2.DiscoveryResponse)
			mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
			client := upstream.NewMock(
				ctx,
				upstream.CallOptions{SendTimeout: time.Second},
				nil,
				upstreamResponseChannelLDS,
				nil,
				nil,
				upstreamResponseChannelCDS,
				func(m interface{}) error { return nil },
				stats.NewMockScope("mock"),
			)
			orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)
			assert.NotNil(t, orchestrator)

			req1Node := corev2.Node{
				Id:      "test-1",
				Cluster: "test-prod",
			}
			gcpReq1 := gcp.Request{
				TypeUrl: resourcev2.ListenerType,
				Node:    &req1Node,
			}
			ldsRespChannel, cancelLDSWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcpReq1))
			assert.NotNil(t, ldsRespChannel)

			req2Node := corev2.Node{
				Id:      "test-2",
				Cluster: "test-prod",
			}
			gcpReq2 := gcp.Request{
				TypeUrl: resourcev2.ClusterType,
				Node:    &req2Node,
			}
			cdsRespChannel, cancelCDSWatch := orchestrator.CreateWatch(transport.NewRequestV2(&gcpReq2))
			assert.NotNil(t, cdsRespChannel)

			listener := &v2.Listener{
				Name: "lds resource",
			}
			listenerAny, err := ptypes.MarshalAny(listener)
			assert.NoError(t, err)
			resp := v2.DiscoveryResponse{
				VersionInfo: "1",
				TypeUrl:     resourcev2.ListenerType,
				Resources: []*any.Any{
					listenerAny,
				},
			}
			upstreamResponseChannelLDS <- &resp
			gotResponse := <-ldsRespChannel.GetChannel().V2
			gotDiscoveryResponse, err := gotResponse.GetDiscoveryResponse()
			assert.NoError(t, err)
			assert.Equal(t, &resp, gotDiscoveryResponse)

			cluster := &v2.Cluster{
				Name: "cds resource",
			}
			clusterAny, err := ptypes.MarshalAny(cluster)
			assert.NoError(t, err)
			resp = v2.DiscoveryResponse{
				VersionInfo: "2",
				TypeUrl:     resourcev2.ClusterType,
				Resources: []*any.Any{
					clusterAny,
				},
			}
			upstreamResponseChannelCDS <- &resp
			gotResponse = <-cdsRespChannel.GetChannel().V2
			gotDiscoveryResponse, err = gotResponse.GetDiscoveryResponse()
			assert.NoError(t, err)
			assert.Equal(t, &resp, gotDiscoveryResponse)

			// Assert cache has two entries before clearing
			cacheKeys, err := orchestrator.GetDownstreamAggregatedKeys()
			assert.NoError(t, err)
			assert.Equal(t, len(cacheKeys), 2)

			req, err := http.NewRequest("POST", url, nil)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := clearCacheHandler(&orchestrator)

			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)

			// Assert cache has expectedCacheCount entries after clearing
			cacheKeys, err = orchestrator.GetDownstreamAggregatedKeys()
			assert.NoError(t, err)
			assert.Equal(t, len(cacheKeys), expectedCacheCount)

			cancelLDSWatch()
			cancelCDSWatch()
		})
	}
}

func TestAdminServer_ClearCacheHandler_EntireCache(t *testing.T) {
	testAdminServerClearCacheHelper(t, []string{clearURL, clearURL + "/", clearURL + "/*"}, 0)
}

func TestAdminServer_ClearCacheHandler_WildcardSuffix(t *testing.T) {
	testAdminServerClearCacheHelper(t, []string{clearURL + "/t*", clearURL + "/tes*", clearURL + "/test*"}, 0)
}

func TestAdminServer_ClearCacheHandler_WildcardSuffix_NotFound(t *testing.T) {
	testAdminServerClearCacheHelper(t, []string{clearURL + "/b*", clearURL + "/tesa*", clearURL + "/t*est*"}, 2)
}

// V3 Clear Cache Handler tests
func testAdminServerClearCacheHelperV3(t *testing.T, urls []string) {
	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			ctx := context.Background()
			mapper := mapper.NewMock(t)
			upstreamResponseChannelLDS := make(chan *discoveryv3.DiscoveryResponse)
			upstreamResponseChannelCDS := make(chan *discoveryv3.DiscoveryResponse)
			mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
			client := upstream.NewMockV3(
				ctx,
				upstream.CallOptions{SendTimeout: time.Second},
				nil,
				upstreamResponseChannelLDS,
				nil,
				nil,
				upstreamResponseChannelCDS,
				func(m interface{}) error { return nil },
				stats.NewMockScope("mock"),
			)
			orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)
			assert.NotNil(t, orchestrator)

			req1Node := envoy_config_core_v3.Node{
				Id:      "test-1",
				Cluster: "test-prod",
			}
			gcpReq1 := gcpv3.Request{
				TypeUrl: resourcev3.ListenerType,
				Node:    &req1Node,
			}
			ldsRespChannel, cancelLDSWatch := orchestrator.CreateWatch(transport.NewRequestV3(&gcpReq1))
			assert.NotNil(t, ldsRespChannel)

			req2Node := envoy_config_core_v3.Node{
				Id:      "test-2",
				Cluster: "test-prod",
			}
			gcpReq2 := gcpv3.Request{
				TypeUrl: resourcev3.ClusterType,
				Node:    &req2Node,
			}
			cdsRespChannel, cancelCDSWatch := orchestrator.CreateWatch(transport.NewRequestV3(&gcpReq2))
			assert.NotNil(t, cdsRespChannel)

			listener := &v2.Listener{
				Name: "lds resource",
			}
			listenerAny, err := ptypes.MarshalAny(listener)
			assert.NoError(t, err)
			resp := discoveryv3.DiscoveryResponse{
				VersionInfo: "1",
				TypeUrl:     resourcev3.ListenerType,
				Resources: []*any.Any{
					listenerAny,
				},
			}
			upstreamResponseChannelLDS <- &resp
			gotResponse := <-ldsRespChannel.GetChannel().V3
			gotDiscoveryResponse, err := gotResponse.GetDiscoveryResponse()
			assert.NoError(t, err)
			assert.Equal(t, &resp, gotDiscoveryResponse)

			cluster := &v2.Cluster{
				Name: "cds resource",
			}
			clusterAny, err := ptypes.MarshalAny(cluster)
			assert.NoError(t, err)
			resp = discoveryv3.DiscoveryResponse{
				VersionInfo: "2",
				TypeUrl:     resourcev3.ClusterType,
				Resources: []*any.Any{
					clusterAny,
				},
			}
			upstreamResponseChannelCDS <- &resp
			gotResponse = <-cdsRespChannel.GetChannel().V3
			gotDiscoveryResponse, err = gotResponse.GetDiscoveryResponse()
			assert.NoError(t, err)
			assert.Equal(t, &resp, gotDiscoveryResponse)

			// Assert cache has two entries before clearing
			cacheKeys, err := orchestrator.GetDownstreamAggregatedKeys()
			assert.NoError(t, err)
			assert.Equal(t, len(cacheKeys), 2)

			req, err := http.NewRequest("POST", url, nil)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := clearCacheHandler(&orchestrator)

			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)

			// Assert cache has zero entries after clearing
			cacheKeys, err = orchestrator.GetDownstreamAggregatedKeys()
			assert.NoError(t, err)
			assert.Equal(t, len(cacheKeys), 0)
			cancelLDSWatch()
			cancelCDSWatch()
		})
	}
}

func TestAdminServer_ClearCacheHandler_EntireCacheV3(t *testing.T) {
	testAdminServerClearCacheHelperV3(t, []string{clearURL, clearURL + "/", clearURL + "/*"})
}

func TestAdminServer_ClearCacheHandler_WildcardSuffixV3(t *testing.T) {
	testAdminServerClearCacheHelperV3(t, []string{clearURL + "/t*", clearURL + "/tes*", clearURL + "/test*"})
}

func verifyCacheOutput(t *testing.T, rr *httptest.ResponseRecorder, isVerbose bool, cdsFile string, ldsFile string) {
	var actualResponse map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	assert.NoError(t, err)

	filecontentsCds, err := ioutil.ReadFile(cdsFile)
	assert.NoError(t, err)
	var expectedCdsResponse map[string]interface{}
	err = json.Unmarshal(filecontentsCds, &expectedCdsResponse)
	assert.NoError(t, err)

	filecontentsLds, err := ioutil.ReadFile(ldsFile)
	assert.NoError(t, err)
	var expectedLdsResponse map[string]interface{}
	err = json.Unmarshal(filecontentsLds, &expectedLdsResponse)
	assert.NoError(t, err)

	actualCacheResponse := actualResponse["Cache"].([]interface{})
	assert.Equal(t, len(actualCacheResponse), 2)
	var actualLdsResponse map[string]interface{}
	var actualCdsResponse map[string]interface{}

	for _, c := range actualCacheResponse {
		if c.(map[string]interface{})["Key"] == "test_lds" {
			actualLdsResponse = c.(map[string]interface{})
		} else {
			actualCdsResponse = c.(map[string]interface{})
		}
	}

	assert.Equal(t, expectedLdsResponse["Key"], actualLdsResponse["Key"])
	assert.Equal(t, expectedCdsResponse["Key"], actualCdsResponse["Key"])
	assert.Equal(t, expectedLdsResponse["Resp"], actualLdsResponse["Resp"])
	assert.Equal(t, expectedCdsResponse["Resp"], actualCdsResponse["Resp"])
	assert.Equal(t, actualLdsResponse["NumRequests"], float64(1))
	assert.Equal(t, actualCdsResponse["NumRequests"], float64(1))
	if isVerbose {
		assert.Equal(t, len(actualLdsResponse["Requests"].([]interface{})), 1)
		assert.Equal(t, len(actualCdsResponse["Requests"].([]interface{})), 1)
	} else {
		assert.NotContains(t, actualLdsResponse, "Requests")
		assert.NotContains(t, actualCdsResponse, "Requests")
	}
	assert.NotNil(t, actualLdsResponse["ExpirationTime"])
	assert.NotNil(t, actualCdsResponse["ExpirationTime"])
}

func verifyEdsLen(t *testing.T, rr *httptest.ResponseRecorder, len int) {
	eds := &marshallable.EDS{}
	err := json.Unmarshal(rr.Body.Bytes(), eds)
	assert.NoError(t, err)
	assert.Len(t, eds.Endpoints, len)
}

func getResponse(t *testing.T, key string, o *orchestrator.Orchestrator) *httptest.ResponseRecorder {
	req, err := http.NewRequest("GET", cacheURL+"/eds/"+key, nil)
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
	req, err := http.NewRequest("GET", "/cache/keys", nil)
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
