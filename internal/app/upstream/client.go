package upstream

import (
	"context"
	"fmt"
	"math/rand"
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
	"google.golang.org/grpc/keepalive"
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
	OpenStream(transport.Request, string) (<-chan transport.Response, func())
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
	SendTimeout time.Duration

	// StreamTimeout is the time to wait before closing and reopening the gRPC stream.
	// If this is set to the Time.IsZero() value (ex: 0s), no stream timeout will be set unless
	// gRPC is blocked on Send/Recv message. If blocked, the maximum of StreamBlockedTimeout or
	// StreamTimeout is used.
	StreamTimeout time.Duration
	// StreamTimeoutJitter is an upper variance added to StreamTimeout to prevent a thundering
	// herd of streams being reopened.
	StreamTimeoutJitter time.Duration

	// ConnKeepaliveTimeout is the gRPC connection keep-alive timeout.
	// Based on https://github.com/grpc/grpc-go/blob/v1.32.x/keepalive/keepalive.go#L27-L45
	// If unset this defaults to 5 minutes.
	ConnKeepaliveTimeout time.Duration
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
	conn, err := grpc.Dial(
		url,
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			PermitWithoutStream: true,
			Time:                callOptions.ConnKeepaliveTimeout,
		}),
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

func (m *client) OpenStream(request transport.Request, aggregatedKey string) (<-chan transport.Response, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	response := make(chan transport.Response)

	go m.handleStreamsWithRetry(ctx, request, response, aggregatedKey)

	// We use context cancellation over using a separate channel for signalling stream shutdown.
	// The reason is cancelling a context tied with the stream is straightforward to signal closure.
	// Also, the shutdown function could potentially be called more than once by a caller.
	// Closing channels is not idempotent while cancelling context is idempotent.
	return response, func() {
		m.logger.With("aggregated_key", aggregatedKey).Debug(ctx, "cancelling stream")
		cancel()
	}
}

