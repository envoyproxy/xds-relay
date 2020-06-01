package orchestrator

import (
	"testing"
	"time"

	"github.com/uber-go/tally"

	"github.com/envoyproxy/xds-relay/internal/app/cache"
	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"github.com/stretchr/testify/assert"
)

func NewOrchestrator(t *testing.T,
	mapper mapper.Mapper,
	upstreamClient upstream.Client,
	scope tally.Scope) Orchestrator {
	return NewMockOrchestrator(t, mapper, upstreamClient, scope)
}

func NewMockOrchestrator(t *testing.T,
	mapper mapper.Mapper,
	upstreamClient upstream.Client,
	scope tally.Scope) Orchestrator {
	orchestrator := &orchestrator{
		logger:                log.New("info"),
		mapper:                mapper,
		upstreamClient:        upstreamClient,
		downstreamResponseMap: newDownstreamResponseMap(scope),
		upstreamResponseMap:   newUpstreamResponseMap(),
	}

	cache, err := cache.NewCache(1000, orchestrator.onCacheEvicted, 10*time.Second)
	assert.NoError(t, err)
	orchestrator.cache = cache

	return orchestrator
}
