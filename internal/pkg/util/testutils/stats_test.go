package testutils

import (
	"testing"
	"time"

	"github.com/uber-go/tally"
)

func TestSnapshotCounters(t *testing.T) {
	s := tally.NewTestScope("foo", make(map[string]string))

	s.Counter("beep").Inc(1)

	snap := s.Snapshot()
	counters := snap.Counters()
	AssertCounterValue(t, counters, "foo.beep", 1)

	s.Counter("beep").Inc(2)
	s.Counter("bop").Inc(42)

	snap = s.Snapshot()
	counters = snap.Counters()
	AssertCounterValue(t, counters, "foo.beep", 3)
	AssertCounterValue(t, counters, "foo.bop", 42)
}

func TestSnapshotGauges(t *testing.T) {
	s := tally.NewTestScope("foo", make(map[string]string))

	s.Gauge("beep").Update(1)

	snap := s.Snapshot()
	gauges := snap.Gauges()
	AssertGaugeValue(t, gauges, "foo.beep", 1)

	s.Gauge("beep").Update(2)
	s.Gauge("bop").Update(42)

	snap = s.Snapshot()
	gauges = snap.Gauges()
	AssertGaugeValue(t, gauges, "foo.beep", 2)
	AssertGaugeValue(t, gauges, "foo.bop", 42)
}

func TestSnapshotTimers(t *testing.T) {
	s := tally.NewTestScope("foo", make(map[string]string))

	s.Timer("beep").Record(time.Microsecond * 1)

	snap := s.Snapshot()
	timers := snap.Timers()
	AssertTimerValue(t, timers, "foo.beep", []time.Duration{1 * time.Microsecond})

	s.Timer("beep").Record(time.Hour * 2)
	s.Timer("bop").Record(time.Second * 42)

	snap = s.Snapshot()
	timers = snap.Timers()
	AssertTimerValue(t, timers, "foo.beep", []time.Duration{1 * time.Microsecond, 2 * time.Hour})
	AssertTimerValue(t, timers, "foo.bop", []time.Duration{42 * time.Second})
}
