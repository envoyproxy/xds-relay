package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	secretv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	runtimev3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/xds-relay/internal/app/cache"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/envoyproxy/xds-relay/internal/pkg/util/stringify"
	"github.com/envoyproxy/xds-relay/pkg/marshallable"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	resource2 "github.com/envoyproxy/go-control-plane/pkg/resource/v2"
	resource3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

func edsDumpHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		edsKey := filepath.Base(req.URL.Path)
		if edsKey == "" {
			w.WriteHeader(http.StatusBadRequest)
			s, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: "Empty key",
			})
			_, _ = w.Write([]byte(s))
		}

		c := orchestrator.Orchestrator.GetReadOnlyCache(*o)
		resp, err := c.FetchReadOnly(edsKey)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			s, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: err.Error(),
			})
			_, _ = w.Write([]byte(s))
			return
		}

		versionedResponse := resp.Resp.Get()
		m := &marshallable.EDS{
			Key:       edsKey,
			Version:   resp.Resp.GetPayloadVersion(),
			Endpoints: make([]string, 0),
		}
		if versionedResponse.V2 != nil {
			r2 := marshalResources(versionedResponse.V2.Resources)
			for _, r := range r2.Endpoints {
				endpoint, _ := r.(*v2.ClusterLoadAssignment)
				if endpoint == nil {
					continue
				}
				for _, e := range endpoint.Endpoints {
					for _, lbe := range e.LbEndpoints {
						if lbe.GetEndpoint() == nil {
							continue
						}
						m.Endpoints = append(m.Endpoints, lbe.GetEndpoint().Address.GetSocketAddress().Address)
					}
				}
			}
		} else if versionedResponse.V3 != nil {
			r3 := marshalResources(versionedResponse.V3.Resources)
			for _, r := range r3.Endpoints {
				endpoint, _ := r.(*endpointv3.ClusterLoadAssignment)
				if endpoint == nil {
					continue
				}
				for _, e := range endpoint.Endpoints {
					for _, lbe := range e.LbEndpoints {
						if lbe.GetEndpoint() == nil {
							continue
						}
						m.Endpoints = append(m.Endpoints, lbe.GetEndpoint().Address.GetSocketAddress().Address)
					}
				}
			}
		}

		x, e := stringify.InterfaceToString(m)
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, e = w.Write([]byte(x))
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func versionHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		key := filepath.Base(req.URL.Path)
		if key == "" {
			w.WriteHeader(http.StatusBadRequest)
			s, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: "Empty key",
			})
			_, _ = w.Write([]byte(s))
		}

		c := orchestrator.Orchestrator.GetReadOnlyCache(*o)
		resp, err := c.FetchReadOnly(key)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			s, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: err.Error(),
			})
			_, _ = w.Write([]byte(s))
			return
		}

		version := &marshallable.Version{
			Version: resp.Resp.GetPayloadVersion(),
		}
		x, e := stringify.InterfaceToString(version)
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, e = w.Write([]byte(x))
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func keyDumpHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		allKeys, err := orchestrator.Orchestrator.GetDownstreamAggregatedKeys(*o)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errMessage, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: fmt.Sprintf("error in getting cache keys: %s", err.Error()),
			})
			_, _ = w.Write([]byte(errMessage))
			return
		}

		keys := make([]string, 0)
		for k := range allKeys {
			keys = append(keys, k)
		}

		response := &marshallable.Key{
			Names: keys,
		}
		marshalledKeys, err := stringify.InterfaceToString(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errMessage, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: fmt.Sprintf("error in marshalling keys: %s", err.Error()),
			})
			_, _ = w.Write([]byte(errMessage))
			return
		}

		_, e := w.Write([]byte(marshalledKeys))
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	}
}

func cacheDumpHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		cacheKey := getParam(req.URL.Path)
		c := orchestrator.Orchestrator.GetReadOnlyCache(*o)
		keysToPrint, err := getRelevantKeys(o, cacheKey, w)
		if err == nil {
			printCacheEntries(keysToPrint, c, w)
		}
	}
}

func clearCacheHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			cacheKey := getParam(req.URL.Path)
			keysToClear, err := getRelevantKeys(o, cacheKey, w)
			if err == nil {
				clearCacheEntries(keysToClear, o, w)
			}
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type marshallableResource struct {
	Key            string
	Resp           *marshalledDiscoveryResponse
	Requests       []types.Resource
	ExpirationTime time.Time
}

type marshallableCache struct {
	Cache []marshallableResource
}

func getRelevantKeys(o *orchestrator.Orchestrator, key string, w http.ResponseWriter) ([]string, error) {
	var relevantKeys []string
	// If wildcard suffix provided, retrieve all cache keys that match the given prefix.
	// If no key is provided, retrieve all keys.
	if hasWildcardSuffix(key) {
		// Retrieve all keys
		allKeys, err := orchestrator.Orchestrator.GetDownstreamAggregatedKeys(*o)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errMessage, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: fmt.Sprintf("error in retrieving downstream aggregated keys: %s", err.Error()),
			})
			_, _ = w.Write([]byte(errMessage))
			return nil, err
		}
		// Find keys that match prefix of wildcard
		rootCacheKeyName := strings.TrimSuffix(key, "*")
		for potentialMatchKey := range allKeys {
			if strings.HasPrefix(potentialMatchKey, rootCacheKeyName) {
				relevantKeys = append(relevantKeys, potentialMatchKey)
			}
		}
	} else {
		// Otherwise return singular key.
		relevantKeys = []string{key}
	}
	return relevantKeys, nil
}

func printCacheEntries(keys []string, cache cache.ReadOnlyCache, w http.ResponseWriter) {
	resp := marshallableCache{}
	for _, key := range keys {
		resource, err := cache.FetchReadOnly(key)
		if err == nil {
			resp.Cache = append(resp.Cache, resourceToPayload(key, resource)...)
		}
	}
	resourceString, err := stringify.InterfaceToString(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "unable to convert resource to string.\n")
		return
	}

	if len(resp.Cache) > 0 {
		fmt.Fprintf(w, "%s\n", resourceString)
	}
}

func clearCacheEntries(keys []string, o *orchestrator.Orchestrator, w http.ResponseWriter) {
	errors := (*o).ClearCacheEntries(keys)
	if len(errors) > 0 {
		var aggregatedError string
		for _, err := range errors {
			aggregatedError += err.Error()
		}
		errMessage, _ := stringify.InterfaceToString(&marshallable.Error{
			Message: aggregatedError,
		})
		_, _ = w.Write([]byte(errMessage))
	}
}

// hasWildcardSuffix returns whether the supplied key contains an empty string or a * suffix.
// Return true in these scenarios.
func hasWildcardSuffix(key string) bool {
	return key == "" || strings.HasSuffix(key, "*")
}

// In order to marshal a Resource from the cache to JSON to be printed,
// the map of requests is converted to a slice of just the keys,
// since the bool value is meaningless.
func resourceToPayload(key string, resource cache.Resource) []marshallableResource {
	var marshallableResources []marshallableResource
	var requests []types.Resource
	resource.Requests.ForEach(func(request transport.Request) {
		if request.GetRaw().V2 != nil {
			requests = append(requests, request.GetRaw().V2)
		} else {
			requests = append(requests, request.GetRaw().V3)
		}
	})

	marshallableResources = append(marshallableResources, marshallableResource{
		Key:            key,
		Resp:           marshalDiscoveryResponse(resource.Resp),
		Requests:       requests,
		ExpirationTime: resource.ExpirationTime,
	})

	return marshallableResources
}

type marshalledDiscoveryResponse struct {
	VersionInfo  string
	Resources    *xDSResources
	Canary       bool
	TypeURL      string
	Nonce        string
	ControlPlane types.Resource
}

type xDSResources struct {
	Endpoints    []types.Resource
	Clusters     []types.Resource
	Routes       []types.Resource
	Listeners    []types.Resource
	Secrets      []types.Resource
	Runtimes     []types.Resource
	Unmarshalled []*any.Any
}

