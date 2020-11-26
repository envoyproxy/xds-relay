package transport

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
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

func (s *streamv2) SendMsg(version string, nonce string) error {
	msg := s.initialRequest.GetRaw().V2
	clone := proto.Clone(msg).(*v2.DiscoveryRequest)
	clone.VersionInfo = version
	clone.ResponseNonce = nonce
	clone.ErrorDetail = nil
	s.logger.With(
		"request_type", clone.GetTypeUrl(),
		"request_version", clone.GetVersionInfo(),
		"request_resource_names", clone.ResourceNames,
	).Debug(context.Background(), "sent message")
	return s.grpcClientStream.SendMsg(clone)
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

func (s *streamv2) CloseSend() error {
	return s.grpcClientStream.CloseSend()
}
