package mapper

import (
	"fmt"
	"regexp"
	"strings"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
)

type matchPredicate = aggregationv1.MatchPredicate
type rule = aggregationv1.KeyerConfiguration_Fragment_Rule

// Mapper defines the interface that Maps an incoming request to an aggregation key
type Mapper interface {
	// GetKey converts a request into an aggregated key
	// Returns error if the regex parsing in the config fails to compile or match
	// A DiscoveryRequest will always contain typeUrl
	// ADS will contain typeUrl https://github.com/envoyproxy/envoy/blob/master/api/envoy/api/v2/discovery.proto#L46
	// Implicit xds requests will have typeUrl set because go-control-plane mutates the DiscoveryRequest
	// ref: https://github.com/envoyproxy/go-control-plane/blob/master/pkg/server/server.go#L310
	GetKey(request v2.DiscoveryRequest) (string, error)
}

type mapper struct {
	config *aggregationv1.KeyerConfiguration
}

const (
	separator = "_"
)

// NewMapper constructs a concrete implementation for the Mapper interface
func NewMapper(config *aggregationv1.KeyerConfiguration) Mapper {
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
			isMatch, err := isMatch(matchPredicate, request.GetTypeUrl(), request.GetNode())
			if err != nil {
				return "", err
			}
			if isMatch {
				result, err := getResult(fragmentRule, request.GetNode())
				if err != nil {
					return "", err
				}
				resultFragments = append(resultFragments, result)
			}
		}
	}

	if len(resultFragments) == 0 {
		return "", fmt.Errorf("Cannot map the input to a key")
	}

	return strings.Join(resultFragments, separator), nil
}

func isMatch(matchPredicate *matchPredicate, typeURL string, node *core.Node) (bool, error) {
	isNodeMatch, err := isNodeMatch(matchPredicate, node)
	if err != nil {
		return false, err
	}
	if isNodeMatch {
		return true, nil
	}

	isAndMatch, err := isAndMatch(matchPredicate, typeURL, node)
	if err != nil {
		return false, err
	}
	if isAndMatch {
		return true, nil
	}

	isOrMatch, err := isOrMatch(matchPredicate, typeURL, node)
	if err != nil {
		return false, err
	}
	if isOrMatch {
		return true, nil
	}

	isNotMatch, err := isNotMatch(matchPredicate, typeURL, node)
	if err != nil {
		return false, err
	}
	if isNotMatch {
		return true, nil
	}

	return isRequestTypeMatch(matchPredicate, typeURL) || isAnyMatch(matchPredicate), nil
}

func isNodeMatch(matchPredicate *matchPredicate, node *core.Node) (bool, error) {
	predicate := matchPredicate.GetRequestNodeMatch()
	if predicate == nil {
		return false, nil
	}
	switch predicate.GetField() {
	case aggregationv1.NodeFieldType_NODE_CLUSTER:
		return compare(predicate, node.GetCluster())
	case aggregationv1.NodeFieldType_NODE_ID:
		return compare(predicate, node.GetId())
	case aggregationv1.NodeFieldType_NODE_LOCALITY_REGION:
		return compare(predicate, node.GetLocality().GetRegion())
	case aggregationv1.NodeFieldType_NODE_LOCALITY_ZONE:
		return compare(predicate, node.GetLocality().GetZone())
	case aggregationv1.NodeFieldType_NODE_LOCALITY_SUBZONE:
		return compare(predicate, node.GetLocality().GetSubZone())
	default:
		return false, nil
	}
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

func isAndMatch(matchPredicate *matchPredicate, typeURL string, node *core.Node) (bool, error) {
	matchset := matchPredicate.GetAndMatch()
	if matchset == nil {
		return false, nil
	}

	for _, rule := range matchset.GetRules() {
		isMatch, err := isMatch(rule, typeURL, node)
		if err != nil {
			return false, err
		}
		if !isMatch {
			return false, nil
		}
	}
	return true, nil
}

func isOrMatch(matchPredicate *matchPredicate, typeURL string, node *core.Node) (bool, error) {
	matchset := matchPredicate.GetOrMatch()
	if matchset == nil {
		return false, nil
	}

	for _, rule := range matchset.GetRules() {
		isMatch, err := isMatch(rule, typeURL, node)
		if err != nil {
			return false, err
		}
		if isMatch {
			return true, nil
		}
	}
	return false, nil
}

func isNotMatch(matchPredicate *matchPredicate, typeURL string, node *core.Node) (bool, error) {
	predicate := matchPredicate.GetNotMatch()
	if predicate == nil {
		return false, nil
	}

	isMatch, err := isMatch(predicate, typeURL, node)
	if err != nil {
		return false, err
	}
	return !isMatch, nil
}

func getResult(fragmentRule *rule, node *core.Node) (string, error) {
	found, result, err := getResultFromRequestNodeFragment(fragmentRule, node)
	if err != nil {
		return "", err
	}
	if found {
		return result, nil
	}

	stringFragment := fragmentRule.GetResult().GetStringFragment()
	return stringFragment, nil
}

func getResultFromRequestNodeFragment(fragmentRule *rule, node *core.Node) (bool, string, error) {
	requestNodeFragment := fragmentRule.GetResult().GetRequestNodeFragment()
	if requestNodeFragment == nil {
		return false, "", nil
	}

	nodeField := requestNodeFragment.GetField()
	var nodeValue = ""
	switch nodeField {
	case aggregationv1.NodeFieldType_NODE_CLUSTER:
		nodeValue = node.GetCluster()
	case aggregationv1.NodeFieldType_NODE_ID:
		nodeValue = node.GetId()
	case aggregationv1.NodeFieldType_NODE_LOCALITY_REGION:
		nodeValue = node.GetLocality().GetRegion()
	case aggregationv1.NodeFieldType_NODE_LOCALITY_ZONE:
		nodeValue = node.GetLocality().GetZone()
	case aggregationv1.NodeFieldType_NODE_LOCALITY_SUBZONE:
		nodeValue = node.GetLocality().GetSubZone()
	}

	action := requestNodeFragment.GetAction()
	if action.GetExact() {
		return true, nodeValue, nil
	}

	regexAction := action.GetRegexAction()
	pattern := regexAction.GetPattern()
	replace := regexAction.GetReplace()

	reg, err := regexp.Compile(pattern)
	if err != nil {
		return false, "", err
	}
	return true, reg.ReplaceAllString(nodeValue, replace), nil
}

func compare(requestNodeMatch *aggregationv1.MatchPredicate_RequestNodeMatch, nodeValue string) (bool, error) {
	exactMatch := requestNodeMatch.GetExactMatch()
	if exactMatch != "" {
		return nodeValue == exactMatch, nil
	}

	regexMatch := requestNodeMatch.GetRegexMatch()
	if regexMatch != "" {
		match, err := regexp.MatchString(regexMatch, nodeValue)
		if err != nil {
			return false, err
		}
		return match, nil
	}

	return false, nil
}
