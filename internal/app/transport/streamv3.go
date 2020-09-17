package transport

import (
	"context"

	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	resource3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"github.com/golang/protobuf/ptypes"
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
		logger:           l,
	}
}

func (s *streamv3) SendMsg(version string, nonce string) error {
	msg := s.initialRequest.GetRaw().V3
	msg.VersionInfo = version
	msg.ResponseNonce = nonce
	s.logger.With(
		"request_type", msg.GetTypeUrl(),
		"request_version", msg.GetVersionInfo(),
		"request_msg", msg.ResourceNames,
	).Debug(context.Background(), "sent message")

	return s.grpcClientStream.SendMsg(msg)
}

func (s *streamv3) RecvMsg() (Response, error) {
	resp := new(v3.DiscoveryResponse)
	if err := s.grpcClientStream.RecvMsg(resp); err != nil {
		return nil, err
	}
	for _, r := range resp.GetResources() {
		switch resp.TypeUrl {
		case resource3.EndpointType:
			e := &endpointv3.ClusterLoadAssignment{}
			err := ptypes.UnmarshalAny(r, e)
			if err == nil {
				s.logger.With(
					"request_type", resp.GetTypeUrl(),
					"request_version", resp.GetVersionInfo(),
					"msg", e,
					"resource_length", len(resp.GetResources()),
				).Debug(context.Background(), "received message")
			}
		}
	}
	/*s.logger.With(
		"request_type", resp.GetTypeUrl(),
		"request_version", resp.GetVersionInfo(),
		"resource_length", len(resp.GetResources()),
	).Debug(context.Background(), "received message")*/
	return NewResponseV3(s.initialRequest.GetRaw().V3, resp), nil
}

func (s *streamv3) CloseSend() error {
	return s.grpcClientStream.CloseSend()
}
