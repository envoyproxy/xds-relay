package server

import (
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Service wraps a grpc service that can be registered.
type Service struct {
	gcpv2 gcpv2.Server
}

// NewService creates a new instance of a service
func NewService(ctx context.Context, o *orchestrator.Orchestrator) Service {
	return Service{
		gcpv2: gcpv2.NewServer(ctx, orchestrator.NewV2(o), nil),
	}
}

// Register the endpoints with grpc server
func (s *Service) Register(srv *grpc.Server) {
	// You must register your new xDS resource handler here for envoymanager
	// to comply with the server interface as well as accept requests
	api.RegisterRouteDiscoveryServiceServer(srv, s.gcpv2)
	api.RegisterClusterDiscoveryServiceServer(srv, s.gcpv2)
	api.RegisterEndpointDiscoveryServiceServer(srv, s.gcpv2)
	api.RegisterListenerDiscoveryServiceServer(srv, s.gcpv2)
}
