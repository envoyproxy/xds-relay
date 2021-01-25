package transport

import (
	"context"
	"time"

	status "google.golang.org/genproto/googleapis/rpc/status"
)

// Stream abstracts the grpc client stream and DiscoveryRequest/Response
type Stream interface {
	SendMsg(version string, nonce string, errorDetail *status.Status) error
	RecvMsg() (Response, error)
	RecvMsgWithTimeout(ctx context.Context, timeout time.Duration) (Response, error)
	CloseSend() error
}
