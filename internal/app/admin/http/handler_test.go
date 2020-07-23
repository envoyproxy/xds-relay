package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/envoyproxy/xds-relay/internal/app/upstream"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/golang/protobuf/ptypes"
	"github.com/uber-go/tally"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
)

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
	req, err := http.NewRequest("GET", "/server_info", nil)
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

	gcpReq := gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
	}
	respChannel, cancelWatch := orchestrator.CreateWatch(gcpReq)
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
	gotResponse := <-respChannel
	gotDiscoveryResponse, err := gotResponse.GetDiscoveryResponse()
	assert.NoError(t, err)
	assert.Equal(t, resp, *gotDiscoveryResponse)

	req, err := http.NewRequest("GET", "/cache/lds", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := cacheDumpHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `{
  "Resp": {
    "VersionInfo": "1",
    "Resources": {
      "Endpoints": null,
      "Clusters": null,
      "Routes": null,
      "Listeners": [
        {
          "name": "lds resource"
        }
      ],
      "Secrets": null,
      "Runtimes": null,
      "Unmarshalled": null
    },
    "Canary": false,
    "TypeURL": "type.googleapis.com/envoy.api.v2.Listener",
    "Nonce": "",
    "ControlPlane": null
  },
  "Requests": [
    {
      "type_url": "type.googleapis.com/envoy.api.v2.Listener"
    }
  ],
  "ExpirationTime": "`)
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
	assert.Equal(t, "no resource for key cds found in cache.\n", rr.Body.String())
}

func TestAdminServer_CacheDumpHandler_EntireCache(t *testing.T) {
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

	gcpReq := gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
	}
	ldsRespChannel, cancelLDSWatch := orchestrator.CreateWatch(gcpReq)
	assert.NotNil(t, ldsRespChannel)

	gcpReq = gcp.Request{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Cluster",
	}
	cdsRespChannel, cancelCDSWatch := orchestrator.CreateWatch(gcpReq)
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
	gotResponse := <-ldsRespChannel
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
	gotResponse = <-cdsRespChannel
	gotDiscoveryResponse, err = gotResponse.GetDiscoveryResponse()
	assert.NoError(t, err)
	assert.Equal(t, resp, *gotDiscoveryResponse)

	req, err := http.NewRequest("GET", "/cache/", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := cacheDumpHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `lds: {
  "Resp": {
    "VersionInfo": "1",
    "Resources": {
      "Endpoints": null,
      "Clusters": null,
      "Routes": null,
      "Listeners": [
        {
          "name": "lds resource"
        }
      ],
      "Secrets": null,
      "Runtimes": null,
      "Unmarshalled": null
    },
    "Canary": false,
    "TypeURL": "type.googleapis.com/envoy.api.v2.Listener",
    "Nonce": "",
    "ControlPlane": null
  },
  "Requests": [
    {
      "type_url": "type.googleapis.com/envoy.api.v2.Listener"
    }
  ],
  "ExpirationTime": "`)
	assert.Contains(t, rr.Body.String(), `cds: {
  "Resp": {
    "VersionInfo": "2",
    "Resources": {
      "Endpoints": null,
      "Clusters": [
        {
          "name": "cds resource",
          "ClusterDiscoveryType": null,
          "LbConfig": null
        }
      ],
      "Routes": null,
      "Listeners": null,
      "Secrets": null,
      "Runtimes": null,
      "Unmarshalled": null
    },
    "Canary": false,
    "TypeURL": "type.googleapis.com/envoy.api.v2.Cluster",
    "Nonce": "",
    "ControlPlane": null
  },
  "Requests": [
    {
      "type_url": "type.googleapis.com/envoy.api.v2.Cluster"
    }
  ],
  "ExpirationTime": "`)
	cancelLDSWatch()
	cancelCDSWatch()
}

func TestGetParam(t *testing.T) {
	path := "127.0.0.1:6070/cache/foo_production_*"
	cacheKey, err := getParam(path)
	assert.NoError(t, err)
	assert.Equal(t, "foo_production_*", cacheKey)
}

func TestGetParam_Empty(t *testing.T) {
	path := "127.0.0.1:6070/cache/"
	cacheKey, err := getParam(path)
	assert.NoError(t, err)
	assert.Equal(t, "", cacheKey)
}

func TestGetParam_Malformed(t *testing.T) {
	path := "127.0.0.1:6070"
	cacheKey, err := getParam(path)
	assert.Error(t, err, "")
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
	logger.Debug(ctx, "bar")
	output = buf.String()
	assert.Contains(t, output, "bar")

	req, err = http.NewRequest("POST", "/log_level/info", nil)
	assert.NoError(t, err)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	logger.Debug(ctx, "baz")
	logger.Info(ctx, "qux")
	output = buf.String()
	assert.NotContains(t, output, "baz")
	assert.Contains(t, output, "qux")
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
	assert.Equal(t, "lds resource", marshalled.Listeners[0].Name)
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
	marshalled := marshalDiscoveryResponse(&resp)
	assert.NotNil(t, marshalled)
	assert.Equal(t, resp.VersionInfo, marshalled.VersionInfo)
	assert.Equal(t, resp.TypeUrl, marshalled.TypeURL)
	assert.Equal(t, listener.Name, marshalled.Resources.Listeners[0].Name)
}
