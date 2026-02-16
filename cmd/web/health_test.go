package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestHealthzHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	healthzHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status got %d, want 200", rr.Code)
	}
	if body := rr.Body.String(); body != "ok" {
		t.Fatalf("body got %q, want ok", body)
	}
}

func TestReadyzHandler(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	handler := readyzHandler(cacheDir)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status got %d, want 200", rr.Code)
	}
	if body := rr.Body.String(); body != "ready" {
		t.Fatalf("body got %q, want ready", body)
	}
}

func TestReadyzHandler_FailsOnInvalidPath(t *testing.T) {
	handler := readyzHandler("")
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status got %d, want 503", rr.Code)
	}
}
