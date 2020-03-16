package yamlprotoconverter_test

import (
	. "github.com/envoyproxy/xds-relay/internal/pkg/util"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
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
			`
string_fragment: abc
`,
			&ResultPredicate{},
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
			`
resource_names_fragment:
  field: 1
  element: 0
  action:
    regex_action:
      pattern: "some_regex"
      replace: "a_replacement"
`,
			&ResultPredicate{},
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
			`
request_node_fragment:
  field: 2
  action:
    regex_action:
      pattern: "some_regex_for_node_fragment"
      replace: "another_replacement"
`,
			&ResultPredicate{},
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
			`
and_result:
  result_predicates:
    - resource_names_fragment:
        field: 1
        element: 0
        action:
          regex_action:
            pattern: "some_regex"
            replace: "a_replacement"
    - request_node_fragment:
        field: 2
        action:
          regex_action:
            pattern: "some_regex_for_node_fragment"
            replace: "another_replacement"
`,
			&ResultPredicate{},
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
			`
fragments:
- rules:
  - match:
      request_type_match:
        types:
        - type.googleapis.com/envoy.api.v2.Endpoint
        - type.googleapis.com/envoy.api.v2.Listener
    result:
      string_fragment: "abc"
`,
			&KeyerConfiguration{},
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

var _ = Describe("Yamlprotoconverter", func() {
	DescribeTable("should be able to convert from yaml to proto",
		func(yml string, protoToUnmarshal proto.Message, expectedProto proto.Message) {
			err := FromYAMLToProto(yml, protoToUnmarshal)
			Expect(err).To(BeNil())
			Expect(proto.Equal(protoToUnmarshal, expectedProto)).To(Equal(true))
		},
		positiveTests...)
})
