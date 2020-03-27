package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// assert all types returns a non-nil logger.
	tests := []struct {
		name     string
		logLevel string
	}{
		{
			name:     "log level info",
			logLevel: "info",
		},
		{
			name:     "log level warn",
			logLevel: "warn",
		},
		{
			name:     "log level invalid",
			logLevel: "invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.logLevel)
			assert.NotNil(t, got)
		})
	}
}
