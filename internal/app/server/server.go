package server

import (
	"context"
	"net"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	gcp "github.com/envoyproxy/go-control-plane/pkg/server"
	"google.golang.org/grpc"
)

const defaultLogLevel = "info" // TODO make configurable

// Run instantiates a running gRPC server for accepting incoming xDS-based
// requests.
func Run() {
	logger := log.New(defaultLogLevel)

	// Cursory implementation of go-control-plane's server.
	// TODO cancel should be invoked by shutdown handlers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	gcpServer := gcp.NewServer(ctx, snapshotCache, nil)
	server := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080") // #nosec
	if err != nil {
		logger.With("err", err).Fatal(ctx, "failed to bind server to listener")
	}

	api.RegisterEndpointDiscoveryServiceServer(server, gcpServer)
	api.RegisterClusterDiscoveryServiceServer(server, gcpServer)
	api.RegisterRouteDiscoveryServiceServer(server, gcpServer)
	api.RegisterListenerDiscoveryServiceServer(server, gcpServer)

	logger.With("address", listener.Addr()).Info(ctx, "Initializing server")
	if err := server.Serve(listener); err != nil {
		logger.With("err", err).Fatal(ctx, "failed to initialize server")
	}
}
