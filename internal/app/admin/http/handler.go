package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	secretv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	runtimev3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"github.com/envoyproxy/xds-relay/internal/pkg/log/zap"

	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	resource2 "github.com/envoyproxy/go-control-plane/pkg/resource/v2"
	resource3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/envoyproxy/xds-relay/internal/pkg/util/stringify"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/xds-relay/internal/app/cache"
	"github.com/envoyproxy/xds-relay/internal/app/transport"

	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"

	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
)

type Handler struct {
	pattern     string
	description string
	handler     http.HandlerFunc
}

func getHandlers(bootstrap *bootstrapv1.Bootstrap,
	orchestrator *orchestrator.Orchestrator,
	logger log.Logger) []Handler {
	handlers := []Handler{
		{
			"/",
			"admin home page",
			func(http.ResponseWriter, *http.Request) {},
		},
		{
			"/cache",
			"print cache entry for a given key. Omitting the key outputs all cache entries. usage: `/cache/<key>`",
			cacheDumpHandler(orchestrator),
		},
		{
			"/log_level",
			"update the log level to `debug`, `info`, `warn`, or `error`. " +
				"Omitting the level outputs the current log level. usage: `/log_level/<level>`",
			logLevelHandler(logger),
		},
		{
			"/server_info",
			"print bootstrap configuration",
			configDumpHandler(bootstrap),
		},
	}
	// The default handler is defined later to avoid infinite recursion.
	handlers[0].handler = defaultHandler(handlers)
	return handlers
}

func RegisterHandlers(bootstrapConfig *bootstrapv1.Bootstrap,
	orchestrator *orchestrator.Orchestrator,
	logger log.Logger) {
	for _, handler := range getHandlers(bootstrapConfig, orchestrator, logger) {
		http.Handle(handler.pattern, handler.handler)
		if !strings.HasSuffix(handler.pattern, "/") {
			http.Handle(handler.pattern+"/", handler.handler)
		}
	}
}

func defaultHandler(handlers []Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		fmt.Fprintf(w, "admin commands are:\n")
		for _, handler := range handlers {
			fmt.Fprintf(w, "  %s: %s\n", handler.pattern, handler.description)
		}
	}
}

func configDumpHandler(bootstrapConfig *bootstrapv1.Bootstrap) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		configString, err := stringify.InterfaceToString(bootstrapConfig)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to dump config: %s\n", err.Error())
			return
		}
		fmt.Fprintf(w, "%s\n", configString)
	}
}

func printCacheEntries(keys []string, cache cache.ReadOnlyCache, w http.ResponseWriter) {
	resp := marshallableCache{}
	for _, key := range keys {
		resource, err := cache.FetchReadOnly(key)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "no resource for key %s found in cache.\n", key)
			continue
		}
		resp.Cache = append(resp.Cache, resourceToPayload(key, resource)...)
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

func cacheDumpHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		cacheKey := getParam(req.URL.Path)
		c := orchestrator.Orchestrator.GetReadOnlyCache(*o)
		var keysToPrint []string

		// If wildcard suffix provided, output all cache entries that match given prefix.
		// If no key is provided, output the entire cache.
		containsWildcardSuffix := strings.HasSuffix(cacheKey, "*")
		if cacheKey == "" || containsWildcardSuffix {
			// Retrieve all keys
			allKeys, err := orchestrator.Orchestrator.GetDownstreamAggregatedKeys(*o)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "error in getting cache keys: %s", err.Error())
				return
			}

			// Find keys that match prefix of wildcard
			if containsWildcardSuffix {
				rootCacheKeyName := strings.TrimSuffix(cacheKey, "*")
				for potentialMatchKey := range allKeys {
					if strings.HasPrefix(potentialMatchKey, rootCacheKeyName) {
						keysToPrint = append(keysToPrint, potentialMatchKey)
					}
				}
				if len(keysToPrint) == 0 {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, "no resource for key %s found in cache.\n", cacheKey)
					return
				}
			} else {
				// Add matched keys to print slice
				for key := range allKeys {
					keysToPrint = append(keysToPrint, key)
				}
			}
		} else {
			// Otherwise return the cache entry corresponding to the given key.
			keysToPrint = []string{cacheKey}
		}
		printCacheEntries(keysToPrint, c, w)
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

// In order to marshal a Resource from the cache to JSON to be printed,
// the map of requests is converted to a slice of just the keys,
// since the bool value is meaningless.
func resourceToPayload(key string, resource cache.Resource) []marshallableResource {
	var marshallableResources []marshallableResource
	var requests []types.Resource
	for request := range resource.Requests {
		if request.GetRaw().V2 != nil {
			requests = append(requests, request.GetRaw().V2)
		} else {
			requests = append(requests, request.GetRaw().V3)
		}
	}

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

func getParam(path string) string {
	// Assumes that the URL is of the format `address/endpoint/parameter` and returns `parameter`.
	splitPath := strings.SplitN(path, "/", 3)
	if len(splitPath) == 3 {
		return splitPath[2]
	}
	return ""
}

func logLevelHandler(l log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "POST" {
			logLevel := getParam(req.URL.Path)

			// If no key is provided, output the current log level.
			if logLevel == "" {
				fmt.Fprintf(w, "Current log level: %s\n", l.GetLevel())
				return
			}

			// Otherwise update the logging level.
			_, parseLogLevelErr := zap.ParseLogLevel(logLevel)
			if parseLogLevelErr != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Invalid log level: %s\n", logLevel)
				return
			}
			l.UpdateLogLevel(logLevel)
			fmt.Fprintf(w, "Current log level: %s\n", l.GetLevel())
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Only POST is supported\n")
		}
	}
}
