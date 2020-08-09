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
)

// Config holds the configuration options for stats reporting.
type Config struct {
	// StatsdAddress that the statsd sink is running on, with format addr:port.
	StatsdAddress string
	// RootPrefix is the prefix for the root scope.
	RootPrefix string
	// The maximum interval for packet sending. If set to 0 it defaults to 300ms.
	FlushInterval time.Duration
}

// NewRootScope creates a new root Scope with the set of configured options and
// statsd reporter.
func NewRootScope(config Config) (tally.Scope, io.Closer, error) {
	// Configure statsd client for reporting stats.
	statsdClient, err := statsd.NewClientWithConfig(&statsd.ClientConfig{
		Address:       config.StatsdAddress,
		UseBuffered:   true,
		FlushInterval: config.FlushInterval,
	})
	if err != nil {
		return nil, nil, err
	}

	reporter := NewStatsdPointTagsReporter(statsdClient)

	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		Prefix:   config.RootPrefix,
		Tags:     map[string]string{},
		Reporter: reporter,
	}, time.Second)

	return scope, closer, nil
}
