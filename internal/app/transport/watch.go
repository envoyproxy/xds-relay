package transport

// Watch interface abstracts v2 and v3 watches
type Watch interface {
	Close()
	Send(Response) bool
}
