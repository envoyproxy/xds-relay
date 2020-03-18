package yamlproto

import (
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// fromYAMLToProto unmarshals a YAML string into a proto message.
func fromYAMLToProto(yml string, pb proto.Message) error {
	js, err := yaml.YAMLToJSON([]byte(yml))
	if err != nil {
		return err
	}
	err = protojson.Unmarshal(js, pb)
	if err != nil {
		return err
	}
	return nil
}

// FromYAMLToKeyerConfiguration unmarshals a YAML string into a KeyerConfiguration and validates it.
func FromYAMLToKeyerConfiguration(yml string, pb *aggregationv1.KeyerConfiguration) error {
	err := fromYAMLToProto(yml, pb)
	if err != nil {
		return err
	}
	err = pb.Validate()
	if err != nil {
		return err
	}
	return nil
}
