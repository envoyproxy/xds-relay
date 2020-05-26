package testutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
)

func AssertCounterValue(t *testing.T, counters map[string]tally.CounterSnapshot, metric string, expected int) {
	// The tally library adds a plus sign to all metrics in the snapshot, this is to support
	// tags.
	assert.EqualValues(t, expected, counters[metric+"+"].Value())
}

func AssertGaugeValue(t *testing.T, gauges map[string]tally.GaugeSnapshot, metric string, expected int) {
	// The tally library adds a plus sign to all metrics in the snapshot, this is to support
	// tags.
	assert.EqualValues(t, expected, gauges[metric+"+"].Value())
}

func AssertTimerValue(t *testing.T, timers map[string]tally.TimerSnapshot, metric string, expected []time.Duration) {
	// The tally library adds a plus sign to all metrics in the snapshot, this is to support
	// tags.
	assert.EqualValues(t, expected, timers[metric+"+"].Values())
}
