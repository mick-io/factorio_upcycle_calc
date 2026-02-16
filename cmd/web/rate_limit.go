package main

import (
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRateLimitRPM       = 300
	defaultRateLimitBurst     = 80
	defaultRateLimitClientTTL = 10 * time.Minute
	defaultRateLimitSweep     = 5 * time.Minute
)

type rateLimitConfig struct {
	Enabled    bool
	RPM        int
	Burst      int
	TrustProxy bool
}

type clientTokenBucket struct {
	Tokens   float64
	LastSeen time.Time
}

type ipRateLimiter struct {
	mu          sync.Mutex
	ratePerSec  float64
	burst       float64
	clientTTL   time.Duration
	sweepEvery  time.Duration
	lastSweepAt time.Time
	clients     map[string]*clientTokenBucket
}

func loadRateLimitConfig() rateLimitConfig {
	return rateLimitConfig{
		Enabled:    parseEnvBool("RATE_LIMIT_ENABLED", true),
		RPM:        parseEnvIntMin("RATE_LIMIT_RPM", defaultRateLimitRPM, 1),
		Burst:      parseEnvIntMin("RATE_LIMIT_BURST", defaultRateLimitBurst, 1),
		TrustProxy: parseEnvBool("RATE_LIMIT_TRUST_PROXY", false),
	}
}

func newIPRateLimiter(rpm int, burst int, clientTTL time.Duration, sweepEvery time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		ratePerSec: float64(rpm) / 60.0,
		burst:      float64(burst),
		clientTTL:  clientTTL,
		sweepEvery: sweepEvery,
		clients:    make(map[string]*clientTokenBucket),
	}
}

func (l *ipRateLimiter) Allow(clientID string, now time.Time) (bool, time.Duration) {
	if strings.TrimSpace(clientID) == "" {
		clientID = "unknown"
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lastSweepAt.IsZero() {
		l.lastSweepAt = now
	} else if now.Sub(l.lastSweepAt) >= l.sweepEvery {
		for key, bucket := range l.clients {
			if now.Sub(bucket.LastSeen) > l.clientTTL {
				delete(l.clients, key)
			}
		}
		l.lastSweepAt = now
	}

	bucket, ok := l.clients[clientID]
	if !ok {
		bucket = &clientTokenBucket{
			Tokens:   l.burst,
			LastSeen: now,
		}
		l.clients[clientID] = bucket
	}

	elapsed := now.Sub(bucket.LastSeen).Seconds()
	if elapsed > 0 {
		bucket.Tokens = math.Min(l.burst, bucket.Tokens+(elapsed*l.ratePerSec))
	}
	bucket.LastSeen = now

	if bucket.Tokens >= 1 {
		bucket.Tokens -= 1
		return true, 0
	}

	deficit := 1 - bucket.Tokens
	waitSec := deficit / l.ratePerSec
	if waitSec < 0 {
		waitSec = 0
	}
	return false, time.Duration(math.Ceil(waitSec * float64(time.Second)))
}

func clientIPFromRequest(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
			first := strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
			if first != "" {
				return first
			}
		}

		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func withRateLimit(next http.Handler, cfg rateLimitConfig) http.Handler {
	if !cfg.Enabled {
		return next
	}

	limiter := newIPRateLimiter(
		cfg.RPM,
		cfg.Burst,
		defaultRateLimitClientTTL,
		defaultRateLimitSweep,
	)

	log.Printf(
		"rate limiting enabled: rpm=%d burst=%d trust_proxy=%t",
		cfg.RPM,
		cfg.Burst,
		cfg.TrustProxy,
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Static assets and operational endpoints should not consume limiter tokens.
		if strings.HasPrefix(r.URL.Path, "/static/") || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		clientID := clientIPFromRequest(r, cfg.TrustProxy)
		allowed, retryAfter := limiter.Allow(clientID, time.Now())
		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}

			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			w.WriteHeader(http.StatusTooManyRequests)
			if strings.EqualFold(r.Header.Get("HX-Request"), "true") {
				_, _ = w.Write([]byte(`<p class="nes-text is-error">Rate limit exceeded. Please wait a moment and try again.</p>`))
				return
			}
			_, _ = w.Write([]byte("rate limit exceeded, please try again later"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func parseEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func parseEnvIntMin(key string, fallback int, min int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	return parsed
}

func parseEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
