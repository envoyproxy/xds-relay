package yamlprotoconverter_test

import (
	. "github.com/envoyproxy/xds-relay/internal/pkg/util"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
)

type ResultPredicate = aggregationv1.ResultPredicate
type StringFragment = aggregationv1.ResultPredicate_StringFragment
type KeyerConfiguration = aggregationv1.KeyerConfiguration
type Fragment = aggregationv1.KeyerConfiguration_Fragment
type FragmentRule = aggregationv1.KeyerConfiguration_Fragment_Rule
type MatchPredicate = aggregationv1.MatchPredicate
type RequestTypeMatch = aggregationv1.MatchPredicate_RequestTypeMatch

var positiveTests = []TableEntry{
	{
		Description: "test result predicate containing a string fragment",
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
		Description: "test 2",
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
