package upstream

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/envoyproxy/xds-relay/internal/app/metrics"
	"github.com/envoyproxy/xds-relay/internal/app/transport"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v2"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"github.com/envoyproxy/xds-relay/internal/pkg/util"
	"github.com/uber-go/tally"
	"google.golang.org/grpc"
)

// UnsupportedResourceError is a custom error for unsupported typeURL
type UnsupportedResourceError struct {
	TypeURL string
}

// Client handles the requests and responses from the origin server.
// The xds client handles each xds request on a separate stream,
// e.g. 2 different cds requests happen on 2 separate streams.
// It is the caller's responsibility to make sure there is one instance of Client overall.
type Client interface {
	// OpenStream creates a stream with the origin server
	//
	// OpenStream should be called once per aggregated key.
	// OpenStream uses one representative node identifier for the entire lifetime of the stream.
	// Therefore, it is not necessary to pass node identifiers on subsequent requests from sidecars.
	// It follows xds protocol https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol
	// to handle the version_info, nonce and error_details and applies the response to the cache.
	//
	// All responses from the origin server are sent back through the callback function.
	//
	// OpenStream uses the retry and timeout configurations to make best effort to get the responses from origin server.
	// If the timeouts are exhausted, receive fails or a irrecoverable error occurs, the response channel is closed.
	// It is the caller's responsibility to send a new request from the last known DiscoveryRequest.
	// The shutdown function should be invoked to signal stream closure.
	// The shutdown function represents the intent that a stream is supposed to be closed.
	// All goroutines that depend on the ctx object should consider ctx.Done to be related to shutdown.
	// All such scenarios need to exit cleanly and are not considered an erroneous situation.
	OpenStream(transport.Request) (<-chan transport.Response, func())
}

type client struct {
	ldsClient   v2.ListenerDiscoveryServiceClient
	rdsClient   v2.RouteDiscoveryServiceClient
	edsClient   v2.EndpointDiscoveryServiceClient
	cdsClient   v2.ClusterDiscoveryServiceClient
	ldsClientV3 listenerservice.ListenerDiscoveryServiceClient
	rdsClientV3 routeservice.RouteDiscoveryServiceClient
	edsClientV3 endpointservice.EndpointDiscoveryServiceClient
	cdsClientV3 clusterservice.ClusterDiscoveryServiceClient
	callOptions CallOptions

	logger log.Logger
	scope  tally.Scope

	shutdown <-chan struct{}
}

// CallOptions contains grpc client call options
type CallOptions struct {
	// Timeout is the time to wait on a blocking grpc SendMsg.
	Timeout time.Duration
}

type version struct {
	version string
	nonce   string
}

// New creates a grpc connection with an upstream origin server.
// Each xds relay server should create a single such upstream connection.
// grpc will handle the actual number of underlying tcp connections.
//
// The method does not block until the underlying connection is up.
// Returns immediately and connecting the server happens in background
func New(
	ctx context.Context,
	url string,
	callOptions CallOptions,
	logger log.Logger,
	scope tally.Scope,
) (Client, error) {
	namedLogger := logger.Named("upstream_client")
	namedLogger.With("address", url).Info(ctx, "Initiating upstream connection")
	subScope := scope.SubScope(metrics.ScopeUpstream)
	// TODO: configure grpc options.https://github.com/envoyproxy/xds-relay/issues/55
	conn, err := grpc.Dial(url, grpc.WithInsecure(),
		grpc.WithStreamInterceptor(ErrorClientStreamInterceptor(namedLogger, subScope)))
	if err != nil {
		return nil, err
	}

	go updateConnectivityMetric(ctx, conn, subScope)

	ldsClient := v2.NewListenerDiscoveryServiceClient(conn)
	rdsClient := v2.NewRouteDiscoveryServiceClient(conn)
	edsClient := v2.NewEndpointDiscoveryServiceClient(conn)
	cdsClient := v2.NewClusterDiscoveryServiceClient(conn)

	ldsClientV3 := listenerservice.NewListenerDiscoveryServiceClient(conn)
	rdsClientV3 := routeservice.NewRouteDiscoveryServiceClient(conn)
	edsClientV3 := endpointservice.NewEndpointDiscoveryServiceClient(conn)
	cdsClientV3 := clusterservice.NewClusterDiscoveryServiceClient(conn)

	shutdownSignal := make(chan struct{})
	go shutDown(ctx, conn, shutdownSignal)

	return &client{
		ldsClient:   ldsClient,
		rdsClient:   rdsClient,
		edsClient:   edsClient,
		cdsClient:   cdsClient,
		ldsClientV3: ldsClientV3,
		rdsClientV3: rdsClientV3,
		edsClientV3: edsClientV3,
		cdsClientV3: cdsClientV3,
		callOptions: callOptions,
		logger:      namedLogger,
		scope:       subScope,
		shutdown:    shutdownSignal,
	}, nil
}

