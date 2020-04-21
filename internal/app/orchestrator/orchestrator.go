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

	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"

	"github.com/envoyproxy/xds-relay/internal/app/cache"
	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"

	discovery "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

const (
	component = "orchestrator"

	// unaggregatedPrefix is the prefix used to label discovery requests that
	// could not be successfully mapped to an aggregation rule.
	unaggregatedPrefix = "unaggregated_"
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
//
// Orchestrator will be using go-control-plane's gRPC server implementation to
// maintain the fanout to downstream clients. go-control-plane keeps an open
// connection with each downstream xDS client. When orchestrator receives an
// upstream response from the forwarded sample request (via a long lived
// channel), Orchestrator will cache the response, and fanout to the
// downstreams by supplying responses to the individual channels corresponding
// to each downstream connection (watcher). See the CreateWatch function for
// more details.
type Orchestrator interface {
	gcp.Cache

	// shutdown takes an aggregated key and shuts down the go routine watching
	// for upstream responses.
	//
	// This is currently used by tests to clean up channels, but can also be
	// used by the main shutdown handler.
	shutdown(string)
}

type orchestrator struct {
	mapper         mapper.Mapper
	cache          cache.Cache
	upstreamClient upstream.Client

	logger log.Logger

	downstreamResponseMap downstreamResponseMap
	upstreamResponseMap   upstreamResponseMap
}

// New instantiates the mapper, cache, upstream client components necessary for
// the orchestrator to operate and returns an instance of the instantiated
// orchestrator.
func New(ctx context.Context,
	l log.Logger,
	mapper mapper.Mapper,
	upstreamClient upstream.Client,
	cacheConfig *bootstrapv1.Cache) Orchestrator {
	orchestrator := &orchestrator{
		logger:         l.Named(component),
		mapper:         mapper,
		upstreamClient: upstreamClient,
		downstreamResponseMap: downstreamResponseMap{
			responseChannel: make(map[*gcp.Request]chan gcp.Response),
		},
		upstreamResponseMap: upstreamResponseMap{
			responseChannel: make(map[string]upstreamResponseChannel),
		},
	}

	// Initialize cache.
	cache, err := cache.NewCache(int(cacheConfig.MaxEntries),
		orchestrator.onCacheEvicted,
		time.Duration(cacheConfig.Ttl.Nanos)*time.Nanosecond)
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
func (o *orchestrator) CreateWatch(req gcp.Request) (chan gcp.Response, func()) {
	ctx := context.Background()

	// If this is the first time we're seeing the request from the
	// downstream client, initialize a channel to feed future responses.
	responseChannel := o.downstreamResponseMap.createChannel(&req)

	aggregatedKey, err := o.mapper.GetKey(req)
	if err != nil {
		// Can't map the request to an aggregated key. Log and continue to
		// propagate the response upstream without aggregation.
		o.logger.With("err", err).With("req node", req.GetNode()).Warn(ctx, "failed to map to aggregated key")
		// Mimic the aggregated key.
		// TODO (https://github.com/envoyproxy/xds-relay/issues/56). This key
		// needs to be made more granular to uniquely identify a request.
		aggregatedKey = fmt.Sprintf("%s%s_%s", unaggregatedPrefix, req.GetNode().GetId(), req.GetTypeUrl())
	}

	// Register the watch for future responses.
	err = o.cache.AddRequest(aggregatedKey, &req)
	if err != nil {
		// If we fail to register the watch, we need to kill this stream by
		// closing the response channel.
		o.logger.With("err", err).With("key", aggregatedKey).With(
			"req node", req.GetNode()).Error(ctx, "failed to add watch")
		closedChannel := o.downstreamResponseMap.delete(&req)
		return closedChannel, nil
	}

	// Check if we have a cached response first.
	cached, err := o.cache.Fetch(aggregatedKey)
	if err != nil {
		// Log, and continue to propagate the response upstream.
		o.logger.With("err", err).With("key", aggregatedKey).Warn(ctx, "failed to fetch aggregated key")
	}

	if cached != nil && cached.Resp != nil && cached.Resp.Raw.GetVersionInfo() != req.GetVersionInfo() {
		// If we have a cached response and the version is different,
		// immediately push the result to the response channel.
		go func() { responseChannel <- convertToGcpResponse(cached.Resp, req) }()
	}

	// Check if we have a upstream stream open for this aggregated key. If not,
	// open a stream with the representative request.
	//
	// Locking is necessary here so that a simultaneous downstream request
	// that maps to the same aggregated key doesn't result in two upstream
	// streams.
	o.upstreamResponseMap.mu.Lock()
	if _, ok := o.upstreamResponseMap.responseChannel[aggregatedKey]; !ok {
		upstreamResponseChan, _, _ := o.upstreamClient.OpenStream(req)
		respChannel := o.upstreamResponseMap.add(aggregatedKey, upstreamResponseChan)
		// Spin up a go routine to watch for upstream responses.
		// One routine is opened per aggregate key.
		go o.watchUpstream(ctx, aggregatedKey, respChannel.response, respChannel.done)

	}
	o.upstreamResponseMap.mu.Unlock()

	return responseChannel, o.onCancelWatch(&req)
}

// Fetch implements the polling method of the config cache using a non-empty request.
func (o *orchestrator) Fetch(context.Context, discovery.DiscoveryRequest) (*gcp.Response, error) {
	return nil, fmt.Errorf("Not implemented")
}

// watchResponse is intended to be called in a goroutine, to receive incoming
// responses and fan out to downstream clients.
func (o *orchestrator) watchUpstream(
	ctx context.Context,
	aggregatedKey string,
	responseChannel <-chan *discovery.DiscoveryResponse,
	done <-chan bool,
) {
	for {
		select {
		case x, more := <-responseChannel:
			if !more {
				// A problem occurred fetching the response upstream, log and
				// return the most recent cached response, so that the
				// downstream will reissue the discovery request.
				o.logger.With("key", aggregatedKey).Error(ctx, "upstream error")
			} else {
				// Cache the response.
				_, err := o.cache.SetResponse(aggregatedKey, *x)
				if err != nil {
					// If we fail to cache the new response, log and return the old one.
					o.logger.With("err", err).With("key", aggregatedKey).
						Error(ctx, "Failed to cache the response")
				}
			}

			// Get downstream watches and fan out.
			// We retrieve from cache rather than directly fanning out the
			// newly received response because the cache does additional
			// resource serialization.
			cached, err := o.cache.Fetch(aggregatedKey)
			if err != nil {
				o.logger.With("err", err).With("key", aggregatedKey).Error(ctx, "cache fetch failed")
				// Can't do anything because we don't know who the watches
				// are. Drop the response.
			} else {
				if cached == nil || cached.Resp == nil {
					// If cache is empty, there is nothing to fan out.
					if !more {
						// Warn. Benefit of the doubt that this is the first request.
						o.logger.With("key", aggregatedKey).
							Warn(ctx, "attempted to fan out with no cached response")
					} else {
						// Error. Sanity check. Shouldn't ever reach this.
						o.logger.With("key", aggregatedKey).
							Error(ctx, "attempted to fan out with no cached response")
					}
				} else {
					// Goldenpath.
					o.fanout(cached.Resp, cached.Requests)
				}
			}
		case <-done:
			// Exit when signaled that the stream has closed.
			return
		}
	}
}

// fanout pushes the response to the response channels of all open downstream
// watches in parallel.
func (o *orchestrator) fanout(resp *cache.Response, watchers []*gcp.Request) {
	for _, watch := range watchers {
		go func(watch *gcp.Request) {
			if channel, ok := o.downstreamResponseMap.get(watch); ok {
				channel <- convertToGcpResponse(resp, *watch)
			}
		}(watch)
	}
}

// onCacheEvicted is called when the cache evicts a response due to TTL or
// other reasons. When this happens, we need to clean up open streams.
// We shut down both the downstream watches and the upstream stream.
func (o *orchestrator) onCacheEvicted(key string, resource cache.Resource) {
	o.downstreamResponseMap.deleteAll(resource.Requests)
	o.upstreamResponseMap.delete(key)
}

// onCancelWatch cleans up the cached watch when called.
func (o *orchestrator) onCancelWatch(req *gcp.Request) func() {
	return func() {
		o.downstreamResponseMap.delete(req)
		// TODO (https://github.com/envoyproxy/xds-relay/issues/57). Clean up
		// watch from cache. Cache needs to expose a function to do so.
	}
}

// shutdown closes the upstream connection for the specified aggregated key.
func (o *orchestrator) shutdown(aggregatedKey string) {
	o.upstreamResponseMap.delete(aggregatedKey)
}

// convertToGcpResponse constructs the go-control-plane response from the
// cached response.
func convertToGcpResponse(resp *cache.Response, req gcp.Request) gcp.Response {
	return gcp.Response{
		Request:            req,
		Version:            resp.Raw.GetVersionInfo(),
		ResourceMarshaled:  true,
		MarshaledResources: resp.MarshaledResources,
	}
}
