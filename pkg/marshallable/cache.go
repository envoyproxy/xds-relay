package marshallable

// CDS is the succinct list of all clusters in the cache for a key
type CDS struct {
	Key     string
	Version string
	Names   []string
}

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
