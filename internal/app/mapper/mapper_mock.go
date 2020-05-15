package mapper

import (
	"io/ioutil"
	"testing"

	"github.com/envoyproxy/xds-relay/internal/pkg/util/yamlproto"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	"github.com/stretchr/testify/assert"
)

func NewMockMapper(t *testing.T) Mapper {
	bytes, err := ioutil.ReadFile("testdata/aggregation_rules.yaml") // key on request type
	assert.NoError(t, err)

	var config aggregationv1.KeyerConfiguration
	err = yamlproto.FromYAMLToKeyerConfiguration(string(bytes), &config)
	assert.NoError(t, err)

	return NewMapper(&config)
}
