package testutils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// AssertSyncMapLen is a custom solution for checking the length of sync.Map.
// sync.Map does not currently offer support for length checks:
// https://github.com/golang/go/issues/20680
func AssertSyncMapLen(t *testing.T, len int, sm *sync.Map) {
	count := 0
	var mu sync.Mutex
	sm.Range(func(key, val interface{}) bool {
		mu.Lock()
		count++
		mu.Unlock()
		return true
	})
	assert.Equal(t, count, len)
}
