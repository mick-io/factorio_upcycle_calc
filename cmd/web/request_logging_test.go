package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithRequestLogging_SetsRequestIDHeader(t *testing.T) {
	handler := withRequestLogging(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
		false,
		log.New(&bytes.Buffer{}, "", 0),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1111"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := strings.TrimSpace(rr.Header().Get("X-Request-ID")); got == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}
}

func TestWithRequestLogging_PreservesIncomingRequestIDAndLogsFields(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", 0)

	handler := withRequestLogging(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created"))
		}),
		true,
		logger,
	)

	req := httptest.NewRequest(http.MethodPost, "/plan", nil)
	req.RemoteAddr = "10.0.0.9:2222"
	req.Header.Set("X-Request-ID", "req-123")
	req.Header.Set("X-Forwarded-For", "203.0.113.20")
	req.Header.Set("User-Agent", "test-agent")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Request-ID"); got != "req-123" {
		t.Fatalf("response request id got %q, want req-123", got)
	}

	line := logBuffer.String()
	expectContains := []string{
		"id=req-123",
		"method=POST",
		"path=/plan",
		"status=201",
		"bytes=7",
		"ip=203.0.113.20",
		`ua="test-agent"`,
	}
	for _, expected := range expectContains {
		if !strings.Contains(line, expected) {
			t.Fatalf("log output %q does not contain %q", line, expected)
		}
	}
}
