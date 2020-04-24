package util

import (
	"context"
	"time"
)

// DoWithTimeout runs f and returns its error.
// If the timeout elapses first, returns a ctx timeout error instead.
func DoWithTimeout(ctx context.Context, f func() error, t time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, t)
	defer cancel()
	errChan := make(chan error, 1)
	go func() {
		errChan <- f()
		close(errChan)
	}()
	select {
	case <-timeoutCtx.Done():
		return timeoutCtx.Err()
	case err := <-errChan:
		return err
	}
}
