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

// Watch interface abstracts v2 and v3 watches
type Watch interface {
	Close()
	GetChannel() *ChannelVersion
	Send(Response) bool
}
