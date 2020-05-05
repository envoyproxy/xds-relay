// Package orchestrator is responsible for instrumenting inbound xDS client
// requests to the correct aggregated key, forwarding a representative request
// to the upstream origin server, and managing the lifecycle of downstream and
// upstream connections and associates streams. It implements
// go-control-plane's Cache interface in order to receive xDS-based requests,
// send responses, and handle gRPC streams.
//
// This file manages the bookkeeping of upstream responses by tracking the
// aggregated key and its corresponding receiver channel for upstream
// responses. The contents of this file are intended to only be used within the
// orchestrator module and should not be exported.
package orchestrator

import (
	"sync"

	discovery "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// upstreamResponseMap is the map of aggregate key to the receive-only upstream
// origin server response channels.
//
// sync.Map was chosen due to:
// - support for concurrent locks on a per-key basis
// - stable keys (when a given key is written once but read many times)
// The main drawback is the lack of type support.
type upstreamResponseMap struct {
	// This is of type *sync.Map[string]upstreamResponseChannel, where the key
	// is the xds-relay aggregated key.
	internal *sync.Map
}

type upstreamResponseChannel struct {
	response <-chan *discovery.DiscoveryResponse
	done     chan bool
}

func newUpstreamResponseMap() upstreamResponseMap {
	return upstreamResponseMap{
		internal: &sync.Map{},
	}
}

// exists returns true if the aggregatedKey exists.
func (u *upstreamResponseMap) exists(aggregatedKey string) bool {
	_, ok := u.internal.Load(aggregatedKey)
	return ok
}

// add sets the response channel for the provided aggregated key. It also
// initializes a done channel to be used during cleanup.
func (u *upstreamResponseMap) add(
	aggregatedKey string,
	responseChannel <-chan *discovery.DiscoveryResponse,
) (upstreamResponseChannel, bool) {
	channel := upstreamResponseChannel{
		response: responseChannel,
		done:     make(chan bool, 1),
	}
	result, exists := u.internal.LoadOrStore(aggregatedKey, channel)
	return result.(upstreamResponseChannel), exists
}

// delete signifies closure of the upstream stream and removes the map entry
// for the specified aggregated key.
func (u *upstreamResponseMap) delete(aggregatedKey string) {
	if channel, ok := u.internal.Load(aggregatedKey); ok {
		close(channel.(upstreamResponseChannel).done)
		// The implementation of sync.Map will already check for key existence
		// prior to issuing the delete, so we don't need worry about deleting
		// a non-existent key due to concurrent race conditions.
		u.internal.Delete(aggregatedKey)
	}
}

// deleteAll signifies closure of all upstream streams and removes the map
// entries. This is called during server shutdown.
func (u *upstreamResponseMap) deleteAll() {
	u.internal.Range(func(aggregatedKey, channel interface{}) bool {
		close(channel.(upstreamResponseChannel).done)
		u.internal.Delete(aggregatedKey)
		return true
	})
}
