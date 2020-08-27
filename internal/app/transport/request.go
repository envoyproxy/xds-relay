package transport

import (
	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	structpb "github.com/golang/protobuf/ptypes/struct"
	status "google.golang.org/genproto/googleapis/rpc/status"
)

// RequestVersion holds either one of the v2/v3 DiscoveryRequests
type RequestVersion struct {
	V2 *discoveryv2.DiscoveryRequest
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
	IsNodeEmpty() bool
	IsEmptyLocality() bool
	GetRegion() string
	GetZone() string
	GetSubZone() string
	GetResponseNonce() string
	GetRaw() *RequestVersion
	CreateWatch() Watch
}

// NewRequestV2 creates a Request objects which wraps v2.DiscoveryRequest
func NewRequestV2(r *discoveryv2.DiscoveryRequest) *RequestV2 {
	return &RequestV2{
		r: r,
	}
}

var _ Request = &RequestV2{}

// RequestV2 is the v2.DiscoveryRequest impl of Request
type RequestV2 struct {
	r *discoveryv2.DiscoveryRequest
}

// GetResourceNames gets the ResourceNames
func (r *RequestV2) GetResourceNames() []string {
	return r.r.GetResourceNames()
}

// GetVersionInfo gets the version info
func (r *RequestV2) GetVersionInfo() string {
	return r.r.GetVersionInfo()
}

// GetNodeID gets the node id
func (r *RequestV2) GetNodeID() string {
	return r.r.GetNode().GetId()
}

// GetNodeMetadata gets version-agnostic node metadata
func (r *RequestV2) GetNodeMetadata() *structpb.Struct {
	if r.r.GetNode() != nil {
		return r.r.GetNode().GetMetadata()
	}
	return nil
}

// GetCluster gets the cluster name
func (r *RequestV2) GetCluster() string {
	return r.r.GetNode().GetCluster()
}

// GetError gets the error details
func (r *RequestV2) GetError() *status.Status {
	return r.r.GetErrorDetail()
}

// GetTypeURL gets the error details
func (r *RequestV2) GetTypeURL() string {
	return r.r.GetTypeUrl()
}

// IsNodeEmpty gets the error details
func (r *RequestV2) IsNodeEmpty() bool {
	return r.r.Node == nil
}

// IsEmptyLocality gets the error details
func (r *RequestV2) IsEmptyLocality() bool {
	return r.r.GetNode().Locality == nil
}

// GetRegion gets the error details
func (r *RequestV2) GetRegion() string {
	return r.r.GetNode().GetLocality().GetRegion()
}

// GetZone gets the error details
func (r *RequestV2) GetZone() string {
	return r.r.GetNode().GetLocality().GetZone()
}

// GetSubZone gets the error details
func (r *RequestV2) GetSubZone() string {
	return r.r.GetNode().GetLocality().GetSubZone()
}

// GetRaw gets the error details
func (r *RequestV2) GetRaw() *RequestVersion {
	return &RequestVersion{V2: r.r}
}

// GetResponseNonce gets the error details
func (r *RequestV2) GetResponseNonce() string {
	return r.r.GetResponseNonce()
}

// CreateWatch creates a versioned Watch
func (r *RequestV2) CreateWatch() Watch {
	return newWatchV2()
}
