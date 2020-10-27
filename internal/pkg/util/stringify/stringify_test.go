package stringify

import (
	"testing"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/stretchr/testify/assert"
)

func TestInterfaceToString(t *testing.T) {
	testRequest := v2.DiscoveryRequest{
		VersionInfo: "version_A",
		Node: &core.Node{
			Id:      "id_A",
			Cluster: "cluster_A",
		},
		ResourceNames: []string{"resource_A"},
		TypeUrl:       "typeURL_A",
		ResponseNonce: "nonce_A",
	}
	requestString, err := InterfaceToString(&testRequest)
	assert.NoError(t, err)
	assert.Equal(t, `{
  "version_info": "version_A",
  "node": {
    "id": "id_A",
    "cluster": "cluster_A",
    "UserAgentVersionType": null
  },
  "resource_names": [
    "resource_A"
  ],
  "type_url": "typeURL_A",
  "response_nonce": "nonce_A"
}`, requestString)
}

func TestInterfaceToString_UnknownTypeError(t *testing.T) {
	testRequestMap := make(map[*v2.DiscoveryRequest]bool)
	requestMapString, err := InterfaceToString(testRequestMap)
	assert.EqualError(t, err, "json: unsupported type: map[*envoy_api_v2.DiscoveryRequest]bool")
	assert.Equal(t, "", requestMapString)
}
