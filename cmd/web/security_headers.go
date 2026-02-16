package main

import (
	"net/http"
	"strings"
)

var defaultCSPDirectives = []string{
	"default-src 'self'",
	"base-uri 'self'",
	"form-action 'self'",
	"frame-ancestors 'none'",
	"object-src 'none'",
	"img-src 'self' data:",
	"style-src 'self' https://unpkg.com https://fonts.googleapis.com",
	"font-src 'self' https://fonts.gstatic.com data:",
	"script-src 'self' https://unpkg.com",
	"connect-src 'self'",
}

func withSecurityHeaders(next http.Handler) http.Handler {
	csp := strings.Join(defaultCSPDirectives, "; ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		headers.Set("Content-Security-Policy", csp)
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		next.ServeHTTP(w, r)
	})
}
