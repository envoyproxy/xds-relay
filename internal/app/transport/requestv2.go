package transport

import (
	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	structpb "github.com/golang/protobuf/ptypes/struct"
	status "google.golang.org/genproto/googleapis/rpc/status"
)

var _ Request = &RequestV2{}

// NewRequestV2 creates a Request objects which wraps v2.DiscoveryRequest
func NewRequestV2(r *discoveryv2.DiscoveryRequest) *RequestV2 {
	return &RequestV2{
		r: r,
	}
}

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

// GetLocality gets the node locality
func (r *RequestV2) GetLocality() *Locality {
	locality := r.r.GetNode().GetLocality()
	return &Locality{
		Region:  locality.GetRegion(),
		Zone:    locality.GetZone(),
		SubZone: locality.GetSubZone(),
	}
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
