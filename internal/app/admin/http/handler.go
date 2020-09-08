package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
	"github.com/envoyproxy/xds-relay/internal/pkg/log/zap"

	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	resource2 "github.com/envoyproxy/go-control-plane/pkg/resource/v2"
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

func cacheDumpHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		cacheKey := getParam(req.URL.Path)
		cache := orchestrator.Orchestrator.GetReadOnlyCache(*o)

		// If no key is provided, output the entire cache.
		if cacheKey == "" || cacheKey == "*" {
			keys, err := orchestrator.Orchestrator.GetDownstreamAggregatedKeys(*o)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "error in getting cache keys: %s", err.Error())
				return
			}
			for key := range keys {
				resource, err := cache.FetchReadOnly(key)
				if err == nil {
					resourceString, err := resourceToString(resource)
					if err == nil {
						fmt.Fprintf(w, "%s: %s\n", key, resourceString)
					}
				}
			}
			return
		}

		// If regex key provided, output all cache entries that match given
		if strings.HasSuffix(cacheKey, "*") {
			rootCacheKeyName := strings.TrimSuffix(cacheKey, "*")

			// Retrieve all keys
			allKeys, err := orchestrator.Orchestrator.GetDownstreamAggregatedKeys(*o)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "error in getting cache keys: %s", err.Error())
				return
			}

			// Find keys that match regex
			var matchedRegexKeys[]string
			for key := range allKeys {
				if strings.HasPrefix(key, rootCacheKeyName) {
					matchedRegexKeys = append(matchedRegexKeys, key)
				}
			}
			if len(matchedRegexKeys) == 0 {
				fmt.Fprintf(w, "no resource for key %s found in cache.\n", cacheKey)
				return
			}

			// Output relevant keys
			for _, key := range matchedRegexKeys {
				resource, err := cache.FetchReadOnly(key)
				if err == nil {
					resourceString, err := resourceToString(resource)
					if err == nil {
						fmt.Fprintf(w, "%s: %s\n", key, resourceString)
					}
				}
			}
			return
		}

		// Otherwise return the cache entry corresponding to the given key.
		resource, err := cache.FetchReadOnly(cacheKey)
		if err != nil {
			fmt.Fprintf(w, "no resource for key %s found in cache.\n", cacheKey)
			return
		}
		resourceString, err := resourceToString(resource)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "unable to convert resource to string.\n")
			return
		}
		fmt.Fprint(w, resourceString)
	}
}

type marshallableResource struct {
	Resp           *marshalledDiscoveryResponse
	Requests       []*v2.DiscoveryRequest
	ExpirationTime time.Time
}

// In order to marshal a Resource from the cache to JSON to be printed,
// the map of requests is converted to a slice of just the keys,
// since the bool value is meaningless.
func resourceToString(resource cache.Resource) (string, error) {
	var requests []*v2.DiscoveryRequest
	for request := range resource.Requests {
		requests = append(requests, request.GetRaw().V2)
	}

	resourceString := &marshallableResource{
		Resp:           marshalDiscoveryResponse(resource.Resp),
		Requests:       requests,
		ExpirationTime: resource.ExpirationTime,
	}

	return stringify.InterfaceToString(resourceString)
}

type marshalledDiscoveryResponse struct {
	VersionInfo  string
	Resources    *xDSResources
	Canary       bool
	TypeURL      string
	Nonce        string
	ControlPlane *envoy_api_v2_core.ControlPlane
}

type xDSResources struct {
	Endpoints    []*v2.ClusterLoadAssignment
	Clusters     []*v2.Cluster
	Routes       []*v2.RouteConfiguration
	Listeners    []*v2.Listener
	Secrets      []*envoy_api_v2_auth.Secret
	Runtimes     []*envoy_service_discovery_v2.Runtime
	Unmarshalled []*any.Any
}

func marshalDiscoveryResponse(r transport.Response) *marshalledDiscoveryResponse {
	if r != nil {
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
