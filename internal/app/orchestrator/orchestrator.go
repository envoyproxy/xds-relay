// Package orchestrator is responsible for instrumenting inbound xDS client
// requests to the correct aggregated key, forwarding a representative request
// to the upstream origin server, and managing the lifecycle of downstream and
// upstream connections and associates streams. It implements
// go-control-plane's Cache interface in order to receive xDS-based requests,
// send responses, and handle gRPC streams.
package orchestrator

import (
	"context"
	"strings"
	"sync"
	"time"

	bootstrapv1 "github.com/envoyproxy/xds-relay/pkg/api/bootstrap/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/uber-go/tally"

	"github.com/envoyproxy/xds-relay/internal/app/cache"
	"github.com/envoyproxy/xds-relay/internal/app/mapper"
	"github.com/envoyproxy/xds-relay/internal/app/metrics"
	"github.com/envoyproxy/xds-relay/internal/app/transport"
	"github.com/envoyproxy/xds-relay/internal/app/upstream"
	"github.com/envoyproxy/xds-relay/internal/pkg/log"
)

const (
	component = "orchestrator"
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
	// This is called by the main shutdown handler and tests to clean up
	// open channels.
	shutdown(ctx context.Context)

	GetReadOnlyCache() cache.ReadOnlyCache

	GetDownstreamAggregatedKeys() []string

	CreateWatch(transport.Request, transport.Watch) func()
}

type orchestrator struct {
	mapper         mapper.Mapper
	cache          cache.Cache
	upstreamClient upstream.Client

	logger log.Logger
	scope  tally.Scope

	downstreamResponseMap downstreamResponseMap
	upstreamResponseMap   upstreamResponseMap
}

