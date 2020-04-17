package upstream

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
)

// NewMockClient creates a mock implementation for testing
func NewMockClient(
	ctx context.Context,
	ldsClient v2.ListenerDiscoveryServiceClient,
	rdsClient v2.RouteDiscoveryServiceClient,
	edsClient v2.EndpointDiscoveryServiceClient,
	cdsClient v2.ClusterDiscoveryServiceClient,
	callOptions CallOptions) Client {
	return &client{
		ldsClient:   ldsClient,
		rdsClient:   rdsClient,
		edsClient:   edsClient,
		cdsClient:   cdsClient,
		callOptions: callOptions,
		logger:      log.New("panic"),
	}
}
