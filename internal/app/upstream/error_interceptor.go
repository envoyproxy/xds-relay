package upstream

import (
	"context"

	"github.com/envoyproxy/xds-relay/internal/app/metrics"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"github.com/uber-go/tally"
	"google.golang.org/grpc"
)

func ErrorClientStreamInterceptor(logger log.Logger, scope tally.Scope) grpc.StreamClientInterceptor {
	return func(ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption) (grpc.ClientStream, error) {
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}
		stream := &wrappedStream{
			clientStream,
			ctx,
			logger.Named("error_interceptor"),
			scope.SubScope(metrics.ScopeErrorInterceptor),
		}
		return stream, nil
	}
}

// wrappedStream wraps around the grpc.CientStream used to send and receive messages,
// intercepting those calls and logging in case an error is encountered.
type wrappedStream struct {
	grpc.ClientStream
	ctx    context.Context
	logger log.Logger
	scope  tally.Scope
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	err := w.ClientStream.RecvMsg(m)
	if err != nil {
		w.logger.With("message", m, "error", err).Warn(w.ctx, "error in RecvMsg")
		w.scope.Counter(metrics.ErrorInterceptorErrorRecvMsg).Inc(1)
	}
	return err
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	err := w.ClientStream.SendMsg(m)
	if err != nil {
		w.logger.With("message", m, "error", err).Warn(w.ctx, "error in SendMsg")
		w.scope.Counter(metrics.ErrorInterceptorErrorSendMsg).Inc(1)
	}
	return err
}
