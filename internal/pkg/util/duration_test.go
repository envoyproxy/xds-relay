package util_test

import (
	"testing"
	"time"

	"github.com/envoyproxy/xds-relay/internal/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestDefaultDuration(t *testing.T) {
	dur, err := util.StringToDuration("", time.Second)
	assert.NoError(t, err)
	assert.Equal(t, dur, time.Second)
}
func TestConversion(t *testing.T) {
	dur, err := util.StringToDuration("10s", time.Second)
	assert.NoError(t, err)
	assert.Equal(t, dur, 10*time.Second)

	dur, err = util.StringToDuration("10m", time.Second)
	assert.NoError(t, err)
	assert.Equal(t, dur, 10*time.Minute)

	dur, err = util.StringToDuration("0s", time.Second)
	assert.NoError(t, err)
	assert.Equal(t, dur, 0*time.Second)

	_, err = util.StringToDuration("invalid", time.Second)
	assert.Error(t, err)
}
