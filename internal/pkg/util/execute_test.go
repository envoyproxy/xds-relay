package util_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/envoyproxy/xds-relay/internal/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestExecuteWithinTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	methodExecuted := false
	err := util.DoWithTimeout(ctx, func() error {
		methodExecuted = true
		return nil
	}, time.Second)

	assert.Nil(t, err)
	assert.Equal(t, methodExecuted, true)
}

func TestExecuteWithinTimeoutReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	returnErr := fmt.Errorf("error")
	err := util.DoWithTimeout(ctx, func() error {
		return returnErr
	}, time.Second)

	assert.NotNil(t, err)
	assert.Equal(t, err, returnErr)
}

func TestExecuteWithinExceedsTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := util.DoWithTimeout(ctx, func() error {
		<-time.After(time.Millisecond)
		return nil
	}, time.Nanosecond)

	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "context deadline exceeded")
}

func TestExecuteCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	methodCalled := make(chan bool, 1)
	go func() {
		<-methodCalled
		cancel()
	}()
	err := util.DoWithTimeout(ctx, func() error {
		methodCalled <- true
		<-time.After(time.Minute)
		return nil
	}, time.Minute)

	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "context canceled")
}
