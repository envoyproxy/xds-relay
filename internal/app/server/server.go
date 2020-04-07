package server

import (
	"context"
	"net"
	"strconv"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	yamlproto "github.com/envoyproxy/xds-relay/internal/pkg/util"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"

	gcp "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	"google.golang.org/grpc"
)

const (
	// TODO (https://github.com/envoyproxy/xds-relay/issues/41) load from configured defaults.
	aggregationRules  = ""
	upstreamClientURL = "localhost:8080"
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

	// TODO cancel should be invoked by shutdown handlers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize upstream client.
	upstreamClient, err := upstream.NewClient(ctx, upstreamClientURL)
	if err != nil {
		logger.With("error", err).Panic(ctx, "failed to initialize upstream client")
	}

	// Initialize request aggregation mapper component.
	var aggregationRulesConfig aggregationv1.KeyerConfiguration
	err = yamlproto.FromYAMLToKeyerConfiguration(aggregationRules, &aggregationRulesConfig)
	if err != nil {
		// TODO Panic when https://github.com/envoyproxy/xds-relay/issues/41 is implemented.
		logger.With("error", err).Warn(ctx, "failed to translate aggregation rules")
	}
	requestMapper := mapper.NewMapper(&aggregationRulesConfig)

	// Initialize orchestrator.
	orchestrator := orchestrator.New(ctx, logger, requestMapper, upstreamClient)

	// Start server.
	if config.Server == nil || config.Server.Address == nil {
		logger.Fatal(ctx, "No server address given, so server was not started.")
	}
	gcpServer := gcp.NewServer(ctx, orchestrator, nil)
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
