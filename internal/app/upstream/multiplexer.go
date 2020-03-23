package multiplexer

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"google.golang.org/grpc"
)

// Multiplexer handles the requests and responses from the origin server.
// The multiplexer handles each xds request on a separate streams, e.g. 2 different cds requests happen on 2 separate streams.
// It is the caller's responsibility to make sure there is one instance of Multipler per unique xds request.
type Multiplexer interface {
	// QueueRequest creates a stream with the origin server
	// All discovery requests to origin server arrive on the request channel
	// All responses from the origin server are sent back through the response channel
	// The caller is responsible for managing the lifetime of the channels.
	// QueueRequest uses the retry and timeout configurations to make best effort to get the responses from origin server.
	// If there's a new request in between retries, the retries are abandoned.
	// The request and response happen asynchronously. Retries are scoped for sending messages to origin server.
	// If the timeouts are exhausted, receive fails or a irrecoverable error occurs, the error is sent back through the error channel.
	// It is the caller's responsibility to send a new request from the last known DiscoveryRequest.
	// Cancellation and cleanup operations will be based on cancellation of the context and closing of channels.
	QueueRequest(context.Context, chan *v2.DiscoveryRequest, chan *v2.DiscoveryResponse) error
}

type multiplexer struct {
	conn *grpc.ClientConn
}

// NewMux creates an instance based on the typeUrl of the resource.
// A new instance of Multiplexer is recommended per xds type.
// e.g. For eds requests of different services, create an instance each.
// TODO: pass retry/timeout configurations
func NewMux(ctx context.Context, conn *grpc.ClientConn, typeURL string) (Multiplexer, error) {
	return &multiplexer{
		conn: conn,
	}, nil
}

func (m *multiplexer) QueueRequest(
	ctx context.Context,
	requestChan chan *v2.DiscoveryRequest,
	responseChan chan *v2.DiscoveryResponse) error {
}
