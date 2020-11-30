package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/uber-go/tally"

	"github.com/envoyproxy/xds-relay/internal/app/cache"
	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"github.com/stretchr/testify/assert"
)

func NewMock(ctx context.Context,
	t *testing.T,
	mapper mapper.Mapper,
	upstreamClient upstream.Client,
	scope tally.Scope) Orchestrator {
	return NewMockOrchestrator(ctx, t, mapper, upstreamClient, scope)
}

func NewMockOrchestrator(
	ctx context.Context,
	t *testing.T,
	mapper mapper.Mapper,
	upstreamClient upstream.Client,
	scope tally.Scope,
) Orchestrator {
	orchestrator := &orchestrator{
		logger:                log.MockLogger,
		scope:                 scope,
		mapper:                mapper,
		upstreamClient:        upstreamClient,
		downstreamResponseMap: newDownstreamResponseMap(ctx),
		upstreamResponseMap:   newUpstreamResponseMap(),
	}

	cache, err := cache.NewCache(1000, orchestrator.onCacheEvicted, 10*time.Second, log.MockLogger, scope)
	assert.NoError(t, err)
	orchestrator.cache = cache

	return orchestrator
}
