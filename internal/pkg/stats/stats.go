// Package stats uses uber-go/tally for reporting hierarchical stats. Tally
// supports multiple sinks including prometheus, m3, and statsd. xds-relay
// currently defaults to statsd, but this will be made configurable in the
// future to support the other sink alternatives.
package stats

import (
	"io"
	"time"

	"github.com/cactus/go-statsd-client/statsd"
	"github.com/uber-go/tally"
	tallystatsd "github.com/uber-go/tally/statsd"
)

// Config holds the configuration options for stats reporting.
type Config struct {
	// StatsdAddress that the statsd sink is running on, with format addr:port.
	StatsdAddress string
	// SampleRate is the metrics emission sample rate. This defaults to 1.0
	SampleRate float32
	// RootPrefix is the prefix for the root scope.
	RootPrefix string
}

// NewScope creates a new root Scope with the set of configured options and
// statsd reporter.
func NewScope(config Config) (tally.Scope, io.Closer, error) {
	// Configure statsd client for reporting stats.
	statsdClient, err := statsd.NewClientWithConfig(&statsd.ClientConfig{
		Address:     config.StatsdAddress,
		Prefix:      "stats",
		UseBuffered: true,
	})
	if err != nil {
		return nil, nil, err
	}

	reporter := tallystatsd.NewReporter(statsdClient, tallystatsd.Options{
		SampleRate: config.SampleRate,
	})

	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		Prefix:   config.RootPrefix,
		Tags:     map[string]string{},
		Reporter: reporter,
	}, time.Second)

	return scope, closer, nil
}
