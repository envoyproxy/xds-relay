package transport

import (
	"testing"

	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/envoyproxy/xds-relay/internal/pkg/stats"
)

const (
	nodeID          = "1"
	resourceName    = "route"
	resourceVersion = "version"
	cluster         = "cluster"
	region          = "region"
	zone            = "zone"
	subzone         = "subzone"
)

var requestV2 = discoveryv2.DiscoveryRequest{
	Node: &corev2.Node{
		Id: nodeID,
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{"a": nil},
		},
		Cluster: cluster,
		Locality: &corev2.Locality{
			Region:  region,
			Zone:    zone,
			SubZone: subzone,
		},
	},
	ResourceNames: []string{resourceName},
	TypeUrl:       "typeUrl",
	VersionInfo:   resourceVersion,
	ErrorDetail:   &status.Status{Code: 0},
	ResponseNonce: "1",
}
var requestV3 = discoveryv3.DiscoveryRequest{
	Node: &corev3.Node{
		Id: nodeID,
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{"a": nil},
		},
		Cluster: cluster,
		Locality: &corev3.Locality{
			Region:  region,
			Zone:    zone,
			SubZone: subzone,
		},
	},
	ResourceNames: []string{resourceName},
	TypeUrl:       "typeUrl",
	VersionInfo:   resourceVersion,
	ErrorDetail:   &status.Status{Code: 0},
	ResponseNonce: "1",
}

func TestGetResourceNames(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	assert.Equal(t, requestv2.GetResourceNames(), requestv3.GetResourceNames())
}

func TestGetTypeURL(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	assert.Equal(t, requestv2.GetTypeURL(), requestv3.GetTypeURL())
}

func TestGetVersionInfo(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	assert.Equal(t, requestv2.GetVersionInfo(), requestv3.GetVersionInfo())
}

func TestNodeId(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	assert.Equal(t, requestv2.GetNodeID(), requestv3.GetNodeID())
}

func TestGetNodeMetadata(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	assert.Equal(t, requestv2.GetNodeMetadata(), requestv3.GetNodeMetadata())
}

func TestGetCluster(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	assert.Equal(t, requestv2.GetCluster(), requestv3.GetCluster())
}

func TestGetError(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	assert.Equal(t, requestv2.GetError(), requestv3.GetError())
}

func TestGetLocality(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	assert.Equal(t, requestv2.GetLocality(), requestv3.GetLocality())
}

func TestGetResponseNonce(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	assert.Equal(t, requestv2.GetResponseNonce(), requestv3.GetResponseNonce())
}

func TestGetRaw(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	assert.Equal(t, requestv2.GetRaw().V2, &requestV2)
	assert.Equal(t, requestv3.GetRaw().V3, &requestV3)
}

func TestCreateWatch(t *testing.T) {
	requestv2 := NewRequestV2(&requestV2)
	requestv3 := NewRequestV3(&requestV3)
	scope := stats.NewMockScope("mockwatch")
	assert.NotNil(t, requestv2.CreateWatch(scope).GetChannel().V2)
	assert.NotNil(t, requestv2.CreateWatch(scope).GetChannel().V2)
	assert.Nil(t, requestv2.CreateWatch(scope).GetChannel().V3)
	assert.Nil(t, requestv3.CreateWatch(scope).GetChannel().V2)
}
