package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/envoyproxy/xds-relay/internal/app/cache"

	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
)

func DefaultHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		// TODO(lisalu): Add more helpful response message, e.g. listing the different endpoints available.
		fmt.Fprintf(w, "xds-relay admin API")
	}
}

func ConfigDumpHandler(bootstrapConfig *bootstrapv1.Bootstrap) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, bootstrapConfig.String())
	}
}

// TODO(lisalu): Support dump of entire cache when no key is provided.
// TODO(lisalu): Support dump of matching resources when cache key regex is provided.
func CacheDumpHandler(c *cache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		cacheKey := getCacheKeyParam(req.URL.Path)
		resource, err := cache.Cache.Fetch(*c, cacheKey)
		if err != nil {
			fmt.Fprintf(w, "no resource for key found in cache")
			return
		}
		fmt.Print(w, resource.Resp.Raw.String())
	}
}

func getCacheKeyParam(path string) string {
	splitPath := strings.SplitN(path, "/", 3)
	return splitPath[2]
}
