package server

import (
	"context"
	"net"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	yamlproto "github.com/envoyproxy/xds-relay/internal/pkg/util"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	"google.golang.org/grpc"
)

const (
	// TODO (https://github.com/envoyproxy/xds-relay/issues/41) load from configured defaults.
	defaultLogLevel   = "info"
	aggregationRules  = ""
	upstreamClientURL = "localhost:8080"
)

// Run instantiates a running gRPC server for accepting incoming xDS-based
// requests.
func Run() {
	logger := log.New(defaultLogLevel)

	// TODO cancel should be invoked by shutdown handlers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize upstream client.
	// TODO: configure CallOptions{} when https://github.com/envoyproxy/xds-relay/pull/49 is merged.
	upstreamClient, err := upstream.NewClient(ctx, upstreamClientURL, upstream.CallOptions{})
	if err != nil {
		logger.With("error", err).Panic(ctx, "failed to initialize upstream client")
	}

	// Initialize request aggregation mapper component.
	var config aggregationv1.KeyerConfiguration
	err = yamlproto.FromYAMLToKeyerConfiguration(aggregationRules, &config)
	if err != nil {
		// TODO Panic when https://github.com/envoyproxy/xds-relay/issues/41 is implemented.
		logger.With("error", err).Warn(ctx, "failed to translate aggregation rules")
	}
	requestMapper := mapper.NewMapper(&config)

	// Initialize orchestrator.
	orchestrator := orchestrator.New(ctx, logger, requestMapper, upstreamClient)

	// Start server.
	gcpServer := gcp.NewServer(ctx, orchestrator, nil)
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
