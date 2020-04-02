// Package orchestrator is responsible for instrumenting inbound xDS client
// requests to the correct aggregated key, forwarding a representative request
// to the upstream origin server, and managing the lifecycle of downstream and
// upstream connections and associates streams. It implements
// go-control-plane's Cache interface in order to receive xDS-based requests,
// send responses, and handle gRPC streams.
package orchestrator

import (
	"context"
	"testing"

	"github.com/envoyproxy/xds-relay/internal/pkg/log"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Trivial test to ensure orchestrator instantiates.
	orchestrator := New(context.Background(), log.New("info"))
	assert.NotNil(t, orchestrator)
}
