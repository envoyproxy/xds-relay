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

var positiveTests = []TableEntry{
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
			getRequestNodeRegexMatch(nodeClusterField, "c*s*r"),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node region regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeRegionField, "^r....[n]"),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node zone regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeZoneField, "z..e$"),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "RequestNodeMatch with node subzone regex match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeSubZoneField, "[^0-9](u|v)b{1}...e"),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "AndMatch RequestNodeMatch",
		Parameters: []interface{}{
			getRequestNodeAndMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeExactMatch(nodeIDField, nodeid),
					getRequestNodeExactMatch(nodeClusterField, nodecluster),
				}),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "AndMatch recursive",
		Parameters: []interface{}{
			getRequestNodeAndMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeAndMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeExactMatch(nodeIDField, nodeid),
						getRequestNodeExactMatch(nodeClusterField, nodecluster),
					}),
					getRequestNodeAndMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeRegexMatch(nodeRegionField, noderegion),
						getRequestNodeRegexMatch(nodeZoneField, nodezone),
					}),
				}),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "OrMatch RequestNodeMatch first predicate",
		Parameters: []interface{}{
			getRequestNodeOrMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeExactMatch(nodeIDField, ""),
					getRequestNodeExactMatch(nodeClusterField, nodecluster),
				}),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "OrMatch RequestNodeMatch second predicate",
		Parameters: []interface{}{
			getRequestNodeOrMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeExactMatch(nodeIDField, nodeid),
					getRequestNodeExactMatch(nodeClusterField, ""),
				}),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "OrMatch recursive",
		Parameters: []interface{}{
			getRequestNodeOrMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeOrMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeExactMatch(nodeIDField, ""),
						getRequestNodeExactMatch(nodeClusterField, nodecluster),
					}),
					getRequestNodeOrMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeRegexMatch(nodeRegionField, noderegion),
						getRequestNodeRegexMatch(nodeZoneField, ""),
					}),
				}),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "AndMatch OrMatch recursive combined",
		Parameters: []interface{}{
			getRequestNodeAndMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeOrMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeExactMatch(nodeIDField, ""),
						getRequestNodeExactMatch(nodeClusterField, nodecluster),
					}),
					getRequestNodeAndMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeRegexMatch(nodeRegionField, noderegion),
						getRequestNodeRegexMatch(nodeZoneField, nodezone),
					}),
				}),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "NotMatch RequestType",
		Parameters: []interface{}{
			getRequestTypeNotMatch([]string{listenerTypeURL}),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node id regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(nodeIDField),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node cluster regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(nodeClusterField),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node region regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(nodeRegionField),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node zone regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(nodeZoneField),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node subzone regex",
		Parameters: []interface{}{
			getRequestNodeRegexNotMatch(nodeSubZoneField),
			getResultStringFragment(),
			clusterTypeURL,
			stringFragment,
		},
	},
	{
		Description: "Not Match recursive",
		Parameters: []interface{}{
			&MatchPredicate{
				Type: &aggregationv1.MatchPredicate_NotMatch{
					NotMatch: getRequestNodeAndMatch(
						[]*aggregationv1.MatchPredicate{
							getRequestNodeOrMatch([]*aggregationv1.MatchPredicate{
								getRequestNodeExactMatch(nodeIDField, ""),
								getRequestNodeExactMatch(nodeClusterField, ""),
							}),
							getRequestNodeAndMatch([]*aggregationv1.MatchPredicate{
								getRequestNodeExactNotMatch(nodeRegionField, ""),
								getRequestNodeRegexMatch(nodeZoneField, nodezone),
							}),
						}),
				},
			},
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
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestTypeMatch does not match with unmatched typeurl",
		Parameters: []interface{}{
			getRequestTypeMatch([]string{""}),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with node id does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeIDField, nodeid+"{5}"),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with node cluster does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeClusterField, "[^a-z]odecluster"),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with node region does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeRegionField, noderegion+"\\d"),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with node zone does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeZoneField, "zon[A-Z]"),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with node subzone does not match",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeSubZoneField, nodesubzone+"+"),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with node id regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeIDField, nodeid+"{5}"),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with node cluster regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeClusterField, "[^a-z]odecluster"),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with node region regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeRegionField, noderegion+"\\d"),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with node zone regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeZoneField, "zon[A-Z]"),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with node subzone regex does not match",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeSubZoneField, nodesubzone+"\\B"),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "RequestNodeMatch with exact match request node id empty",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeIDField, nodeid),
			getResultStringFragment(),
			getDiscoveryRequestWithNode(getNode("", nodecluster, noderegion, nodezone, nodesubzone)),
		},
	},
	{
		Description: "RequestNodeMatch with regex match request node id empty",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeIDField, nodeid),
			getResultStringFragment(),
			getDiscoveryRequestWithNode(getNode("", nodecluster, noderegion, nodezone, nodesubzone)),
		},
	},
	{
		Description: "RequestNodeMatch with exact match request node cluster empty",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeClusterField, nodecluster),
			getResultStringFragment(),
			getDiscoveryRequestWithNode(getNode(nodeid, "", noderegion, nodezone, nodesubzone)),
		},
	},
	{
		Description: "RequestNodeMatch with regex match request node cluster empty",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeClusterField, nodecluster),
			getResultStringFragment(),
			getDiscoveryRequestWithNode(getNode(nodeid, "", noderegion, nodezone, nodesubzone)),
		},
	},
	{
		Description: "RequestNodeMatch with exact match request node region empty",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeRegionField, noderegion),
			getResultStringFragment(),
			getDiscoveryRequestWithNode(getNode(nodeid, nodecluster, "", nodezone, nodesubzone)),
		},
	},
	{
		Description: "RequestNodeMatch with regex match request node region empty",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeRegionField, noderegion),
			getResultStringFragment(),
			getDiscoveryRequestWithNode(getNode(nodeid, nodecluster, "", nodezone, nodesubzone)),
		},
	},
	{
		Description: "RequestNodeMatch with exact match request node zone empty",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeZoneField, nodezone),
			getResultStringFragment(),
			getDiscoveryRequestWithNode(getNode(nodeid, nodecluster, noderegion, "", nodesubzone)),
		},
	},
	{
		Description: "RequestNodeMatch with regex match request node zone empty",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeZoneField, nodezone),
			getResultStringFragment(),
			getDiscoveryRequestWithNode(getNode(nodeid, nodecluster, noderegion, "", nodesubzone)),
		},
	},
	{
		Description: "RequestNodeMatch with exact match request node subzone empty",
		Parameters: []interface{}{
			getRequestNodeExactMatch(nodeSubZoneField, nodesubzone),
			getResultStringFragment(),
			getDiscoveryRequestWithNode(getNode(nodeid, nodecluster, noderegion, nodezone, "")),
		},
	},
	{
		Description: "RequestNodeMatch with regex match request node subzone empty",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeSubZoneField, nodesubzone),
			getResultStringFragment(),
			getDiscoveryRequestWithNode(getNode(nodeid, nodecluster, noderegion, nodezone, "")),
		},
	},
	{
		Description: "AndMatch RequestNodeMatch does not match first predicate",
		Parameters: []interface{}{
			getRequestNodeAndMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeExactMatch(nodeIDField, "nonmatchingnode"),
					getRequestNodeExactMatch(nodeClusterField, nodecluster)}),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "AndMatch RequestNodeMatch does not match second predicate",
		Parameters: []interface{}{
			getRequestNodeAndMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeExactMatch(nodeIDField, nodeid),
					getRequestNodeExactMatch(nodeClusterField, "nomatch")}),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "AndMatch recursive",
		Parameters: []interface{}{
			getRequestNodeAndMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeAndMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeExactMatch(nodeIDField, "nonmatchingnode"),
						getRequestNodeExactMatch(nodeClusterField, nodecluster),
					}),
					getRequestNodeExactMatch(nodeClusterField, nodecluster)}),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "OrMatch RequestNodeMatch does not match",
		Parameters: []interface{}{
			getRequestNodeOrMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeExactMatch(nodeIDField, ""),
					getRequestNodeExactMatch(nodeClusterField, "")}),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "OrMatch recursive",
		Parameters: []interface{}{
			getRequestNodeOrMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeOrMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeExactMatch(nodeIDField, ""),
						getRequestNodeExactMatch(nodeClusterField, ""),
					}),
					getRequestNodeOrMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeRegexMatch(nodeRegionField, ""),
						getRequestNodeRegexMatch(nodeZoneField, ""),
					}),
				}),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "AndMatch OrMatch recursive combined",
		Parameters: []interface{}{
			getRequestNodeAndMatch(
				[]*aggregationv1.MatchPredicate{
					getRequestNodeOrMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeExactMatch(nodeIDField, ""),
						getRequestNodeExactMatch(nodeClusterField, ""),
					}),
					getRequestNodeAndMatch([]*aggregationv1.MatchPredicate{
						getRequestNodeRegexMatch(nodeRegionField, ""),
						getRequestNodeRegexMatch(nodeZoneField, ""),
					}),
				}),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "NotMatch RequestType",
		Parameters: []interface{}{
			getRequestTypeNotMatch([]string{clusterTypeURL}),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node id regex",
		Parameters: []interface{}{
			getRequestNodeExactNotMatch(nodeIDField, nodeid),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node cluster regex",
		Parameters: []interface{}{
			getRequestNodeExactNotMatch(nodeClusterField, nodecluster),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node region regex",
		Parameters: []interface{}{
			getRequestNodeExactNotMatch(nodeRegionField, noderegion),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node zone regex",
		Parameters: []interface{}{
			getRequestNodeExactNotMatch(nodeZoneField, nodezone),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "Not Match RequestNodeMatch with node subzone regex",
		Parameters: []interface{}{
			getRequestNodeExactNotMatch(nodeSubZoneField, nodesubzone),
			getResultStringFragment(),
			getDiscoveryRequest(),
		},
	},
	{
		Description: "Not Match recursive",
		Parameters: []interface{}{
			&MatchPredicate{
				Type: &aggregationv1.MatchPredicate_NotMatch{
					NotMatch: getRequestNodeAndMatch(
						[]*aggregationv1.MatchPredicate{
							getRequestNodeOrMatch([]*aggregationv1.MatchPredicate{
								getRequestNodeExactMatch(nodeIDField, nodeid),
								getRequestNodeExactMatch(nodeClusterField, nodecluster),
							}),
							getRequestNodeAndMatch([]*aggregationv1.MatchPredicate{
								getRequestNodeRegexMatch(nodeRegionField, noderegion),
								getRequestNodeRegexMatch(nodeZoneField, nodezone),
							}),
						}),
				},
			},
			getResultStringFragment(),
			getDiscoveryRequest(),
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

var regexpErrorCases = []TableEntry{
	{
		Description: "Regex compilation failure in Nodefragment should return error",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeSubZoneField, "\xbd\xb2"),
			getResultStringFragment(),
		},
	},
}

var regexpErrorCasesMultipleFragments = []TableEntry{
	{
		Description: "Regex parse failure in first request predicate",
		Parameters: []interface{}{
			getRequestNodeRegexMatch(nodeSubZoneField, "\xbd\xb2"),
			getAnyMatch(true),
			getResultStringFragment(),
			getResultStringFragment(),
		},
	},
	{
		Description: "Regex parse failure in second request predicate",
		Parameters: []interface{}{
			getAnyMatch(true),
			getRequestNodeRegexMatch(nodeSubZoneField, "\xbd\xb2"),
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
			mapper := NewMapper(&protoConfig)
			request := getDiscoveryRequest()
			request.TypeUrl = typeurl
			key, err := mapper.GetKey(request)
			Expect(key).To(Equal(assert))
			Expect(err).Should(BeNil())
		}, positiveTests...)

	DescribeTable("should be able to return error",
		func(match *MatchPredicate, result *ResultPredicate, request v2.DiscoveryRequest) {
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
			mapper := NewMapper(&protoConfig)
			key, err := mapper.GetKey(request)
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
			mapper := NewMapper(&protoConfig)
			key, err := mapper.GetKey(getDiscoveryRequest())
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
			mapper := NewMapper(&protoConfig)
			key, err := mapper.GetKey(getDiscoveryRequest())
			Expect(key).To(Equal(""))
			Expect(err).Should(Equal(fmt.Errorf("Cannot map the input to a key")))
		},
		multiFragmentNegativeTests...)

	DescribeTable("should be able to return error for regex evaluation failures",
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
			mapper := NewMapper(&protoConfig)
			key, err := mapper.GetKey(getDiscoveryRequest())
			Expect(key).To(Equal(""))
			Expect(err.Error()).Should(Equal("error parsing regexp: invalid UTF-8: `\xbd\xb2`"))
		},
		regexpErrorCases...)

	DescribeTable("should return error for multiple fragments regex failure",
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
			mapper := NewMapper(&protoConfig)
			key, err := mapper.GetKey(getDiscoveryRequest())
			Expect(key).To(Equal(""))
			Expect(err.Error()).Should(Equal("error parsing regexp: invalid UTF-8: `\xbd\xb2`"))
		},
		regexpErrorCasesMultipleFragments...)

	It("TypeUrl should not be empty", func() {
		mapper := NewMapper(&KeyerConfiguration{})
		request := getDiscoveryRequest()
		request.TypeUrl = ""
		key, err := mapper.GetKey(request)
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

func getRequestTypeNotMatch(typeurls []string) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_NotMatch{
			NotMatch: getRequestTypeMatch(typeurls),
		},
	}
}

func getRequestNodeRegexNotMatch(field aggregationv1.NodeFieldType) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_NotMatch{
			NotMatch: getRequestNodeRegexMatch(field, "notmatchregex"),
		},
	}
}

func getRequestNodeExactNotMatch(field aggregationv1.NodeFieldType, regex string) *MatchPredicate {
	return &MatchPredicate{
		Type: &aggregationv1.MatchPredicate_NotMatch{
			NotMatch: getRequestNodeExactMatch(field, regex),
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
	return getDiscoveryRequestWithNode(getNode(nodeid, nodecluster, noderegion, nodezone, nodesubzone))
}

func getDiscoveryRequestWithNode(node *core.Node) v2.DiscoveryRequest {
	return v2.DiscoveryRequest{
		Node:          node,
		VersionInfo:   "version",
		ResourceNames: []string{},
		TypeUrl:       clusterTypeURL,
	}
}

func getNode(id string, cluster string, region string, zone string, subzone string) *core.Node {
	return &core.Node{
		Id:      id,
		Cluster: cluster,
		Locality: &core.Locality{
			Region:  region,
			Zone:    zone,
			SubZone: subzone,
		},
	}
}
