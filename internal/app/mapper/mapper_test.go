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
	clusterTypeURL  = "type.googleapis.com/envoy.api.v2.Cluster"
	listenerTypeURL = "type.googleapis.com/envoy.api.v2.Listener"
	nodeid          = "nodeid"
	nodecluster     = "cluster"
	noderegion      = "region"
	nodezone        = "zone"
	nodesubzone     = "subzone"
	stringfragment  = "stringfragment"
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
}

var multiFragmentPositiveTests = []TableEntry{
	{
		Description: "both fragments match",
		Parameters: []interface{}{
			getAnyMatch(true),
			getAnyMatch(true),
			getResultStringFragment(),
			getResultStringFragment(),
			stringfragment + "_" + stringfragment,
		},
	},
	{
		Description: "first fragment match",
		Parameters: []interface{}{
			getAnyMatch(true),
			getAnyMatch(false),
			getResultStringFragment(),
			getResultStringFragment(),
			stringfragment,
		},
	},
	{
		Description: "second fragment match",
		Parameters: []interface{}{
			getAnyMatch(false),
			getAnyMatch(true),
			getResultStringFragment(),
			getResultStringFragment(),
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

func getResultStringFragment() *ResultPredicate {
	return &ResultPredicate{
		Type: &aggregationv1.ResultPredicate_StringFragment{
			StringFragment: stringfragment,
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
