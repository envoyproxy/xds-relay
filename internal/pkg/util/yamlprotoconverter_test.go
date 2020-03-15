package yamlprotoconverter_test

import (
	"fmt"
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

var s1 = `
string_fragment: abc
`

var _ = Describe("Yamlprotoconverter", func() {
	type TestCase struct {
		YAML             string
		ProtoToUnmarshal proto.Message
		ExpectedProto    proto.Message
	}
	DescribeTable("should be able to convert from yaml to proto",
		func(tc TestCase) {
			fmt.Println(tc.YAML)
			err := FromYAMLToProto(tc.YAML, tc.ProtoToUnmarshal)
			Expect(err).To(BeNil())
			Expect(proto.Equal(tc.ProtoToUnmarshal, tc.ExpectedProto)).To(Equal(true))
		},
		Entry("test 1", TestCase{
			YAML: `
string_fragment: abc
`,
			ProtoToUnmarshal: &ResultPredicate{},
			ExpectedProto: &ResultPredicate{
				Type: &StringFragment{
					StringFragment: "abc",
				},
			},
		}),
		// 		Entry("test 2", `
		// fragments:
		// - rules:
		//   - match:
		//       request_type_match:
		//         types:
		//         - type.googleapis.com/envoy.api.v2.Endpoint
		//         - type.googleapis.com/envoy.api.v2.Listener
		//     result:
		//       string_fragment: "abc"
		// `,
		// 			KeyerConfiguration{},
		// 			KeyerConfiguration{
		// 				Fragments: []*Fragment{
		// 					{
		// 						Rules: []*FragmentRule{
		// 							{
		// 								Match: &MatchPredicate{
		// 									Type: &aggregationv1.MatchPredicate_RequestTypeMatch_{
		// 										RequestTypeMatch: &RequestTypeMatch{
		// 											Types: []string{
		// 												"type.googleapis.com/envoy.api.v2.Endpoint",
		// 												"type.googleapis.com/envoy.api.v2.Listener",
		// 											},
		// 										},
		// 									},
		// 								},
		// 								Result: &ResultPredicate{
		// 									Type: &StringFragment{
		// 										StringFragment: "abc",
		// 									},
		// 								},
		// 							},
		// 						},
		// 					},
		// 				},
		// 			},
		// 		),
	)
})
