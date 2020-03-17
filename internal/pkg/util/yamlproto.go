package yamlproto

import (
	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// FromYAMLToProto unmarshals a YAML string into a proto message.
func FromYAMLToProto(yml string, pb proto.Message) error {
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
