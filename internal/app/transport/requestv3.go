package transport

import (
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	status "google.golang.org/genproto/googleapis/rpc/status"
)

var _ Request = &RequestV3{}

// NewRequestV3 creates a Request objects which wraps v2.DiscoveryRequest
func NewRequestV3(r *discoveryv3.DiscoveryRequest) *RequestV3 {
	return &RequestV3{
		r: r,
	}
}

// RequestV3 is the v2.DiscoveryRequest impl of Request
type RequestV3 struct {
	r *discoveryv3.DiscoveryRequest
}

// GetResourceNames gets the ResourceNames
func (r *RequestV3) GetResourceNames() []string {
	return r.r.GetResourceNames()
}

// GetVersionInfo gets the version info
func (r *RequestV3) GetVersionInfo() string {
	return r.r.GetVersionInfo()
}

// GetNodeID gets the node id
func (r *RequestV3) GetNodeID() string {
	return r.r.GetNode().GetId()
}

// GetNodeMetadata gets version-agnostic node metadata
func (r *RequestV3) GetNodeMetadata() *structpb.Struct {
	if r.r.GetNode() != nil {
		return r.r.GetNode().GetMetadata()
	}
	return nil
}

// GetCluster gets the cluster name
func (r *RequestV3) GetCluster() string {
	return r.r.GetNode().GetCluster()
}

// GetError gets the error details
func (r *RequestV3) GetError() *status.Status {
	return r.r.GetErrorDetail()
}

// GetTypeURL gets the error details
func (r *RequestV3) GetTypeURL() string {
	return r.r.GetTypeUrl()
}

// GetRegion gets the error details
func (r *RequestV3) GetRegion() string {
	return r.r.GetNode().GetLocality().GetRegion()
}

// GetZone gets the error details
func (r *RequestV3) GetZone() string {
	return r.r.GetNode().GetLocality().GetZone()
}

// GetSubZone gets the error details
func (r *RequestV3) GetSubZone() string {
	return r.r.GetNode().GetLocality().GetSubZone()
}

// GetRaw gets the error details
func (r *RequestV3) GetRaw() *RequestVersion {
	return &RequestVersion{V3: r.r}
}

// GetResponseNonce gets the error details
func (r *RequestV3) GetResponseNonce() string {
	return r.r.GetResponseNonce()
}

// CreateWatch creates a versioned Watch
func (r *RequestV3) CreateWatch() Watch {
	return newWatchV3()
}
