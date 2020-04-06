package server

import (
	"context"
	"net"
	"strconv"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	"google.golang.org/grpc"
)

// Run instantiates a running gRPC server for accepting incoming xDS-based
// requests.
func Run(config *bootstrapv1.Bootstrap, logLevel string) {
	// The command line input for the log level overrides the log level set in the bootstrap config. If no log level is
	// set in the config, the default is INFO.
	var logger log.Logger
	if logLevel != "" {
		logger = log.New(logLevel)
	} else {
		logger = log.New(config.Logging.Level.String())
	}

	// Cursory implementation of go-control-plane's server.
	// TODO cancel should be invoked by shutdown handlers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	gcpServer := gcp.NewServer(ctx, snapshotCache, nil)

	if config.Server == nil || config.Server.Address == nil {
		logger.Fatal(ctx, "No server address given, so server was not started.")
	}
	server := grpc.NewServer()
	port := strconv.FormatUint(uint64(config.Server.Address.PortValue), 10)
	address := net.JoinHostPort(config.Server.Address.Address, port)
	listener, err := net.Listen("tcp", address) // #nosec
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
