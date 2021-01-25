package transport

import (
	"context"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	status "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
)

var _ Stream = &streamv2{}

type streamv2 struct {
	grpcClientStream grpc.ClientStream
	initialRequest   Request
	logger           log.Logger
}

// NewStreamV2 creates a new wrapped transport stream
func NewStreamV2(clientStream grpc.ClientStream, req Request, l log.Logger) Stream {
	return &streamv2{
		grpcClientStream: clientStream,
		initialRequest:   req,
		logger:           l.Named("stream"),
	}
}

func (s *streamv2) SendMsg(version string, nonce string, errorDetail *status.Status) error {
	msg := s.initialRequest.GetRaw().V2
	msg.VersionInfo = version
	msg.ResponseNonce = nonce
	msg.ErrorDetail = errorDetail
	s.logger.With(
		"request_type", msg.GetTypeUrl(),
		"request_version", msg.GetVersionInfo(),
		"request_resource_names", msg.ResourceNames,
	).Debug(context.Background(), "sent message")
	return s.grpcClientStream.SendMsg(msg)
}

func (s *streamv2) RecvMsg() (Response, error) {
	resp := new(v2.DiscoveryResponse)
	if err := s.grpcClientStream.RecvMsg(resp); err != nil {
		return nil, err
	}
	s.logger.With(
		"response_type", resp.GetTypeUrl(),
		"request_version", resp.GetVersionInfo(),
		"resource_length", len(resp.GetResources()),
	).Debug(context.Background(), "received message")
	return NewResponseV2(s.initialRequest.GetRaw().V2, resp), nil
}

func (s *streamv2) RecvMsgWithTimeout(ctx context.Context, timeout time.Duration) (Response, error) {
	type response struct {
		resp Response
		err  error
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	respChan := make(chan response, 1)
	go func() {
		resp, err := s.RecvMsg()
		respChan <- response{resp: resp, err: err}
		close(respChan)
	}()
	select {
	case <-timeoutCtx.Done():
		return nil, timeoutCtx.Err()
	case r := <-respChan:
		return r.resp, r.err
	}
}

func (s *streamv2) CloseSend() error {
	return s.grpcClientStream.CloseSend()
}
