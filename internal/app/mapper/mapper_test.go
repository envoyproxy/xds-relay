package mapper

import (
	"fmt"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
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
	clusterTypeURL   = "type.googleapis.com/envoy.api.v2.Cluster"
	listenerTypeURL  = "type.googleapis.com/envoy.api.v2.Listener"
	nodeid           = "nodeid"
	nodecluster      = "cluster"
	noderegion       = "region"
	nodezone         = "zone"
	nodesubzone      = "subzone"
	stringFragment   = "stringFragment"
	nodeIDField      = aggregationv1.NodeFieldType_NODE_ID
	nodeClusterField = aggregationv1.NodeFieldType_NODE_CLUSTER
	nodeRegionField  = aggregationv1.NodeFieldType_NODE_LOCALITY_REGION
	nodeZoneField    = aggregationv1.NodeFieldType_NODE_LOCALITY_ZONE
	nodeSubZoneField = aggregationv1.NodeFieldType_NODE_LOCALITY_SUBZONE
)

var postivetests = []TableEntry{
	{
		Description: "AnyMatch returns StringFragment",
		Parameters: []interface{}{
			getAnyMatch(true),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestTypeMatch Matches with a single typeurl",
		Parameters: []interface{}{
			getRequestTypeMatch([]string{clusterTypeURL}),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestTypeMatch Matches with first among multiple typeurl",
		Parameters: []interface{}{
			getRequestTypeMatch([]string{clusterTypeURL, listenerTypeURL}),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestTypeMatch Matches with second among multiple typeurl",
		Parameters: []interface{}{
			getRequestTypeMatch([]string{listenerTypeURL, clusterTypeURL}),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node id exact match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeIDField, nodeid),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node cluster exact match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeClusterField, nodecluster),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node region exact match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeRegionField, noderegion),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node zone exact match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeZoneField, nodezone),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node subzone exact match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeSubZoneField, nodesubzone),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node id regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeIDField, "n....d"),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node cluster regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeClusterField, "c.....r"),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node region regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeRegionField, "r....n"),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node zone regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeZoneField, "z..e"),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node subzone regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeSubZoneField, "s.....e"),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
}

var multiFragmentPositiveTests = []TableEntry{
	{
		Description: "both fragments match",
		Parameters: []interface{}{
			getAnyMatch(true),
			getAnyMatch(true),
			getResultStringFragment(),
			getResultStringFragment(),
			stringFragment + "_" + stringFragment,
		},
	},
	{
		Description: "first fragment match",
		Parameters: []interface{}{
			getAnyMatch(true),
			getAnyMatch(false),
			getResultStringFragment(),
			getResultStringFragment(),
			stringFragment,
		},
	},
	{
		Description: "second fragment match",
		Parameters: []interface{}{
			getAnyMatch(false),
			getAnyMatch(true),
			getResultStringFragment(),
			getResultStringFragment(),
			stringFragment,
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
			getRequestNodeExactMatch(nodeIDField, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node cluster does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeClusterField, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node region does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeRegionField, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node zone does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeZoneField, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node subzone does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeSubZoneField, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node id regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeIDField, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node cluster regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeClusterField, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node region regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeRegionField, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node zone regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeZoneField, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
	{
		Description: "RequestNodeMatch with node subzone regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeSubZoneField, "nonmatchingnode"),
			getResultStringFragment(),
		},
	},
}

var multiFragmentNegativeTests = []TableEntry{
	{
		Description: "no fragments match",
		Parameters: []interface{}{
			getAnyMatch(false),
			getAnyMatch(false),
			getResultStringFragment(),
			getResultStringFragment(),
		},
	},
}

var _ = Describe("GetKey", func() {
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
			key, err := mapper.GetKey(getDiscoveryRequest(), typeurl)
			Expect(key).To(Equal(assert))
			Expect(err).Should(BeNil())
		}, postivetests...)

	DescribeTable("should be able to return error",
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
			key, err := mapper.GetKey(getDiscoveryRequest(), clusterTypeURL)
			Expect(key).To(Equal(""))
			Expect(err).Should(Equal(fmt.Errorf("Cannot map the input to a key")))
		},
		negativeTests...)

	DescribeTable("should be able to join multiple fragments",
		func(match1 *MatchPredicate,
			match2 *MatchPredicate,
			result1 *ResultPredicate,
			result2 *ResultPredicate,
			expectedKey string) {
			protoConfig := KeyerConfiguration{
				Fragments: []*Fragment{
					{
						Rules: []*FragmentRule{
							{
								Match:  match1,
								Result: result1,
							},
						},
					},
					{
						Rules: []*FragmentRule{
							{
								Match:  match2,
								Result: result2,
							},
						},
					},
				},
			}
			mapper := NewMapper(protoConfig)
			key, err := mapper.GetKey(getDiscoveryRequest(), clusterTypeURL)
			Expect(expectedKey).To(Equal(key))
			Expect(err).Should(BeNil())
		},
		multiFragmentPositiveTests...)

	DescribeTable("should be able to return error for non matching multiple fragments",
		func(match1 *MatchPredicate, match2 *MatchPredicate, result1 *ResultPredicate, result2 *ResultPredicate) {
			protoConfig := KeyerConfiguration{
				Fragments: []*Fragment{
					{
						Rules: []*FragmentRule{
							{
								Match:  match1,
								Result: result1,
							},
						},
					},
					{
						Rules: []*FragmentRule{
							{
								Match:  match2,
								Result: result2,
							},
						},
					},
				},
			}
			mapper := NewMapper(protoConfig)
			key, err := mapper.GetKey(getDiscoveryRequest(), clusterTypeURL)
			Expect(key).To(Equal(""))
			Expect(err).Should(Equal(fmt.Errorf("Cannot map the input to a key")))
		},
		multiFragmentNegativeTests...)

	It("TypeUrl should not be empty", func() {
		mapper := NewMapper(KeyerConfiguration{})
		key, err := mapper.GetKey(getDiscoveryRequest(), "")
		Expect(key).To(Equal(""))
		Expect(err).Should(Equal(fmt.Errorf("typeURL is empty")))
	})
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

func getResultStringFragment() *ResultPredicate {
	return &ResultPredicate{
		Type: &aggregationv1.ResultPredicate_StringFragment{
			StringFragment: stringFragment,
		},
	}
}

func getDiscoveryRequest() v2.DiscoveryRequest {
	return v2.DiscoveryRequest{
		Node:          getNode(),
		VersionInfo:   "version",
		ResourceNames: []string{},
		TypeUrl:       clusterTypeURL,
	}
}

func getNode() *core.Node {
	return &core.Node{
		Id:      nodeid,
		Cluster: nodecluster,
		Locality: &core.Locality{
			Region:  noderegion,
			Zone:    nodezone,
			SubZone: nodesubzone,
		},
	}
}
