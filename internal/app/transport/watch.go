package transport

import "github.com/uber-go/tally"

// Watch interface abstracts v2 and v3 watches to the downstream sidecars.
type Watch interface {
	// Close is idempotent with Send.
	// When Close and Send are called from separate goroutines they are guaranteed to not panic
	// Close waits until all Send operations drain.
	Close()
	// Send is a mutex protected function to send responses to the downstream sidecars.
	// It provides guarantee to never panic when called in tandem with Close from separate
	// goroutines. This also guarantees that stale responses are dropped in the event that a
	// newer response arrives.
	Send(Response) error
	// SetScope sets scope name for metrics/observability.
	SetScope(scope tally.Scope)
}
