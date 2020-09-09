package transport

import (
	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc"
)

var _ Stream = &streamv3{}

type streamv3 struct {
	grpcClientStream grpc.ClientStream
	initialRequest   Request
}

// NewStreamV3 creates a new wrapped transport stream
func NewStreamV3(clientStream grpc.ClientStream, req Request) Stream {
	return &streamv3{
		grpcClientStream: clientStream,
		initialRequest:   req,
	}
}

func (s *streamv3) SendMsg(version string, nonce string) error {
	msg := s.initialRequest.GetRaw().V3
	msg.VersionInfo = version
	msg.ResponseNonce = nonce
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
