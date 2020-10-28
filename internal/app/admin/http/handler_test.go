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
	"github.com/envoyproxy/xds-relay/internal/pkg/stats"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/golang/protobuf/ptypes"
	"github.com/uber-go/tally"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
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
		true,
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
		true,
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

	req, err := http.NewRequest("GET", "/cache/cds", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := cacheDumpHandler(&orchestrator)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "", rr.Body.String())
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
