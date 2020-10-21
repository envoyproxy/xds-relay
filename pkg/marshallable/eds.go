package marshallable

// EDS is the succinct list of all endpoints in the cache for a key
type EDS struct {
	Key       string
	Version   string
	Endpoints []string
}
