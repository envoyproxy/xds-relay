package cache

import (
	"testing"
	"time"

	"github.com/onsi/gomega"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
)

const testKeyA = "key_A"
const testKeyB = "key_B"

func onEvict(key uint64, conflict uint64, value interface{}, cost int64) {
	// TODO: Simulate eviction behavior, e.g. closing of streams.

}

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

func TestAddRequestAndFetch(t *testing.T) {
	gomega.RegisterTestingT(t)
	cache, err := NewCache(10, 1048576, time.Second*60, onEvict)
	assert.NoError(t, err)

	// Simulate cache miss and setting of new request.
	response, err := cache.Fetch(testKeyA)
	assert.EqualError(t, err, "No value found for key: key_A")
	assert.Nil(t, response)
	isStreamOpen, err := cache.AddRequest(testKeyA, testRequest)
	assert.NoError(t, err)
	assert.True(t, isStreamOpen)
	gomega.Eventually(func() (*envoy_api_v2.DiscoveryResponse, error) {
		return cache.Fetch(testKeyA)
	}).Should(gomega.Equal((*envoy_api_v2.DiscoveryResponse)(nil)))
}

func TestSetResponseAndFetch(t *testing.T) {
	gomega.RegisterTestingT(t)
	cache, err := NewCache(10, 1048576, time.Second*60, onEvict)
	assert.NoError(t, err)

	// Simulate cache miss and setting of new response.
	response, err := cache.Fetch(testKeyA)
	assert.EqualError(t, err, "No value found for key: key_A")
	assert.Nil(t, response)
	requests, err := cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)
	assert.Nil(t, requests)
	gomega.Eventually(func() (*envoy_api_v2.DiscoveryResponse, error) {
		return cache.Fetch(testKeyA)
	}).Should(gomega.Equal(&testResponse))
}

func TestAddRequestAndSetResponse(t *testing.T) {
	gomega.RegisterTestingT(t)
	cache, err := NewCache(10, 1048576, time.Second*60, onEvict)
	assert.NoError(t, err)

	isStreamOpen, err := cache.AddRequest(testKeyA, testRequest)
	assert.NoError(t, err)
	assert.True(t, isStreamOpen)

	gomega.Eventually(func() ([]*envoy_api_v2.DiscoveryRequest, error) {
		return cache.SetResponse(testKeyA, testResponse)
	}).Should(gomega.HaveLen(1))
	requests, err := cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)
	assert.Equal(t, testRequest, *requests[0])

	response, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)
}

func TestTTL(t *testing.T) {
	cache, err := NewCache(10, 1048576, time.Millisecond*10, onEvict)
	assert.NoError(t, err)
	_, err = cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)
	time.Sleep(1 * time.Millisecond)
	response, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)
	time.Sleep(9 * time.Millisecond)
	response, err = cache.Fetch(testKeyA)
	assert.EqualError(t, err, "No value found for key: key_A")
	assert.Nil(t, response)
}

func TestMemoryOverflow(t *testing.T) {
	gomega.RegisterTestingT(t)
	cache, err := NewCache(10, 40, time.Second*60, onEvict)
	assert.NoError(t, err)
	_, err = cache.SetResponse(testKeyA, testResponse)
	assert.NoError(t, err)
	gomega.Eventually(func() (*envoy_api_v2.DiscoveryResponse, error) {
		return cache.Fetch(testKeyA)
	}).Should(gomega.Equal(&testResponse))
	_, err = cache.AddRequest(testKeyA, testRequest)
	assert.NoError(t, err)
	response, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, *response)

	_, err = cache.AddRequest(testKeyB, testRequest)
	assert.NoError(t, err)
	time.Sleep(1 * time.Millisecond)
	response, err = cache.Fetch(testKeyA)
	assert.EqualError(t, err, "No value found for key: key_A")
	assert.Nil(t, response)
	response, err = cache.Fetch(testKeyB)
	assert.NoError(t, err)
	assert.Nil(t, response)
}
