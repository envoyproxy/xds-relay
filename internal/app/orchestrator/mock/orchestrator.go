package orchestrator

import (
	"testing"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
)

func NewOrchestrator(t *testing.T, mapper mapper.Mapper, upstreamClient upstream.Client) orchestrator.Orchestrator {
	return orchestrator.NewMockOrchestrator(t, mapper, upstreamClient)
}
