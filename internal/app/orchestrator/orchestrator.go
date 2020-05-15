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
	"sync"
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

	// This is called by the main shutdown handler and tests to clean up
	// open channels.
	shutdown(ctx context.Context)

	GetReadOnlyCache() cache.ReadOnlyCache
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
func New(
	ctx context.Context,
	l log.Logger,
	mapper mapper.Mapper,
	upstreamClient upstream.Client,
	cacheConfig *bootstrapv1.Cache,
) Orchestrator {
	orchestrator := &orchestrator{
		logger:                l.Named(component),
		mapper:                mapper,
		upstreamClient:        upstreamClient,
		downstreamResponseMap: newDownstreamResponseMap(),
		upstreamResponseMap:   newUpstreamResponseMap(),
	}

	// Initialize cache.
	cache, err := cache.NewCache(
		int(cacheConfig.MaxEntries),
		orchestrator.onCacheEvicted,
		time.Duration(cacheConfig.Ttl.Nanos)*time.Nanosecond,
	)
	if err != nil {
		orchestrator.logger.With("error", err).Panic(ctx, "failed to initialize cache")
	}
	orchestrator.cache = cache

	go orchestrator.shutdown(ctx)

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
	o.logger.With("node ID", req.GetNode().GetId()).With("type", req.GetTypeUrl()).Debug(ctx, "creating watch")

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
	if !o.upstreamResponseMap.exists(aggregatedKey) {
		upstreamResponseChan, shutdown, err := o.upstreamClient.OpenStream(req)
		if err != nil {
			// TODO implement retry/back-off logic on error scenario.
			// https://github.com/envoyproxy/xds-relay/issues/68
			o.logger.With("err", err).With("key", aggregatedKey).Error(ctx, "Failed to open stream to origin server")
		} else {
			respChannel, upstreamOpenedPreviously := o.upstreamResponseMap.add(aggregatedKey, upstreamResponseChan)
			if upstreamOpenedPreviously {
				// A stream was opened previously due to a race between
				// concurrent downstreams for the same aggregated key, between
				// exists and add operations. In this event, simply close the
				// slower stream and return the existing one.
				shutdown()
			} else {
				// Spin up a go routine to watch for upstream responses.
				// One routine is opened per aggregate key.
				go o.watchUpstream(ctx, aggregatedKey, respChannel.response, respChannel.done, shutdown)
			}
		}
	}

	return responseChannel, o.onCancelWatch(aggregatedKey, &req)
}

