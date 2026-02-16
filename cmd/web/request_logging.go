package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"
)

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (w *statusCapturingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusCapturingResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func withRequestLogging(next http.Handler, trustProxy bool, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := requestIDFromRequest(r)
		w.Header().Set("X-Request-ID", requestID)

		start := time.Now()
		recorder := &statusCapturingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		statusCode := recorder.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		clientIP := clientIPFromRequest(r, trustProxy)
		latency := time.Since(start)
		logger.Printf(
			`request id=%s method=%s path=%s status=%d bytes=%d duration_ms=%.3f ip=%s ua=%q`,
			requestID,
			r.Method,
			r.URL.Path,
			statusCode,
			recorder.bytes,
			float64(latency.Microseconds())/1000.0,
			clientIP,
			r.UserAgent(),
		)
	})
}

func requestIDFromRequest(r *http.Request) string {
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}

	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "req-fallback"
	}
	return hex.EncodeToString(raw)
}
