package stats

import (
	"github.com/uber-go/tally"
)

// NewMockScope mocks a stats scope for testing.
func NewMockScope(prefix string) tally.TestScope {
	return tally.NewTestScope(prefix, make(map[string]string))
}