func (m *client) OpenStream(request transport.Request) (<-chan transport.Response, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	response := make(chan transport.Response)

	go m.handleStreamsWithRetry(ctx, request, response)

	// We use context cancellation over using a separate channel for signalling stream shutdown.
	// The reason is cancelling a context tied with the stream is straightforward to signal closure.
	// Also, the shutdown function could potentially be called more than once by a caller.
	// Closing channels is not idempotent while cancelling context is idempotent.
	return response, func() { cancel() }
}

func (m *client) handleStreamsWithRetry(
	ctx context.Context,
	request transport.Request,
	respCh chan transport.Response) {
	var (
		s      grpc.ClientStream
		stream transport.Stream
		err    error
		scope  tally.Scope
	)
	for {
		childCtx, cancel := context.WithCancel(ctx)
		select {
		case _, ok := <-m.shutdown:
			if !ok {
				cancel()
				close(respCh)
				return
			}
		case <-ctx.Done():
			// Context was cancelled, hence this is not an erroneous scenario.
			// Context is cancelled only when shutdown is called or any of the send/recv goroutines error out.
			// The shutdown can be called by the caller in many cases, during app shutdown/ttl expiry, etc
			cancel()
			close(respCh)
			return
		default:
			switch request.GetTypeURL() {
			case resource.ListenerType:
				s, err = m.ldsClient.StreamListeners(childCtx)
				stream = transport.NewStreamV2(s, request, m.logger)
				scope = m.scope.SubScope(metrics.ScopeUpstreamLDS)
			case resource.ClusterType:
				s, err = m.cdsClient.StreamClusters(childCtx)
				stream = transport.NewStreamV2(s, request, m.logger)
				scope = m.scope.SubScope(metrics.ScopeUpstreamCDS)
			case resource.RouteType:
				s, err = m.rdsClient.StreamRoutes(childCtx)
				stream = transport.NewStreamV2(s, request, m.logger)
				scope = m.scope.SubScope(metrics.ScopeUpstreamRDS)
			case resource.EndpointType:
				s, err = m.edsClient.StreamEndpoints(childCtx)
				stream = transport.NewStreamV2(s, request, m.logger)
				scope = m.scope.SubScope(metrics.ScopeUpstreamEDS)
			case resourcev3.ListenerType:
				s, err = m.ldsClientV3.StreamListeners(childCtx)
				stream = transport.NewStreamV3(s, request, m.logger)
				scope = m.scope.SubScope(metrics.ScopeUpstreamLDS)
			case resourcev3.ClusterType:
				s, err = m.cdsClientV3.StreamClusters(childCtx)
				stream = transport.NewStreamV3(s, request, m.logger)
				scope = m.scope.SubScope(metrics.ScopeUpstreamCDS)
			case resourcev3.RouteType:
				s, err = m.rdsClientV3.StreamRoutes(childCtx)
				stream = transport.NewStreamV3(s, request, m.logger)
				scope = m.scope.SubScope(metrics.ScopeUpstreamRDS)
			case resourcev3.EndpointType:
				s, err = m.edsClientV3.StreamEndpoints(childCtx)
				stream = transport.NewStreamV3(s, request, m.logger)
				scope = m.scope.SubScope(metrics.ScopeUpstreamEDS)
			default:
				handleError(ctx, m.logger, "Unsupported Type Url", func() {}, fmt.Errorf(request.GetTypeURL()))
				cancel()
				close(respCh)
				return
			}
			if err != nil {
				m.logger.With("request_type", request.GetTypeURL()).Warn(ctx, "stream failed")
				scope.Counter(metrics.UpstreamStreamCreationFailure).Inc(1)
				cancel()
				continue
			}

			signal := make(chan *version, 1)
			m.logger.With("request_type", request.GetTypeURL()).Info(ctx, "stream opened")
			scope.Counter(metrics.UpstreamStreamOpened).Inc(1)
			// The xds protocol https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#ack
			// specifies that the first request be empty nonce and empty version.
			// The origin server will respond with the latest version.
			signal <- &version{nonce: "", version: ""}
			var wg sync.WaitGroup
			wg.Add(2)

			go send(childCtx, wg.Done, m.logger, cancel, stream, signal, m.callOptions)
			go recv(childCtx, wg.Done, cancel, m.logger, respCh, stream, signal)

			wg.Wait()
			close(signal)
		}
	}
}

