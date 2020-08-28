package transport

// Stream abstracts the grpc client stream and DiscoveryRequest/Response
type Stream interface {
	SendMsg(version string, nonce string) error
	RecvMsg() (Response, error)
	CloseSend() error
}
