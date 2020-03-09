package mapper

import (
	"regexp"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
)

type Rule = aggregationv1.KeyerConfiguration_Fragment_Rule

type Mapper interface {
	GetKeys(node core.Node, typeUrl string) (string, error)
}

type mapper struct {
	config aggregationv1.KeyerConfiguration
}

func NewMapper(config aggregationv1.KeyerConfiguration) Mapper {
	return &mapper{
		config: config,
	}
}

func (mapper *mapper) GetKeys(node core.Node, typeUrl string) (string, error) {
	for _, fragment := range mapper.config.GetFragments() {
		fragmentrules := fragment.GetRules()
		for _, fragmentrule := range fragmentrules {
			matchpredicate := fragmentrule.GetMatch()
			if matchpredicate.GetAnyMatch() {
				if isAnyMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule, node)
				}
			}

			andMatch := matchpredicate.GetAndMatch()
			if andMatch != nil {
				if isAndMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule, node)
				}
			}

			orMatch := matchpredicate.GetOrMatch()
			if orMatch != nil {
				if isOrMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule, node)
				}
			}

			notMatch := matchpredicate.GetNotMatch()
			if notMatch != nil {
				if isNotMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule, node)
				}
			}

			requestNodeMatch := matchpredicate.GetRequestNodeMatch()
			if requestNodeMatch != nil {
				if isNodeTypeMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule, node)
				}
			}

			requestTypeMatch := matchpredicate.GetRequestTypeMatch()
			if requestTypeMatch != nil {
				if isRequestTypeMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule, node)
				}
			}
		}
	}
	return "", nil
}

func isAnyMatch(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) bool {
	return matchPredicate.GetAnyMatch()
}

func isAndMatch(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) bool {
	matchset := matchPredicate.GetAndMatch()
	for _, rule := range matchset.GetRules() {
		if !isMatchPredicate(rule, node, typeUrl) {
			return false
		}
	}
	return true
}

func isOrMatch(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) bool {
	matchset := matchPredicate.GetOrMatch()
	for _, rule := range matchset.GetRules() {
		if !isMatchPredicate(rule, node, typeUrl) {
			return true
		}
	}
	return false
}

func isNotMatch(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) bool {
	predicate := matchPredicate.GetNotMatch()
	if isMatchPredicate(predicate, node, typeUrl) {
		return false
	}
	return true
}

func isRequestTypeMatch(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) bool {
	predicate := matchPredicate.GetRequestTypeMatch()
	types := predicate.GetTypes()
	if types != nil {
		for _, t := range types {
			if t == typeUrl {
				return true
			}
		}
	}
	return false
}

func isNodeTypeMatch(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) bool {
	predicate := matchPredicate.GetRequestNodeMatch()
	nodeField := predicate.GetField()
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

	return compare(predicate, nodeValue)
}

func compare(requestNodeMatch *aggregationv1.MatchPredicate_RequestNodeMatch, nodeValue string) bool {
	exactMatch := requestNodeMatch.GetExactMatch()
	if exactMatch != "" {
		return nodeValue == exactMatch
	}

	regexMatch := requestNodeMatch.GetRegexMatch()
	if regexMatch != "" {
		match, _ := regexp.MatchString(regexMatch, nodeValue)
		return match
	}

	return false
}

func isMatchPredicate(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) bool {
	if matchPredicate.GetAnyMatch() {
		return isAnyMatch(matchPredicate, node, typeUrl)
	}

	if matchPredicate.GetAndMatch() != nil {
		return isAndMatch(matchPredicate, node, typeUrl)
	}

	if matchPredicate.GetOrMatch() != nil {
		return isOrMatch(matchPredicate, node, typeUrl)
	}

	if matchPredicate.GetNotMatch() != nil {
		return isNotMatch(matchPredicate, node, typeUrl)
	}

	if matchPredicate.GetRequestTypeMatch() != nil {
		return isRequestTypeMatch(matchPredicate, node, typeUrl)
	}

	return false
}

func getResult(fragmentRule *aggregationv1.KeyerConfiguration_Fragment_Rule, node core.Node) (string, error) {
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

		reg, _ := regexp.Compile(pattern)
		return reg.ReplaceAllString(nodeValue, replace), nil
	}

	if fragmentRule.GetResult().GetResultPredicate() != nil {
		return getResultPredicate(fragmentRule.GetResult().GetResultPredicate(), node), nil
	}
	return "", nil
}

func getStringResult(predicate *aggregationv1.ResultPredicate) string {
	return predicate.GetStringFragment()
}

func getRequestNodeFragment(predicate *aggregationv1.ResultPredicate, node core.Node) string {
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
		return nodeValue
	}

	regexAction := action.GetRegexAction()
	pattern := regexAction.GetPattern()
	replace := regexAction.GetReplace()

	reg, _ := regexp.Compile(pattern)
	result := reg.ReplaceAllString(nodeValue, replace)
	return result
}

func getResultPredicate(resultPredicate *aggregationv1.ResultPredicate_RepeatedResultPredicate, node core.Node) string {
	results := resultPredicate.GetAndResult()
	var resultfragments = ""
	for _, result := range results {
		if result.GetStringFragment() != "" {
			resultfragments = resultfragments + result.GetStringFragment()
		}

		if result.GetRequestNodeFragment() != nil {
			resultfragments = resultfragments + getRequestNodeFragment(result, node)
		}

		if result.GetResultPredicate() != nil {
			resultfragments = resultfragments + getResultPredicate(result.GetResultPredicate(), node)
		}
	}
	return resultfragments
}
