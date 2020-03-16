package mapper

import (
	"fmt"
	"strings"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
)

type matchPredicate = aggregationv1.MatchPredicate
type rule = aggregationv1.KeyerConfiguration_Fragment_Rule

// Mapper defines the interface that Maps an incoming request to an aggregation key
type Mapper interface {
	// GetKey converts a request into an aggregated key
	// Returns error if the regex parsing in the config fails to compile or match
	// An DiscoveryRequest will always contain typeUrl
	// ADS will contain typeUrl https://github.com/envoyproxy/envoy/blob/master/api/envoy/api/v2/discovery.proto#L46
	// Implicit xds requests will have typeUrl set because go-control-plane mutates the DiscoveeryRequest
	// ref: https://github.com/envoyproxy/go-control-plane/blob/master/pkg/server/server.go#L310
	GetKey(request v2.DiscoveryRequest) (string, error)
}

type mapper struct {
	config aggregationv1.KeyerConfiguration
}

const (
	separator = "_"
)

// NewMapper constructs a concrete implementation for the Mapper interface
func NewMapper(config aggregationv1.KeyerConfiguration) Mapper {
	return &mapper{
		config: config,
	}
}

// GetKey converts a request into an aggregated key
func (mapper *mapper) GetKey(request v2.DiscoveryRequest) (string, error) {
	if request.GetTypeUrl() == "" {
		return "", fmt.Errorf("typeURL is empty")
	}

	var resultFragments []string
	for _, fragment := range mapper.config.GetFragments() {
		fragmentRules := fragment.GetRules()
		for _, fragmentRule := range fragmentRules {
			matchPredicate := fragmentRule.GetMatch()
			isMatch := isMatch(matchPredicate, request.GetTypeUrl())
			if isMatch {
				result := getResult(fragmentRule)
				resultFragments = append(resultFragments, result)
			}
		}
	}

	if len(resultFragments) == 0 {
		return "", fmt.Errorf("Cannot map the input to a key")
	}

	return strings.Join(resultFragments, separator), nil
}

func isMatch(matchPredicate *matchPredicate, typeURL string) bool {
	return isRequestTypeMatch(matchPredicate, typeURL) || isAnyMatch(matchPredicate)
}

func isRequestTypeMatch(matchPredicate *matchPredicate, typeURL string) bool {
	predicate := matchPredicate.GetRequestTypeMatch()
	if predicate == nil {
		return false
	}

	for _, t := range predicate.GetTypes() {
		if t == typeURL {
			return true
		}
	}
	return false
}

func isAnyMatch(matchPredicate *matchPredicate) bool {
	return matchPredicate.GetAnyMatch()
}

func getResult(fragmentRule *rule) string {
	stringFragment := fragmentRule.GetResult().GetStringFragment()
	return stringFragment
}
