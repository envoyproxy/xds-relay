package upstream

import (
	"context"
	"fmt"
	"sync"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	listenerTypeURL = "type.googleapis.com/envoy.api.v2.Listener"
	clusterTypeURL  = "type.googleapis.com/envoy.api.v2.Cluster"
	endpointTypeURL = "ype.googleapis.com/envoy.api.v2.ClusterLoadAssignment"
	routeTypeURL    = "type.googleapis.com/envoy.api.v2.RouteConfiguration"
)

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
	OpenStream(context.Context, *v2.DiscoveryRequest) (chan *Response, error)
}

type client struct {
	ldsClient   v2.ListenerDiscoveryServiceClient
	rdsClient   v2.RouteDiscoveryServiceClient
	edsClient   v2.EndpointDiscoveryServiceClient
	cdsClient   v2.ClusterDiscoveryServiceClient
	callOptions CallOptions
}

// Response struct is a holder for the result from a single request.
// A request can result in a response from origin server or an error
// Only one of the fields is valid at any time. If the error is set, the response will be ignored.
type Response struct {
	Response *v2.DiscoveryResponse
	Err      error
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
// TODO: pass retry configurations
func NewClient(ctx context.Context, url string, callOptions CallOptions) (Client, error) {
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
	}, nil
}

// NewMockClient creates a mock implementation for testing
func NewMockClient(
	ctx context.Context,
	ldsClient v2.ListenerDiscoveryServiceClient,
	rdsClient v2.RouteDiscoveryServiceClient,
	edsClient v2.EndpointDiscoveryServiceClient,
	cdsClient v2.ClusterDiscoveryServiceClient,
	callOptions CallOptions) Client {
	return &client{
		ldsClient:   ldsClient,
		rdsClient:   rdsClient,
		edsClient:   edsClient,
		cdsClient:   cdsClient,
		callOptions: callOptions,
	}
}

func (m *client) OpenStream(ctx context.Context, request *v2.DiscoveryRequest) (chan *Response, error) {
	var stream grpc.ClientStream
	var err error
	switch request.GetTypeUrl() {
	case listenerTypeURL:
		stream, err = m.ldsClient.StreamListeners(ctx)
	case clusterTypeURL:
		stream, err = m.cdsClient.StreamClusters(ctx)
	case routeTypeURL:
		stream, err = m.rdsClient.StreamRoutes(ctx)
	case endpointTypeURL:
		stream, err = m.edsClient.StreamEndpoints(ctx)
	default:
		return nil, fmt.Errorf("Unsupported request type")
	}

	if err != nil {
		return nil, err
	}

	signal := make(chan *version, 1)
	signal <- &version{}
	response := make(chan *Response)

	var wg sync.WaitGroup
	wg.Add(2)

	go send(ctx, &wg, request, response, stream, signal, m.callOptions)
	go recv(ctx, &wg, response, stream, signal)

	go func() {
		wg.Wait()
		closeChannel(signal, response)
	}()

	return response, nil
}

func send(
	ctx context.Context,
	wg *sync.WaitGroup,
	request *v2.DiscoveryRequest,
	response chan *Response,
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
			err := doWithTimeout(func() error {
				return stream.SendMsg(request)
			}, callOptions.Timeout)
			if err != nil {
				select {
				case <-ctx.Done():
					wg.Done()
					return
				default:
					response <- &Response{Err: err}
					wg.Done()
					return
				}
			}
		case <-ctx.Done():
			_ = stream.CloseSend()
			wg.Done()
			return
		}
	}
}

func recv(
	ctx context.Context,
	wg *sync.WaitGroup,
	response chan *Response,
	stream grpc.ClientStream,
	signal chan *version) {
	for {
		resp := new(v2.DiscoveryResponse)
		if err := stream.RecvMsg(resp); err != nil {
			select {
			case <-ctx.Done():
				wg.Done()
			default:
				response <- &Response{Err: err}
			}
			return
		}
		select {
		case <-ctx.Done():
			wg.Done()
			return
		default:
			response <- &Response{Response: resp}
			signal <- &version{version: resp.GetVersionInfo(), nonce: resp.GetNonce()}
		}
	}
}

func closeChannel(versionChan chan *version, responseChan chan *Response) {
	close(versionChan)
	close(responseChan)
}

func shutDown(ctx context.Context, conn *grpc.ClientConn) {
	<-ctx.Done()
	conn.Close()
}

// DoWithTimeout runs f and returns its error.  If the deadline d elapses first,
// it returns a grpc DeadlineExceeded error instead.
// Ref: https://github.com/grpc/grpc-go/issues/1229#issuecomment-302755717s
func doWithTimeout(f func() error, d time.Duration) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- f()
		close(errChan)
	}()
	timer := time.NewTimer(d)
	select {
	case <-timer.C:
		return status.Errorf(codes.DeadlineExceeded, "timeout")
	case err := <-errChan:
		if !timer.Stop() {
			<-timer.C
		}
		return err
	}
}
