package transport

import (
	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	status "google.golang.org/genproto/googleapis/rpc/status"
)

// RequestVersion holds either one of the v2/v3 DiscoveryRequests
type RequestVersion struct {
	V2 *discoveryv2.DiscoveryRequest
	V3 *discoveryv3.DiscoveryRequest
}

// Locality is an interface to abstract the differences between the v2 and v3 Locality type
type Locality struct {
	Region  string
	Zone    string
	SubZone string
}

// Request is the generic interface to abstract v2 and v3 DiscoveryRequest types
type Request interface {
	GetResourceNames() []string
	GetTypeURL() string
	GetVersionInfo() string
	GetNodeID() string
	GetNodeMetadata() *structpb.Struct
	GetCluster() string
	GetError() *status.Status
	GetLocality() *Locality
	GetResponseNonce() string
	GetRaw() *RequestVersion
}
