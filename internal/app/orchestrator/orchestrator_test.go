package orchestrator

import (
	"context"
	"testing"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	upstream_mock "github.com/envoyproxy/xds-relay/test/mocks/upstream"

	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Trivial test to ensure orchestrator instantiates.
	upstreamClient := upstream_mock.NewClient(
		context.Background(),
		upstream.CallOptions{},
		nil,
		nil,
		func(m interface{}) error { return nil })

	config := aggregationv1.KeyerConfiguration{
		Fragments: []*aggregationv1.KeyerConfiguration_Fragment{
			{
				Rules: []*aggregationv1.KeyerConfiguration_Fragment_Rule{},
			},
		},
	}
	requestMapper := mapper.NewMapper(&config)

	orchestrator := New(context.Background(), log.New("info"), requestMapper, upstreamClient)
	assert.NotNil(t, orchestrator)
}
