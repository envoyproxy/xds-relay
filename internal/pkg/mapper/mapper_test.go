package mapper

import (
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type KeyerConfiguration = aggregationv1.KeyerConfiguration
type Fragment = aggregationv1.KeyerConfiguration_Fragment
type FragmentRule = aggregationv1.KeyerConfiguration_Fragment_Rule
type MatchPredicate = aggregationv1.MatchPredicate
type ResultPredicate = aggregationv1.ResultPredicate

var _ = Describe("Test1", func() {
	It("Test it 2", func() {
		protoConfig := KeyerConfiguration{
			Fragments: []*Fragment{
				&Fragment{
					Rules: []*FragmentRule{
						&FragmentRule{
							Match: &MatchPredicate{
								Type: &aggregationv1.MatchPredicate_AnyMatch{
									AnyMatch: true,
								},
							},
							Result: &ResultPredicate{
								Type: &aggregationv1.ResultPredicate_StringFragment{
									StringFragment: "abc",
								},
							},
						},
					},
				},
			},
		}
		mapper := NewMapper(protoConfig)
		key, _ := mapper.GetKeys(core.Node{}, "")
		Expect(key).To(Equal("abc"))
	})
})
