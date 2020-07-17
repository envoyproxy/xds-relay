package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/envoyproxy/xds-relay/internal/app/metrics"

	handler "github.com/envoyproxy/xds-relay/internal/app/admin/http"
	"github.com/envoyproxy/xds-relay/internal/pkg/stats"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"github.com/envoyproxy/xds-relay/internal/pkg/util"

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	RunWithContext(ctx, cancel, bootstrapConfig, aggregationRulesConfig, logLevel, mode)
}

func RunAdminServer(ctx context.Context, adminServer *http.Server, logger log.Logger) {
	logger.With("address", adminServer.Addr).Info(ctx, "Starting admin server")
	if err := adminServer.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatal(ctx, "Failed to start admin server with ListenAndServe: %v", err)
	}
}

func RunWithContext(ctx context.Context, cancel context.CancelFunc, bootstrapConfig *bootstrapv1.Bootstrap,
	aggregationRulesConfig *aggregationv1.KeyerConfiguration, logLevel string, mode string) {
	// Initialize logger. The command line input for the log level overrides the log level set in the bootstrap config.
	// If no log level is set in the config, the default is INFO.
	// If no log path is set in the config, the default is output to stderr.
	var logger log.Logger
	writeTo := os.Stderr
	if bootstrapConfig.Logging.Path != "" {
		var err error
		if writeTo, err = os.Create(bootstrapConfig.Logging.Path); err != nil {
			panic(fmt.Errorf("failed to create/truncate log file at path %s, %v", bootstrapConfig.Logging.Path, err))
		}
	}
	if logLevel != "" {
		logger = log.New(logLevel, writeTo)
	} else {
		logger = log.New(bootstrapConfig.Logging.Level.String(), writeTo)
	}
	// Initialize metrics sink. For now we default to statsd.
	statsdPort := strconv.FormatUint(uint64(bootstrapConfig.MetricsSink.GetStatsd().Address.PortValue), 10)
	statsdAddress := net.JoinHostPort(bootstrapConfig.MetricsSink.GetStatsd().Address.Address, statsdPort)
	scope, scopeCloser, err := stats.NewScope(stats.Config{
		StatsdAddress: statsdAddress,
		RootPrefix:    bootstrapConfig.MetricsSink.GetStatsd().RootPrefix,
		FlushInterval: time.Duration(bootstrapConfig.MetricsSink.GetStatsd().FlushInterval.Nanos),
	})
	defer func() {
		if err := scopeCloser.Close(); err != nil {
			panic(err)
		}
	}()

	if err != nil {
		logger.With("error", err).Panic(ctx, "failed to configure stats client")
	}

	// Initialize upstream client.
	upstreamPort := strconv.FormatUint(uint64(bootstrapConfig.OriginServer.Address.PortValue), 10)
	upstreamAddress := net.JoinHostPort(bootstrapConfig.OriginServer.Address.Address, upstreamPort)
	// TODO: configure timeout param from bootstrap config.
	// https://github.com/envoyproxy/xds-relay/issues/55
	upstreamClient, err := upstream.New(
		ctx,
		upstreamAddress,
		upstream.CallOptions{Timeout: time.Minute},
		logger,
		scope,
	)
	if err != nil {
		logger.With("error", err).Panic(ctx, "failed to initialize upstream client")
	}

	// Initialize request aggregation mapper component.
	requestMapper := mapper.New(aggregationRulesConfig, scope)

	// Initialize orchestrator.
	orchestrator := orchestrator.New(ctx, logger, scope, requestMapper, upstreamClient, bootstrapConfig.Cache)

	// Configure admin server.
	adminPort := strconv.FormatUint(uint64(bootstrapConfig.Admin.Address.PortValue), 10)
	adminAddress := net.JoinHostPort(bootstrapConfig.Admin.Address.Address, adminPort)
	adminServer := &http.Server{
		Addr: adminAddress,
	}
	handler.RegisterHandlers(bootstrapConfig, &orchestrator, logger)

	// Start server.
	gcpServer := gcp.NewServer(ctx, orchestrator, nil)
	server := grpc.NewServer()
	serverPort := strconv.FormatUint(uint64(bootstrapConfig.Server.Address.PortValue), 10)
	serverAddress := net.JoinHostPort(bootstrapConfig.Server.Address.Address, serverPort)
	listener, err := net.Listen("tcp", serverAddress) // #nosec
	if err != nil {
		logger.With("error", err).Fatal(ctx, "failed to bind server to listener")
	}

	api.RegisterEndpointDiscoveryServiceServer(server, gcpServer)
	api.RegisterClusterDiscoveryServiceServer(server, gcpServer)
	api.RegisterRouteDiscoveryServiceServer(server, gcpServer)
	api.RegisterListenerDiscoveryServiceServer(server, gcpServer)

	if mode != "serve" {
		return
	}

	go RunAdminServer(ctx, adminServer, logger)

	registerShutdownHandler(ctx, cancel, server.GracefulStop, adminServer.Shutdown, logger, time.Second*30)
	logger.With("address", listener.Addr()).Info(ctx, "Initializing server")
	scope.SubScope(metrics.ScopeServer).Counter(metrics.ServerAlive).Inc(1)

	if err := server.Serve(listener); err != nil {
		logger.With("error", err).Fatal(ctx, "failed to initialize server")
	}
}

func registerShutdownHandler(
	ctx context.Context,
	cancel context.CancelFunc,
	gracefulStop func(),
	adminShutdown func(context.Context) error,
	logger log.Logger,
	waitTime time.Duration) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Info(ctx, "received interrupt signal:", sig.String())
		err := util.DoWithTimeout(ctx, func() error {
			logger.Info(ctx, "initiating admin server shutdown")
			if shutdownErr := adminShutdown(ctx); shutdownErr != nil {
				logger.With("error", shutdownErr).Error(ctx, "admin shutdown error: ", shutdownErr.Error())
			}
			logger.Info(ctx, "initiating grpc graceful stop")
			gracefulStop()
			_ = logger.Sync()
			cancel()
			return nil
		}, waitTime)
		if err != nil {
			logger.Error(ctx, "shutdown error: ", err.Error())
		}
	}()
}
