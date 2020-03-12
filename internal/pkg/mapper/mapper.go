package mapper

import (
	"fmt"
	"regexp"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
)

type rule = aggregationv1.KeyerConfiguration_Fragment_Rule
type matchPredicate = aggregationv1.MatchPredicate
type repeatedResultPredicate = aggregationv1.ResultPredicate_RepeatedResultPredicate

// Mapper defines the interface that Maps an incoming request to an aggregation key
type Mapper interface {
	GetKey(node core.Node, typeURL string) (string, error)
}

type mapper struct {
	config aggregationv1.KeyerConfiguration
}

// NewMapper construts a concrete implementation for the Mapper interface
func NewMapper(config aggregationv1.KeyerConfiguration) Mapper {
	return &mapper{
		config: config,
	}
}

// GetKey converts a request into an aggregated key
func (mapper *mapper) GetKey(node core.Node, typeURL string) (string, error) {
	for _, fragment := range mapper.config.GetFragments() {
		fragmentrules := fragment.GetRules()
		for _, fragmentrule := range fragmentrules {
			matchpredicate := fragmentrule.GetMatch()
			isMatch, err := isMatchPredicate(matchpredicate, node, typeURL)
			if err != nil {
				return "", err
			}
			if isMatch {
				return getResult(fragmentrule, node)
			}
		}
	}
	return "", fmt.Errorf("Cannot map the input to a key")
}

func isAnyMatch(matchPredicate *matchPredicate) bool {
	return matchPredicate.GetAnyMatch()
}

func isAndMatch(matchPredicate *matchPredicate, node core.Node, typeURL string) (bool, error) {
	matchset := matchPredicate.GetAndMatch()
	for _, rule := range matchset.GetRules() {
		isMatch, err := isMatchPredicate(rule, node, typeURL)
		if err != nil {
			return false, err
		}
		if !isMatch {
			return false, nil
		}
	}
	return true, nil
}

func isOrMatch(matchPredicate *matchPredicate, node core.Node, typeURL string) (bool, error) {
	matchset := matchPredicate.GetOrMatch()
	for _, rule := range matchset.GetRules() {
		isMatch, err := isMatchPredicate(rule, node, typeURL)
		if err != nil {
			return false, err
		}
		if isMatch {
			return true, nil
		}
	}
	return false, nil
}

func isNotMatch(matchPredicate *matchPredicate, node core.Node, typeURL string) (bool, error) {
	predicate := matchPredicate.GetNotMatch()
	isMatch, err := isMatchPredicate(predicate, node, typeURL)
	if err != nil {
		return false, err
	}
	return !isMatch, nil
}

func isRequestTypeMatch(matchPredicate *matchPredicate, typeURL string) bool {
	predicate := matchPredicate.GetRequestTypeMatch()
	for _, t := range predicate.GetTypes() {
		if t == typeURL {
			return true
		}
	}
	return false
}

func isNodeTypeMatch(matchPredicate *matchPredicate, node core.Node) (bool, error) {
	predicate := matchPredicate.GetRequestNodeMatch()
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

func isMatchPredicate(matchPredicate *matchPredicate, node core.Node, typeURL string) (bool, error) {
	if matchPredicate.GetAnyMatch() {
		return isAnyMatch(matchPredicate), nil
	} else if matchPredicate.GetAndMatch() != nil {
		return isAndMatch(matchPredicate, node, typeURL)
	} else if matchPredicate.GetOrMatch() != nil {
		return isOrMatch(matchPredicate, node, typeURL)
	} else if matchPredicate.GetNotMatch() != nil {
		return isNotMatch(matchPredicate, node, typeURL)
	} else if matchPredicate.GetRequestTypeMatch() != nil {
		return isRequestTypeMatch(matchPredicate, typeURL), nil
	} else if matchPredicate.GetRequestNodeMatch() != nil {
		return isNodeTypeMatch(matchPredicate, node)
	}

	return false, nil
}

func getResult(fragmentRule *rule, node core.Node) (string, error) {
	stringFragment := fragmentRule.GetResult().GetStringFragment()
	if stringFragment != "" {
		return stringFragment, nil
	}

	requestNodeFragment := fragmentRule.GetResult().GetRequestNodeFragment()
	if requestNodeFragment != nil {
		nodeField := requestNodeFragment.GetField()
		var nodeValue = ""
		if nodeField == aggregationv1.NodeFieldType_NODE_CLUSTER {
			nodeValue = node.GetCluster()
		} else if nodeField == aggregationv1.NodeFieldType_NODE_ID {
			nodeValue = node.GetId()
		} else if nodeField == aggregationv1.NodeFieldType_NODE_LOCALITY_REGION {
			nodeValue = node.GetLocality().GetRegion()
		} else if nodeField == aggregationv1.NodeFieldType_NODE_LOCALITY_ZONE {
			nodeValue = node.GetLocality().GetZone()
		} else if nodeField == aggregationv1.NodeFieldType_NODE_LOCALITY_SUBZONE {
			nodeValue = node.GetLocality().GetSubZone()
		}

		action := requestNodeFragment.GetAction()
		if action.GetExact() {
			return nodeValue, nil
		}

		regexAction := action.GetRegexAction()
		pattern := regexAction.GetPattern()
		replace := regexAction.GetReplace()

		reg, err := regexp.Compile(pattern)
		if err != nil {
			return "", err
		}
		return reg.ReplaceAllString(nodeValue, replace), nil
	}

	if fragmentRule.GetResult().GetResultPredicate() != nil {
		return getResultPredicate(fragmentRule.GetResult().GetResultPredicate(), node)
	}
	return "", nil
}

func getRequestNodeFragment(predicate *aggregationv1.ResultPredicate, node core.Node) (string, error) {
	nodeField := predicate.GetRequestNodeFragment().GetField()
	var nodeValue = ""
	if nodeField == aggregationv1.NodeFieldType_NODE_CLUSTER {
		nodeValue = node.GetCluster()
	} else if nodeField == aggregationv1.NodeFieldType_NODE_ID {
		nodeValue = node.GetId()
	} else if nodeField == aggregationv1.NodeFieldType_NODE_LOCALITY_REGION {
		nodeValue = node.GetLocality().GetRegion()
	} else if nodeField == aggregationv1.NodeFieldType_NODE_LOCALITY_ZONE {
		nodeValue = node.GetLocality().GetZone()
	} else if nodeField == aggregationv1.NodeFieldType_NODE_LOCALITY_SUBZONE {
		nodeValue = node.GetLocality().GetSubZone()
	}

	action := predicate.GetRequestNodeFragment().GetAction()
	if action.GetExact() {
		return nodeValue, nil
	}

	regexAction := action.GetRegexAction()
	pattern := regexAction.GetPattern()
	replace := regexAction.GetReplace()

	reg, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	result := reg.ReplaceAllString(nodeValue, replace)
	return result, nil
}

func getResultPredicate(resultPredicate *repeatedResultPredicate, node core.Node) (string, error) {
	results := resultPredicate.GetAndResult()
	var resultfragments = ""
	for _, result := range results {
		if result.GetStringFragment() != "" {
			resultfragments = resultfragments + result.GetStringFragment()
		}

		if result.GetRequestNodeFragment() != nil {
			requestNodeFragment, err := getRequestNodeFragment(result, node)
			if err != nil {
				return "", err
			}
			resultfragments = resultfragments + requestNodeFragment
		}

		if result.GetResultPredicate() != nil {
			result, err := getResultPredicate(result.GetResultPredicate(), node)
			if err != nil {
				return "", err
			}
			resultfragments = resultfragments + result
		}
	}
	return resultfragments, nil
}
