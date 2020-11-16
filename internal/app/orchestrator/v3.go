package orchestrator

import (
	"context"
	"fmt"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
)

var _ gcpv3.Cache = &V3{}

// V3 cache wraps a shared cache
type V3 struct {
	orchestrator Orchestrator
}

// NewV3 creates a shared cache for v2 requests
func NewV3(o Orchestrator) *V3 {
	return &V3{
		orchestrator: o,
	}
}

// CreateWatch is the grpc backed xds handler
func (v *V3) CreateWatch(r *gcpv3.Request, resp chan<- gcpv3.Response) func() {
	req := transport.NewRequestV3(r)
	return v.orchestrator.CreateWatch(req, transport.NewWatchV3(resp))
}

// Fetch implements the polling method of the config cache using a non-empty request.
func (v *V3) Fetch(context.Context, *discovery.DiscoveryRequest) (gcpv3.Response, error) {
	return nil, fmt.Errorf("Not implemented")
}
