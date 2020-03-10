package mapper

import (
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type KeyerConfiguration = aggregationv1.KeyerConfiguration
type Fragment = aggregationv1.KeyerConfiguration_Fragment
type FragmentRule = aggregationv1.KeyerConfiguration_Fragment_Rule
type MatchPredicate = aggregationv1.MatchPredicate
type ResultPredicate = aggregationv1.ResultPredicate

const (
	clusterTypeURL = "type.googleapis.com/envoy.api.v2.Cluster"
	nodeid         = "nodeid"
	nodecluster    = "cluster"
	noderegion     = "region"
	nodezone       = "zone"
	snodesubzone   = "subzone"
)

var _ = Describe("GetKeys should be able to return fragment for", func() {
	stringfragment := "abc"
	DescribeTable("AnyMatch", func(match *MatchPredicate, result *ResultPredicate, assert string) {
		protoConfig := KeyerConfiguration{
			Fragments: []*Fragment{
				&Fragment{
					Rules: []*FragmentRule{
						&FragmentRule{
							Match:  match,
							Result: result,
						},
					},
				},
			},
		}
		mapper := NewMapper(protoConfig)
		key, _ := mapper.GetKeys(getNode(), clusterTypeURL)
		Expect(key).To(Equal(assert))
	},
		Entry("AnyMatch True returns StringFragment", getAnyMatch(true), getResultStringFragment(stringfragment), stringfragment),
		Entry("AnyMatch False returns empty String", getAnyMatch(false), getResultStringFragment(stringfragment), ""),
		Entry("AnyMatch True With Exact Node Id match", getAnyMatch(true), getResultRequestNodeFragment(aggregationv1.NodeFieldType_NODE_ID, getExactAction()), nodeid),
		Entry("AnyMatch True With Exact Node Cluster match", getAnyMatch(true), getResultRequestNodeFragment(aggregationv1.NodeFieldType_NODE_CLUSTER, getExactAction()), nodecluster),
		Entry("AnyMatch True With Exact Node Region match", getAnyMatch(true), getResultRequestNodeFragment(aggregationv1.NodeFieldType_NODE_LOCALITY_REGION, getExactAction()), noderegion),
		Entry("AnyMatch True With Exact Node Zone match", getAnyMatch(true), getResultRequestNodeFragment(aggregationv1.NodeFieldType_NODE_LOCALITY_ZONE, getExactAction()), nodezone),
		Entry("AnyMatch True With Exact Node Subzone match", getAnyMatch(true), getResultRequestNodeFragment(aggregationv1.NodeFieldType_NODE_LOCALITY_SUBZONE, getExactAction()), snodesubzone),
		Entry("AnyMatch True With result concatenation", getAnyMatch(true), getRepeatedResultPredicate1(), nodeid+nodecluster),
		Entry("AnyMatch True With regex action", getAnyMatch(true), getResultRequestNodeFragment(aggregationv1.NodeFieldType_NODE_ID, getRegexAction()), "nTid"))
})

func getAnyMatch(any bool) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_AnyMatch{
			AnyMatch: any,
		},
	}
}

func getResultStringFragment(fragment string) *ResultPredicate {
	return &ResultPredicate{
		Type: &aggregationv1.ResultPredicate_StringFragment{
			StringFragment: fragment,
		},
	}
}

func getResultRequestNodeFragment(
	field aggregationv1.NodeFieldType,
	action *aggregationv1.ResultPredicate_ResultAction) *ResultPredicate {
	return &ResultPredicate{
		Type: &aggregationv1.ResultPredicate_RequestNodeFragment_{
			RequestNodeFragment: &aggregationv1.ResultPredicate_RequestNodeFragment{
				Field:  field,
				Action: action,
			},
		},
	}
}

func getRepeatedResultPredicate1() *ResultPredicate {
	return &ResultPredicate{
		Type: &aggregationv1.ResultPredicate_ResultPredicate{
			ResultPredicate: &aggregationv1.ResultPredicate_RepeatedResultPredicate{
				AndResult: []*aggregationv1.ResultPredicate{
					getResultRequestNodeFragment(aggregationv1.NodeFieldType_NODE_ID, getExactAction()),
					getResultRequestNodeFragment(aggregationv1.NodeFieldType_NODE_CLUSTER, getExactAction()),
				},
			},
		},
	}
}

func getExactAction() *aggregationv1.ResultPredicate_ResultAction {
	return &aggregationv1.ResultPredicate_ResultAction{
		Action: &aggregationv1.ResultPredicate_ResultAction_Exact{
			Exact: true,
		},
	}
}

func getRegexAction() *aggregationv1.ResultPredicate_ResultAction {
	return &aggregationv1.ResultPredicate_ResultAction{
		Action: &aggregationv1.ResultPredicate_ResultAction_RegexAction_{
			RegexAction: &aggregationv1.ResultPredicate_ResultAction_RegexAction{
				Pattern: "ode",
				Replace: "T",
			},
		},
	}
}

func getNode() core.Node {
	return core.Node{
		Id:      nodeid,
		Cluster: nodecluster,
		Locality: &core.Locality{
			Region:  noderegion,
			Zone:    nodezone,
			SubZone: snodesubzone,
		},
	}
}
