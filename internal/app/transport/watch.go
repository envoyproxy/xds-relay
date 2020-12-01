package transport

// Watch interface abstracts v2 and v3 watches
type Watch interface {
	// Send is a mutex protected function to send responses to the downstream sidecars.
	// It provides guarantee to never panic when calling in tandem with Close from separate goroutines.
	Send(Response) error
}
