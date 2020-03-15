package t

import (
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"testing"
)

type ResultPredicate = aggregationv1.ResultPredicate
type StringFragment = aggregationv1.ResultPredicate_StringFragment
type KeyerConfiguration = aggregationv1.KeyerConfiguration
type Fragment = aggregationv1.KeyerConfiguration_Fragment
type FragmentRule = aggregationv1.KeyerConfiguration_Fragment_Rule
type MatchPredicate = aggregationv1.MatchPredicate
type RequestTypeMatch = aggregationv1.MatchPredicate_RequestTypeMatch

func TestUnmarshaling(t *testing.T) {
	source := `
string_fragment: abc
`
	j2, err := yaml.YAMLToJSON([]byte(source))
	if err != nil {
		t.Error(err)
	}

	var rp ResultPredicate
	err = protojson.Unmarshal(j2, &rp)
	if err != nil {
		t.Error(err)
	}
	expected_rp := ResultPredicate{
		Type: &StringFragment{
			StringFragment: "abc",
		},
	}
	require.Equal(t, expected_rp.String(), rp.String())
}

func TestKeyerConfigurationUnmarshaling(t *testing.T) {
	source := `
fragments:
- rules:
  - match:
      request_type_match:
        types:
        - type.googleapis.com/envoy.api.v2.Endpoint
        - type.googleapis.com/envoy.api.v2.Listener
    result:
      string_fragment: "abc"
`
	yaml_contents, err := yaml.YAMLToJSON([]byte(source))
	if err != nil {
		t.Error(err)
	}

	kc := KeyerConfiguration{}
	err = protojson.Unmarshal(yaml_contents, &kc)
	expected_kc := KeyerConfiguration{
		Fragments: []*Fragment{
			{
				Rules: []*FragmentRule{
					{
						Match: &MatchPredicate{
							Type: &aggregationv1.MatchPredicate_RequestTypeMatch_{
								RequestTypeMatch: &RequestTypeMatch{
									Types: []string{
										"type.googleapis.com/envoy.api.v2.Endpoint",
										"type.googleapis.com/envoy.api.v2.Listener",
									},
								},
							},
						},
						Result: &ResultPredicate{
							Type: &StringFragment{
								StringFragment: "abc",
							},
						},
					},
				},
			},
		},
	}
	require.Equal(t, expected_kc.String(), kc.String())
}
