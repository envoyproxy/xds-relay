package handler

import (
	"fmt"
	"net/http"
	"strings"

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
			defaultHandler(nil),
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

// TODO(lisalu): Make config output more readable.
func configDumpHandler(bootstrapConfig *bootstrapv1.Bootstrap) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, bootstrapConfig.String())
	}
}

// TODO(lisalu): Support dump of entire cache when no key is provided.
// TODO(lisalu): Support dump of matching resources when cache key regex is provided.
func cacheDumpHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		cacheKey := getCacheKeyParam(req.URL.Path)
		cache := orchestrator.Orchestrator.GetReadOnlyCache(*o)
		resource, err := cache.Fetch(cacheKey)
		if err != nil {
			fmt.Fprintf(w, "no resource for key found in cache")
			return
		}
		fmt.Print(w, resource.Resp.Raw.String())
	}
}

func getCacheKeyParam(path string) string {
	// Assumes that the URL is of the format `address/cache/parameter` and returns `parameter`.
	splitPath := strings.SplitN(path, "/", 3)
	return splitPath[2]
}
