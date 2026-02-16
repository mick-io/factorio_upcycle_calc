package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestIPRateLimiter_EnforcesBurstAndRefill(t *testing.T) {
	limiter := newIPRateLimiter(60, 2, time.Minute, time.Minute)
	now := time.Unix(0, 0)

	if allowed, _ := limiter.Allow("1.2.3.4", now); !allowed {
		t.Fatal("first request unexpectedly blocked")
	}
	if allowed, _ := limiter.Allow("1.2.3.4", now); !allowed {
		t.Fatal("second request unexpectedly blocked")
	}

	allowed, retryAfter := limiter.Allow("1.2.3.4", now)
	if allowed {
		t.Fatal("third request should be blocked")
	}
	if retryAfter <= 0 {
		t.Fatalf("retry_after got %v, want > 0", retryAfter)
	}

	if allowed, _ := limiter.Allow("1.2.3.4", now.Add(1*time.Second)); !allowed {
		t.Fatal("request should be allowed after token refill")
	}
}

func TestClientIPFromRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.5:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.2")
	r.Header.Set("X-Real-IP", "198.51.100.7")

	if got := clientIPFromRequest(r, false); got != "10.0.0.5" {
		t.Fatalf("ip without trusted proxy got %q, want 10.0.0.5", got)
	}
	if got := clientIPFromRequest(r, true); got != "203.0.113.10" {
		t.Fatalf("ip with trusted proxy got %q, want 203.0.113.10", got)
	}
}

func TestWithRateLimit_Returns429(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	handler := withRateLimit(next, rateLimitConfig{
		Enabled:    true,
		RPM:        1,
		Burst:      1,
		TrustProxy: false,
	})

	req1 := httptest.NewRequest(http.MethodGet, "/partials/recycler-stats", nil)
	req1.RemoteAddr = "127.0.0.1:5000"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first request status got %d, want 200", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/partials/recycler-stats", nil)
	req2.RemoteAddr = "127.0.0.1:5001"
	req2.Header.Set("HX-Request", "true")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status got %d, want 429", rr2.Code)
	}
	if retryAfter := strings.TrimSpace(rr2.Header().Get("Retry-After")); retryAfter == "" {
		t.Fatal("expected Retry-After header to be set")
	}
	if !strings.Contains(rr2.Body.String(), "Rate limit exceeded") {
		t.Fatalf("429 body got %q, expected htmx-friendly error message", rr2.Body.String())
	}
}

func TestWithRateLimit_StaticBypass(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	handler := withRateLimit(next, rateLimitConfig{
		Enabled:    true,
		RPM:        1,
		Burst:      1,
		TrustProxy: false,
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/static/js/app.js", nil)
		req.RemoteAddr = "127.0.0.1:6000"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("static request %d status got %d, want 200", i+1, rr.Code)
		}
	}
}

func TestParseEnvDuration(t *testing.T) {
	key := "TEST_PARSE_ENV_DURATION"
	t.Setenv(key, "2s")
	if got := parseEnvDuration(key, time.Second); got != 2*time.Second {
		t.Fatalf("parsed duration got %v, want 2s", got)
	}

	t.Setenv(key, "invalid")
	if got := parseEnvDuration(key, time.Second); got != time.Second {
		t.Fatalf("invalid value fallback got %v, want 1s", got)
	}

	_ = os.Unsetenv(key)
	if got := parseEnvDuration(key, 3*time.Second); got != 3*time.Second {
		t.Fatalf("missing env fallback got %v, want 3s", got)
	}
}
