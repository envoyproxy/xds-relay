package mapper

import (
	"fmt"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
)

// Mapper defines the interface that Maps an incoming request to an aggregation key
type Mapper interface {
	// GetKey converts a request into an aggregated key
	// Returns error if the regex parsing in the config fails to compile or match
	// Returns error if the typeURL is empty. An empty typeURL signifies an ADS request.
	// ref: envoyproxy/envoy/blob/d1a36f1ea24b38fc414d06ea29c5664f419066ef/api/envoy/api/v2/discovery.proto#L43-L46
	// ref: envoyproxy/go-control-plane/blob/1152177914f2ec0037411f65c56d9beae526870a/pkg/server/server.go#L305-L310
	// The go-control-plane will always traslate typeUrl implicitly to one of the resource typeUrls.
	GetKey(request v2.DiscoveryRequest, typeURL string) (string, error)
}

type mapper struct {
	config aggregationv1.KeyerConfiguration
}

// NewMapper constructs a concrete implementation for the Mapper interface
func NewMapper(config aggregationv1.KeyerConfiguration) Mapper {
	return &mapper{
		config: config,
	}
}

// GetKey converts a request into an aggregated key
func (mapper *mapper) GetKey(request v2.DiscoveryRequest, typeURL string) (string, error) {
	if typeURL == "" {
		return "", fmt.Errorf("Typeurl is empty")
	}
	return "", fmt.Errorf("Cannot map the input to a key")
}
