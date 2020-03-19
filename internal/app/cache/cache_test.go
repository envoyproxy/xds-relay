package cache

import (
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
)

const testKeyA = "key_A"

//const testKeyB = "key_B"

//func onEvict(key uint64, conflict uint64, value interface{}, cost int64) {
// TODO: Simulate eviction behavior, e.g. closing of streams.
//}

var testRequest = v2.DiscoveryRequest{
	VersionInfo: "version_A",
	Node: &core.Node{
		Id:      "id_A",
		Cluster: "cluster_A",
	},
	ResourceNames: []string{"resource_A"},
	TypeUrl:       "typeURL_A",
	ResponseNonce: "nonce_A",
}

var testResponse = v2.DiscoveryResponse{
	VersionInfo: "version_A",
	Resources:   []*any.Any{},
	Canary:      false,
	TypeUrl:     "typeURL_A",
	Nonce:       "nonce_A",
	ControlPlane: &core.ControlPlane{
		Identifier: "identifier_A",
	},
}

func TestAddRequestAndFetch(t *testing.T) {
	cache := NewCache(time.Second * 60)
	response, err := cache.Fetch(testKeyA)
	assert.EqualError(t, err, "no value found for key: key_A")
	assert.Nil(t, response)

	isStreamOpen, err := cache.AddRequest(testKeyA, testRequest)
	assert.NoError(t, err)
	assert.True(t, isStreamOpen)

	response, err = cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Nil(t, response)
}

func TestSetResponseAndFetch(t *testing.T) {
	cache := NewCache(time.Second * 60)

	// Simulate cache miss and setting of new response.
	response, err := cache.Fetch(testKeyA)
	assert.EqualError(t, err, "no value found for key: key_A")
	assert.Nil(t, response)

	requests, err := cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)
	assert.Nil(t, requests)

	response, err = cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)
}

func TestAddRequestAndSetResponse(t *testing.T) {
	cache := NewCache(time.Second * 60)

	isStreamOpen, err := cache.AddRequest(testKeyA, testRequest)
	assert.NoError(t, err)
	assert.True(t, isStreamOpen)

	isStreamOpen, err = cache.AddRequest(testKeyA, testRequest)
	assert.NoError(t, err)
	assert.True(t, isStreamOpen)

	requests, err := cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(requests))
	assert.Equal(t, testRequest, *requests[0])
	assert.Equal(t, testRequest, *requests[1])

	response, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)
}

// TODO: Implement test exercising TTL eviction.
//func TestTTL(t *testing.T) {
//}
