package orchestrator

import (
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache"
)

// Orchestrator provides an interface that handles the discovery requests from sidecars.
type Orchestrator interface {
	// CreateWatch returns a new open watch from a non-empty request.
	// Returns a channel with Response and a function which is a callback for handling stream cancellations.
	CreateWatch(req gcp.Request) (chan gcp.Response, func())
}

// orchestrator provides a mechanism to implement the Orchestrator interface.
// The orchestrator maintains all the relevant caches and makes sure of the following:
// 1. Returns long lived streams from the sidecars.
// 2. Aggregates similar requests from a key.
// 3. Maintains upstream request streams with the upstream origin server for each such unique discovery request.
// 4. When a new response is available on the origin server,
//    orchestrator relays the response back on the streams associated with the sidecars.
type orchestrator struct {
}

// NewOrchestrator returns an instance of an orchestrator
func NewOrchestrator() Orchestrator {
	return &orchestrator{}
}

func (c *orchestrator) CreateWatch(req gcp.Request) (chan gcp.Response, func()) {
	return nil, nil
}
