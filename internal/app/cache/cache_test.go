package cache

import (
	"testing"
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
)

const testKeyA = "key_A"
const testKeyB = "key_B"

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

	assert.False(t, cache.Exists(testKeyA))
}

func TestAddWatchAndFetch(t *testing.T) {
	cache, err := NewCache(1048576, 60)
	assert.NoError(t, err)

	// Simulate cache miss and setting of new watch.
	response, err := cache.Fetch(testKeyA)
	assert.EqualError(t, err, "No value found for key: key_A")
	assert.Nil(t, response)
	isStreamOpen, err := cache.AddWatch(testKeyA, testRequest)
	assert.NoError(t, err)
	assert.True(t, isStreamOpen)
	time.Sleep(1 * time.Millisecond)
	assert.True(t, cache.Exists(testKeyA))
	response, err = cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Nil(t, response)
}

func TestSetResponseAndFetch(t *testing.T) {
	cache, err := NewCache(1048576, 60)
	assert.NoError(t, err)

	// Simulate cache miss and setting of new response.
	response, err := cache.Fetch(testKeyA)
	assert.EqualError(t, err, "No value found for key: key_A")
	assert.Nil(t, response)
	openWatches, err := cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)
	assert.Nil(t, openWatches)
	time.Sleep(1 * time.Millisecond)
	response, err = cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)
}

// This test demonstrates behavior unique to ristretto caching, i.e. if Set is applied on a new key, it may take
// a few milliseconds after the call returns, but if the key already exists in the cache, the update is done instantly.
func TestAddWatchAndSetResponse(t *testing.T) {
	cache, err := NewCache(1048576, 60)
	assert.NoError(t, err)

	isStreamOpen, err := cache.AddWatch(testKeyA, testRequest)
	assert.NoError(t, err)
	assert.True(t, isStreamOpen)
	time.Sleep(1 * time.Millisecond)
	isStreamOpen, err = cache.AddWatch(testKeyA, testRequest)
	assert.NoError(t, err)
	assert.True(t, isStreamOpen)

	openWatches, err := cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(openWatches))
	assert.Equal(t, testRequest, *openWatches[0])
	assert.Equal(t, testRequest, *openWatches[1])

	response, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)
}

func TestTTL(t *testing.T) {
	cache, err := NewCache(1048576, 1)
	assert.NoError(t, err)
	_, err = cache.AddWatch(testKeyA, testRequest)
	assert.NoError(t, err)
	time.Sleep(1 * time.Millisecond)
	assert.True(t, cache.Exists(testKeyA))
	time.Sleep(1 * time.Second)
	assert.False(t, cache.Exists(testKeyA))
}

func TestMemoryOverflow(t *testing.T) {
	cache, err := NewCache(40, 60)
	assert.NoError(t, err)
	_, err = cache.AddWatch(testKeyA, testRequest)
	assert.NoError(t, err)
	time.Sleep(1 * time.Millisecond)
	assert.True(t, cache.Exists(testKeyA))
	_, err = cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)
	assert.True(t, cache.Exists(testKeyA))

	_, err = cache.AddWatch(testKeyB, testRequest)
	assert.NoError(t, err)
	time.Sleep(1 * time.Millisecond)
	assert.False(t, cache.Exists(testKeyA))
	assert.True(t, cache.Exists(testKeyB))
}
