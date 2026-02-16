package main

import (
	"net/http"
	"strings"
)

func withHTTPSRedirect(next http.Handler, enforce bool, trustProxy bool) http.Handler {
	if !enforce {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestIsHTTPS(r, trustProxy) {
			next.ServeHTTP(w, r)
			return
		}

		target := "https://" + r.Host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	})
}

func requestIsHTTPS(r *http.Request, trustProxy bool) bool {
	if r.TLS != nil {
		return true
	}
	if trustProxy {
		proto := strings.TrimSpace(strings.ToLower(r.Header.Get("X-Forwarded-Proto")))
		if proto == "https" {
			return true
		}
	}
	return false
}
