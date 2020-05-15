package testutils

import (
	"sync"
	"testing"
)

func TestAssertSyncMapLen(t *testing.T) {
	var sm sync.Map

	sm.Store("foo", 1)
	sm.Store("foo", 2)
	sm.Store("bar", 1)
	AssertSyncMapLen(t, 2, &sm)

	sm.Delete("foo")
	AssertSyncMapLen(t, 1, &sm)
}
