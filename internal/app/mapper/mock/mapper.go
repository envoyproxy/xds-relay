package mock

import (
	"testing"

	"github.com/envoyproxy/xds-relay/internal/app/mapper"
)

func NewMapper(t *testing.T) mapper.Mapper {
	return mapper.NewMockMapper(t)
}
