package handler

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/golang/protobuf/ptypes"
	"github.com/uber-go/tally"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev2 "github.com/envoyproxy/go-control-plane/pkg/resource/v2"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
)

var dateRegex = regexp.MustCompile(`"....-..-..T..:..:.....*"`)

func TestAdminServer_DefaultHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := defaultHandler([]Handler{{
		"/foo",
		"does nothing",
		http.HandlerFunc(nil),
	}})

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "admin commands are:\n  /foo: does nothing\n", rr.Body.String())
}

func TestAdminServer_DefaultHandler_NotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/not-implemented", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := defaultHandler([]Handler{{
		"/foo",
		"does nothing",
		http.HandlerFunc(nil),
	}})

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "404 page not found\n", rr.Body.String())
}

func TestAdminServer_ConfigDumpHandler(t *testing.T) {
	for _, url := range []string{"/server_info", "/server_info/"} {
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		rr := httptest.NewRecorder()
		handler := configDumpHandler(&bootstrapv1.Bootstrap{
			Server: &bootstrapv1.Server{Address: &bootstrapv1.SocketAddress{
				Address:   "127.0.0.1",
				PortValue: 9991,
			}},
			OriginServer: nil,
			Logging:      nil,
			Cache:        nil,
		})

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t,
			`{
  "server": {
    "address": {
      "address": "127.0.0.1",
      "port_value": 9991
    }
  }
}
`,
			rr.Body.String())
	}
}

