package cache

import (
	"testing"
	"time"

	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/envoyproxy/xds-relay/internal/pkg/stats"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/groupcache/lru"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"
)

const (
	testKeyA = "key_A"
	testKeyB = "key_B"
)

type panicValues struct {
	key    lru.Key
	reason string
}

func testOnEvict(key string, value Resource) {
	// TODO: Simulate eviction behavior, e.g. closing of streams.
	panic(panicValues{
		key:    key,
		reason: "testOnEvict called",
	})
}

var testRequestA = v2.DiscoveryRequest{
	VersionInfo: "version_A",
	Node: &core.Node{
		Id:      "id_A",
		Cluster: "cluster_A",
	},
	ResourceNames: []string{"resource_A"},
	TypeUrl:       "typeURL_A",
	ResponseNonce: "nonce_A",
}

var testRequestB = v2.DiscoveryRequest{
	VersionInfo: "version_B",
	Node: &core.Node{
		Id:      "id_B",
		Cluster: "cluster_B",
	},
	ResourceNames: []string{"resource_B"},
	TypeUrl:       "typeURL_B",
	ResponseNonce: "nonce_B",
}

var testDiscoveryResponse = v2.DiscoveryResponse{
	VersionInfo: "version_A",
	Resources: []*any.Any{
		{
			Value: []byte("test"),
		},
	},
	Canary:  false,
	TypeUrl: "typeURL_A",
	Nonce:   "nonce_A",
	ControlPlane: &core.ControlPlane{
		Identifier: "identifier_A",
	},
}

var testResource = Resource{
	Resp: transport.NewResponseV2(&v2.DiscoveryRequest{}, &testDiscoveryResponse),
}

func TestSetResponseAndFetch(t *testing.T) {
	cache, err := NewCache(1, testOnEvict, time.Second*60, log.MockLogger, stats.NewMockScope("cache"))
	assert.NoError(t, err)

	// Simulate cache miss and setting of new response.
	resource, err := cache.Fetch(testKeyA)
	assert.EqualError(t, err, "no value found for key: key_A")
	assert.Nil(t, resource)

	err = cache.SetResponse(testKeyA, testResource.Resp)
	assert.NoError(t, err)

	resource, err = cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, &testDiscoveryResponse, resource.Resp.Get().V2)
}

func TestMaxEntries(t *testing.T) {
	cache, err := NewCache(1, testOnEvict, time.Second*60, log.MockLogger, stats.NewMockScope("cache"))
	assert.NoError(t, err)

	err = cache.SetResponse(testKeyA, testResource.Resp)
	assert.NoError(t, err)

	resource, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, &testDiscoveryResponse, resource.Resp.Get().V2)

	assert.PanicsWithValue(t, panicValues{
		key:    testKeyA,
		reason: "testOnEvict called",
	}, func() {
		err = cache.SetResponse(testKeyB, testResource.Resp)
		assert.NoError(t, err)
	})

	resource, err = cache.Fetch(testKeyA)
	assert.EqualError(t, err, "no value found for key: key_A")
	assert.Nil(t, resource)

	resource, err = cache.Fetch(testKeyB)
	assert.NoError(t, err)
	assert.NotNil(t, resource.Resp)
}

func TestTTL_Enabled(t *testing.T) {
	cache, err := NewCache(1, testOnEvict, time.Millisecond*10, log.MockLogger, stats.NewMockScope("cache"))
	assert.NoError(t, err)

	err = cache.SetResponse(testKeyA, testResource.Resp)
	assert.NoError(t, err)

	resource, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, &testDiscoveryResponse, resource.Resp.Get().V2)

	time.Sleep(time.Millisecond * 10)
	assert.PanicsWithValue(t, panicValues{
		key:    testKeyA,
		reason: "testOnEvict called",
	}, func() {
		resource, err = cache.Fetch(testKeyA)
		assert.NoError(t, err)
		assert.Nil(t, resource)
	})

	resource, err = cache.Fetch(testKeyA)
	assert.EqualError(t, err, "no value found for key: key_A")
	assert.Nil(t, resource)
}

func TestTTL_Disabled(t *testing.T) {
	gomega.RegisterTestingT(t)
	cache, err := NewCache(1, testOnEvict, 0, log.MockLogger, stats.NewMockScope("cache"))
	assert.NoError(t, err)

	err = cache.SetResponse(testKeyA, testResource.Resp)
	assert.NoError(t, err)

	resource, err := cache.Fetch(testKeyA)
	assert.NoError(t, err)
	assert.Equal(t, &testDiscoveryResponse, resource.Resp.Get().V2)

	gomega.Consistently(func() (*Resource, error) {
		return cache.Fetch(testKeyA)
	}).Should(gomega.Equal(&testResource))
}

func TestTTL_Negative(t *testing.T) {
	cache, err := NewCache(1, testOnEvict, -1, log.MockLogger, stats.NewMockScope("cache"))
	assert.EqualError(t, err, "ttl must be nonnegative but was set to -1ns")
	assert.Nil(t, cache)
}

func TestIsExpired(t *testing.T) {
	var resource Resource

	// The expiration time is 0, meaning TTL is disabled, so the resource is not considered expired.
	resource.ExpirationTime = time.Time{}
	assert.False(t, resource.isExpired(time.Now()))

	resource.ExpirationTime = time.Now()
	assert.False(t, resource.isExpired(time.Time{}))

	resource.ExpirationTime = time.Now()
	assert.True(t, resource.isExpired(resource.ExpirationTime.Add(1)))
}

func TestGetExpirationTime(t *testing.T) {
	var c cache

	c.ttl = 0
	assert.Equal(t, time.Time{}, c.getExpirationTime(time.Now()))

	c.ttl = time.Second
	currentTime := time.Date(0, 0, 0, 0, 0, 1, 0, time.UTC)
	expirationTime := time.Date(0, 0, 0, 0, 0, 2, 0, time.UTC)
	assert.Equal(t, expirationTime, c.getExpirationTime(currentTime))
}
