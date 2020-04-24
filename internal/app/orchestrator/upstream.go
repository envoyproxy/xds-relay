package orchestrator

import (
	"sync"

	discovery "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// upstream is the map of aggregate key to the receive-only upstream
// origin server response channels.
type upstreamResponseMap struct {
	mu               sync.RWMutex
	responseChannels map[string]upstreamResponseChannel
}

type upstreamResponseChannel struct {
	response <-chan *discovery.DiscoveryResponse
	done     chan bool
}

func newUpstreamResponseMap() upstreamResponseMap {
	return upstreamResponseMap{
		responseChannels: make(map[string]upstreamResponseChannel),
	}
}

// add sets the response channel for the provided aggregated key. It also
// initializes a done channel to be used during cleanup.
//
// Expects orchestrator to manage the lock since this is called simultaneously
// with stream open.
func (u *upstreamResponseMap) add(
	aggregatedKey string,
	responseChannel <-chan *discovery.DiscoveryResponse,
) upstreamResponseChannel {
	channel := upstreamResponseChannel{
		response: responseChannel,
		done:     make(chan bool, 1),
	}
	u.responseChannels[aggregatedKey] = channel
	return channel
}

// delete signifies closure of the upstream stream and removes the map entry
// for the specified aggregated key.
func (u *upstreamResponseMap) delete(aggregatedKey string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if channel, ok := u.responseChannels[aggregatedKey]; ok {
		close(channel.done)
		delete(u.responseChannels, aggregatedKey)
	}
}

// deleteAll signifies closure of all upstream streams and removes the map
// entries.
func (u *upstreamResponseMap) deleteAll() {
	u.mu.Lock()
	defer u.mu.Unlock()
	for aggregatedKey, responseChannels := range u.responseChannels {
		close(responseChannels.done)
		delete(u.responseChannels, aggregatedKey)
	}
}
