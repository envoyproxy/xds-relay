package client

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"google.golang.org/grpc"
)

// XdsClient handles the requests and responses from the origin server.
// The xds client handles each xds request on a separate stream,
// e.g. 2 different cds requests happen on 2 separate streams.
// It is the caller's responsibility to make sure there is one instance of XdsClient per unique xds request.
type XdsClient interface {
	// QueueRequest creates a stream with the origin server
	// All discovery requests to origin server arrive on the request channel
	// All responses from the origin server are sent back through the response channel
	// The caller is responsible for managing the lifetime of the channels.
	// QueueRequest uses the retry and timeout configurations to make best effort to get the responses from origin server.
	// If there's a new request in between retries, the retries are abandoned.
	// The request and response happen asynchronously. Retries are scoped for sending messages to origin server.
	// If the timeouts are exhausted, receive fails or a irrecoverable error occurs, the error is sent back to the caller.
	// It is the caller's responsibility to send a new request from the last known DiscoveryRequest.
	// Cancellation and cleanup operations will be based on cancellation of the context and closing of channels.
	QueueRequest(context.Context, chan *v2.DiscoveryRequest, chan *Response) error
}

type xdsClient struct {
	conn *grpc.ClientConn
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

// NewMux creates an instance based on the typeUrl of the resource.
// A new instance of XdsClient is recommended per xds type.
// e.g. For eds requests of different services, create an instance each.
// TODO: pass retry/timeout configurations
func NewMux(ctx context.Context, conn *grpc.ClientConn, typeURL string) (XdsClient, error) {
	return &xdsClient{
		conn: conn,
	}, nil
}

func (m *xdsClient) QueueRequest(
	ctx context.Context,
	requestChan chan *v2.DiscoveryRequest,
	responseChan chan *Response) error {
	return nil
}
