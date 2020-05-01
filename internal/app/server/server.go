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

func RunAdminServer(ctx context.Context, logger log.Logger) {
	adminServer := &http.Server{
		//TODO(lisalu): Possibly make below address configurable.
		Addr:              "127.0.0.1:6070",
		Handler:           nil,
		TLSConfig:         nil,
		ReadTimeout:       0,
		ReadHeaderTimeout: 0,
		WriteTimeout:      0,
		IdleTimeout:       0,
		MaxHeaderBytes:    0,
		TLSNextProto:      nil,
		ConnState:         nil,
		ErrorLog:          nil,
		BaseContext:       nil,
		ConnContext:       nil,
	}
	defaultHandler := func(w http.ResponseWriter, req *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		fmt.Fprintf(w, "hello world!")
	}
	http.HandleFunc("/", defaultHandler)

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := adminServer.Shutdown(context.Background()); err != nil {
			//TODO(lisalu): Use diff. context/logger.
			logger.Error(ctx, "shutdown error: ", err)
		}
	}()
	if err := adminServer.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatal(ctx, "HTTP server ListenAndServe: %v", err)
	}
}

func RunWithContext(ctx context.Context, cancel context.CancelFunc, bootstrapConfig *bootstrapv1.Bootstrap,
	aggregationRulesConfig *aggregationv1.KeyerConfiguration, logLevel string, mode string) {
	// Initialize logger. The command line input for the log level overrides the log level set in the bootstrap config.
	// If no log level is set in the config, the default is INFO.
	var logger log.Logger
	if logLevel != "" {
		logger = log.New(logLevel)
	} else {
		logger = log.New(bootstrapConfig.Logging.Level.String())
	}

	// Initialize upstream client.
	upstreamPort := strconv.FormatUint(uint64(bootstrapConfig.OriginServer.Address.PortValue), 10)
	upstreamAddress := net.JoinHostPort(bootstrapConfig.OriginServer.Address.Address, upstreamPort)
	// TODO: configure timeout param from bootstrap config.
	// https://github.com/envoyproxy/xds-relay/issues/55
	upstreamClient, err := upstream.NewClient(
		ctx,
		upstreamAddress,
		upstream.CallOptions{Timeout: time.Minute},
		logger,
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

	if mode != "serve" {
		return
	}

	go RunAdminServer(ctx, logger)

	registerShutdownHandler(ctx, cancel, server.GracefulStop, logger, time.Second*30)
	logger.With("address", listener.Addr()).Info(ctx, "Initializing server")
	if err := server.Serve(listener); err != nil {
		logger.With("err", err).Fatal(ctx, "failed to initialize server")
	}
}

func registerShutdownHandler(
	ctx context.Context,
	cancel context.CancelFunc,
	gracefulStop func(),
	logger log.Logger,
	waitTime time.Duration) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Info(ctx, "received interrupt signal:", sig.String())
		err := util.DoWithTimeout(ctx, func() error {
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