func (m *client) handleStreamsWithRetry(
	ctx context.Context,
	request transport.Request,
	respCh chan transport.Response,
	aggregatedKey string) {
	var (
		s      grpc.ClientStream
		stream transport.Stream
		err    error
		scope  tally.Scope

		// childCtx completion signifies that the the client has closed the stream.
		childCtx context.Context
		// cancel is called during shutdown or clean up operations from the caller. This will close the child context.
		cancel context.CancelFunc
	)
	for {
		if m.callOptions.StreamTimeout != 0*time.Second {
			timeout := m.callOptions.getStreamTimeout()
			m.logger.With("aggregated_key", aggregatedKey).Debug(ctx, "Connecting to upstream with timeout: %ds", timeout.Seconds())
			childCtx, cancel = context.WithTimeout(ctx, timeout)
		} else {
			m.logger.With("aggregated_key", aggregatedKey).Debug(ctx, "Connecting to upstream with timeout")
			childCtx, cancel = context.WithCancel(ctx)
		}
		select {
		case _, ok := <-m.shutdown:
			if !ok {
				cancel()
				close(respCh)
				m.logger.With("aggregated_key", aggregatedKey).Debug(ctx, "stream shutdown")
				return
			}
		case <-ctx.Done():
			// Context was cancelled, hence this is not an erroneous scenario.
			// Context is cancelled only when shutdown is called or any of the send/recv goroutines
			// error out or context timeout is reached. The shutdown can be called by the caller in
			// many cases, during app shutdown/ttl expiry, etc.
			cancel()
			close(respCh)
			m.logger.With("aggregated_key", aggregatedKey, "err", ctx.Err()).Debug(ctx, "context cancelled")
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
				handleError(ctx, m.logger, aggregatedKey, "Unsupported Type Url", func() {}, fmt.Errorf(request.GetTypeURL()))
				cancel()
				close(respCh)
				return
			}
			scope = scope.Tagged(map[string]string{metrics.TagName: aggregatedKey})
			if err != nil {
				m.logger.With("request_type", request.GetTypeURL(), "aggregated_key", aggregatedKey).Warn(ctx, "stream failed")
				scope.Counter(metrics.UpstreamStreamCreationFailure).Inc(1)
				cancel()
				continue
			}

			signal := make(chan *version, 1)
			m.logger.With("request_type", request.GetTypeURL(), "aggregated_key", aggregatedKey).Info(ctx, "stream opened")
			scope.Counter(metrics.UpstreamStreamOpened).Inc(1)
			// The xds protocol https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#ack
			// specifies that the first request be empty nonce and empty version.
			// The origin server will respond with the latest version.
			signal <- &version{nonce: "", version: ""}
			var wg sync.WaitGroup
			wg.Add(2)

			go send(childCtx, wg.Done, m.logger, cancel, stream, signal, m.callOptions, aggregatedKey)
			go recv(childCtx, wg.Done, cancel, m.logger, respCh, stream, signal, aggregatedKey)

			wg.Wait()
			scope.Counter(metrics.UpstreamStreamRetry).Inc(1)
			m.logger.With("aggregated_key", aggregatedKey).Info(ctx, "retrying")
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
	callOptions CallOptions,
	aggregatedKey string) {
	defer complete()
	for {
		select {
		case sig, ok := <-signal:
			if !ok {
				// This shouldn't happen since the signal channel is only closed during garbage
				// collection, but in the case of an erroneous error, cancelling allows recv() to
				// exit cleanly.
				logger.With("aggregated_key", aggregatedKey).Debug(ctx, "send(): chan closed")
				cancelFunc()
				return
			}
			logger.With("aggregated_key", aggregatedKey,
				"version", sig.version).Debug(ctx, "send(): sending version and nonce to upstream (ACK)")
			// Ref: https://github.com/grpc/grpc-go/issues/1229#issuecomment-302755717
			// Call SendMsg in a timeout because it can block in some cases.
			err := util.DoWithTimeout(ctx, func() error {
				return stream.SendMsg(sig.version, sig.nonce)
			}, callOptions.SendTimeout)
			if err != nil {
				handleError(ctx, logger, aggregatedKey, "send(): error", cancelFunc, err)
				return
			}
			logger.With("aggregated_key", aggregatedKey, "version", sig.version).Debug(ctx, "send(): complete")
		case <-ctx.Done():
			_ = stream.CloseSend()
			logger.With("aggregated_key", aggregatedKey).Debug(ctx, "send(): context done")
			return
		}
	}
}

// recv is an infinite loop which blocks on RecvMsg.
// The only ways to exit the goroutine is by cancelling the context or when an error occurs.
// response channel is used by the caller (orchestrator) to propagate responses to downstream clients.
// signal channel is read by send() in order to send an ACK to the upstream client.
func recv(
	ctx context.Context,
	complete func(),
	cancelFunc context.CancelFunc,
	logger log.Logger,
	response chan transport.Response,
	stream transport.Stream,
	signal chan *version,
	aggregatedKey string) {
	defer complete()
	for {
		logger.With("aggregated_key", aggregatedKey).Debug(ctx, "recv(): listening for message")
		resp, err := stream.RecvMsg()
		if err != nil {
			handleError(ctx, logger, aggregatedKey, "recv(): error", cancelFunc, err)
			return
		}

		select {
		case <-ctx.Done():
			logger.With("aggregated_key", aggregatedKey,
				"version", resp.GetPayloadVersion()).Debug(ctx, "recv(): context done")
			return
		default:
			logger.With("aggregated_key", aggregatedKey,
				"version", resp.GetPayloadVersion()).Debug(ctx, "recv(): received response from upstream")
			// recv() will be blocking if the response channel is blocked from the receiver
			// (orchestrator). Timeout after 5 seconds. TODO make receive timeout configurable
			select {
			case response <- resp:
				signal <- &version{version: resp.GetPayloadVersion(), nonce: resp.GetNonce()}
			case <-time.After(5 * time.Second):
				handleError(ctx, logger, aggregatedKey, "recv(): blocked on receiver", cancelFunc, err)
			}
		}
	}
}

func (co CallOptions) getStreamTimeout() time.Duration {
	// note: if the combined jitter and timeout duration is > 292.47 calendar years, expect
	// unsupported behavior/overflows in the below logic due to int64 max range. :)
	//
	// nanoseconds is the Time library's lowest granularity.
	timeout := co.StreamTimeout.Nanoseconds()
	if co.StreamTimeoutJitter != 0*time.Nanosecond {
		jitter := rand.Int63n(co.StreamTimeoutJitter.Nanoseconds())
		timeout = timeout + jitter
	}
	return time.Duration(timeout) * time.Nanosecond
}

func handleError(ctx context.Context, logger log.Logger, key string, errMsg string,
	cancelFunc context.CancelFunc, err error) {
	defer cancelFunc()
	select {
	case <-ctx.Done():
		// Context was cancelled, hence this is not an erroneous scenario.
		// Context is cancelled only when shutdown is called or any of the send/recv goroutines error out.
		// The shutdown can be called by the caller in many cases, during app shutdown/ttl expiry, etc
		logger.With("aggregated_key", key).Debug(ctx, "context cancelled")
	default:
		logger.With("aggregated_key", key).Error(ctx, "%s: %s", errMsg, err.Error())
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
