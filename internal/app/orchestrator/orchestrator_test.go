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
