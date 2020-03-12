package mapper

import (
	"errors"
	"fmt"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
)


// Mapper defines the interface that Maps an incoming request to an aggregation key
type Mapper interface {
	GetKey(node core.Node, typeURL string) (string, error)
}

type mapper struct {
	config aggregationv1.KeyerConfiguration
}

// NewMapper construts a concrete implementation for the Mapper interface
func NewMapper(config aggregationv1.KeyerConfiguration) Mapper {
	return &mapper{
		config: config,
	}
}

// GetKey converts a request into an aggregated key
func (mapper *mapper) GetKey(node core.Node, typeURL string) (string, error) {
	return "", fmt.Errorf("Cannot map the input to a key")
}
