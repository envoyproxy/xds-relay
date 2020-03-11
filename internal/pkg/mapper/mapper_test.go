package mapper

import (
	"fmt"

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
	clusterTypeURL  = "type.googleapis.com/envoy.api.v2.Cluster"
	listenerTypeURL = "type.googleapis.com/envoy.api.v2.Listener"
	nodeid          = "nodeid"
	nodecluster     = "cluster"
	noderegion      = "region"
	nodezone        = "zone"
	nodesubzone     = "subzone"
	stringfragment  = "stringfragment"
	NodeID          = aggregationv1.NodeFieldType_NODE_ID
	NodeCluster     = aggregationv1.NodeFieldType_NODE_CLUSTER
	NodeRegion      = aggregationv1.NodeFieldType_NODE_LOCALITY_REGION
	NodeZone        = aggregationv1.NodeFieldType_NODE_LOCALITY_ZONE
	NodeSubZone     = aggregationv1.NodeFieldType_NODE_LOCALITY_SUBZONE
)

var _ = Describe("GetKeys", func() {
	DescribeTable("should be able to return fragment for", func(match *MatchPredicate, result *ResultPredicate, typeurl string, assert string) {
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
		key, _ := mapper.GetKeys(getNode(), typeurl)
		Expect(key).To(Equal(assert))
	},
		Entry("AnyMatch returns StringFragment", getAnyMatch(true), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("AnyMatch With Exact Node Id match", getAnyMatch(true), getResultRequestNodeFragment(NodeID, getExactAction()), clusterTypeURL, nodeid),
		Entry("AnyMatch With Exact Node Cluster match", getAnyMatch(true), getResultRequestNodeFragment(NodeCluster, getExactAction()), clusterTypeURL, nodecluster),
		Entry("AnyMatch With Exact Node Region match", getAnyMatch(true), getResultRequestNodeFragment(NodeRegion, getExactAction()), clusterTypeURL, noderegion),
		Entry("AnyMatch With Exact Node Zone match", getAnyMatch(true), getResultRequestNodeFragment(NodeZone, getExactAction()), clusterTypeURL, nodezone),
		Entry("AnyMatch With Exact Node Subzone match", getAnyMatch(true), getResultRequestNodeFragment(NodeSubZone, getExactAction()), clusterTypeURL, nodesubzone),
		Entry("AnyMatch With result concatenation", getAnyMatch(true), getRepeatedResultPredicate1(), clusterTypeURL, nodeid+nodecluster),
		Entry("AnyMatch With regex action1", getAnyMatch(true), getResultRequestNodeFragment(NodeID, getRegexAction()), clusterTypeURL, "nTid"),
		Entry("AnyMatch With regex action2", getAnyMatch(true), getRepeatedResultPredicate2(), clusterTypeURL, "str"+noderegion+nodezone+"nTid"+nodecluster),
		Entry("RequestTypeMatch Matches with a single typeurl", getRequestTypeMatch([]string{clusterTypeURL}), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestTypeMatch Matches with multiple typeurl", getRequestTypeMatch([]string{clusterTypeURL, listenerTypeURL}), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestTypeMatch does not match with unmatched typeurl", getRequestTypeMatch([]string{""}), getResultStringFragment(stringfragment), clusterTypeURL, ""),
		Entry("RequestNodeMatch with node id exact match", getRequestNodeExactMatch(NodeID, nodeid), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestNodeMatch with node cluster exact match", getRequestNodeExactMatch(NodeCluster, nodecluster), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestNodeMatch with node region exact match", getRequestNodeExactMatch(NodeRegion, noderegion), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestNodeMatch with node zone exact match", getRequestNodeExactMatch(NodeZone, nodezone), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestNodeMatch with node subzone exact match", getRequestNodeExactMatch(NodeSubZone, nodesubzone), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestNodeMatch with node id regex match", getRequestNodeRegexMatch(NodeID, "n....d"), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestNodeMatch with node cluster regex match", getRequestNodeRegexMatch(NodeCluster, "c.....r"), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestNodeMatch with node region regex match", getRequestNodeRegexMatch(NodeRegion, "r....n"), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestNodeMatch with node zone regex match", getRequestNodeRegexMatch(NodeZone, "z..e"), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("RequestNodeMatch with node subzone regex match", getRequestNodeRegexMatch(NodeSubZone, "s.....e"), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("NotMatch RequestType", getRequestTypeNotMatch([]string{listenerTypeURL}), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("Not Match RequestNodeMatch with node id regex", getRequestNodeRegexNotMatch(NodeID, "notmatchregex"), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("Not Match RequestNodeMatch with node cluster regex", getRequestNodeRegexNotMatch(NodeCluster, "notmatchregex"), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("Not Match RequestNodeMatch with node region regex", getRequestNodeRegexNotMatch(NodeRegion, "notmatchregex"), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("Not Match RequestNodeMatch with node zone regex", getRequestNodeRegexNotMatch(NodeZone, "notmatchregex"), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("Not Match RequestNodeMatch with node subzone regex", getRequestNodeRegexNotMatch(NodeSubZone, "notmatchregex"), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("AndMatch RequestNodeMatch", getRequestNodeAndMatch([]*aggregationv1.MatchPredicate{getRequestNodeExactMatch(NodeID, nodeid), getRequestNodeExactMatch(NodeCluster, nodecluster)}), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment),
		Entry("OrMatch RequestNodeMatch", getRequestNodeOrMatch([]*aggregationv1.MatchPredicate{getRequestNodeExactMatch(NodeID, ""), getRequestNodeExactMatch(NodeCluster, nodecluster)}), getResultStringFragment(stringfragment), clusterTypeURL, stringfragment))

	DescribeTable("should be able to return error for non matching predicate", func(match *MatchPredicate, result *ResultPredicate, typeurl string) {
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
		key, err := mapper.GetKeys(getNode(), typeurl)
		Expect(key).To(Equal(""))
		Expect(err).Should(Equal(fmt.Errorf("Cannot map the input to a key")))
	},
		Entry("AnyMatch returns empty String", getAnyMatch(false), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestTypeMatch does not match with unmatched typeurl", getRequestTypeMatch([]string{""}), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestNodeMatch with node id does not match", getRequestNodeExactMatch(NodeID, "nonmatchingnode"), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestNodeMatch with node cluster does not match", getRequestNodeExactMatch(NodeCluster, "nonmatchingnode"), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestNodeMatch with node region does not match", getRequestNodeExactMatch(NodeRegion, "nonmatchingnode"), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestNodeMatch with node zone does not match", getRequestNodeExactMatch(NodeZone, "nonmatchingnode"), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestNodeMatch with node subzone does not match", getRequestNodeExactMatch(NodeSubZone, "nonmatchingnode"), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestNodeMatch with node id regex does not match", getRequestNodeRegexMatch(NodeID, "nonmatchingnode"), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestNodeMatch with node cluster regex does not match", getRequestNodeRegexMatch(NodeCluster, "nonmatchingnode"), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestNodeMatch with node region regex does not match", getRequestNodeRegexMatch(NodeRegion, "nonmatchingnode"), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestNodeMatch with node zone regex does not match", getRequestNodeRegexMatch(NodeZone, "nonmatchingnode"), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("RequestNodeMatch with node subzone regex does not match", getRequestNodeRegexMatch(NodeSubZone, "nonmatchingnode"), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("NotMatch RequestType", getRequestTypeNotMatch([]string{clusterTypeURL}), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("Not Match RequestNodeMatch with node id regex", getRequestNodeRegexNotMatch(NodeID, nodeid), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("Not Match RequestNodeMatch with node cluster regex", getRequestNodeRegexNotMatch(NodeCluster, nodecluster), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("Not Match RequestNodeMatch with node region regex", getRequestNodeRegexNotMatch(NodeRegion, noderegion), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("Not Match RequestNodeMatch with node zone regex", getRequestNodeRegexNotMatch(NodeZone, nodezone), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("Not Match RequestNodeMatch with node subzone regex", getRequestNodeRegexNotMatch(NodeSubZone, nodesubzone), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("AndMatch RequestNodeMatch does not match", getRequestNodeAndMatch([]*aggregationv1.MatchPredicate{getRequestNodeExactMatch(NodeID, "nonmatchingnode"), getRequestNodeExactMatch(NodeCluster, nodecluster)}), getResultStringFragment(stringfragment), clusterTypeURL),
		Entry("OrMatch RequestNodeMatch does not match", getRequestNodeOrMatch([]*aggregationv1.MatchPredicate{getRequestNodeExactMatch(NodeID, ""), getRequestNodeExactMatch(NodeCluster, "")}), getResultStringFragment(stringfragment), clusterTypeURL))
})

func getAnyMatch(any bool) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_AnyMatch{
			AnyMatch: any,
		},
	}
}

func getRequestTypeMatch(typeurls []string) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_RequestTypeMatch_{
			RequestTypeMatch: &aggregationv1.MatchPredicate_RequestTypeMatch{
				Types: typeurls,
			},
		},
	}
}

func getRequestNodeExactMatch(field aggregationv1.NodeFieldType, exact string) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_RequestNodeMatch_{
			RequestNodeMatch: &aggregationv1.MatchPredicate_RequestNodeMatch{
				Field: field,
				Type: &aggregationv1.MatchPredicate_RequestNodeMatch_ExactMatch{
					ExactMatch: exact,
				},
			},
		},
	}
}

func getRequestTypeNotMatch(typeurls []string) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_NotMatch{
			NotMatch: getRequestTypeMatch(typeurls),
		},
	}
}

func getRequestNodeRegexNotMatch(field aggregationv1.NodeFieldType, regex string) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_NotMatch{
			NotMatch: getRequestNodeRegexMatch(field, regex),
		},
	}
}

func getRequestNodeRegexMatch(field aggregationv1.NodeFieldType, regex string) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_RequestNodeMatch_{
			RequestNodeMatch: &aggregationv1.MatchPredicate_RequestNodeMatch{
				Field: field,
				Type: &aggregationv1.MatchPredicate_RequestNodeMatch_RegexMatch{
					RegexMatch: regex,
				},
			},
		},
	}
}

func getRequestNodeAndMatch(predicates []*MatchPredicate) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_AndMatch{
			AndMatch: &aggregationv1.MatchPredicate_MatchSet{
				Rules: predicates,
			},
		},
	}
}

func getRequestNodeOrMatch(predicates []*MatchPredicate) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_OrMatch{
			OrMatch: &aggregationv1.MatchPredicate_MatchSet{
				Rules: predicates,
			},
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
					getResultRequestNodeFragment(NodeID, getExactAction()),
					getResultRequestNodeFragment(NodeCluster, getExactAction()),
				},
			},
		},
	}
}
func getRepeatedResultPredicate2() *ResultPredicate {
	return &ResultPredicate{
		Type: &aggregationv1.ResultPredicate_ResultPredicate{
			ResultPredicate: &aggregationv1.ResultPredicate_RepeatedResultPredicate{
				AndResult: []*aggregationv1.ResultPredicate{
					&aggregationv1.ResultPredicate{
						Type: &aggregationv1.ResultPredicate_ResultPredicate{
							ResultPredicate: &aggregationv1.ResultPredicate_RepeatedResultPredicate{
								AndResult: []*aggregationv1.ResultPredicate{
									&aggregationv1.ResultPredicate{
										Type: &aggregationv1.ResultPredicate_StringFragment{
											StringFragment: "str",
										},
									},
									getResultRequestNodeFragment(NodeRegion, getExactAction()),
									getResultRequestNodeFragment(NodeZone, getExactAction()),
								},
							},
						},
					},
					getResultRequestNodeFragment(NodeID, getRegexAction()),
					getResultRequestNodeFragment(NodeCluster, getExactAction()),
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
			SubZone: nodesubzone,
		},
	}
}
