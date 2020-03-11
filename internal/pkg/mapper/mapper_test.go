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

var postivetests = []TableEntry{
	{
		Description: "AnyMatch returns StringFragment",
		Parameters: []interface{}{
			getAnyMatch(true),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "AnyMatch With Exact Node Id match",
		Parameters: []interface{}{
			getAnyMatch(true),
			getResultRequestNodeFragment(NodeID, getExactAction()),
			clusterTypeURL,
			nodeid,
		},
	},
	{
		Description: "AnyMatch With Exact Node Cluster match",
		Parameters: []interface{}{
			getAnyMatch(true),
			getResultRequestNodeFragment(NodeCluster, getExactAction()),
			clusterTypeURL,
			nodecluster,
		},
	},
	{
		Description: "AnyMatch With Exact Node Region match",
		Parameters: []interface{}{
			getAnyMatch(true),
			getResultRequestNodeFragment(NodeRegion, getExactAction()),
			clusterTypeURL,
			noderegion,
		},
	},
	{
		Description: "AnyMatch With Exact Node Zone match",
		Parameters: []interface{}{
			getAnyMatch(true),
			getResultRequestNodeFragment(NodeZone, getExactAction()),
			clusterTypeURL,
			nodezone,
		},
	},
	{
		Description: "AnyMatch With Exact Node Subzone match",
		Parameters: []interface{}{
			getAnyMatch(true),
			getResultRequestNodeFragment(NodeSubZone, getExactAction()),
			clusterTypeURL,
			nodesubzone,
		},
	},
	{
		Description: "AnyMatch With result concatenation",
		Parameters: []interface{}{
			getAnyMatch(true),
			getRepeatedResultPredicate1(),
			clusterTypeURL,
			nodeid + nodecluster,
		},
	},
	{
		Description: "AnyMatch With regex action1",
		Parameters: []interface{}{
			getAnyMatch(true),
			getResultRequestNodeFragment(NodeID, getRegexAction()),
			clusterTypeURL,
			"nTid",
		},
	},
	{
		Description: "AnyMatch With regex action2",
		Parameters: []interface{}{
			getAnyMatch(true),
			getRepeatedResultPredicate2(),
			clusterTypeURL,
			"str" + noderegion + nodezone + "nTid" + nodecluster,
		},
	},
	{
		Description: "RequestTypeMatch Matches with a single typeurl",
		Parameters: []interface{}{
			getRequestTypeMatch([]string{clusterTypeURL}),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestTypeMatch Matches with multiple typeurl",
		Parameters: []interface{}{
			getRequestTypeMatch([]string{clusterTypeURL, listenerTypeURL}),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestTypeMatch does not match with unmatched typeurl",
		Parameters: []interface{}{
			getRequestTypeMatch([]string{""}),
			getResultStringFragment(),
			clusterTypeURL,
			"",
		},
	},
	{
		Description: "RequestNodeMatch with node id exact match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(NodeID, nodeid),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestNodeMatch with node cluster exact match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(NodeCluster, nodecluster),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestNodeMatch with node region exact match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(NodeRegion, noderegion),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestNodeMatch with node zone exact match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(NodeZone, nodezone),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestNodeMatch with node subzone exact match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(NodeSubZone, nodesubzone),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestNodeMatch with node id regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(NodeID, "n....d"),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestNodeMatch with node cluster regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(NodeCluster, "c.....r"),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestNodeMatch with node region regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(NodeRegion, "r....n"),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestNodeMatch with node zone regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(NodeZone, "z..e"),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "RequestNodeMatch with node subzone regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(NodeSubZone, "s.....e"),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "NotMatch RequestType",
		Parameters: []interface{}{
			getRequestTypeNotMatch([]string{listenerTypeURL}),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node id regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(NodeID, "notmatchregex"),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node cluster regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(NodeCluster, "notmatchregex"),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node region regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(NodeRegion, "notmatchregex"),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node zone regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(NodeZone, "notmatchregex"),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node subzone regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(NodeSubZone, "notmatchregex"),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "AndMatch RequestNodeMatch",
		Parameters: []interface{}{
			getRequestNodeAndMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeExactMatch(NodeID, nodeid),
					getRequestNodeExactMatch(NodeCluster, nodecluster),
				}),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
	{
		Description: "OrMatch RequestNodeMatch",
		Parameters: []interface{}{
			getRequestNodeOrMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeExactMatch(NodeID, ""),
					getRequestNodeExactMatch(NodeCluster, nodecluster),
				}),
			getResultStringFragment(),
			clusterTypeURL,
			stringfragment,
		},
	},
}

var negativeTests = []TableEntry{
	{
		Description: "AnyMatch returns empty String",
		Parameters: []interface{}{
			getAnyMatch(false),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestTypeMatch does not match with unmatched typeurl",
		Parameters: []interface{}{
			getRequestTypeMatch([]string{""}),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node id does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(NodeID, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node cluster does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(NodeCluster, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node region does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(NodeRegion, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node zone does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(NodeZone, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node subzone does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(NodeSubZone, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node id regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(NodeID, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node cluster regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(NodeCluster, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node region regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(NodeRegion, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node zone regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(NodeZone, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node subzone regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(NodeSubZone, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "NotMatch RequestType",
		Parameters: []interface{}{
			getRequestTypeNotMatch([]string{clusterTypeURL}),
			getResultStringFragment(),
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node id regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(NodeID, nodeid),
			getResultStringFragment(),
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node cluster regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(NodeCluster, nodecluster),
			getResultStringFragment(),
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node region regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(NodeRegion, noderegion),
			getResultStringFragment(),
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node zone regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(NodeZone, nodezone),
			getResultStringFragment(),
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node subzone regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(NodeSubZone, nodesubzone),
			getResultStringFragment(),
		},
	},
	{
		Description: "AndMatch RequestNodeMatch does not match",
		Parameters: []interface{}{
			getRequestNodeAndMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeExactMatch(NodeID, "nonmatchingnode"),
					getRequestNodeExactMatch(NodeCluster, nodecluster)}),
			getResultStringFragment(),
		},
	},
	{
		Description: "OrMatch RequestNodeMatch does not match",
		Parameters: []interface{}{
			getRequestNodeOrMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeExactMatch(NodeID, ""),
					getRequestNodeExactMatch(NodeCluster, "")}),
			getResultStringFragment(),
		},
	},
}

var _ = Describe("GetKeys", func() {
	DescribeTable("should be able to return fragment for",
		func(match *MatchPredicate, result *ResultPredicate, typeurl string, assert string) {
			protoConfig := KeyerConfiguration{
				Fragments: []*Fragment{
					{
						Rules: []*FragmentRule{
							{
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
		}, postivetests...)

	DescribeTable("should be able to return error for non matching predicate",
		func(match *MatchPredicate, result *ResultPredicate) {
			protoConfig := KeyerConfiguration{
				Fragments: []*Fragment{
					{
						Rules: []*FragmentRule{
							{
								Match:  match,
								Result: result,
							},
						},
					},
				},
			}
			mapper := NewMapper(protoConfig)
			key, err := mapper.GetKeys(getNode(), clusterTypeURL)
			Expect(key).To(Equal(""))
			Expect(err).Should(Equal(fmt.Errorf("Cannot map the input to a key")))
		}, negativeTests...)
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

func getResultStringFragment() *ResultPredicate {
	return &ResultPredicate{
		Type: &aggregationv1.ResultPredicate_StringFragment{
			StringFragment: stringfragment,
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
					{
						Type: &aggregationv1.ResultPredicate_ResultPredicate{
							ResultPredicate: &aggregationv1.ResultPredicate_RepeatedResultPredicate{
								AndResult: []*aggregationv1.ResultPredicate{
									{
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
