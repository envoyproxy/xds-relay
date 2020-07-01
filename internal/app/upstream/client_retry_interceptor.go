package upstream

import (
	"context"

	backoff "github.com/cenkalti/backoff"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func RetryClientStreamInterceptor(logger log.Logger) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}
		stream := &wrappedClientStream{
			ctx:    ctx,
			logger: logger.Named("wrapped-clientStream"),
			inner:  clientStream,
		}
		return stream, nil
	}
}

type wrappedClientStream struct {
	inner  grpc.ClientStream
	ctx    context.Context
	logger log.Logger
}

func (wcs *wrappedClientStream) SendMsg(m interface{}) error {
	wcs.logger.Info(wcs.ctx, "wrapped SendMsg")
	return wcs.inner.SendMsg(m)
}

func (wcs *wrappedClientStream) RecvMsg(m interface{}) error {
	wcs.logger.Info(wcs.ctx, "wrapped RecvMsg")
	if err := wcs.inner.RecvMsg(m); err != nil {
		return backoff.Retry(func() error {
			resp := new(v2.DiscoveryResponse)
			wcs.logger.Info(wcs.ctx, "wrapped RecvMsg - retry")
			if err := wcs.inner.RecvMsg(resp); err != nil {
				if isErrorRetryable(err) {
					return err
				}
				return backoff.Permanent(err)
			}
			wcs.logger.Info(wcs.ctx, "successful retry")
			return nil
		}, backoff.NewExponentialBackOff())
	}
	return nil
}

func isErrorRetryable(err error) bool {
	errorCode := status.Code(err)
	return errorCode == codes.Unavailable || errorCode == codes.Unknown
}

func (wcs *wrappedClientStream) CloseSend() error {
	return wcs.inner.CloseSend()
}

func (wcs *wrappedClientStream) Context() context.Context {
	return wcs.ctx
}

func (wcs *wrappedClientStream) Header() (metadata.MD, error) {
	return wcs.inner.Header()
}

func (wcs *wrappedClientStream) Trailer() metadata.MD {
	return wcs.inner.Trailer()
}
