package transport

import (
	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes/any"
)

// ResponseVersion holds either one of the v2/v3 DiscoveryRequests
type ResponseVersion struct {
	V2 *discoveryv2.DiscoveryResponse
	V3 *discoveryv3.DiscoveryResponse
}

// Response is the generic response interface
type Response interface {
	GetPayloadVersion() string
	GetNonce() string
	GetTypeURL() string
	GetRequest() *RequestVersion
	Get() *ResponseVersion
	GetResources() []*any.Any
}
