package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type serverRuntimeConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type appConfig struct {
	CacheDir        string
	Server          serverRuntimeConfig
	RateLimit       rateLimitConfig
	RequireNonRoot  bool
	EnforceHTTPS    bool
	HTTPSTrustProxy bool
}

func loadAppConfigFromEnv() (appConfig, error) {
	addr := strings.TrimSpace(os.Getenv("SERVER_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	if addr == "" {
		return appConfig{}, fmt.Errorf("SERVER_ADDR is required")
	}

	cacheDir := strings.TrimSpace(os.Getenv("CACHE_DIR"))
	if cacheDir == "" {
		cacheDir = "docs/cache"
	}
	if cacheDir == "" {
		return appConfig{}, fmt.Errorf("CACHE_DIR is required")
	}

	readTimeout, err := parseEnvDurationStrict("SERVER_READ_TIMEOUT", 10*time.Second, time.Millisecond)
	if err != nil {
		return appConfig{}, err
	}
	writeTimeout, err := parseEnvDurationStrict("SERVER_WRITE_TIMEOUT", 30*time.Second, time.Millisecond)
	if err != nil {
		return appConfig{}, err
	}
	idleTimeout, err := parseEnvDurationStrict("SERVER_IDLE_TIMEOUT", 60*time.Second, time.Millisecond)
	if err != nil {
		return appConfig{}, err
	}
	shutdownTimeout, err := parseEnvDurationStrict("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second, time.Millisecond)
	if err != nil {
		return appConfig{}, err
	}

	rateEnabled, err := parseEnvBoolStrict("RATE_LIMIT_ENABLED", true)
	if err != nil {
		return appConfig{}, err
	}
	rateRPM, err := parseEnvIntStrict("RATE_LIMIT_RPM", defaultRateLimitRPM, 1, 1000000)
	if err != nil {
		return appConfig{}, err
	}
	rateBurst, err := parseEnvIntStrict("RATE_LIMIT_BURST", defaultRateLimitBurst, 1, 1000000)
	if err != nil {
		return appConfig{}, err
	}
	rateTrustProxy, err := parseEnvBoolStrict("RATE_LIMIT_TRUST_PROXY", false)
	if err != nil {
		return appConfig{}, err
	}
	requireNonRoot, err := parseEnvBoolStrict("REQUIRE_NON_ROOT", false)
	if err != nil {
		return appConfig{}, err
	}
	enforceHTTPS, err := parseEnvBoolStrict("ENFORCE_HTTPS", false)
	if err != nil {
		return appConfig{}, err
	}
	httpsTrustProxy, err := parseEnvBoolStrict("HTTPS_TRUST_PROXY", true)
	if err != nil {
		return appConfig{}, err
	}
	if requireNonRoot && os.Geteuid() == 0 {
		return appConfig{}, fmt.Errorf("REQUIRE_NON_ROOT is enabled but process is running as root")
	}

	return appConfig{
		CacheDir: cacheDir,
		Server: serverRuntimeConfig{
			Addr:            addr,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			IdleTimeout:     idleTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
		RateLimit: rateLimitConfig{
			Enabled:    rateEnabled,
			RPM:        rateRPM,
			Burst:      rateBurst,
			TrustProxy: rateTrustProxy,
		},
		RequireNonRoot:  requireNonRoot,
		EnforceHTTPS:    enforceHTTPS,
		HTTPSTrustProxy: httpsTrustProxy,
	}, nil
}

func parseEnvBoolStrict(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback, nil
	}

	switch raw {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean value", key)
	}
}

func parseEnvIntStrict(key string, fallback int, min int, max int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	if parsed < min {
		return 0, fmt.Errorf("%s must be >= %d", key, min)
	}
	if max > 0 && parsed > max {
		return 0, fmt.Errorf("%s must be <= %d", key, max)
	}
	return parsed, nil
}

func parseEnvDurationStrict(key string, fallback time.Duration, min time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration (example: 10s)", key)
	}
	if parsed < min {
		return 0, fmt.Errorf("%s must be >= %s", key, min.String())
	}
	return parsed, nil
}