// New instantiates the mapper, cache, upstream client components necessary for
// the orchestrator to operate and returns an instance of the instantiated
// orchestrator.
func New(
	ctx context.Context,
	logger log.Logger,
	scope tally.Scope,
	mapper mapper.Mapper,
	upstreamClient upstream.Client,
	cacheConfig *bootstrapv1.Cache,
) Orchestrator {
	orchestrator := &orchestrator{
		logger:                logger.Named(component),
		scope:                 scope.SubScope(metrics.ScopeOrchestrator),
		mapper:                mapper,
		upstreamClient:        upstreamClient,
		downstreamResponseMap: newDownstreamResponseMap(),
		upstreamResponseMap:   newUpstreamResponseMap(),
	}

	// Initialize cache.
	ttl, err := ptypes.Duration(cacheConfig.Ttl)
	if err != nil {
		orchestrator.logger.With("error", err).Panic(ctx, "failed to convert ttl from durationpb.Duration to time.Duration")
	}
	cache, err := cache.NewCache(
		int(cacheConfig.MaxEntries),
		orchestrator.onCacheEvicted,
		ttl,
		logger,
		scope,
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
func (o *orchestrator) CreateWatch(req transport.Request, watch transport.Watch) func() {
	ctx := context.Background()

	aggregatedKey, err := o.mapper.GetKey(req)
	if err != nil {
		// Can't map the request to an aggregated key. Log error and return.
		o.logger.With(
			"error", err,
			"request_type", req.GetTypeURL(),
			"node_id", req.GetNodeID(),
		).Error(ctx, "failed to map to aggregated key")

		// TODO (https://github.com/envoyproxy/xds-relay/issues/56)
		// Support unnaggregated keys.
		// nil is a kill pill which will cause go-control-plane to error out the stream
		_ = o.send(nil, watch, aggregatedKey)
		return nil
	}

	o.logger.With(
		"node_id", req.GetNodeID(),
		"request_type", req.GetTypeURL(),
		"request_version", req.GetVersionInfo(),
		"nonce", req.GetResponseNonce(),
		"error", req.GetError(),
		"aggregated_key", aggregatedKey,
	).Debug(ctx, "creating watch")

	metrics.OrchestratorWatchSubscope(o.scope, aggregatedKey).Counter(metrics.OrchestratorWatchCreated).Inc(1)

	// Log + stat to investigate NACK behavior
	if isNackRequest(req) {
		resourceString := ""
		if req.GetResourceNames() != nil {
			resourceString = strings.Join(req.GetResourceNames()[:], ",")
		}
		o.logger.With(
			"request_version", req.GetVersionInfo(),
			"resource_names", resourceString,
			"nonce", req.GetResponseNonce(),
			"request_type", req.GetTypeURL(),
			"error", req.GetError(),
			"aggregated_key", aggregatedKey,
		).Debug(ctx, "NACK request")
		metrics.OrchestratorWatchSubscope(o.scope, aggregatedKey).Counter(metrics.OrchestratorNackWatchCreated).Inc(1)
	}

	// Check if we have a cached response first.
	cached, err := o.cache.Fetch(aggregatedKey)
	if err != nil {
		// Log, and continue to propagate the response upstream.
		o.logger.With("error", err).With("aggregated_key", aggregatedKey).Warn(ctx, "failed to fetch aggregated key")
	}

	cachedResponse := false

	if cached != nil && cached.Resp != nil &&
		cached.Resp.GetPayloadVersion() != req.GetVersionInfo() &&
		req.GetError() == nil {
		cachedResponse = true
		// If we have a cached response and the version is different,
		// immediately push the result to the response channel.
		err := o.send(cached.Resp, watch, aggregatedKey)
		if err == nil {
			metrics.OrchestratorWatchSubscope(o.scope, aggregatedKey).Counter(metrics.OrchestratorWatchFanouts).Inc(1)
		}
	}

	if !cachedResponse {
		// If this is the first time we're seeing the request from the
		// downstream client, initialize a channel to feed future responses.
		o.downstreamResponseMap.addWatch(aggregatedKey, watch)
	}

	// Check if we have a upstream stream open for this aggregated key. If not,
	// open a stream with the representative request.
	if !o.upstreamResponseMap.exists(aggregatedKey) {
		upstreamResponseChan, shutdown := o.upstreamClient.OpenStream(req, aggregatedKey)
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
			o.logger.With("aggregated_key", aggregatedKey).Debug(ctx, "watching upstream")
			go o.watchUpstream(ctx, aggregatedKey, respChannel.response, respChannel.done, shutdown)
		}
	}

	if !cachedResponse {
		return o.onCancelWatch(aggregatedKey, watch)
	}
	return nil
}

// GetReadOnlyCache returns the request/response cache with only read-only methods exposed.
func (o *orchestrator) GetReadOnlyCache() cache.ReadOnlyCache {
	return o.cache.GetReadOnlyCache()
}

// GetDownstreamAggregatedKeys returns the aggregated keys for all requests stored in the downstream response map.
func (o *orchestrator) GetDownstreamAggregatedKeys() []string {
	return o.downstreamResponseMap.getAggregatedKeys()
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
	responseChannel <-chan transport.Response,
	done <-chan bool,
	shutdownUpstream func(),
) {
	for {
		select {
		case x, more := <-responseChannel:
			if !more {
				// A problem occurred fetching the response upstream, log retry.
				o.logger.With("aggregated_key", aggregatedKey).Error(ctx, "upstream error")
				metrics.OrchestratorWatchErrorsSubscope(o.scope, aggregatedKey).Counter(metrics.ErrorUpstreamFailure).Inc(1)

				// TODO implement retry/back-off logic on error scenario.
				// https://github.com/envoyproxy/xds-relay/issues/68
				o.onCacheEvicted(aggregatedKey, cache.Resource{})

				return
			}
			// Cache the response.
			err := o.cache.SetResponse(aggregatedKey, x)
			if err != nil {
				// TODO if set fails, we may need to retry upstream as well.
				// Currently the fallback is to rely on a future response, but
				// that probably isn't ideal.
				// https://github.com/envoyproxy/xds-relay/issues/70s
				//
				// If we fail to cache the new response, log and return the old one.
				o.logger.With("error", err).With("aggregated_key", aggregatedKey).
					Error(ctx, "Failed to cache the response")
			}

			// Get downstream watchers and fan out.
			// We retrieve from cache rather than directly fanning out the
			// newly received response because the cache does additional
			// resource serialization.
			cached, err := o.cache.Fetch(aggregatedKey)
			if err != nil {
				o.logger.With("error", err).With("aggregated_key", aggregatedKey).Error(ctx, "cache fetch failed")
				// Can't do anything because we don't know who the watchers
				// are. Drop the response.
			} else {
				if cached == nil || cached.Resp == nil {
					// If cache is empty, there is nothing to fan out.
					// Error. Sanity check. Shouldn't ever reach this since we
					// just set the response, but it's a rare scenario that can
					// happen if the cache TTL is set very short.
					o.logger.With("aggregated_key", aggregatedKey).Error(ctx, "attempted to fan out with no cached response")
					metrics.OrchestratorWatchErrorsSubscope(o.scope, aggregatedKey).Counter(metrics.ErrorCacheMiss).Inc(1)
				} else {
					// Goldenpath.
					o.logger.With(
						"aggregated_key", aggregatedKey,
						"response_type", cached.Resp.GetTypeURL(),
						"response_version", cached.Resp.GetPayloadVersion(),
					).Debug(ctx, "response fanout initiated")
					o.fanout(cached.Resp, aggregatedKey)
				}
			}
		case <-done:
			// Exit when signaled that the stream has closed.
			o.logger.With("aggregated_key", aggregatedKey).Info(ctx, "shutting down upstream watch")
			shutdownUpstream()
			return
		}
	}
}

// fanout pushes the response to the response channels of all open downstream
// watchers in parallel.
func (o *orchestrator) fanout(resp transport.Response, aggregatedKey string) {
	start := time.Now()
	var wg sync.WaitGroup
	watchSnapshot, ok := o.downstreamResponseMap.getSnapshot(aggregatedKey)
	if !ok {
		return
	}
	for _, watch := range watchSnapshot {
		wg.Add(1)
		go func(w transport.Watch) {
			defer wg.Done()
			// TODO https://github.com/envoyproxy/xds-relay/issues/119
			_ = o.send(resp, w, aggregatedKey)
			o.logger.With(
				"aggregated_key", aggregatedKey,
				"response_version", resp.GetPayloadVersion(),
				"response_type", resp.GetTypeURL(),
			).Debug(context.Background(), "response sent")
			metrics.OrchestratorWatchSubscope(o.scope, aggregatedKey).Counter(metrics.OrchestratorWatchFanouts).Inc(1)
		}(watch)
	}
	o.scope.Timer(metrics.TimerFanoutTime).Record(time.Since(start))
	// Wait for all fanouts to complete.
	wg.Wait()
	o.scope.Timer(metrics.TimerSendTime).Record(time.Since(start))
}

// onCacheEvicted is called when the cache evicts a response due to TTL or
// other reasons. When this happens, we need to clean up open streams.
// We shut down both the downstream watchers and the upstream stream.
func (o *orchestrator) onCacheEvicted(key string, resource cache.Resource) {
	// TODO Potential for improvements here to handle the thundering herd
	// problem: https://github.com/envoyproxy/xds-relay/issues/71
	metrics.OrchestratorCacheEvictSubscope(o.scope, key).Counter(metrics.OrcheestratorCacheEvictCount).Inc(1)
	metrics.OrchestratorCacheEvictSubscope(o.scope, key).Counter(
		metrics.OrchestratorOnCacheEvictedRequestCount).Inc(int64(len(o.downstreamResponseMap.get(key))))
	o.logger.With("aggregated_key", key).Debug(context.Background(), "cache eviction called")
	if s, ok := o.downstreamResponseMap.getSnapshot(key); ok {
		for _, d := range s {
			_ = o.send(nil, d, key)
		}
	}
	o.upstreamResponseMap.delete(key)
}

// onCancelWatch cleans up the cached watch when called.
func (o *orchestrator) onCancelWatch(aggregatedKey string, w transport.Watch) func() {
	return func() {
		o.downstreamResponseMap.delete(w)
		metrics.OrchestratorWatchSubscope(o.scope, aggregatedKey).Counter(metrics.OrchestratorWatchCanceled).Inc(1)
	}
}

// shutdown closes all upstream connections when ctx.Done is called.
func (o *orchestrator) shutdown(ctx context.Context) {
	<-ctx.Done()
	o.upstreamResponseMap.deleteAll()
	for key := range o.downstreamResponseMap.watches {
		o.downstreamResponseMap.deleteAll(key)
	}
}

func isNackRequest(req transport.Request) bool {
	return req.GetError() != nil
}

func (o *orchestrator) send(resp transport.Response, w transport.Watch, aggregatedKey string) error {
	err := w.Send(resp)
	if err != nil {
		// If the channel is blocked, we simply drop subsequent
		// requests and error. Alternative possibilities are
		// discussed here:
		// https://github.com/envoyproxy/xds-relay/pull/53#discussion_r420325553
		o.logger.With("aggregated_key", aggregatedKey).
			Error(context.Background(), "channel blocked during fanout")
		metrics.OrchestratorWatchErrorsSubscope(o.scope, aggregatedKey).Counter(metrics.ErrorChannelFull).Inc(1)
	}
	return err
}
