package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithSecurityHeaders_SetsExpectedHeaders(t *testing.T) {
	handler := withSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("missing Content-Security-Policy header")
	}
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options got %q, want nosniff", got)
	}
	if got := rr.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options got %q, want DENY", got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("Referrer-Policy got %q", got)
	}
	if got := rr.Header().Get("Permissions-Policy"); !strings.Contains(got, "camera=()") {
		t.Fatalf("Permissions-Policy got %q", got)
	}
}

func TestWithSecurityHeaders_AppliesToRateLimitedResponses(t *testing.T) {
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	chain := withSecurityHeaders(withRateLimit(base, rateLimitConfig{
		Enabled: true,
		RPM:     1,
		Burst:   1,
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/partials/recycler-stats", nil)
	req1.RemoteAddr = "127.0.0.1:7000"
	rr1 := httptest.NewRecorder()
	chain.ServeHTTP(rr1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/partials/recycler-stats", nil)
	req2.RemoteAddr = "127.0.0.1:7001"
	rr2 := httptest.NewRecorder()
	chain.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("status got %d, want 429", rr2.Code)
	}
	if got := rr2.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("missing Content-Security-Policy header on 429")
	}
}
