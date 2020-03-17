package cache

import (
	"testing"
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
)

const testKey = "key_A"

var testRequest = envoy_api_v2.DiscoveryRequest{
	VersionInfo: "version_A",
	Node: &core.Node{
		Id:      "id_A",
		Cluster: "cluster_A",
	},
	ResourceNames: []string{"resource_A"},
	TypeUrl:       "typeURL_A",
	ResponseNonce: "nonce_A",
}

var testResponse = envoy_api_v2.DiscoveryResponse{
	VersionInfo: "version_A",
	Resources:   []*any.Any{},
	Canary:      false,
	TypeUrl:     "typeURL_A",
	Nonce:       "nonce_A",
	ControlPlane: &core.ControlPlane{
		Identifier: "identifier_A",
	},
}

func TestExists_EmptyCache(t *testing.T) {
	cache, err := NewCache(1048576, 60)
	assert.NoError(t, err)
	assert.Equal(t, false, cache.Exists(testKey))
}

func TestSetResponseAndFetch(t *testing.T) {
	cache, err := NewCache(1048576, 60)
	assert.NoError(t, err)

	// Simulate cache miss and setting of new watch.
	assert.Equal(t, false, cache.Exists(testKey))
	response, err := cache.Fetch(testKey)
	assert.EqualError(t, err, "No value found for key: key_A")
	assert.Nil(t, response)
	isStreamOpen, err := cache.AddWatch(testKey, testRequest)
	assert.NoError(t, err)
	assert.True(t, isStreamOpen)
	time.Sleep(1 * time.Millisecond)
	assert.True(t, cache.Exists(testKey))
	response, err = cache.Fetch(testKey)
	assert.NoError(t, err)
	assert.Nil(t, response)

	// Simulate setting new response.
	openWatches, err := cache.SetResponse(testKey, testResponse)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(openWatches))
	assert.Equal(t, testRequest, *openWatches[0])
	response, err = cache.Fetch(testKey)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)
}
