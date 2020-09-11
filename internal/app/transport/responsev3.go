package transport

import (
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes/any"
)

var _ Response = &ResponseV3{}

// ResponseV3 is the v3.DiscoveryRequest impl of Response
type ResponseV3 struct {
	req  *discoveryv3.DiscoveryRequest
	resp *discoveryv3.DiscoveryResponse
}

// NewResponseV3 creates a new instance of wrapped Response
func NewResponseV3(req *discoveryv3.DiscoveryRequest, resp *discoveryv3.DiscoveryResponse) Response {
	return &ResponseV3{
		req:  req,
		resp: resp,
	}
}

// GetPayloadVersion gets the api version
func (r *ResponseV3) GetPayloadVersion() string {
	return r.resp.GetVersionInfo()
}

// GetTypeURL returns the typeUrl for the resource
func (r *ResponseV3) GetTypeURL() string {
	return r.resp.GetTypeUrl()
}

// GetNonce gets the response nonce
func (r *ResponseV3) GetNonce() string {
	return r.resp.GetNonce()
}

// GetRequest returns the original request associated with the response
func (r *ResponseV3) GetRequest() *RequestVersion {
	return &RequestVersion{V3: r.req}
}

// Get returns the original discovery response
func (r *ResponseV3) Get() *ResponseVersion {
	return &ResponseVersion{V3: r.resp}
}

// GetResources returns the original discovery response
func (r *ResponseV3) GetResources() []*any.Any {
	return r.resp.GetResources()
}
