package stats

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
)

type mockReporter struct {
	tally.StatsReporter

	actualName string
}

func (m *mockReporter) ReportCounter(name string, tags map[string]string, value int64) {
	m.actualName = name
}

var tests = []struct {
	name         string
	tags         map[string]string
	expectedTags map[string]string
}{
	{
		name:         "a.b.c",
		tags:         map[string]string{"tag1": "value1"},
		expectedTags: map[string]string{"tag1": "value1"},
	},
	{
		name:         "a-b.c-d",
		tags:         map[string]string{"tag1": "value|1", "tag2": "value.2"},
		expectedTags: map[string]string{"tag1": "value_1", "tag2": "value.2"},
	},
	{
		name:         "a",
		tags:         map[string]string{"tag1": "value-1", "tag2": "value 2", "tag:3": "value:3"},
		expectedTags: map[string]string{"tag1": "value-1", "tag2": "value_2", "tag_3": "value_3"},
	},
}

func TestPointTagReporter(t *testing.T) {
	for idx, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			t.Parallel()

			m := &mockReporter{}
			p := pointTagsReporter{
				StatsReporter: m,
			}
			p.ReportCounter(tt.name, tt.tags, 10)

			actualTags := map[string]string{}
			for _, v := range strings.Split(m.actualName, ".__")[1:] {
				ss := strings.Split(v, "=")
				actualTags[ss[0]] = ss[1]
			}

			assert.Equal(t, tt.expectedTags, actualTags)
			assert.True(t, strings.HasPrefix(m.actualName, tt.name))
		})
	}
}
