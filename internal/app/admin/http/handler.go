package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/envoyproxy/xds-relay/internal/pkg/util/stringify"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/xds-relay/internal/app/cache"

	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"

	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
)

type Handler struct {
	prefix      string
	description string
	handler     http.HandlerFunc
}

func getHandlers(bootstrap *bootstrapv1.Bootstrap, orchestrator *orchestrator.Orchestrator) []Handler {
	handlers := []Handler{
		{
			"/",
			"admin home page",
			func(http.ResponseWriter, *http.Request) {},
		},
		{
			"/cache/",
			"print cache entry for a given key. usage: `/cache/<key>`",
			cacheDumpHandler(orchestrator),
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

func RegisterHandlers(bootstrapConfig *bootstrapv1.Bootstrap, orchestrator *orchestrator.Orchestrator) {
	for _, handler := range getHandlers(bootstrapConfig, orchestrator) {
		http.Handle(handler.prefix, handler.handler)
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
			fmt.Fprintf(w, "  %s: %s\n", handler.prefix, handler.description)
		}
	}
}

func configDumpHandler(bootstrapConfig *bootstrapv1.Bootstrap) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		configString, err := stringify.InterfaceToString(bootstrapConfig)
		if err != nil {
			fmt.Fprintf(w, "Failed to dump config: %s\n", err.Error())
		}
		fmt.Fprintf(w, "%s\n", configString)
	}
}

// TODO(lisalu): Support dump of entire cache when no key is provided.
// TODO(lisalu): Support dump of matching resources when cache key regex is provided.
func cacheDumpHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		cacheKey, err := getCacheKeyParam(req.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "unable to parse cache key from path: %s", err.Error())
		}
		cache := orchestrator.Orchestrator.GetReadOnlyCache(*o)
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
	Resp           *v2.DiscoveryResponse
	Requests       []*v2.DiscoveryRequest
	ExpirationTime time.Time
}

// In order to marshal a Resource from the cache to JSON to be printed,
// the map of requests is converted to a slice of just the keys,
// since the bool value is meaningless.
// TODO(lisalu): More intelligent unmarshalling of DiscoveryResponse.
func resourceToString(resource cache.Resource) (string, error) {
	var requests []*v2.DiscoveryRequest
	for request := range resource.Requests {
		requests = append(requests, request)
	}

	resourceString := &marshallableResource{
		Resp:           resource.Resp,
		Requests:       requests,
		ExpirationTime: resource.ExpirationTime,
	}

	return stringify.InterfaceToString(resourceString)
}

func getCacheKeyParam(path string) (string, error) {
	// Assumes that the URL is of the format `address/cache/parameter` and returns `parameter`.
	splitPath := strings.SplitN(path, "/", 3)
	if len(splitPath) == 3 {
		return splitPath[2], nil
	}
	return "", fmt.Errorf("unable to parse cache key from path: %s", path)
}
