package cache

import (
	"context"

	discovery "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
)

// GcpCache implements the cache needed for integrating with go-control-plane.
// It implements the CreateWatch function which is required to create xDS gRPC streams with xDS clients.
type GcpCache struct {
	orchestrator orchestrator.Orchestrator
}

// NewGcpCache creates an instance of GcpCache.
func NewGcpCache(orchestrator orchestrator.Orchestrator) *GcpCache {
	return &GcpCache{
		orchestrator: orchestrator,
	}
}

// CreateWatch returns a new open watch from a non-empty request.
//
// Value channel produces requested resources, once they are available.  If
// the channel is closed prior to cancellation of the watch, an unrecoverable
// error has occurred in the producer, and the consumer should close the
// corresponding stream.
//
// Cancel is an optional function to release resources in the producer. If
// provided, the consumer may call this function multiple times.
func (c *GcpCache) CreateWatch(req gcp.Request) (chan gcp.Response, func()) {
	return c.orchestrator.CreateWatch(req)
}

// Fetch implements the polling method of the config cache using a non-empty request.
func (c *GcpCache) Fetch(context.Context, discovery.DiscoveryRequest) (*gcp.Response, error) {
	return nil, nil
}

// GetStatusInfo retrieves status information for a node ID.
func (c *GcpCache) GetStatusInfo(string) gcp.StatusInfo {
	return nil
}

// GetStatusKeys retrieves node IDs for all statuses.
func (c *GcpCache) GetStatusKeys() []string {
	return nil
}
