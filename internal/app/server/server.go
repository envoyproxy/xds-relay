package server

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"

	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	"google.golang.org/grpc"
)

// Run instantiates a running gRPC server for accepting incoming xDS-based requests.
func Run(bootstrapConfig *bootstrapv1.Bootstrap,
	aggregationRulesConfig *aggregationv1.KeyerConfiguration,
	logLevel string, mode string) {
	// Initialize logger. The command line input for the log level overrides the log level set in the bootstrap config.
	// If no log level is set in the config, the default is INFO.
	var logger log.Logger
	if logLevel != "" {
		logger = log.New(logLevel)
	} else {
		logger = log.New(bootstrapConfig.Logging.Level.String())
	}

	// TODO cancel should be invoked by shutdown handlers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize upstream client.
	upstreamPort := strconv.FormatUint(uint64(bootstrapConfig.OriginServer.Address.PortValue), 10)
	upstreamAddress := net.JoinHostPort(bootstrapConfig.OriginServer.Address.Address, upstreamPort)
	// TODO: configure timeout param from bootstrap config.
	// https://github.com/envoyproxy/xds-relay/issues/55
	upstreamClient, err := upstream.NewClient(
		ctx,
		upstreamAddress,
		upstream.CallOptions{Timeout: time.Minute},
		logger.Named("xdsclient"),
	)
	if err != nil {
		logger.With("error", err).Panic(ctx, "failed to initialize upstream client")
	}

	// Initialize request aggregation mapper component.
	requestMapper := mapper.NewMapper(aggregationRulesConfig)

	// Initialize orchestrator.
	orchestrator := orchestrator.New(ctx, logger, requestMapper, upstreamClient, bootstrapConfig.Cache)

	// Start server.
	gcpServer := gcp.NewServer(ctx, orchestrator, nil)
	server := grpc.NewServer()
	serverPort := strconv.FormatUint(uint64(bootstrapConfig.Server.Address.PortValue), 10)
	serverAddress := net.JoinHostPort(bootstrapConfig.Server.Address.Address, serverPort)
	listener, err := net.Listen("tcp", serverAddress) // #nosec
	if err != nil {
		logger.With("err", err).Fatal(ctx, "failed to bind server to listener")
	}

	api.RegisterEndpointDiscoveryServiceServer(server, gcpServer)
	api.RegisterClusterDiscoveryServiceServer(server, gcpServer)
	api.RegisterRouteDiscoveryServiceServer(server, gcpServer)
	api.RegisterListenerDiscoveryServiceServer(server, gcpServer)

	if mode == "serve" {
		logger.With("address", listener.Addr()).Info(ctx, "Initializing server")
		if err := server.Serve(listener); err != nil {
			logger.With("err", err).Fatal(ctx, "failed to initialize server")
		}
	}
}
