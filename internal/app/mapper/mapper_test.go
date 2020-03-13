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
	clusterTypeURL = "type.googleapis.com/envoy.api.v2.Cluster"
	nodeid         = "nodeid"
	nodecluster    = "cluster"
	noderegion     = "region"
	nodezone       = "zone"
	nodesubzone    = "subzone"
	stringfragment = "stringfragment"
)

var _ = Describe("GetKey", func() {
	DescribeTable("should be able to return error",
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
			Expect(err).Should(Equal(fmt.Errorf("Cannot map the input to a key")))
		},
		Entry("for all requests", getAnyMatch(false), getResultStringFragment(), clusterTypeURL, ""))

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
