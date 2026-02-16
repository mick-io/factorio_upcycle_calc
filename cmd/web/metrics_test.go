package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithMetrics_CountsRequestsAnd429(t *testing.T) {
	metrics := &appMetrics{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/limited" {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("limited"))
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	handler := withMetrics(next, metrics)

	req1 := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/limited", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if got := metrics.requestsTotal.Load(); got != 2 {
		t.Fatalf("requests_total got %d, want 2", got)
	}
	if got := metrics.requests429.Load(); got != 1 {
		t.Fatalf("requests_429 got %d, want 1", got)
	}
}

func TestMetricsHandler_OutputsPrometheusText(t *testing.T) {
	metrics := &appMetrics{}
	metrics.requestsTotal.Store(3)
	metrics.requests429.Store(1)
	metrics.itemCacheHits.Store(2)
	metrics.itemCacheMisses.Store(1)
	metrics.machineCacheHits.Store(4)
	metrics.machineCacheMisses.Store(4)
	metrics.wikiFetchTotal.Store(10)
	metrics.wikiFetchErrors.Store(2)

	handler := metricsHandler(metrics)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status got %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	expected := []string{
		"app_requests_total 3",
		"app_requests_429_total 1",
		"app_wiki_fetch_total 10",
		"app_wiki_fetch_errors_total 2",
		"app_item_cache_hits_total 2",
		"app_item_cache_misses_total 1",
		"app_machine_cache_hits_total 4",
		"app_machine_cache_misses_total 4",
	}
	for _, token := range expected {
		if !strings.Contains(body, token) {
			t.Fatalf("metrics output missing %q", token)
		}
	}
}
