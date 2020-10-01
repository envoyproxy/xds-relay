package mapper

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/envoyproxy/xds-relay/internal/app/metrics"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"google.golang.org/protobuf/types/known/structpb"

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

	// By construction only one of these is set at any point in time, so checking one by one
	// sequentially is ok.
	idMatch := predicate.GetIdMatch()
	if idMatch != nil {
		return compareString(idMatch, req.GetNodeID())
	}

	clusterMatch := predicate.GetClusterMatch()
	if clusterMatch != nil {
		return compareString(clusterMatch, req.GetCluster())
	}

	localityMatch := predicate.GetLocalityMatch()
	if localityMatch != nil {
		return compareLocality(localityMatch, req.GetLocality())
	}

	nodeMetadataMatch := predicate.GetNodeMetadataMatch()
	if nodeMetadataMatch != nil {
		return compareNodeMetadata(nodeMetadataMatch, req.GetNodeMetadata())
	}

	return false, fmt.Errorf("RequestNodeMatch is invalid")
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

	var resultFragment string
	var err error
	if requestNodeFragment.GetIdAction() != nil {
		resultFragment, err = getResultFragmentFromAction(req.GetNodeID(), requestNodeFragment.GetIdAction())
	} else if requestNodeFragment.GetClusterAction() != nil {
		resultFragment, err = getResultFragmentFromAction(req.GetCluster(), requestNodeFragment.GetClusterAction())
	} else if requestNodeFragment.GetLocalityAction() != nil {
		resultFragment, err = getFragmentFromLocalityAction(req.GetLocality(), requestNodeFragment.GetLocalityAction())
	} else if requestNodeFragment.GetNodeMetadataAction() != nil {
		resultFragment, err = getFragmentFromNodeMetadataAction(req.GetNodeMetadata(),
			requestNodeFragment.GetNodeMetadataAction())
	}

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

func getFragmentFromLocalityAction(
	locality *transport.Locality,
	action *aggregationv1.ResultPredicate_LocalityResultAction) (string, error) {
	var matches []string
	if action.RegionAction != nil {
		fragment, err := getResultFragmentFromAction(locality.Region, action.RegionAction)
		if err != nil {
			return "", err
		}
		matches = append(matches, fragment)
	}
	if action.ZoneAction != nil {
		fragment, err := getResultFragmentFromAction(locality.Zone, action.ZoneAction)
		if err != nil {
			return "", err
		}
		matches = append(matches, fragment)
	}
	if action.SubzoneAction != nil {
		fragment, err := getResultFragmentFromAction(locality.SubZone, action.SubzoneAction)
		if err != nil {
			return "", err
		}
		matches = append(matches, fragment)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("RequestNodeFragment match resulted in an empty fragment")
	}

	// N.B.: join matches using "|" to indicate they all came from the locality object.
	return strings.Join(matches, "|"), nil
}

func getFragmentFromNodeMetadataAction(
	nodeMetadata *structpb.Struct,
	action *aggregationv1.ResultPredicate_NodeMetadataAction) (string, error) {
	// Traverse to the right node
	var value *structpb.Value = nil
	var ok bool
	for _, segment := range action.GetPath() {
		fields := nodeMetadata.GetFields()
		value, ok = fields[segment.Key]
		if !ok {
			// TODO what to do if the key doesn't map to a valid struct field?
			return "", fmt.Errorf("Path to key is inexistent")
		}
		nodeMetadata = value.GetStructValue()
	}

	// Compare value with the one in the action
	// TODO: We need to stringify values other than strings (bool, integers, etc) before
	// extracting the fragment via getResultFragmentFromAction
	return getResultFragmentFromAction(value.GetStringValue(), action.GetAction())
}

func compareString(stringMatch *aggregationv1.StringMatch, nodeValue string) (bool, error) {
	if nodeValue == "" {
		return false, fmt.Errorf("MatchPredicate Node field cannot be empty")
	}
	exactMatch := stringMatch.GetExactMatch()
	if exactMatch != "" {
		return nodeValue == exactMatch, nil
	}

	regexMatch := stringMatch.GetRegexMatch()
	if regexMatch != "" {
		match, err := regexp.MatchString(regexMatch, nodeValue)
		if err != nil {
			return false, err
		}
		return match, nil
	}

	return false, nil
}

func compareBool(boolMatch *aggregationv1.BoolMatch, boolValue bool) bool {
	return boolMatch.ValueMatch == boolValue
}

func compareLocality(localityMatch *aggregationv1.LocalityMatch,
	reqNodeLocality *transport.Locality) (bool, error) {
	if reqNodeLocality == nil {
		return false, fmt.Errorf("Locality Node field cannot be empty")
	}

	regionMatch := true
	var err error
	if localityMatch.GetRegion() != nil {
		regionMatch, err = compareString(localityMatch.Region, reqNodeLocality.Region)
		if err != nil {
			return false, err
		}
	}

	zoneMatch := true
	if localityMatch.GetZone() != nil {
		zoneMatch, err = compareString(localityMatch.Zone, reqNodeLocality.Zone)
		if err != nil {
			return false, err
		}
	}

	subZoneMatch := true
	if localityMatch.GetSubZone() != nil {
		subZoneMatch, err = compareString(localityMatch.SubZone, reqNodeLocality.SubZone)
		if err != nil {
			return false, err
		}
	}

	return regionMatch && zoneMatch && subZoneMatch, nil
}

func compareNodeMetadata(nodeMetadataMatch *aggregationv1.NodeMetadataMatch,
	nodeMetadata *structpb.Struct) (bool, error) {
	if nodeMetadata == nil {
		return false, fmt.Errorf("Metadata Node field cannot be empty")
	}

	var value *structpb.Value = nil
	var ok bool
	for _, segment := range nodeMetadataMatch.GetPath() {
		// Starting from the second iteration, make sure that we're dealing with structs
		if value != nil {
			if value.GetStructValue() != nil {
				nodeMetadata = value.GetStructValue()
			} else {
				// TODO: signal that the field is not a struct
				return false, nil
			}
		}
		fields := nodeMetadata.GetFields()
		value, ok = fields[segment.Key]
		if !ok {
			return false, nil
		}
	}

	// TODO: implement the other structpb.Value types.
	if nodeMetadataMatch.Match.GetStringMatch() != nil {
		return compareString(nodeMetadataMatch.Match.GetStringMatch(), value.GetStringValue())
	} else if nodeMetadataMatch.Match.GetBoolMatch() != nil {
		return compareBool(nodeMetadataMatch.Match.GetBoolMatch(), value.GetBoolValue()), nil
	} else {
		return false, fmt.Errorf("Invalid NodeMetadata Match")
	}
}
