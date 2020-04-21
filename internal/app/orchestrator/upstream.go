package orchestrator

import (
	"sync"

	discovery "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// upstream is the map of aggregate key to the receive-only upstream
// origin server response channels.
type upstreamResponseMap struct {
	mu              sync.RWMutex
	responseChannel map[string]upstreamResponseChannel
}

type upstreamResponseChannel struct {
	response <-chan *discovery.DiscoveryResponse
	done     chan bool
}

// add sets the response channel for the provided aggregated key. It also
// initializes a done channel to be used during cleanup.
//
// Expects orchestrator to manage the lock since this is called simultaneously
// with stream open.
func (u *upstreamResponseMap) add(aggregatedKey string,
	responseChannel <-chan *discovery.DiscoveryResponse) upstreamResponseChannel {
	channel := upstreamResponseChannel{
		response: responseChannel,
		done:     make(chan bool, 1),
	}
	u.responseChannel[aggregatedKey] = channel
	return channel
}

// delete closes the done channel and removes both channels from the map.
// TODO We can't close the upstream receiver only channel, need to signify to
// the upstream client to do so.
func (u *upstreamResponseMap) delete(aggregatedKey string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if channel, ok := u.responseChannel[aggregatedKey]; ok {
		close(channel.done)
		delete(u.responseChannel, aggregatedKey)
	}
}
