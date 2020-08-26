package mapper

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/envoyproxy/xds-relay/internal/app/metrics"
	"github.com/envoyproxy/xds-relay/internal/app/transport"

	"github.com/uber-go/tally"

	aggregationv1 "github.com/envoyproxy/xds-relay/pkg/api/aggregation/v1"
)

type matchPredicate = aggregationv1.MatchPredicate
type rule = aggregationv1.KeyerConfiguration_Fragment_Rule
type resultPredicate = aggregationv1.ResultPredicate

// Mapper defines the interface that Maps an incoming request to an aggregation key
type Mapper interface {
	// GetKey converts a request into an aggregated key
	// Returns error if the regex parsing in the config fails to compile or match
	// A DiscoveryRequest will always contain typeUrl
	// ADS will contain typeUrl https://github.com/envoyproxy/envoy/blob/master/api/envoy/api/v2/discovery.proto#L46
	// Implicit xds requests will have typeUrl set because go-control-plane mutates the DiscoveryRequest
	// ref: https://github.com/envoyproxy/go-control-plane/blob/master/pkg/server/server.go#L310
	GetKey(request transport.Request) (string, error)
}

type mapper struct {
	config *aggregationv1.KeyerConfiguration

	scope tally.Scope
}

const (
	separator = "_"
)

// New constructs a concrete implementation for the Mapper interface
func New(config *aggregationv1.KeyerConfiguration, scope tally.Scope) Mapper {
	return &mapper{
		config: config,
		scope:  scope.SubScope(metrics.ScopeMapper),
	}
}

// GetKey converts a request into an aggregated key
func (mapper *mapper) GetKey(request transport.Request) (string, error) {
	if request.GetTypeURL() == "" {
		mapper.scope.Counter(metrics.MapperError).Inc(1)
		return "", fmt.Errorf("typeURL is empty")
	}

	var resultFragments []string
	for _, fragment := range mapper.config.GetFragments() {
		fragmentRules := fragment.GetRules()
		for _, fragmentRule := range fragmentRules {
			matchPredicate := fragmentRule.GetMatch()
			isMatch, err := isMatch(matchPredicate, request.GetTypeURL(), request)
			if err != nil {
				mapper.scope.Counter(metrics.MapperError).Inc(1)
				return "", err
			}
			if isMatch {
				result, err := getResult(fragmentRule, request, request.GetResourceNames())
				if err != nil {
					mapper.scope.Counter(metrics.MapperError).Inc(1)
					return "", err
				}
				resultFragments = append(resultFragments, result)
			}
		}
	}

	if len(resultFragments) == 0 {
		mapper.scope.Counter(metrics.MapperError).Inc(1)
		return "", fmt.Errorf("Cannot map the input to a key")
	}

	mapper.scope.Counter(metrics.MapperSuccess).Inc(1)
	return strings.Join(resultFragments, separator), nil
}

func isMatch(matchPredicate *matchPredicate, typeURL string, req transport.Request) (bool, error) {
	isNodeMatch, err := isNodeMatch(matchPredicate, req)
	if err != nil {
		return false, err
	}
	if isNodeMatch {
		return true, nil
	}

	isAndMatch, err := isAndMatch(matchPredicate, typeURL, req)
	if err != nil {
		return false, err
	}
	if isAndMatch {
		return true, nil
	}

	isOrMatch, err := isOrMatch(matchPredicate, typeURL, req)
	if err != nil {
		return false, err
	}
	if isOrMatch {
		return true, nil
	}

	isNotMatch, err := isNotMatch(matchPredicate, typeURL, req)
	if err != nil {
		return false, err
	}
	if isNotMatch {
		return true, nil
	}

	return isRequestTypeMatch(matchPredicate, typeURL) || isAnyMatch(matchPredicate), nil
}

