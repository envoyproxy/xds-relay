package upstream

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// Client handles the requests and responses from the origin server.
// The client handles each xds request on a separate stream,
// e.g. 2 different cds requests happen on 2 separate streams.
// It is the caller's responsibility to make sure there is one instance of client overall and
// Start is called per unique key.
type Client interface {
	// Start creates a stream with the origin server
	//
	// Start should be called once per aggregated key.
	// Start uses one representative node identifier for the entire lifetime of the stream.
	// Therefore, it is not necessary to pass node identifiers on subsequent requests from sidecars.
	// It follows xds protocol https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol
	// to handle the version_info, nonce and error_details and applies the response to the cache.
	//
	// All responses from the origin server are sent back through the callback function.
	//
	// Start uses the retry and timeout configurations to make best effort to get the responses from origin server.
	// The request and response happen asynchronously on separate goroutines.
	// If the timeouts are exhausted, receive fails or a irrecoverable error occurs, the error is sent back to the caller
	// and ressources(stream/goroutine, etc) are released.
	// It is the caller's responsibility to send a new request from the last known DiscoveryRequest.
	// Cancellation of the context cleans up all outstanding streams and releases all resources.
	Start(ctx context.Context, request *v2.DiscoveryRequest, responseCb func(*Response), typeURL string) error
}

type client struct {
	//nolint
	ldsClient v2.ListenerDiscoveryServiceClient
}

// Response struct is a holder for the result from a single request.
// A request can result in a response from origin server or an error
// Only one of the fields is valid at any time. If the error is set, the response will be ignored.
type Response struct {
	//nolint
	response v2.DiscoveryResponse
	//nolint
	err error
}

// NewClient creates a grpc connection with an upstream origin server.
// Each xds relay server should create a single such upstream connection.
// grpc will handle the actual number of underlying tcp connections.
//
// The method does not block until the underlying connection is up.
// Returns immediately and connecting the server happens in background
// TODO: pass retry/timeout configurations
func NewClient(ctx context.Context, url string) (Client, error) {
	return &client{}, nil
}

func (m *client) Start(
	ctx context.Context,
	request *v2.DiscoveryRequest,
	responseFunc func(*Response),
	typeURL string) error {
	return nil
}
