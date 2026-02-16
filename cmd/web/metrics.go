package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

type appMetrics struct {
	requestsTotal atomic.Uint64
	requests429   atomic.Uint64

	wikiFetchTotal     atomic.Uint64
	wikiFetchErrors    atomic.Uint64
	wikiFetchLatencyNs atomic.Uint64

	itemCacheHits      atomic.Uint64
	itemCacheMisses    atomic.Uint64
	machineCacheHits   atomic.Uint64
	machineCacheMisses atomic.Uint64
}

var metricsCollector = &appMetrics{}

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *metricsResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *metricsResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(p)
}

func withMetrics(next http.Handler, metrics *appMetrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &metricsResponseWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		metrics.requestsTotal.Add(1)
		if writer.statusCode == http.StatusTooManyRequests {
			metrics.requests429.Add(1)
		}
	})
}

func metricsHandler(metrics *appMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		requestsTotal := metrics.requestsTotal.Load()
		requests429 := metrics.requests429.Load()
		wikiFetchTotal := metrics.wikiFetchTotal.Load()
		wikiFetchErrors := metrics.wikiFetchErrors.Load()
		wikiFetchLatencyNs := metrics.wikiFetchLatencyNs.Load()

		itemHits := metrics.itemCacheHits.Load()
		itemMisses := metrics.itemCacheMisses.Load()
		itemRatio := hitRatio(itemHits, itemMisses)

		machineHits := metrics.machineCacheHits.Load()
		machineMisses := metrics.machineCacheMisses.Load()
		machineRatio := hitRatio(machineHits, machineMisses)

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(
			w,
			`# HELP app_requests_total Total HTTP requests.
# TYPE app_requests_total counter
app_requests_total %d
# HELP app_requests_429_total Total HTTP 429 responses.
# TYPE app_requests_429_total counter
app_requests_429_total %d
# HELP app_wiki_fetch_total Total wiki fetch attempts.
# TYPE app_wiki_fetch_total counter
app_wiki_fetch_total %d
# HELP app_wiki_fetch_errors_total Total wiki fetch errors.
# TYPE app_wiki_fetch_errors_total counter
app_wiki_fetch_errors_total %d
# HELP app_wiki_fetch_latency_seconds_sum Sum of wiki fetch latency in seconds.
# TYPE app_wiki_fetch_latency_seconds_sum counter
app_wiki_fetch_latency_seconds_sum %.6f
# HELP app_item_cache_hits_total Total item cache hits.
# TYPE app_item_cache_hits_total counter
app_item_cache_hits_total %d
# HELP app_item_cache_misses_total Total item cache misses.
# TYPE app_item_cache_misses_total counter
app_item_cache_misses_total %d
# HELP app_item_cache_hit_ratio Item cache hit ratio.
# TYPE app_item_cache_hit_ratio gauge
app_item_cache_hit_ratio %.6f
# HELP app_machine_cache_hits_total Total machine cache hits.
# TYPE app_machine_cache_hits_total counter
app_machine_cache_hits_total %d
# HELP app_machine_cache_misses_total Total machine cache misses.
# TYPE app_machine_cache_misses_total counter
app_machine_cache_misses_total %d
# HELP app_machine_cache_hit_ratio Machine cache hit ratio.
# TYPE app_machine_cache_hit_ratio gauge
app_machine_cache_hit_ratio %.6f
`,
			requestsTotal,
			requests429,
			wikiFetchTotal,
			wikiFetchErrors,
			float64(wikiFetchLatencyNs)/float64(time.Second),
			itemHits,
			itemMisses,
			itemRatio,
			machineHits,
			machineMisses,
			machineRatio,
		)
	}
}

func hitRatio(hits uint64, misses uint64) float64 {
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}
