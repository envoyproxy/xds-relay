package transport

import (
	gcpv2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// ChannelVersion wraps v2 and v3 response channels
type ChannelVersion struct {
	V2 chan gcpv2.Response
	V3 chan gcpv3.Response
}

// Watch interface abstracts v2 and v3 watches to the downstream sidecars.
type Watch interface {
	// Close is idempotent with Send.
	// When Close and Send are called from separate goroutines they are guaranteed to not panic
	Close()
	GetChannel() *ChannelVersion
	// Send is a mutex protected function to send responses to the downstream sidecars.
	// It provides guarantee to never panic when calling in tandem with Close from separate goroutines.
	// Close waits until all Send operations drain.
	Send(Response) error
}
