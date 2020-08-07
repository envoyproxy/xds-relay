package stats

import (
	"strings"
	"time"

	"github.com/cactus/go-statsd-client/statsd"
	"github.com/uber-go/tally"
	tallystatsd "github.com/uber-go/tally/statsd"
)

func NewStatsdPointTagsReporter(statsdClient statsd.Statter) tally.StatsReporter {
	reporter := tallystatsd.NewReporter(statsdClient, tallystatsd.Options{})
	return &pointTagsReporter{
		StatsReporter: reporter,
	}
}

type pointTagsReporter struct {
	tally.StatsReporter
}

func (r *pointTagsReporter) ReportCounter(name string, tags map[string]string, value int64) {
	r.StatsReporter.ReportCounter(r.buildTaggedName(name, tags), nil, value)
}

func (r *pointTagsReporter) ReportGauge(name string, tags map[string]string, value float64) {
	r.StatsReporter.ReportGauge(r.buildTaggedName(name, tags), nil, value)
}

func (r *pointTagsReporter) ReportTimer(name string, tags map[string]string, interval time.Duration) {
	r.StatsReporter.ReportTimer(r.buildTaggedName(name, tags), nil, interval)
}

func (r *pointTagsReporter) buildTaggedName(name string, tags map[string]string) string {
	var b strings.Builder
	b.WriteString(name)
	for k, v := range tags {
		// We assume that the tags are of the form "a.b.c.__tag1=value1.__tag2=value2", where "a.b.c" is
		// a metric name.
		b.WriteString(".__")
		b.WriteString(sanitize(k))
		b.WriteByte('=')
		b.WriteString(sanitize(v))
	}
	return b.String()
}

// Instead of panic'ing in the case of invalid tags, we simply replace the invalid characters with
// the character '_'.
func sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isValidChar(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func isValidChar(r rune) bool {
	return ('0' <= r && r <= '9') || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || r == '-' || r == '_' || r == '.'
}