// Fetch implements the polling method of the config cache using a non-empty request.
func (o *orchestrator) Fetch(context.Context, discovery.DiscoveryRequest) (*gcp.Response, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (o *orchestrator) GetReadOnlyCache() cache.ReadOnlyCache {
	return o.cache.GetReadOnlyCache()
}

// watchUpstream is intended to be called in a go routine, to receive incoming
// responses, cache the response, and fan out to downstream clients or
// "watchers". There is a corresponding go routine for each aggregated key.
//
// This goroutine continually listens for upstream responses from the passed
// `responseChannel`. For each response, we will:
// - cache this latest response, replacing the previous stale response.
// - retrieve the downstream watchers from the cache for this `aggregated key`.
// - trigger the fanout process to downstream watchers by pushing to the
//   individual downstream response channels in separate go routines.
//
// Additionally this function tracks a `done` channel and a `shutdownUpstream`
// function. `done` is a channel that gets closed in two places:
// 1. when server shutdown is triggered. See the `shutdown` function in this
//    file for more information.
// 2. when cache TTL expires for this aggregated key. See the `onCacheEvicted`
//    function in this file for more information.
// When the `done` channel is closed, we call the `shutdownUpstream` callback
// function. This will signify to the upstream client that we no longer require
// responses from this stream because the downstream connections have been
// terminated. The upstream client will clean up the stream accordingly.
func (o *orchestrator) watchUpstream(
	ctx context.Context,
	aggregatedKey string,
	responseChannel <-chan *discovery.DiscoveryResponse,
	done <-chan bool,
	shutdownUpstream func(),
) {
	for {
		select {
		case x, more := <-responseChannel:
			if !more {
				// A problem occurred fetching the response upstream, log retry.
				// TODO implement retry/back-off logic on error scenario.
				// https://github.com/envoyproxy/xds-relay/issues/68
				o.logger.With("key", aggregatedKey).Error(ctx, "upstream error")
				return
			}
			// Cache the response.
			_, err := o.cache.SetResponse(aggregatedKey, *x)
			if err != nil {
				// TODO if set fails, we may need to retry upstream as well.
				// Currently the fallback is to rely on a future response, but
				// that probably isn't ideal.
				// https://github.com/envoyproxy/xds-relay/issues/70
				//
				// If we fail to cache the new response, log and return the old one.
				o.logger.With("err", err).With("key", aggregatedKey).
					Error(ctx, "Failed to cache the response")
			}

			// Get downstream watchers and fan out.
			// We retrieve from cache rather than directly fanning out the
			// newly received response because the cache does additional
			// resource serialization.
			cached, err := o.cache.Fetch(aggregatedKey)
			if err != nil {
				o.logger.With("err", err).With("key", aggregatedKey).Error(ctx, "cache fetch failed")
				// Can't do anything because we don't know who the watchers
				// are. Drop the response.
			} else {
				if cached == nil || cached.Resp == nil {
					// If cache is empty, there is nothing to fan out.
					// Error. Sanity check. Shouldn't ever reach this since we
					// just set the response, but it's a rare scenario that can
					// happen if the cache TTL is set very short.
					o.logger.With("key", aggregatedKey).Error(ctx, "attempted to fan out with no cached response")
				} else {
					// Goldenpath.
					o.logger.With("key", aggregatedKey).With("response", cached.Resp).Debug(ctx, "response fanout initiated")
					o.fanout(cached.Resp, cached.Requests, aggregatedKey)
				}
			}
		case <-done:
			// Exit when signaled that the stream has closed.
			shutdownUpstream()
			return
		}
	}
}

// fanout pushes the response to the response channels of all open downstream
// watchers in parallel.
func (o *orchestrator) fanout(resp *cache.Response, watchers map[*gcp.Request]bool, aggregatedKey string) {
	var wg sync.WaitGroup
	for watch := range watchers {
		wg.Add(1)
		go func(watch *gcp.Request) {
			defer wg.Done()
			if channel, ok := o.downstreamResponseMap.get(watch); ok {
				select {
				case channel <- convertToGcpResponse(resp, *watch):
					o.logger.With("key", aggregatedKey).With("node ID", watch.GetNode().GetId()).
						Debug(context.Background(), "response sent")
				default:
					// If the channel is blocked, we simply drop subsequent requests and error.
					// Alternative possibilities are discussed here:
					// https://github.com/envoyproxy/xds-relay/pull/53#discussion_r420325553
					o.logger.With("key", aggregatedKey).With("node ID", watch.GetNode().GetId()).
						Error(context.Background(), "channel blocked during fanout")
				}
			}
		}(watch)
	}
	// Wait for all fanouts to complete.
	wg.Wait()
}

// onCacheEvicted is called when the cache evicts a response due to TTL or
// other reasons. When this happens, we need to clean up open streams.
// We shut down both the downstream watchers and the upstream stream.
func (o *orchestrator) onCacheEvicted(key string, resource cache.Resource) {
	// TODO Potential for improvements here to handle the thundering herd
	// problem: https://github.com/envoyproxy/xds-relay/issues/71
	o.downstreamResponseMap.deleteAll(resource.Requests)
	o.upstreamResponseMap.delete(key)
}

// onCancelWatch cleans up the cached watch when called.
func (o *orchestrator) onCancelWatch(aggregatedKey string, req *gcp.Request) func() {
	return func() {
		o.downstreamResponseMap.delete(req)
		if err := o.cache.DeleteRequest(aggregatedKey, req); err != nil {
			o.logger.With("key", aggregatedKey).With("err", err).Warn(context.Background(), "Failed to delete from cache")
		}
	}
}

// shutdown closes all upstream connections when ctx.Done is called.
func (o *orchestrator) shutdown(ctx context.Context) {
	<-ctx.Done()
	o.upstreamResponseMap.deleteAll()
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
