package transport

// Watch interface abstracts v2 and v3 watches
type Watch interface {
	// Close is idempotent with Send.
	// When Close and Send are called from separate goroutines they are guaranteed to not panic
	// Close waits until all Send operations drain.
	Close()
	// Send is a mutex protected function to send responses to the downstream sidecars.
	// It provides guarantee to never panic when calling in tandem with Close from separate goroutines.
	Send(Response) error
}
