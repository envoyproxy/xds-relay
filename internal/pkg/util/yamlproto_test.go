package yamlproto

import (
	"fmt"

	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"

	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
)

type AndResult = aggregationv1.ResultPredicate_AndResult
type Fragment = aggregationv1.KeyerConfiguration_Fragment
type FragmentRule = aggregationv1.KeyerConfiguration_Fragment_Rule
type KeyerConfiguration = aggregationv1.KeyerConfiguration
type MatchPredicate = aggregationv1.MatchPredicate
type RegexAction = aggregationv1.ResultPredicate_ResultAction_RegexAction
type RequestNodeFragment = aggregationv1.ResultPredicate_RequestNodeFragment
type RequestTypeMatch = aggregationv1.MatchPredicate_RequestTypeMatch
type ResourceNamesFragment = aggregationv1.ResultPredicate_ResourceNamesFragment
type ResultAction = aggregationv1.ResultPredicate_ResultAction
type ResultPredicate = aggregationv1.ResultPredicate
type StringFragment = aggregationv1.ResultPredicate_StringFragment

var positiveTests = []TableEntry{
	{
		Description: "test result predicate containing string_fragment",
		Parameters: []interface{}{
			"string_fragment.yaml",
			&ResultPredicate{
				Type: &StringFragment{
					StringFragment: "abc",
				},
			},
		},
	},
	{
		Description: "test result predicate containing resource_names_fragment",
		Parameters: []interface{}{
			"resource_names_fragment.yaml",
			&ResultPredicate{
				Type: &aggregationv1.ResultPredicate_ResourceNamesFragment_{
					ResourceNamesFragment: &ResourceNamesFragment{
						Field:   1,
						Element: 0,
						Action: &ResultAction{
							Action: &aggregationv1.ResultPredicate_ResultAction_RegexAction_{
								RegexAction: &RegexAction{
									Pattern: "some_regex",
									Replace: "a_replacement",
								},
							},
						},
					},
				},
			},
		},
	},
	{
		Description: "test result predicate containing request_node_fragment",
		Parameters: []interface{}{
			"request_node_fragment.yaml",
			&ResultPredicate{
				Type: &aggregationv1.ResultPredicate_RequestNodeFragment_{
					RequestNodeFragment: &RequestNodeFragment{
						Field: 2,
						Action: &ResultAction{
							Action: &aggregationv1.ResultPredicate_ResultAction_RegexAction_{
								RegexAction: &RegexAction{
									Pattern: "some_regex_for_node_fragment",
									Replace: "another_replacement",
								},
							},
						},
					},
				},
			},
		},
	},
	{
		Description: "test result predicate containing and_result",
		Parameters: []interface{}{
			"and_result.yaml",
			&ResultPredicate{
				Type: &aggregationv1.ResultPredicate_AndResult_{
					AndResult: &AndResult{
						ResultPredicates: []*ResultPredicate{
							{
								Type: &aggregationv1.ResultPredicate_ResourceNamesFragment_{
									ResourceNamesFragment: &ResourceNamesFragment{
										Field:   1,
										Element: 0,
										Action: &ResultAction{
											Action: &aggregationv1.ResultPredicate_ResultAction_RegexAction_{
												RegexAction: &RegexAction{
													Pattern: "some_regex",
													Replace: "a_replacement",
												},
											},
										},
									},
								},
							},
							{
								Type: &aggregationv1.ResultPredicate_RequestNodeFragment_{
									RequestNodeFragment: &RequestNodeFragment{
										Field: 2,
										Action: &ResultAction{
											Action: &aggregationv1.ResultPredicate_ResultAction_RegexAction_{
												RegexAction: &RegexAction{
													Pattern: "some_regex_for_node_fragment",
													Replace: "another_replacement",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
	{
		Description: "KeyerConfiguration + RequestTypeMatch + StringFragment",
		Parameters: []interface{}{
			"keyer_configuration_request_type_match_string_fragment.yaml",
			&KeyerConfiguration{
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
			},
		},
	},
}

var negativeTests = []TableEntry{
	{
		Description: "typo in fragments definition, instead of fragments we have fragmentss",
		Parameters: []interface{}{
			"typo_in_fragments.yaml",
			&KeyerConfiguration{},
			"proto: (line 1:2): unknown field \"fragmentss\"",
		},
	},
	{
		Description: "inexistent field in fragment",
		Parameters: []interface{}{
			"inexistent_field.yaml",
			&KeyerConfiguration{},
			"proto: (line 1:16): unknown field \"inexistent_field_in_fragment\"",
		},
	},
	{
		Description: "empty yaml",
		Parameters: []interface{}{
			"empty.yaml",
			&KeyerConfiguration{},
			"proto: syntax error (line 1:1): unexpected token null",
		},
	},
	{
		Description: "malformed second rule",
		Parameters: []interface{}{
			"error_in_second_rule.yaml",
			&KeyerConfiguration{},
			"proto: (line 1:16): unknown field \"ruless\"",
		},
	},
	{
		Description: "malformed yaml",
		Parameters: []interface{}{
			"invalid.yaml",
			&ResultPredicate{},
			"proto: syntax error (line 1:1): unexpected token \"some crazy yaml\\nbla\\n42\"",
		},
	},
}

var _ = Describe("Yamlprotoconverter", func() {
	DescribeTable("should be able to convert from yaml to proto",
		func(ymlFixtureFilename string, expectedProto proto.Message) {
			ymlBytes, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", ymlFixtureFilename))
			Expect(err).To(BeNil())
			// Get an empty copy of the expected proto to use as a recipient of the unmarshaling.
			protoToUnmarshal := proto.Clone(expectedProto)
			proto.Reset(protoToUnmarshal)
			err = FromYAMLToProto(string(ymlBytes), protoToUnmarshal)
			Expect(err).To(BeNil())
			Expect(proto.Equal(protoToUnmarshal, expectedProto)).To(Equal(true))
		},
		positiveTests...)

	DescribeTable("should not be able to convert from yaml to proto",
		func(ymlFixtureFilename string, protoToUnmarshal proto.Message, expectedErrorMessage string) {
			ymlBytes, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", ymlFixtureFilename))
			Expect(err).To(BeNil())
			err = FromYAMLToProto(string(ymlBytes), protoToUnmarshal)
			Expect(err.Error()).Should(Equal(expectedErrorMessage))
		},
		negativeTests...)
})
