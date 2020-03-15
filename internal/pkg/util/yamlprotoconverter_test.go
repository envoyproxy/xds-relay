package yamlprotoconverter

import (
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	"github.com/stretchr/testify/require"
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
	yml := `
string_fragment: abc
`
	var rp ResultPredicate
	err := FromYAMLToProto(yml, &rp)
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
	yml := `
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

	kc := KeyerConfiguration{}
	err := FromYAMLToProto(yml, &kc)
	if err != nil {
		t.Error(err)
	}
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