func isNodeMatch(matchPredicate *matchPredicate, req transport.Request) (bool, error) {
	predicate := matchPredicate.GetRequestNodeMatch()
	if predicate == nil {
		return false, nil
	}
	switch predicate.GetField() {
	case aggregationv1.NodeFieldType_NODE_CLUSTER:
		return compare(predicate, req.GetCluster())
	case aggregationv1.NodeFieldType_NODE_ID:
		return compare(predicate, req.GetNodeID())
	case aggregationv1.NodeFieldType_NODE_LOCALITY_REGION:
		return compare(predicate, req.GetRegion())
	case aggregationv1.NodeFieldType_NODE_LOCALITY_ZONE:
		return compare(predicate, req.GetZone())
	case aggregationv1.NodeFieldType_NODE_LOCALITY_SUBZONE:
		return compare(predicate, req.GetSubZone())
	default:
		return false, fmt.Errorf("RequestNodeMatch does not have a valid NodeFieldType")
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

func isAndMatch(matchPredicate *matchPredicate, typeURL string, req transport.Request) (bool, error) {
	matchset := matchPredicate.GetAndMatch()
	if matchset == nil {
		return false, nil
	}

	for _, rule := range matchset.GetRules() {
		isMatch, err := isMatch(rule, typeURL, req)
		if err != nil {
			return false, err
		}
		if !isMatch {
			return false, nil
		}
	}
	return true, nil
}

func isOrMatch(matchPredicate *matchPredicate, typeURL string, req transport.Request) (bool, error) {
	matchset := matchPredicate.GetOrMatch()
	if matchset == nil {
		return false, nil
	}

	for _, rule := range matchset.GetRules() {
		isMatch, err := isMatch(rule, typeURL, req)
		if err != nil {
			return false, err
		}
		if isMatch {
			return true, nil
		}
	}
	return false, nil
}

func isNotMatch(matchPredicate *matchPredicate, typeURL string, req transport.Request) (bool, error) {
	predicate := matchPredicate.GetNotMatch()
	if predicate == nil {
		return false, nil
	}

	isMatch, err := isMatch(predicate, typeURL, req)
	if err != nil {
		return false, err
	}
	return !isMatch, nil
}

func getResult(fragmentRule *rule, req transport.Request, resourceNames []string) (string, error) {
	found, result, err := getResultFromRequestNodeFragmentRule(fragmentRule, req)
	if err != nil {
		return "", err
	}
	if found {
		return result, nil
	}

	found, result, err = getResultFromAndResultFragmentRule(fragmentRule, req, resourceNames)
	if err != nil {
		return "", err
	}
	if found {
		return result, nil
	}

	found, result, err = getResultFromResourceNamesFragmentRule(fragmentRule, resourceNames)
	if err != nil {
		return "", err
	}
	if found {
		return result, nil
	}

	return fragmentRule.GetResult().GetStringFragment(), nil
}

func getResultFromRequestNodeFragmentRule(fragmentRule *rule, req transport.Request) (bool, string, error) {
	resultPredicate := fragmentRule.GetResult()
	if resultPredicate == nil {
		return false, "", nil
	}

	return getResultFromRequestNodePredicate(resultPredicate, req)
}

func getResultFromAndResultFragmentRule(
	fragmentRule *rule,
	req transport.Request,
	resourceNames []string) (bool, string, error) {
	resultPredicate := fragmentRule.GetResult()
	if resultPredicate == nil {
		return false, "", nil
	}
	return getResultFromAndResultPredicate(resultPredicate, req, resourceNames)
}

func getResultFromRequestNodePredicate(predicate *resultPredicate, req transport.Request) (bool, string, error) {
	requestNodeFragment := predicate.GetRequestNodeFragment()
	if requestNodeFragment == nil {
		return false, "", nil
	}

	nodeValue, err := getNodeValue(requestNodeFragment.GetField(), req)
	if err != nil {
		return false, "", err
	}
	resultFragment, err := getResultFragmentFromAction(nodeValue, requestNodeFragment.GetAction())
	if err != nil {
		return false, "", err
	}

	return true, resultFragment, nil
}

func getResultFromAndResultPredicate(
	resultPredicate *resultPredicate,
	req transport.Request,
	resourceNames []string) (bool, string, error) {
	if resultPredicate == nil {
		return false, "", nil
	}
	if resultPredicate.GetAndResult() == nil {
		return false, "", nil
	}

	results := resultPredicate.GetAndResult().GetResultPredicates()
	var resultfragments strings.Builder
	for _, result := range results {
		if result.GetStringFragment() != "" {
			resultfragments.WriteString(result.GetStringFragment())
		}

		found, fragment, err := getResultFromRequestNodePredicate(result, req)
		if err != nil {
			return false, "", err
		}
		if found {
			resultfragments.WriteString(fragment)
		}

		found, fragment, err = getResultFromResourceNamesPredicate(result, resourceNames)
		if err != nil {
			return false, "", err
		}
		if found {
			resultfragments.WriteString(fragment)
		}

		found, fragment, err = getResultFromAndResultPredicate(result, req, resourceNames)
		if err != nil {
			return false, "", err
		}
		if found {
			resultfragments.WriteString(fragment)
		}
	}
	return true, resultfragments.String(), nil
}

func getResultFromResourceNamesFragmentRule(
	fragmentRule *rule,
	resourceNames []string) (bool, string, error) {
	resultPredicate := fragmentRule.GetResult()
	if resultPredicate == nil {
		return false, "", nil
	}

	return getResultFromResourceNamesPredicate(resultPredicate, resourceNames)
}

func getResultFromResourceNamesPredicate(
	predicate *resultPredicate,
	resourceNames []string) (bool, string, error) {
	if predicate == nil {
		return false, "", nil
	}
	if predicate.GetResourceNamesFragment() == nil {
		return false, "", nil
	}

	resourceNamesFragment := predicate.GetResourceNamesFragment()

	index := resourceNamesFragment.GetElement()
	if index < 0 || index >= int32(len(resourceNames)) {
		return false, "", fmt.Errorf("ResourceNamesFragment.Element cannot be negative or larger than length")
	}
	resource := resourceNames[index]

	action := resourceNamesFragment.GetAction()
	result, err := getResultFragmentFromAction(resource, action)
	if err != nil {
		return false, "", err
	}
	return true, result, nil
}

func getNodeValue(nodeField aggregationv1.NodeFieldType, req transport.Request) (string, error) {
	var nodeValue string
	switch nodeField {
	case aggregationv1.NodeFieldType_NODE_CLUSTER:
		nodeValue = req.GetCluster()
	case aggregationv1.NodeFieldType_NODE_ID:
		nodeValue = req.GetNodeID()
	case aggregationv1.NodeFieldType_NODE_LOCALITY_REGION:
		nodeValue = req.GetRegion()
	case aggregationv1.NodeFieldType_NODE_LOCALITY_ZONE:
		nodeValue = req.GetZone()
	case aggregationv1.NodeFieldType_NODE_LOCALITY_SUBZONE:
		nodeValue = req.GetSubZone()
	default:
		return "", fmt.Errorf("RequestNodeFragment Invalid NodeFieldType")
	}

	return nodeValue, nil
}

func getResultFragmentFromAction(
	nodeValue string,
	action *aggregationv1.ResultPredicate_ResultAction) (string, error) {
	if action.GetExact() {
		if nodeValue == "" {
			return "", fmt.Errorf("RequestNodeFragment exact match resulted in an empty fragment")
		}
		return nodeValue, nil
	}

	regexAction := action.GetRegexAction()
	pattern := regexAction.GetPattern()
	replace := regexAction.GetReplace()

	reg, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	replacedFragment := reg.ReplaceAllString(nodeValue, replace)
	if replacedFragment == "" {
		return "", fmt.Errorf("RequestNodeFragment regex match resulted in an empty fragment")
	}

	return replacedFragment, nil
}

func compare(requestNodeMatch *aggregationv1.MatchPredicate_RequestNodeMatch, nodeValue string) (bool, error) {
	if nodeValue == "" {
		return false, fmt.Errorf("MatchPredicate Node field cannot be empty")
	}
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
