package cache

import (
	"regexp"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
)

type Rule = aggregationv1.KeyerConfiguration_Fragment_Rule

type Mapper interface {
	GetKeys(node core.Node, typeUrl string) string
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
				if isAnyMatch(matchpredicate) {
					return getResult(fragmentrule)
				}
			}

			andMatch := matchpredicate.GetAndMatch()
			if andMatch != nil {
				if isAndMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule)
				}
			}

			orMatch := matchpredicate.GetOrMatch()
			if orMatch != nil {
				if isOrMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule)
				}
			}

			notMatch := matchpredicate.GetNotMatch()
			if notMatch != nil {
				if isNotMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule)
				}
			}

			requestNodeMatch := matchpredicate.GetRequestNodeMatch()
			if requestNodeMatch != nil {
				if isNodeTypeMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule)
				}
			}

			requestTypeMatch := matchpredicate.GetRequestTypeMatch()
			if requestTypeMatch != nil {
				if isRequestTypeMatch(matchpredicate, node, typeUrl) {
					return getResult(fragmentrule)
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
		if !isMatchPredicate(rule) {
			return false
		}
	}
	return true
}

func isOrMatch(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) bool {
	matchset := matchPredicate.GetOrMatch()
	for _, rule := range matchset.GetRules() {
		if !isMatchPredicate(rule) {
			return true
		}
	}
	return false
}

func isNotMatch(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) bool {
	predicate := matchPredicate.GetNotMatch()
	if isMatchPredicate(predicate) {
		return false
	}
	return true
}

func isRequestTypeMatch(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) {
	predicate := matchPredicate.GetRequestTypeMatch()
	types := predicate.GetTypes()
	if types != nil {
		for _, type := range types {
			if type == typeUrl {
				return true
			}
		}
	}
	return false
}

func isNodeTypeMatch(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) {
	predicate := matchPredicate.GetRequestNodeMatch()
	nodeField := predicate.GetField()
	matchType := predicate.GetType()
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

	return return compare(predicate *aggregationv1.MatchPredicate, nodeValue string)
}

func compare(str string, predicate aggregationv1.MatchPredicate) bool {
	exactMatch := predicate.GetRequestNodeMatch().GetExactMatch()
	if exactMatch != "" {
		return str == exactMatch
	}

	regexMatch := predicate.GetRequestNodeMatch().GetRegexMatch()
	if regexMatch != "" {
		match, _ := regexp.MatchString(regexMatch, str)
		return match
	}

	return false
}

func isMatchPredicate(matchPredicate *aggregationv1.MatchPredicate, node core.Node, typeUrl string) bool {
	if matchPredicate.GetAnyMatch() {
		return isAnyMatch(matchPredicate)
	}

	if matchPredicate.GetAndMatch() != nil {
		return isAndMatch(matchPredicate)
	}

	if matchPredicate.GetOrMatch() != nil {
		return isOrMatch(matchPredicate)
	}

	if matchPredicate.GetNotMatch() != nil {
		return isNotMatch(matchPredicate)
	}

	if matchPredicate.GetRequestTypeMatch() != nil {
		return isRequestTypeMatch(matchPredicate)
	}

	return false
}

func getResult(fragmentRule *aggregationv1.KeyerConfiguration_Fragment_Rule) (string, error) {
	return "", nil
}
