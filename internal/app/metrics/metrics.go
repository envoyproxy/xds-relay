// Package metrics contains all the metrics constants used in xds-relay.
package metrics

import (
	"github.com/uber-go/tally"
)

const TagName = "key"

// .server
const (
	// scope: .server.*
	ScopeServer = "server"

	ServerAlive = "alive" // counter, 1 indicates that the server is running
)

// .orchestrator
const (
	// scope: .orchestrator.$aggregated_key.*
	ScopeOrchestrator = "orchestrator"

	// scope: .orchestrator.$aggregated_key.watch.*
	ScopeOrchestratorWatch    = "watch"
	OrchestratorWatchCreated  = "created"  // counter, # of watches created per aggregated key
	OrchestratorWatchCanceled = "canceled" // counter, # of watch cancels initiated per aggregated key
	OrchestratorWatchFanouts  = "fanout"   // counter, # of responses pushed downstream

	// scope: .orchestrator.$aggregated_key.cache_evict.*
	ScopeOrchestratorCacheEvict            = "cache_evict"
	OrcheestratorCacheEvictCount           = "calls"            // counter, # of times cache evict is called
	OrchestratorOnCacheEvictedRequestCount = "requests_evicted" // counter, # of requests that were evicted

	// scope: .orchestrator.$aggregated_key.watch.errors.*
	ScopeOrchestratorWatchErrors = "errors"
	ErrorRegisterWatch           = "register"     // counter, # of errors as a result of watch registration in the cache
	ErrorChannelFull             = "channel_full" // counter, # of response fanout failures due to blocked channels
	ErrorUpstreamFailure         = "upstream"     // counter, # of errors as a result of a problem upstream
	ErrorCacheMiss               = "cache_miss"   // counter, # of errors due to a fanout attempt with no cached response
)

// .upstream
const (
	// scope: .upstream.$xds.*
	ScopeUpstream = "upstream"

	// scope: .upstream.lds.*
	ScopeUpstreamLDS = "lds"
	// scope: .upstream.cds.*
	ScopeUpstreamCDS = "cds"
	// scope: .upstream.rds.*
	ScopeUpstreamRDS = "rds"
	// scope: .upstream.eds.*
	ScopeUpstreamEDS = "eds"

	UpstreamStreamOpened = "stream_opened" // counter, # of times a gRPC stream was opened to the origin server.
)

// .cache
const (
	// scope: .cache.$aggregated_key.*
	ScopeCache = "cache"

	// scope: .cache.$aggregated_key.fetch.*
	ScopeCacheFetch   = "fetch"
	CacheFetchAttempt = "attempt" // counter, # of cache fetches called
	CacheFetchMiss    = "miss"    // counter, # of cache fetches that resulted in a miss
	CacheFetchError   = "error"   // counter, # of errors while calling cache fetch
	CacheFetchExpired = "expired" // counter, # of cache entries expired while calling fetch

	// scope: .cache.$aggregated_key.set_response.*
	ScopeCacheSet   = "set_response"
	CacheSetAttempt = "attempt" // counter, # of cache sets called
	CacheSetSuccess = "success" // counter, # of cache sets succeeded
	CacheSetError   = "error"   // counter, # of errors while calling cache set

	// scope: .cache.$aggregated_key.add_request.*
	ScopeCacheAdd   = "add_request"
	CacheAddAttempt = "attempt" // counter, # of cache add requests called
	CacheAddSuccess = "success" // counter, # of cache add requests succeeded
	CacheAddError   = "error"   // counter, # of errors while calling cache add

	// scope: .cache.$aggregated_key.delete_request.*
	ScopeCacheDelete   = "delete_request"
	CacheDeleteAttempt = "attempt" // counter, # of cache delete requests called
	CacheDeleteSuccess = "success" // counter, # of cache delete requests succeeded
	CacheDeleteError   = "error"   // counter, # of errors while calling cache delete
)

// .mapper
const (
	// scope: .mapper.*
	ScopeMapper = "mapper"

	MapperSuccess = "success" // counter, # of successfully converted request to aggregated keys
	MapperError   = "error"   // counter, # of errors when converting a request to an aggregated key
)

// .upstream.error_interceptor
const (
	// scope: .upstream.error_interceptor.*
	ScopeErrorInterceptor = "error_interceptor"

	ErrorInterceptorErrorSendMsg = "error_sendmsg"
	ErrorInterceptorErrorRecvMsg = "error_recvmsg"
)

// OrchestratorWatchSubscope gets the orchestor watch subscope for the aggregated key.
// ex: .orchestrator.$aggregated_key.watch
func OrchestratorWatchSubscope(parent tally.Scope, aggregatedKey string) tally.Scope {
	return parent.SubScope(aggregatedKey).SubScope(ScopeOrchestratorWatch)
}

// OrchestratorWatchErrorsSubscope gets the orchestor watch errora subscope for the aggregated key.
// ex: .orchestrator.$aggregated_key.watch.errors
func OrchestratorWatchErrorsSubscope(parent tally.Scope, aggregatedKey string) tally.Scope {
	return parent.SubScope(aggregatedKey).SubScope(ScopeOrchestratorWatch).SubScope(ScopeOrchestratorWatchErrors)
}

// OrchestratorCacheEvictSubscope gets the orchestor cache evict subscope for the aggregated key.
// ex: .orchestrator.$aggregated_key.cache_evict
func OrchestratorCacheEvictSubscope(parent tally.Scope, aggregatedKey string) tally.Scope {
	return parent.SubScope(aggregatedKey).SubScope(ScopeOrchestratorCacheEvict)
}

// CacheFetchSubscope gets the cache fetch subscope for the aggregated key.
// ex: .cache.$aggregated_key.fetch
func CacheFetchSubscope(parent tally.Scope, aggregatedKey string) tally.Scope {
	return parent.SubScope(aggregatedKey).SubScope(ScopeCacheFetch)
}

// CacheSetSubscope gets the cache set subscope and adds the aggregated key as a point tag.
// ex: .cache.set_response+key=$aggregated_key
func CacheSetSubscope(parent tally.Scope, aggregatedKey string) tally.Scope {
	return parent.SubScope(ScopeCacheSet).Tagged(map[string]string{TagName: aggregatedKey})
}

// CacheAddRequestSubscope gets the cache add request subscope and adds the aggregated key
// as a point tag.
// ex: .cache.add_request+key=$aggregated_key
func CacheAddRequestSubscope(parent tally.Scope, aggregatedKey string) tally.Scope {
	return parent.SubScope(ScopeCacheAdd).Tagged(map[string]string{TagName: aggregatedKey})
}

// CacheDeleteRequestSubscope gets the cache delete request subscope and adds the aggregated key
// as a point tag.
// ex: .cache.delete_request+key=$aggregated_key
func CacheDeleteRequestSubscope(parent tally.Scope, aggregatedKey string) tally.Scope {
	return parent.SubScope(ScopeCacheDelete).Tagged(map[string]string{TagName: aggregatedKey})
}