func TestAdminServer_CacheDumpHandler(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamResponseChannel := make(chan *v2.DiscoveryResponse)
	mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
	client := upstream.NewMock(
		ctx,
		upstream.CallOptions{Timeout: time.Second},
		nil,
		upstreamResponseChannel,
		nil,
		nil,
		nil,
		func(m interface{}) error { return nil },
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
	assert.Equal(t, resp, *gotDiscoveryResponse)

	req, err := http.NewRequest("GET", "/cache/test_lds", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := cacheDumpHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	body := dateRegex.ReplaceAllString(rr.Body.String(), "\"\"")
	filecontents, err := ioutil.ReadFile("testdata/lds_response.json")
	assert.NoError(t, err)
	assert.Equal(t, body, string(filecontents))
	cancelWatch()
}

func TestAdminServer_CacheDumpHandler_NotFound(t *testing.T) {
	ctx := context.Background()
	mapper := mapper.NewMock(t)
	upstreamResponseChannel := make(chan *v2.DiscoveryResponse)
	mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
	client := upstream.NewMock(
		ctx,
		upstream.CallOptions{Timeout: time.Second},
		nil,
		nil,
		nil,
		nil,
		upstreamResponseChannel,
		func(m interface{}) error { return nil },
	)
	orchestrator := orchestrator.NewMock(t, mapper, client, mockScope)
	assert.NotNil(t, orchestrator)

	req, err := http.NewRequest("GET", "/cache/cds", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := cacheDumpHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "", rr.Body.String())
}

func TestAdminServer_CacheDumpHandler_EntireCache(t *testing.T) {
	for _, url := range []string{"/cache", "/cache/", "/cache/*"} {
		ctx := context.Background()
		mapper := mapper.NewMock(t)
		upstreamResponseChannelLDS := make(chan *v2.DiscoveryResponse)
		upstreamResponseChannelCDS := make(chan *v2.DiscoveryResponse)
		mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
		client := upstream.NewMock(
			ctx,
			upstream.CallOptions{Timeout: time.Second},
			nil,
			upstreamResponseChannelLDS,
			nil,
			nil,
			upstreamResponseChannelCDS,
			func(m interface{}) error { return nil },
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
		assert.Equal(t, resp, *gotDiscoveryResponse)

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
		assert.Equal(t, resp, *gotDiscoveryResponse)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := cacheDumpHandler(&orchestrator)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		body := dateRegex.ReplaceAllString(rr.Body.String(), "\"\"")
		filecontentsCds, err := ioutil.ReadFile("testdata/entire_cachev2_cds.json")
		assert.NoError(t, err)
		filecontentsLds, err := ioutil.ReadFile("testdata/entire_cachev2_lds.json")
		assert.NoError(t, err)
		assert.Contains(t, body, string(filecontentsCds))
		assert.Contains(t, body, string(filecontentsLds))

		cancelLDSWatch()
		cancelCDSWatch()
	}
}

func TestAdminServer_CacheDumpHandler_EntireCacheV3(t *testing.T) {
	for _, url := range []string{"/cache", "/cache/", "/cache/*"} {
		ctx := context.Background()
		mapper := mapper.NewMock(t)
		upstreamResponseChannelLDS := make(chan *discoveryv3.DiscoveryResponse)
		upstreamResponseChannelCDS := make(chan *discoveryv3.DiscoveryResponse)
		mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
		client := upstream.NewMockV3(
			ctx,
			upstream.CallOptions{Timeout: time.Second},
			nil,
			upstreamResponseChannelLDS,
			nil,
			nil,
			upstreamResponseChannelCDS,
			func(m interface{}) error { return nil },
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
		assert.Equal(t, resp, *gotDiscoveryResponse)

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
		assert.Equal(t, resp, *gotDiscoveryResponse)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := cacheDumpHandler(&orchestrator)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		body := dateRegex.ReplaceAllString(rr.Body.String(), "\"\"")
		filecontentsCds, err := ioutil.ReadFile("testdata/entire_cachev3_cds.json")
		assert.NoError(t, err)
		filecontentsLds, err := ioutil.ReadFile("testdata/entire_cachev3_lds.json")
		assert.NoError(t, err)
		assert.Contains(t, body, string(filecontentsCds))
		assert.Contains(t, body, string(filecontentsLds))
		cancelLDSWatch()
		cancelCDSWatch()
	}
}

func TestAdminServer_CacheDumpHandler_WildcardSuffix(t *testing.T) {
	for _, url := range []string{"/cache/t*", "/cache/tes*", "/cache/test*"} {
		ctx := context.Background()
		mapper := mapper.NewMock(t)
		upstreamResponseChannelLDS := make(chan *v2.DiscoveryResponse)
		upstreamResponseChannelCDS := make(chan *v2.DiscoveryResponse)
		mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
		client := upstream.NewMock(
			ctx,
			upstream.CallOptions{Timeout: time.Second},
			nil,
			upstreamResponseChannelLDS,
			nil,
			nil,
			upstreamResponseChannelCDS,
			func(m interface{}) error { return nil },
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
		assert.Equal(t, resp, *gotDiscoveryResponse)

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
		assert.Equal(t, resp, *gotDiscoveryResponse)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := cacheDumpHandler(&orchestrator)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		body := dateRegex.ReplaceAllString(rr.Body.String(), "\"\"")
		filecontentsCds, err := ioutil.ReadFile("testdata/entire_cachev2_cds.json")
		assert.NoError(t, err)
		filecontentsLds, err := ioutil.ReadFile("testdata/entire_cachev2_lds.json")
		assert.NoError(t, err)
		assert.Contains(t, body, string(filecontentsCds))
		assert.Contains(t, body, string(filecontentsLds))

		cancelLDSWatch()
		cancelCDSWatch()
	}
}

func TestAdminServer_CacheDumpHandler_WildcardSuffixV3(t *testing.T) {
	for _, url := range []string{"/cache/t*", "/cache/tes*", "/cache/test*"} {
		ctx := context.Background()
		mapper := mapper.NewMock(t)
		upstreamResponseChannelLDS := make(chan *discoveryv3.DiscoveryResponse)
		upstreamResponseChannelCDS := make(chan *discoveryv3.DiscoveryResponse)
		mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
		client := upstream.NewMockV3(
			ctx,
			upstream.CallOptions{Timeout: time.Second},
			nil,
			upstreamResponseChannelLDS,
			nil,
			nil,
			upstreamResponseChannelCDS,
			func(m interface{}) error { return nil },
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
		assert.Equal(t, resp, *gotDiscoveryResponse)

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
		assert.Equal(t, resp, *gotDiscoveryResponse)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := cacheDumpHandler(&orchestrator)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		body := dateRegex.ReplaceAllString(rr.Body.String(), "\"\"")
		filecontentsCds, err := ioutil.ReadFile("testdata/entire_cachev3_cds.json")
		assert.NoError(t, err)
		filecontentsLds, err := ioutil.ReadFile("testdata/entire_cachev3_lds.json")
		assert.NoError(t, err)
		assert.Contains(t, body, string(filecontentsCds))
		assert.Contains(t, body, string(filecontentsLds))

		cancelLDSWatch()
		cancelCDSWatch()
	}
}

func TestAdminServer_CacheDumpHandler_WildcardSuffix_NotFound(t *testing.T) {
	wildcardKeys := []string{"b*", "tesa*", "t*est*"}
	for _, key := range wildcardKeys {
		url := "/cache/" + key
		ctx := context.Background()
		mapper := mapper.NewMock(t)
		upstreamResponseChannelLDS := make(chan *v2.DiscoveryResponse)
		upstreamResponseChannelCDS := make(chan *v2.DiscoveryResponse)
		mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
		client := upstream.NewMock(
			ctx,
			upstream.CallOptions{Timeout: time.Second},
			nil,
			upstreamResponseChannelLDS,
			nil,
			nil,
			upstreamResponseChannelCDS,
			func(m interface{}) error { return nil },
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
		assert.Equal(t, resp, *gotDiscoveryResponse)

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
		assert.Equal(t, resp, *gotDiscoveryResponse)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := cacheDumpHandler(&orchestrator)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "", rr.Body.String())

		cancelLDSWatch()
		cancelCDSWatch()
	}
}

func TestGetParam(t *testing.T) {
	path := "127.0.0.1:6070/cache/foo_production_*"
	cacheKey := getParam(path)
	assert.Equal(t, "foo_production_*", cacheKey)
}

func TestGetParam_Empty(t *testing.T) {
	path := "127.0.0.1:6070/cache/"
	cacheKey := getParam(path)
	assert.Equal(t, "", cacheKey)
}

func TestGetParam_Malformed(t *testing.T) {
	path := "127.0.0.1:6070"
	cacheKey := getParam(path)
	assert.Equal(t, "", cacheKey)
}

func TestAdminServer_LogLevelHandler(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	logger := log.NewMock("error", &buf)
	assert.Equal(t, 0, buf.Len())

	logger.Error(ctx, "foo")
	logger.Debug(ctx, "bar")
	output := buf.String()
	assert.Contains(t, output, "foo")
	assert.NotContains(t, output, "bar")

	req, err := http.NewRequest("POST", "/log_level/debug", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := logLevelHandler(logger)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, rr.Body.String(), "Current log level: debug\n")
	logger.Debug(ctx, "bar")
	output = buf.String()
	assert.Contains(t, output, "bar")

	req, err = http.NewRequest("POST", "/log_level/info", nil)
	assert.NoError(t, err)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Current log level: info\n")
	logger.Debug(ctx, "baz")
	logger.Info(ctx, "qux")
	output = buf.String()
	assert.NotContains(t, output, "baz")
	assert.Contains(t, output, "qux")
}

func TestAdminServer_LogLevelHandler_GetLevel(t *testing.T) {
	for _, url := range []string{"/log_level", "/log_level/"} {
		var buf bytes.Buffer
		logger := log.NewMock("error", &buf)
		assert.Equal(t, 0, buf.Len())

		req, err := http.NewRequest("POST", url, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := logLevelHandler(logger)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, rr.Body.String(), "Current log level: error\n")

		req, err = http.NewRequest("POST", "/log_level/info", nil)
		assert.NoError(t, err)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		req, err = http.NewRequest("POST", url, nil)
		assert.NoError(t, err)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Current log level: info\n")
	}
}

func TestMarshalResources(t *testing.T) {
	listener := &v2.Listener{
		Name: "lds resource",
	}
	listenerAny, err := ptypes.MarshalAny(listener)
	assert.NoError(t, err)
	marshalled := marshalResources([]*any.Any{
		listenerAny,
	})
	assert.NotNil(t, marshalled)
	assert.Equal(t, 1, len(marshalled.Listeners))
	assert.Equal(t, "lds resource", marshalled.Listeners[0].(*v2.Listener).Name)
}

func TestMarshalDiscoveryResponse(t *testing.T) {
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
	marshalled := marshalDiscoveryResponse(transport.NewResponseV2(&gcp.Request{}, &resp))
	assert.NotNil(t, marshalled)
	assert.Equal(t, resp.VersionInfo, marshalled.VersionInfo)
	assert.Equal(t, resp.TypeUrl, marshalled.TypeURL)
	assert.Equal(t, listener.Name, marshalled.Resources.Listeners[0].(*v2.Listener).Name)
}
