package orchestrator

import (
	"testing"

	"github.com/uber-go/tally"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
)

func NewOrchestrator(t *testing.T,
	mapper mapper.Mapper,
	upstreamClient upstream.Client,
	scope tally.Scope) orchestrator.Orchestrator {
	return orchestrator.NewMockOrchestrator(t, mapper, upstreamClient, scope)
}
