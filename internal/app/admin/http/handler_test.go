package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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

type mockSimpleUpstreamClient struct {
	responseChan <-chan *v2.DiscoveryResponse
}

func (m mockSimpleUpstreamClient) OpenStream(req v2.DiscoveryRequest) (<-chan *v2.DiscoveryResponse, func(), error) {
	return m.responseChan, func() {}, nil
}

func TestAdminServer_CacheDumpHandler(t *testing.T) {
	upstreamResponseChannel := make(chan *v2.DiscoveryResponse)
	mapper := mapper.NewMock(t)
	mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
	orchestrator := orchestrator.NewMock(t, mapper,
		mockSimpleUpstreamClient{responseChan: upstreamResponseChannel}, mockScope)
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
	upstreamResponseChannel := make(chan *v2.DiscoveryResponse)
	mapper := mapper.NewMock(t)
	mockScope := tally.NewTestScope("mock_orchestrator", make(map[string]string))
	orchestrator := orchestrator.NewMock(t, mapper,
		mockSimpleUpstreamClient{responseChan: upstreamResponseChannel}, mockScope)
	assert.NotNil(t, orchestrator)

	req, err := http.NewRequest("GET", "/cache/cds", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := cacheDumpHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "no resource for key cds found in cache.\n", rr.Body.String())
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
