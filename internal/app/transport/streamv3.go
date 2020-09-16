package transport

import (
	"context"

	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"google.golang.org/grpc"
)

var _ Stream = &streamv3{}

type streamv3 struct {
	grpcClientStream grpc.ClientStream
	initialRequest   Request
	logger           log.Logger
}

// NewStreamV3 creates a new wrapped transport stream
func NewStreamV3(clientStream grpc.ClientStream, req Request, l log.Logger) Stream {
	return &streamv3{
		grpcClientStream: clientStream,
		initialRequest:   req,
	}
}

func (s *streamv3) SendMsg(version string, nonce string) error {
	msg := s.initialRequest.GetRaw().V3
	msg.VersionInfo = version
	msg.ResponseNonce = nonce
	s.logger.With(
		"request_type", msg.GetTypeUrl(),
		"request_version", msg.GetVersionInfo(),
	).Debug(context.Background(), "sent message")

	return s.grpcClientStream.SendMsg(msg)
}

func (s *streamv3) RecvMsg() (Response, error) {
	resp := new(v3.DiscoveryResponse)
	if err := s.grpcClientStream.RecvMsg(resp); err != nil {
		return nil, err
	}
	return NewResponseV3(s.initialRequest.GetRaw().V3, resp), nil
}

func (s *streamv3) CloseSend() error {
	return s.grpcClientStream.CloseSend()
}
