package upstream

import (
	"context"
	"fmt"
	"sync"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"google.golang.org/grpc"
)

const (
	// ListenerTypeURL is the resource url for listener
	ListenerTypeURL = "type.googleapis.com/envoy.api.v2.Listener"
	// ClusterTypeURL is the resource url for cluster
	ClusterTypeURL = "type.googleapis.com/envoy.api.v2.Cluster"
	// EndpointTypeURL is the resource url for endpoints
	EndpointTypeURL = "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"
	// RouteTypeURL is the resource url for route
	RouteTypeURL = "type.googleapis.com/envoy.api.v2.RouteConfiguration"
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
	// If the timeouts are exhausted, receive fails or a irrecoverable error occurs, the error is sent back to the caller.
	// It is the caller's responsibility to send a new request from the last known DiscoveryRequest.
	// Cancellation of the context cleans up all outstanding streams and releases all resources.
	OpenStream(v2.DiscoveryRequest) (<-chan *v2.DiscoveryResponse, chan bool, error)
}

type client struct {
	ldsClient   v2.ListenerDiscoveryServiceClient
	rdsClient   v2.RouteDiscoveryServiceClient
	edsClient   v2.EndpointDiscoveryServiceClient
	cdsClient   v2.ClusterDiscoveryServiceClient
	callOptions CallOptions
	logger      log.Logger
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

// NewClient creates a grpc connection with an upstream origin server.
// Each xds relay server should create a single such upstream connection.
// grpc will handle the actual number of underlying tcp connections.
//
// The method does not block until the underlying connection is up.
// Returns immediately and connecting the server happens in background
func NewClient(ctx context.Context, url string, callOptions CallOptions, logger log.Logger) (Client, error) {
	// TODO: configure grpc options.https://github.com/envoyproxy/xds-relay/issues/55
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	ldsClient := v2.NewListenerDiscoveryServiceClient(conn)
	rdsClient := v2.NewRouteDiscoveryServiceClient(conn)
	edsClient := v2.NewEndpointDiscoveryServiceClient(conn)
	cdsClient := v2.NewClusterDiscoveryServiceClient(conn)

	go shutDown(ctx, conn)

	return &client{
		ldsClient:   ldsClient,
		rdsClient:   rdsClient,
		edsClient:   edsClient,
		cdsClient:   cdsClient,
		callOptions: callOptions,
		logger:      logger,
	}, nil
}

func (m *client) OpenStream(request v2.DiscoveryRequest) (<-chan *v2.DiscoveryResponse, chan bool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	var stream grpc.ClientStream
	var err error
	switch request.GetTypeUrl() {
	case ListenerTypeURL:
		stream, err = m.ldsClient.StreamListeners(ctx)
	case ClusterTypeURL:
		stream, err = m.cdsClient.StreamClusters(ctx)
	case RouteTypeURL:
		stream, err = m.rdsClient.StreamRoutes(ctx)
	case EndpointTypeURL:
		stream, err = m.edsClient.StreamEndpoints(ctx)
	default:
		cancel()
		m.logger.Error(ctx, "Unsupported Type Url %s", request.GetTypeUrl())
		return nil, nil, &UnsupportedResourceError{TypeURL: request.GetTypeUrl()}
	}

	if err != nil {
		cancel()
		return nil, nil, err
	}

	signal := make(chan *version, 1)
	// The xds protocol https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#ack
	// specifies that the first request be empty nonce and empty version.
	// The origin server will respond with the latest version.
	signal <- &version{nonce: "", version: ""}

	done := make(chan bool, 1)
	response := make(chan *v2.DiscoveryResponse, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	go send(ctx, m.logger, cancel, done, &wg, &request, stream, signal, m.callOptions)
	go recv(ctx, cancel, m.logger, done, &wg, response, stream, signal)

	go func() {
		wg.Wait()
		closeChannels(signal, response)
	}()

	return response, done, nil
}

func send(
	ctx context.Context,
	logger log.Logger,
	cancelFunc context.CancelFunc,
	done <-chan bool,
	wg *sync.WaitGroup,
	request *v2.DiscoveryRequest,
	stream grpc.ClientStream,
	signal chan *version,
	callOptions CallOptions) {
	for {
		select {
		case sig, ok := <-signal:
			if !ok {
				return
			}
			request.ResponseNonce = sig.nonce
			request.VersionInfo = sig.version
			err := doWithTimeout(ctx, func() error {
				return stream.SendMsg(request)
			}, callOptions.Timeout)
			if err != nil {
				select {
				case <-done:
					// This situation indicates that the caller closed the channel.
					// Hence, this is not an erroneous scenario.
				default:
					logger.Error(ctx, "Error in SendMsg: %s", err.Error())
				}
				cancelFunc()
				wg.Done()
				return
			}
		case <-done:
			cancelFunc()
			_ = stream.CloseSend()
			wg.Done()
			return
		}
	}
}

func recv(
	ctx context.Context,
	cancelFunc context.CancelFunc,
	logger log.Logger,
	done <-chan bool,
	wg *sync.WaitGroup,
	response chan *v2.DiscoveryResponse,
	stream grpc.ClientStream,
	signal chan *version) {
	for {
		resp := new(v2.DiscoveryResponse)
		if err := stream.RecvMsg(resp); err != nil {
			select {
			case <-done:
				// This situation indicates that the caller closed the channel.
				// Hence, this is not an erroneous scenario.
			default:
				logger.Error(ctx, "Error in RecvMsg %s", err.Error())
			}
			cancelFunc()
			wg.Done()
			return
		}
		select {
		case <-done:
			cancelFunc()
			wg.Done()
			return
		default:
			response <- resp
			signal <- &version{version: resp.GetVersionInfo(), nonce: resp.GetNonce()}
		}
	}
}

// closeChannels is called whenever the context is cancelled (ctx.Done) in Send and Recv goroutines.
// It is also called when an irrecoverable error occurs and the error is passed to the caller.
func closeChannels(versionChan chan *version, responseChan chan *v2.DiscoveryResponse) {
	close(versionChan)
	close(responseChan)
}

// shutDown should be called in a separate goroutine.
// This is a blocking function that closes the upstream connection on context completion.
func shutDown(ctx context.Context, conn *grpc.ClientConn) {
	<-ctx.Done()
	conn.Close()
}

// DoWithTimeout runs f and returns its error.  If the deadline d elapses first,
// it returns a DeadlineExceeded error instead.
// Ref: https://github.com/grpc/grpc-go/issues/1229#issuecomment-302755717
func doWithTimeout(ctx context.Context, f func() error, d time.Duration) error {
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(d))
	errChan := make(chan error, 1)
	go func() {
		errChan <- f()
		close(errChan)
	}()
	select {
	case <-deadlineCtx.Done():
		cancel()
		return deadlineCtx.Err()
	case err := <-errChan:
		cancel()
		return err
	}
}

func (e *UnsupportedResourceError) Error() string {
	return fmt.Sprintf("Unsupported resource typeUrl: %s", e.TypeURL)
}
