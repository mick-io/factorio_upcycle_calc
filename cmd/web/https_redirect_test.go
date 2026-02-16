package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithHTTPSRedirect_RedirectsInsecureRequests(t *testing.T) {
	handler := withHTTPSRedirect(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}), true, true)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/plan", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPermanentRedirect {
		t.Fatalf("status got %d, want 308", rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "https://example.com/plan" {
		t.Fatalf("location got %q, want https://example.com/plan", got)
	}
}

func TestWithHTTPSRedirect_AllowsForwardedHTTPS(t *testing.T) {
	handler := withHTTPSRedirect(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}), true, true)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/plan", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status got %d, want 200", rr.Code)
	}
}
