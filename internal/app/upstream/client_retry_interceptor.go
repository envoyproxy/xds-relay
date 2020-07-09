package upstream

import (
	"context"
	"sync"
	"time"

	backoff "github.com/cenkalti/backoff"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func RetryClientStreamInterceptor(logger log.Logger) grpc.StreamClientInterceptor {
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
		stream := &wrappedClientStream{
			ctx:    ctx,
			logger: logger.Named("retry-interceptor"),
			inner:  clientStream,
			newStream: func(ctx context.Context) (grpc.ClientStream, error) {
				return streamer(ctx, desc, cc, method, opts...)
			},
		}
		return stream, nil
	}
}

type wrappedClientStream struct {
	inner     grpc.ClientStream
	ctx       context.Context
	logger    log.Logger
	newStream func(ctx context.Context) (grpc.ClientStream, error)
	buffer    []interface{}
	mu        sync.RWMutex
}

func (wcs *wrappedClientStream) SendMsg(m interface{}) error {
	wcs.mu.Lock()
	wcs.buffer = append(wcs.buffer, m)
	wcs.mu.Unlock()
	if err := wcs.getStream().SendMsg(m); err != nil {
		return err
	}
	return nil
}

func (wcs *wrappedClientStream) RecvMsg(m interface{}) error {
	if err := wcs.getStream().RecvMsg(m); err != nil {
		wcs.logger.Info(wcs.ctx, "retry RecvMsg")
		exponentialBackoff := backoff.NewExponentialBackOff()
		exponentialBackoff.InitialInterval = 1 * time.Second
		exponentialBackoff.Multiplier = 2.0
		exponentialBackoff.MaxInterval = 1 * time.Second
		exponentialBackoff.MaxElapsedTime = 5 * time.Second

		return backoff.Retry(func() error {
			wcs.logger.Info(wcs.ctx, "inside backoff retry RecvMsg")
			if err := wcs.retrySend(); err != nil {
				if isErrorRetriable(err) {
					return err
				}
				return backoff.Permanent(err)
			}
			if err := wcs.getStream().RecvMsg(m); err != nil {
				if isErrorRetriable(err) {
					return err
				}
				return backoff.Permanent(err)
			}
			return nil
		}, exponentialBackoff) // TODO: parametrize
	}
	return nil
}

func (wcs *wrappedClientStream) retrySend() error {
	return backoff.Retry(func() error {
		stream, err := wcs.newStream(context.Background())
		if err != nil {
			return err
		}

		wcs.mu.RLock()
		buffer := wcs.buffer
		wcs.mu.RUnlock()
		for _, m := range buffer {
			if err := stream.SendMsg(m); err != nil {
				wcs.logger.With("message", m, "error", err).Info(wcs.ctx, "SendMsg error - retry")
				if isErrorRetriable(err) {
					return err
				}
				return backoff.Permanent(err)
			}
		}

		wcs.setStream(stream)
		return nil
		// TODO parametrize
	}, backoff.NewConstantBackOff(1*time.Second))
}

func (wcs *wrappedClientStream) setStream(stream grpc.ClientStream) {
	wcs.mu.Lock()
	wcs.inner = stream
	wcs.mu.Unlock()
}

func (wcs *wrappedClientStream) getStream() grpc.ClientStream {
	wcs.mu.RLock()
	defer wcs.mu.RUnlock()
	return wcs.inner
}

func (wcs *wrappedClientStream) CloseSend() error {
	return wcs.getStream().CloseSend()
}

func (wcs *wrappedClientStream) Context() context.Context {
	return wcs.ctx
}

func (wcs *wrappedClientStream) Header() (metadata.MD, error) {
	return wcs.getStream().Header()
}

func (wcs *wrappedClientStream) Trailer() metadata.MD {
	return wcs.getStream().Trailer()
}

func isErrorRetriable(err error) bool {
	errorCode := status.Code(err)
	return errorCode == codes.Unavailable || errorCode == codes.Unknown
}
