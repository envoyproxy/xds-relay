package marshallable

// EDS is the succinct list of all endpoints in the cache for a key
type EDS struct {
	Key       string
	Version   string
	Endpoints []string
}

// Key is the marshallable list of all keys in the cache
type Key struct {
	Names []string
}

// Stream is the marshallable snapshot of a stream to the upstream control plane
type Stream struct {
	Key     string
	Version string
}
