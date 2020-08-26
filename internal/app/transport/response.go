package transport

import (
	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// Response is the generic response interface
type Response interface {
	GetPayloadVersion() string
	GetNonce() string
	GetTypeURL() string
	GetRequest() interface{}
	Get() interface{}
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
func (r *ResponseV2) GetRequest() interface{} {
	return r.req
}

// Get returns the original discovery response
func (r *ResponseV2) Get() interface{} {
	return r.resp
}
