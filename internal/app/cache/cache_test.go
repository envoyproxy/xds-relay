package cache

import (
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/golang/groupcache/lru"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
)

const testKeyA = "key_A"

const testKeyB = "key_B"

func testOnEvict(key lru.Key, value interface{}) {
	// TODO: Simulate eviction behavior, e.g. closing of streams.
}

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
	cache, err := NewCache(1, testOnEvict, time.Second*60)
	assert.NoError(t, err)

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
	cache, err := NewCache(1, testOnEvict, time.Second*60)
	assert.NoError(t, err)

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
	cache, err := NewCache(1, testOnEvict, time.Second*60)
	assert.NoError(t, err)

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

func TestMaxEntries(t *testing.T) {
	cache, err := NewCache(1, testOnEvict, time.Second*60)
	assert.NoError(t, err)

	_, err = cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)

	response, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)

	_, err = cache.AddRequest(testKeyB, testRequest)
	assert.NoError(t, err)

	response, err = cache.Fetch(testKeyA)
	assert.EqualError(t, err, "no value found for key: key_A")
	assert.Nil(t, response)

	response, err = cache.Fetch(testKeyB)
	assert.NoError(t, err)
	assert.Nil(t, response)
}

func TestTTL_Enabled(t *testing.T) {
	gomega.RegisterTestingT(t)
	cache, err := NewCache(1, testOnEvict, time.Millisecond*10)
	assert.NoError(t, err)

	_, err = cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)

	response, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)

	gomega.Eventually(func() (*v2.DiscoveryResponse, error) {
		return cache.Fetch(testKeyA)
	}).Should(gomega.BeNil())
}

func TestTTL_Disabled(t *testing.T) {
	gomega.RegisterTestingT(t)
	cache, err := NewCache(1, testOnEvict, 0)
	assert.NoError(t, err)

	_, err = cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)

	response, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)

	gomega.Consistently(func() (*v2.DiscoveryResponse, error) {
		return cache.Fetch(testKeyA)
	}).Should(gomega.Equal(&testResponse))
}

func TestTTL_Negative(t *testing.T) {
	cache, err := NewCache(1, testOnEvict, -1)
	assert.EqualError(t, err, "ttl must be nonnegative but was set to -1ns")
	assert.Nil(t, cache)
}
