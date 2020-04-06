// Package orchestrator is responsible for instrumenting inbound xDS client
// requests to the correct aggregated key, forwarding a representative request
// to the upstream origin server, and managing the lifecycle of downstream and
// upstream connections and associates streams. It implements
// go-control-plane's Cache interface in order to receive xDS-based requests,
// send responses, and handle gRPC streams.
package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/envoyproxy/xds-relay/internal/app/cache"
	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"

	discovery "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

const (
	component = "orchestrator"

	// TODO (https://github.com/envoyproxy/xds-relay/issues/41) load from configured defaults.
	cacheMaxEntries = 1000
	cacheTTL        = 60 * time.Minute
)

// Orchestrator has the following responsibilities:
//
// 1. Aggregates similar requests abiding by the aggregated keyer
//    configurations.
// 2. Maintains long lived streams with the upstream origin server for each
//    representative discovery request.
// 3. Maintains long lived streams with each downstream xDS client using
//    go-control-plane's CreateWatch function.
// 4. When new responses are available upstream, orchestrator relays and fans
//    out the response back on the streams associated with the xDS clients.
// 5. Updates the xds-relay cache with the latest state of the world.
type Orchestrator interface {
	gcp.Cache
}

type orchestrator struct {
	mapper         mapper.Mapper
	cache          cache.Cache
	upstreamClient upstream.Client

	logger log.Logger
}

// New instantiates the mapper, cache, upstream client components necessary for
// the orchestrator to operate and returns an instance of the instantiated
// orchestrator.
func New(ctx context.Context, l log.Logger, mapper mapper.Mapper, upstreamClient upstream.Client) Orchestrator {
	orchestrator := &orchestrator{
		logger:         l.Named(component),
		mapper:         mapper,
		upstreamClient: upstreamClient,
	}

	cache, err := cache.NewCache(cacheMaxEntries, orchestrator.onCacheEvicted, cacheTTL)
	if err != nil {
		orchestrator.logger.With("error", err).Panic(ctx, "failed to initialize cache")
	}
	orchestrator.cache = cache

	return orchestrator
}

// CreateWatch is managed by the underlying go-control-plane gRPC server.
//
// Orchestrator will populate the response channel with the corresponding
// responses once new resources are available upstream.
//
// If the channel is closed prior to cancellation of the watch, an
// unrecoverable error has occurred in the producer, and the consumer should
// close the corresponding stream.
//
// Cancel is an optional function to release resources in the producer. If
// provided, the consumer may call this function multiple times.
func (c *orchestrator) CreateWatch(req gcp.Request) (chan gcp.Response, func()) {
	// TODO implement.
	return nil, nil
}

// Fetch implements the polling method of the config cache using a non-empty request.
func (c *orchestrator) Fetch(context.Context, discovery.DiscoveryRequest) (*gcp.Response, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (c *orchestrator) onCacheEvicted(key string, resource cache.Resource) {
	// TODO implement.
}
