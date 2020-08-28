package transport

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"google.golang.org/grpc"
)

// Stream abstracts the grpc client stream and DiscoveryRequest/Response
type Stream interface {
	SendMsg() error
	RecvMsg() (Response, error)
	CloseSend() error
}

var _ Stream = &streamv2{}

type streamv2 struct {
	grpcClientStream grpc.ClientStream
	initialRequest   Request
}

// NewStreamV2 creates a new wrapped transport stream
func NewStreamV2(clientStream grpc.ClientStream, req Request) Stream {
	return &streamv2{
		grpcClientStream: clientStream,
		initialRequest:   req,
	}
}

func (s *streamv2) SendMsg() error {
	return s.grpcClientStream.SendMsg(s.initialRequest.GetRaw().V2)
}

func (s *streamv2) RecvMsg() (Response, error) {
	resp := new(v2.DiscoveryResponse)
	if err := s.grpcClientStream.RecvMsg(resp); err != nil {
		return nil, err
	}
	return NewResponseV2(s.initialRequest.GetRaw().V2, resp), nil
}

func (s *streamv2) CloseSend() error {
	return s.grpcClientStream.CloseSend()
}
