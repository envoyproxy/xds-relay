package transport

// Watch interface abstracts v2 and v3 watches
// It is a only once use object. Each new DiscoveryRequest from the downstream sidecars lead to
// the creation of new watch object.
type Watch interface {
	// Send is a mutex protected function to send responses to the downstream sidecars.
	// It can only be used once and subsequent invocations are noop.
	// This design was adopted due to the way go-control-plane uses CreateWatch and channels.
	// Every new DiscoveryRequest leads to the invocation of separate CreateWatch. However, the same
	// response channel is used across multiple invocations.
	// Due to the highly concurrent nature of the server, a fanout can happen in parallel while
	// cancel is called. In order to prevent multiple responses on a single watch, subsequent calls will be noop.
	// This is safe because after the response a new CreateWatch will create a new watch and send responses over it.
	Send(Response) error
}
