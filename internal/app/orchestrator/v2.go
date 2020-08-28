package orchestrator

import (
	"context"
	"fmt"

	discovery "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
)

var _ gcpv2.Cache = &V2{}

// V2 cache wraps a shared cache
type V2 struct {
	orchestrator Orchestrator
}

// NewV2 creates a shared cache for v2 requests
func NewV2(o Orchestrator) *V2 {
	return &V2{
		orchestrator: o,
	}
}

// CreateWatch is the grpc backed xds handler
func (v *V2) CreateWatch(r gcpv2.Request) (chan gcpv2.Response, func()) {
	req := transport.NewRequestV2(&r)
	w, f := v.orchestrator.CreateWatch(req)
	return w.GetChannel().V2, f
}

// Fetch implements the polling method of the config cache using a non-empty request.
func (v *V2) Fetch(context.Context, discovery.DiscoveryRequest) (gcpv2.Response, error) {
	return nil, fmt.Errorf("Not implemented")
}
