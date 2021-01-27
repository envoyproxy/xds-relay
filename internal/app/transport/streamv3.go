package transport

import (
	"context"

	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
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
		logger:           l.Named("stream"),
	}
}

func (s *streamv3) SendMsg(version string, nonce string, metadata string) error {
	msg := s.initialRequest.GetRaw().V3
	msg.VersionInfo = version
	msg.ErrorDetail = nil
	msg.ResponseNonce = nonce
	if s.initialRequest.GetNodeMetadata() == nil {
		fields := make(map[string]*structpb.Value)
		msg.Node.Metadata = &structpb.Struct{
			Fields: fields,
		}
	}
	msg.Node.Metadata.Fields["xdsrelay_node_metadata"] = &structpb.Value{
		Kind: &structpb.Value_StringValue{
			StringValue: metadata,
		},
	}
	s.logger.With(
		"request_type", msg.GetTypeUrl(),
		"request_version", msg.GetVersionInfo(),
		"request_resource_names", msg.ResourceNames,
		"xdsrelay_node_metadata", metadata,
	).Debug(context.Background(), "sent message")

	return s.grpcClientStream.SendMsg(msg)
}

func (s *streamv3) RecvMsg() (Response, error) {
	resp := new(v3.DiscoveryResponse)
	if err := s.grpcClientStream.RecvMsg(resp); err != nil {
		return nil, err
	}

	s.logger.With(
		"response_type", resp.GetTypeUrl(),
		"request_version", resp.GetVersionInfo(),
		"resource_length", len(resp.GetResources()),
	).Debug(context.Background(), "received message")
	return NewResponseV3(s.initialRequest.GetRaw().V3, resp), nil
}

func (s *streamv3) CloseSend() error {
	return s.grpcClientStream.CloseSend()
}
