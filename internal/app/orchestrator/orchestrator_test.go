package orchestrator

import (
	"context"
	"testing"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
	"github.com/golang/protobuf/ptypes/duration"

	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Trivial test to ensure orchestrator instantiates.
	upstreamClient, err := upstream.NewClient(context.Background(), "example.com")
	assert.NoError(t, err)

	config := aggregationv1.KeyerConfiguration{
		Fragments: []*aggregationv1.KeyerConfiguration_Fragment{
			{
				Rules: []*aggregationv1.KeyerConfiguration_Fragment_Rule{},
			},
		},
	}
	requestMapper := mapper.NewMapper(&config)

	cacheConfig := bootstrapv1.Cache{
		Ttl: &duration.Duration{
			Seconds: 10,
		},
		MaxEntries: 10,
	}

	orchestrator := New(context.Background(), log.New("info"), requestMapper, upstreamClient, &cacheConfig)
	assert.NotNil(t, orchestrator)
}
