package server

import (
	"context"
	"net"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/server"
	internalcache "github.com/envoyproxy/xds-relay/internal/app/cache"
	orchestrator "github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"google.golang.org/grpc"
)

// Run instantiates a running gRPC server for accepting incoming xDS-based
// requests.
func Run() {
	logger := log.New().Sugar()

	// Cursory implementation of go-control-plane's server.
	// TODO cancel should be invoked by shutdown handlers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gcpCache := internalcache.NewGcpCache(orchestrator.NewOrchestrator())
	gcpServer := gcp.NewServer(ctx, gcpCache, nil)
	server := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080") // #nosec
	if err != nil {
		logger.Fatalw("failed to bind server to listener", "err", err)
	}

	api.RegisterEndpointDiscoveryServiceServer(server, gcpServer)
	api.RegisterClusterDiscoveryServiceServer(server, gcpServer)
	api.RegisterRouteDiscoveryServiceServer(server, gcpServer)
	api.RegisterListenerDiscoveryServiceServer(server, gcpServer)

	logger.Info("Initializing server at", listener.Addr().String())
	if err := server.Serve(listener); err != nil {
		logger.Fatalw("failed to initialize server", "err", err)
	}
}