func marshalDiscoveryResponse(r transport.Response) *marshalledDiscoveryResponse {
	if r != nil {
		if r.Get().V2 != nil {
			resp := r.Get().V2
			marshalledResp := marshalledDiscoveryResponse{
				VersionInfo:  resp.VersionInfo,
				Canary:       resp.Canary,
				TypeURL:      resp.TypeUrl,
				Resources:    marshalResources(resp.Resources),
				Nonce:        resp.Nonce,
				ControlPlane: resp.ControlPlane,
			}
			return &marshalledResp
		}
		resp := r.Get().V3
		marshalledResp := marshalledDiscoveryResponse{
			VersionInfo:  resp.VersionInfo,
			Canary:       resp.Canary,
			TypeURL:      resp.TypeUrl,
			Resources:    marshalResources(resp.Resources),
			Nonce:        resp.Nonce,
			ControlPlane: resp.ControlPlane,
		}
		return &marshalledResp
	}
	return nil
}

func marshalResources(Resources []*any.Any) *xDSResources {
	var marshalledResources xDSResources
	for _, resource := range Resources {
		switch resource.TypeUrl {
		case resource2.EndpointType:
			e := &v2.ClusterLoadAssignment{}
			err := ptypes.UnmarshalAny(resource, e)
			if err == nil {
				marshalledResources.Endpoints = append(marshalledResources.Endpoints, e)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource2.ClusterType:
			c := &v2.Cluster{}
			err := ptypes.UnmarshalAny(resource, c)
			if err == nil {
				marshalledResources.Clusters = append(marshalledResources.Clusters, c)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource2.RouteType:
			r := &v2.RouteConfiguration{}
			err := ptypes.UnmarshalAny(resource, r)
			if err == nil {
				marshalledResources.Routes = append(marshalledResources.Routes, r)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource2.ListenerType:
			l := &v2.Listener{}
			err := ptypes.UnmarshalAny(resource, l)
			if err == nil {
				marshalledResources.Listeners = append(marshalledResources.Listeners, l)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource2.SecretType:
			s := &envoy_api_v2_auth.Secret{}
			err := ptypes.UnmarshalAny(resource, s)
			if err == nil {
				marshalledResources.Secrets = append(marshalledResources.Secrets, s)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource2.RuntimeType:
			r := &envoy_service_discovery_v2.Runtime{}
			err := ptypes.UnmarshalAny(resource, r)
			if err == nil {
				marshalledResources.Runtimes = append(marshalledResources.Runtimes, r)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource3.EndpointType:
			e := &endpointv3.ClusterLoadAssignment{}
			err := ptypes.UnmarshalAny(resource, e)
			if err == nil {
				marshalledResources.Endpoints = append(marshalledResources.Endpoints, e)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource3.ClusterType:
			c := &clusterv3.Cluster{}
			err := ptypes.UnmarshalAny(resource, c)
			if err == nil {
				marshalledResources.Clusters = append(marshalledResources.Clusters, c)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource3.RouteType:
			r := &routev3.RouteConfiguration{}
			err := ptypes.UnmarshalAny(resource, r)
			if err == nil {
				marshalledResources.Routes = append(marshalledResources.Routes, r)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource3.ListenerType:
			l := &listenerv3.Listener{}
			err := ptypes.UnmarshalAny(resource, l)
			if err == nil {
				marshalledResources.Listeners = append(marshalledResources.Listeners, l)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource3.SecretType:
			s := &secretv3.Secret{}
			err := ptypes.UnmarshalAny(resource, s)
			if err == nil {
				marshalledResources.Secrets = append(marshalledResources.Secrets, s)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		case resource3.RuntimeType:
			r := &runtimev3.Runtime{}
			err := ptypes.UnmarshalAny(resource, r)
			if err == nil {
				marshalledResources.Runtimes = append(marshalledResources.Runtimes, r)
			} else {
				marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
			}
		default:
			marshalledResources.Unmarshalled = append(marshalledResources.Unmarshalled, resource)
		}
	}
	return &marshalledResources
}
