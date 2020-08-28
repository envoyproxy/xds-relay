package upstream

import (
	"context"
	"fmt"
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
	OpenStream(transport.Request) (<-chan transport.Response, func(), error)
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

	ldsClient := v2.NewListenerDiscoveryServiceClient(conn)
	rdsClient := v2.NewRouteDiscoveryServiceClient(conn)
	edsClient := v2.NewEndpointDiscoveryServiceClient(conn)
	cdsClient := v2.NewClusterDiscoveryServiceClient(conn)

	ldsClientV3 := listenerservice.NewListenerDiscoveryServiceClient(conn)
	rdsClientV3 := routeservice.NewRouteDiscoveryServiceClient(conn)
	edsClientV3 := endpointservice.NewEndpointDiscoveryServiceClient(conn)
	cdsClientV3 := clusterservice.NewClusterDiscoveryServiceClient(conn)

	go shutDown(ctx, conn)

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
	}, nil
}

func (m *client) OpenStream(request transport.Request) (<-chan transport.Response, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	var (
		s      grpc.ClientStream
		stream transport.Stream
		err    error
		scope  tally.Scope
	)
	switch request.GetTypeURL() {
	case resource.ListenerType:
		s, err = m.ldsClient.StreamListeners(ctx)
		stream = transport.NewStreamV2(s, request)
		scope = m.scope.SubScope(metrics.ScopeUpstreamLDS)
	case resource.ClusterType:
		s, err = m.cdsClient.StreamClusters(ctx)
		stream = transport.NewStreamV2(s, request)
		scope = m.scope.SubScope(metrics.ScopeUpstreamCDS)
	case resource.RouteType:
		s, err = m.rdsClient.StreamRoutes(ctx)
		stream = transport.NewStreamV2(s, request)
		scope = m.scope.SubScope(metrics.ScopeUpstreamRDS)
	case resource.EndpointType:
		s, err = m.edsClient.StreamEndpoints(ctx)
		stream = transport.NewStreamV2(s, request)
		scope = m.scope.SubScope(metrics.ScopeUpstreamEDS)
	case resourcev3.ListenerType:
		s, err = m.ldsClientV3.StreamListeners(ctx)
		stream = transport.NewStreamV2(s, request)
		scope = m.scope.SubScope(metrics.ScopeUpstreamLDS)
	case resourcev3.ClusterType:
		s, err = m.cdsClientV3.StreamClusters(ctx)
		stream = transport.NewStreamV2(s, request)
		scope = m.scope.SubScope(metrics.ScopeUpstreamCDS)
	case resourcev3.RouteType:
		s, err = m.rdsClientV3.StreamRoutes(ctx)
		stream = transport.NewStreamV2(s, request)
		scope = m.scope.SubScope(metrics.ScopeUpstreamRDS)
	case resourcev3.EndpointType:
		s, err = m.edsClientV3.StreamEndpoints(ctx)
		stream = transport.NewStreamV2(s, request)
		scope = m.scope.SubScope(metrics.ScopeUpstreamEDS)
	default:
		defer cancel()
		m.logger.Error(ctx, "Unsupported Type Url %s", request.GetTypeURL())
		return nil, nil, &UnsupportedResourceError{TypeURL: request.GetTypeURL()}
	}

	if err != nil {
		defer cancel()
		return nil, nil, err
	}
	scope.Counter(metrics.UpstreamStreamOpened).Inc(1)

	signal := make(chan *version, 1)
	// The xds protocol https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#ack
	// specifies that the first request be empty nonce and empty version.
	// The origin server will respond with the latest version.
	signal <- &version{nonce: "", version: ""}

	response := make(chan transport.Response)

	go send(ctx, m.logger, cancel, request, stream, signal, m.callOptions)
	go recv(ctx, cancel, m.logger, response, stream, signal)

	m.logger.With("request_type", request.GetTypeURL()).Info(ctx, "stream opened")

	// We use context cancellation over using a separate channel for signalling stream shutdown.
	// The reason is cancelling a context tied with the stream is straightforward to signal closure.
	// Also, the shutdown function could potentially be called more than once by a caller.
	// Closing channels is not idempotent while cancelling context is idempotent.
	return response, func() { cancel() }, nil
}

// It is safe to assume send goroutine will not leak as long as these conditions are true:
// - SendMsg is performed with timeout.
// - send is a receiver for signal and exits when signal is closed by the owner.
// - send also exits on context cancellations.
func send(
	ctx context.Context,
	logger log.Logger,
	cancelFunc context.CancelFunc,
	request transport.Request,
	stream transport.Stream,
	signal chan *version,
	callOptions CallOptions) {
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
			logger.With(
				"request_type", request.GetTypeURL(),
				"request_version", request.GetVersionInfo(),
			).Debug(ctx, "sent message")
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
	cancelFunc context.CancelFunc,
	logger log.Logger,
	response chan transport.Response,
	stream transport.Stream,
	signal chan *version) {
	for {
		resp, err := stream.RecvMsg()
		if err != nil {
			handleError(ctx, logger, "Error in RecvMsg", cancelFunc, err)
			break
		}
		logger.With(
			"response_version", resp.GetVersionInfo(),
			"response_type", resp.GetTypeURL(),
			"resource_length", len(resp.GetResources()),
		).Debug(context.Background(), "received message")
		select {
		case <-ctx.Done():
			break
		default:
			response <- resp
			signal <- &version{version: resp.GetVersionInfo(), nonce: resp.GetNonce()}
		}
	}
	closeChannels(signal, response)
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

// closeChannels is called whenever the context is cancelled (ctx.Done) in Send and Recv goroutines.
// It is also called when an irrecoverable error occurs and the error is passed to the caller.
func closeChannels(versionChan chan *version, responseChan chan transport.Response) {
	close(versionChan)
	close(responseChan)
}

// shutDown should be called in a separate goroutine.
// This is a blocking function that closes the upstream connection on context completion.
func shutDown(ctx context.Context, conn *grpc.ClientConn) {
	<-ctx.Done()
	conn.Close()
}

func (e *UnsupportedResourceError) Error() string {
	return fmt.Sprintf("Unsupported resource typeUrl: %s", e.TypeURL)
}
