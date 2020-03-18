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
			"(line 1:2): unknown field \"fragmentss\"",
		},
	},
	{
		Description: "inexistent field in fragment",
		Parameters: []interface{}{
			"inexistent_field.yaml",
			&KeyerConfiguration{},
			"(line 1:16): unknown field \"inexistent_field_in_fragment\"",
		},
	},
	{
		Description: "empty yaml",
		Parameters: []interface{}{
			"empty.yaml",
			&KeyerConfiguration{},
			"syntax error (line 1:1): unexpected token null",
		},
	},
	{
		Description: "malformed second rule",
		Parameters: []interface{}{
			"error_in_second_rule.yaml",
			&KeyerConfiguration{},
			"(line 1:16): unknown field \"ruless\"",
		},
	},
	{
		Description: "malformed yaml",
		Parameters: []interface{}{
			"invalid.yaml",
			&ResultPredicate{},
			"syntax error (line 1:1): unexpected token \"some crazy yaml\\nbla\\n42\"",
		},
	},
}

var positiveTestsForKeyerConfigurationProto = []TableEntry{
	{
		Description: "should be able to load valid KeyerConfiguration protos",
		Parameters: []interface{}{
			"keyer_configuration_request_type_match_string_fragment.yaml",
		},
	},
	{
		Description: "should be able to load valid KeyerConfiguration containing a not_match",
		Parameters: []interface{}{
			"keyer_configuration_not_match_request_type_match.yaml",
		},
	},
	{
		Description: "should be able to load valid KeyerConfiguration containing an or_match",
		Parameters: []interface{}{
			"keyer_configuration_or_match_request_type_match.yaml",
		},
	},
	{
		Description: "should be able to load valid KeyerConfiguration containing request_node_match",
		Parameters: []interface{}{
			"keyer_configuration_request_node_match_string_fragment.yaml",
		},
	},
}

var negativeTestsForKeyerConfigurationProto = []TableEntry{
	{
		Description: "empty list of RequestTypeMatch in the single rule",
		Parameters: []interface{}{
			"keyer_configuration_empty_request_type_match.yaml",
			`invalid KeyerConfiguration.Fragments[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment.Rules[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment_Rule.Match: embedded message failed validation | caused by: invalid MatchPredicate.RequestTypeMatch: embedded message failed validation | caused by: invalid MatchPredicate_RequestTypeMatch.Types: value must contain at least 1 item(s)`,
		},
	},
	{
		Description: "empty list of RequestTypeMatch in the second fragment",
		Parameters: []interface{}{
			"keyer_configuration_empty_request_type_math_in_second_rule.yaml",
			`invalid KeyerConfiguration.Fragments[1]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment.Rules[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment_Rule.Match: embedded message failed validation | caused by: invalid MatchPredicate.RequestTypeMatch: embedded message failed validation | caused by: invalid MatchPredicate_RequestTypeMatch.Types: value must contain at least 1 item(s)`,
		},
	},
	{
		Description: "single rule missing ResultPredicate",
		Parameters: []interface{}{
			"keyer_configuration_missing_result_predicate.yaml",
			`invalid KeyerConfiguration.Fragments[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment.Rules[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment_Rule.Result: value is required`,
		},
	},
	{
		Description: "single rule missing MatchPredicate",
		Parameters: []interface{}{
			"keyer_configuration_missing_match_predicate.yaml",
			`invalid KeyerConfiguration.Fragments[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment.Rules[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment_Rule.Match: value is required`,
		},
	},
	{
		Description: "any_match containing invalid boolean",
		Parameters: []interface{}{
			"keyer_configuration_match_predicate_invalid_bool.yaml",
			`invalid KeyerConfiguration.Fragments[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment.Rules[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment_Rule.Match: embedded message failed validation | caused by: invalid MatchPredicate.AnyMatch: value must equal true`,
		},
	},
	{
		Description: "not_match containing empty request_type_match",
		Parameters: []interface{}{
			"keyer_configuration_not_match_empty_request_type_match.yaml",
			`invalid KeyerConfiguration.Fragments[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment.Rules[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment_Rule.Match: embedded message failed validation | caused by: invalid MatchPredicate.NotMatch: embedded message failed validation | caused by: invalid MatchPredicate.RequestTypeMatch: embedded message failed validation | caused by: invalid MatchPredicate_RequestTypeMatch.Types: value must contain at least 1 item(s)`,
		},
	},
	{
		Description: "or_match containing a single matchset",
		Parameters: []interface{}{
			"keyer_configuration_or_match_invalid_request_type_match.yaml",
			`invalid KeyerConfiguration.Fragments[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment.Rules[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment_Rule.Match: embedded message failed validation | caused by: invalid MatchPredicate.OrMatch: embedded message failed validation | caused by: invalid MatchPredicate_MatchSet.Rules: value must contain at least 2 item(s)`,
		},
	},
	{
		Description: "reqesut_node_match cotaining invalid enum value",
		Parameters: []interface{}{
			"keyer_configuration_request_node_match_invalid_enum.yaml",
			`invalid KeyerConfiguration.Fragments[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment.Rules[0]: embedded message failed validation | caused by: invalid KeyerConfiguration_Fragment_Rule.Match: embedded message failed validation | caused by: invalid MatchPredicate.RequestNodeMatch: embedded message failed validation | caused by: invalid MatchPredicate_RequestNodeMatch.Field: value must be one of the defined enum values`,
		},
	},
}

var _ = Describe("yamlproto tests", func() {
	DescribeTable("should be able to convert from yaml to proto",
		func(ymlFixtureFilename string, expectedProto proto.Message) {
			ymlBytes, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", ymlFixtureFilename))
			Expect(err).To(BeNil())

			// Get an empty copy of the expected proto to use as a recipient of the unmarshaling.
			protoToUnmarshal := proto.Clone(expectedProto)
			proto.Reset(protoToUnmarshal)
			err = fromYAMLToProto(string(ymlBytes), protoToUnmarshal)
			Expect(err).To(BeNil())
			Expect(proto.Equal(protoToUnmarshal, expectedProto)).To(Equal(true))
		},
		positiveTests...)

	DescribeTable("should not be able to convert from yaml to proto",
		func(ymlFixtureFilename string, protoToUnmarshal proto.Message, expectedErrorMessage string) {
			ymlBytes, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", ymlFixtureFilename))
			Expect(err).To(BeNil())

			err = fromYAMLToProto(string(ymlBytes), protoToUnmarshal)
			Expect(err.Error()).Should(HaveSuffix(expectedErrorMessage))
		},
		negativeTests...)

	DescribeTable("should load and validate KeyerConfiguration",
		func(ymlFixtureFilename string) {
			ymlBytes, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", ymlFixtureFilename))
			Expect(err).To(BeNil())

			var kc KeyerConfiguration
			err = FromYAMLToKeyerConfiguration(string(ymlBytes), &kc)
			Expect(err).To(BeNil())
		},
		positiveTestsForKeyerConfigurationProto...)

	DescribeTable("should load yaml but validation for KeyerConfiguration should fail",
		func(ymlFixtureFilename string, expectedErrorMessage string) {
			ymlBytes, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", ymlFixtureFilename))
			Expect(err).To(BeNil())

			var kc KeyerConfiguration
			err = FromYAMLToKeyerConfiguration(string(ymlBytes), &kc)
			Expect(err.Error()).To(Equal(expectedErrorMessage))
		},
		negativeTestsForKeyerConfigurationProto...)
})
