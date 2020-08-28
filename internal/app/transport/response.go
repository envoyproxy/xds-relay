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
	GetVersionInfo() string
	GetResources() []*any.Any
}

var _ Response = &ResponseV2{}

// ResponseV2 is the v2.DiscoveryRequest impl of Response
type ResponseV2 struct {
	req  *discoveryv2.DiscoveryRequest
	resp *discoveryv2.DiscoveryResponse
}

// NewResponseV2 creates a new instance of wrapped Response
func NewResponseV2(req *discoveryv2.DiscoveryRequest, resp *discoveryv2.DiscoveryResponse) Response {
	return &ResponseV2{
		req:  req,
		resp: resp,
	}
}

// GetPayloadVersion gets the api version
func (r *ResponseV2) GetPayloadVersion() string {
	return r.resp.GetVersionInfo()
}

// GetTypeURL returns the typeUrl for the resource
func (r *ResponseV2) GetTypeURL() string {
	return r.resp.GetTypeUrl()
}

// GetNonce gets the response nonce
func (r *ResponseV2) GetNonce() string {
	return r.resp.GetNonce()
}

// GetRequest returns the original request associated with the response
func (r *ResponseV2) GetRequest() *RequestVersion {
	return &RequestVersion{V2: r.req}
}

// Get returns the original discovery response
func (r *ResponseV2) Get() *ResponseVersion {
	return &ResponseVersion{V2: r.resp}
}

// GetVersionInfo returns the original discovery response
func (r *ResponseV2) GetVersionInfo() string {
	return r.resp.GetVersionInfo()
}

// GetResources returns the original discovery response
func (r *ResponseV2) GetResources() []*any.Any {
	return r.resp.GetResources()
}

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

// GetVersionInfo returns the original discovery response
func (r *ResponseV3) GetVersionInfo() string {
	return r.resp.GetVersionInfo()
}

// GetResources returns the original discovery response
func (r *ResponseV3) GetResources() []*any.Any {
	return r.resp.GetResources()
}