// It is safe to assume send goroutine will not leak as long as these conditions are true:
// - SendMsg is performed with timeout.
// - send is a receiver for signal and exits when signal is closed by the owner.
// - send also exits on context cancellations.
func send(
	ctx context.Context,
	complete func(),
	logger log.Logger,
	cancelFunc context.CancelFunc,
	stream transport.Stream,
	signal chan *version,
	callOptions CallOptions) {
	defer complete()
	for {
		select {
		case sig, ok := <-signal:
			if !ok {
				return
			}
			// Ref: https://github.com/grpc/grpc-go/issues/1229#issuecomment-302755717
			// Call SendMsg in a timeout because it can block in some cases.
			err := util.DoWithTimeout(ctx, func() error {
				return stream.SendMsg(sig.version, sig.nonce)
			}, callOptions.Timeout)
			if err != nil {
				handleError(ctx, logger, "Error in SendMsg", cancelFunc, err)
				return
			}
		case <-ctx.Done():
			_ = stream.CloseSend()
			return
		}
	}
}

// recv is an infinite loop which blocks on RecvMsg.
// The only ways to exit the goroutine is by cancelling the context or when an error occurs.
func recv(
	ctx context.Context,
	complete func(),
	cancelFunc context.CancelFunc,
	logger log.Logger,
	response chan transport.Response,
	stream transport.Stream,
	signal chan *version) {
	defer complete()
	for {
		resp, err := stream.RecvMsg()
		if err != nil {
			handleError(ctx, logger, "Error in RecvMsg", cancelFunc, err)
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
			response <- resp
			signal <- &version{version: resp.GetPayloadVersion(), nonce: resp.GetNonce()}
		}
	}
}

func handleError(ctx context.Context, logger log.Logger, errMsg string, cancelFunc context.CancelFunc, err error) {
	defer cancelFunc()
	select {
	case <-ctx.Done():
		// Context was cancelled, hence this is not an erroneous scenario.
		// Context is cancelled only when shutdown is called or any of the send/recv goroutines error out.
		// The shutdown can be called by the caller in many cases, during app shutdown/ttl expiry, etc
	default:
		logger.Error(ctx, "%s: %s", errMsg, err.Error())
	}
}

// shutDown should be called in a separate goroutine.
// This is a blocking function that closes the upstream connection on context completion.
func shutDown(ctx context.Context, conn *grpc.ClientConn, signal chan struct{}) {
	<-ctx.Done()
	conn.Close()
	close(signal)
}

func (e *UnsupportedResourceError) Error() string {
	return fmt.Sprintf("Unsupported resource typeUrl: %s", e.TypeURL)
}

func updateConnectivityMetric(ctx context.Context, conn *grpc.ClientConn, scope tally.Scope) {
	for {
		isChanged := conn.WaitForStateChange(ctx, conn.GetState())
		// Based on the grpc implementation, isChanged is false only when ctx expires
		// https://github.com/grpc/grpc-go/blob/0f7e218c2cf49c7b0ca8247711b0daed2a07e79a/clientconn.go#L509-L510
		// We set an unusually high value in case of ctx expires to show the connection is dead.
		if !isChanged {
			// The possible states are https://godoc.org/google.golang.org/grpc/connectivity
			// ctx expired is not part of state enum and we've chosen 100 as a sufficiently high
			// number to avoid collision. Using -1 is possible too, but we're not sure about using
			// negative numbers in gauges.
			scope.Gauge(metrics.UpstreamConnected).Update(100)
			return
		}

		scope.Gauge(metrics.UpstreamConnected).Update(float64(conn.GetState()))
	}
}
