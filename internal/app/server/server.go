package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/xds-relay/internal/app/metrics"
	"github.com/uber-go/tally"
	"google.golang.org/grpc/channelz/service"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	handler "github.com/envoyproxy/xds-relay/internal/app/admin/http"
	"github.com/envoyproxy/xds-relay/internal/pkg/stats"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"

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
	scope, scopeCloser, err := stats.NewRootScope(stats.Config{
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
	upstreamClient, err := upstream.New(
		ctx,
		upstreamAddress,
		upstream.CallOptions{
			SendTimeout:              time.Minute,
			UpstreamKeepaliveTimeout: bootstrapConfig.OriginServer.KeepAliveTime,
		},
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

	weboff := make(chan bool, 1)
	weboff <- true
	handler.RegisterHandlers(bootstrapConfig, &orchestrator, weboff, logger)

	defer close(weboff)

	serverPort := strconv.FormatUint(uint64(bootstrapConfig.Server.Address.PortValue), 10)
	serverAddress := net.JoinHostPort(bootstrapConfig.Server.Address.Address, serverPort)
	listener, err := net.Listen("tcp", serverAddress) // #nosec
	if err != nil {
		logger.With("error", err).Fatal(ctx, "failed to bind server to listener")
	}

	if mode != "serve" {
		return
	}

	go RunAdminServer(ctx, adminServer, logger)

	currentState := false
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	var server *grpc.Server
	var stopMutex sync.Mutex
	for {
		select {
		case state := <-weboff:
			if state == currentState {
				logger.Info(ctx, "ignoring desired state(%t) = current state(%t)", state, currentState)
				break
			}

			currentState = state
			if currentState {
				logger.Info(ctx, "working on desired state(%t), current state(%t)", state, currentState)
				stopMutex.Lock()
				server = grpc.NewServer()
				registerEndpoints(ctx, server, orchestrator)
				go startServer(ctx, logger, scope, listener, server)
				stopMutex.Unlock()
			} else {
				stopMutex.Lock()
				server.GracefulStop()
				stopMutex.Unlock()
			}
		case sig := <-sigs:
			logger.Info(ctx, "received interrupt signal:", sig.String())
			logger.Info(ctx, "initiating admin server shutdown")
			if shutdownErr := adminServer.Shutdown(ctx); shutdownErr != nil {
				logger.With("error", shutdownErr).Error(ctx, "admin shutdown error: ", shutdownErr.Error())
			}
			logger.Info(ctx, "initiating grpc graceful stop")
			cancel()
			stopMutex.Lock()
			server.GracefulStop()
			stopMutex.Unlock()
			_ = logger.Sync()
			return
		}
	}
}

func startServer(ctx context.Context, logger log.Logger, scope tally.Scope, listener net.Listener, server *grpc.Server) {
	logger.With("address", listener.Addr()).Info(ctx, "Initializing server")
	scope.SubScope(metrics.ScopeServer).Counter(metrics.ServerAlive).Inc(1)
	if err := server.Serve(listener); err != nil {
		if err == grpc.ErrServerStopped {
			logger.With("error", err).Fatal(ctx, "trying to start a stopped server")
			return
		}
		logger.With("error", err).Fatal(ctx, "failed to initialize server")
	}
}

func registerEndpoints(ctx context.Context, g *grpc.Server, o orchestrator.Orchestrator) {
	gcpv2 := gcpv2.NewServer(ctx, orchestrator.NewV2(o), nil)
	api.RegisterRouteDiscoveryServiceServer(g, gcpv2)
	api.RegisterClusterDiscoveryServiceServer(g, gcpv2)
	api.RegisterEndpointDiscoveryServiceServer(g, gcpv2)
	api.RegisterListenerDiscoveryServiceServer(g, gcpv2)

	gcpv3 := gcpv3.NewServer(ctx, orchestrator.NewV3(o), nil)
	routeservice.RegisterRouteDiscoveryServiceServer(g, gcpv3)
	clusterservice.RegisterClusterDiscoveryServiceServer(g, gcpv3)
	endpointservice.RegisterEndpointDiscoveryServiceServer(g, gcpv3)
	listenerservice.RegisterListenerDiscoveryServiceServer(g, gcpv3)

	// Use https://github.com/grpc/grpc-experiments/tree/master/gdebug to debug grpc channel issues
	service.RegisterChannelzServiceToServer(g)
}
