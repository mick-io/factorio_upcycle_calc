package main

import (
	"strings"
	"testing"
	"time"
)

func TestLoadAppConfigFromEnv_Defaults(t *testing.T) {
	t.Setenv("SERVER_ADDR", "")
	t.Setenv("CACHE_DIR", "")
	t.Setenv("SERVER_READ_TIMEOUT", "")
	t.Setenv("RATE_LIMIT_ENABLED", "")

	cfg, err := loadAppConfigFromEnv()
	if err != nil {
		t.Fatalf("loadAppConfigFromEnv returned unexpected error: %v", err)
	}

	if cfg.Server.Addr != ":8080" {
		t.Fatalf("server addr got %q, want :8080", cfg.Server.Addr)
	}
	if cfg.CacheDir != "docs/cache" {
		t.Fatalf("cache dir got %q, want docs/cache", cfg.CacheDir)
	}
	if cfg.Server.ReadTimeout != 10*time.Second {
		t.Fatalf("read timeout got %v, want 10s", cfg.Server.ReadTimeout)
	}
	if !cfg.RateLimit.Enabled {
		t.Fatal("rate limiting should default to enabled")
	}
	if cfg.EnforceHTTPS {
		t.Fatal("ENFORCE_HTTPS should default to false")
	}
	if !cfg.HTTPSTrustProxy {
		t.Fatal("HTTPS_TRUST_PROXY should default to true")
	}
}

func TestLoadAppConfigFromEnv_InvalidValueFailsFast(t *testing.T) {
	t.Setenv("SERVER_READ_TIMEOUT", "not-a-duration")
	_, err := loadAppConfigFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid timeout value")
	}
	if !strings.Contains(err.Error(), "SERVER_READ_TIMEOUT") {
		t.Fatalf("error %q should mention SERVER_READ_TIMEOUT", err.Error())
	}
}

func TestLoadAppConfigFromEnv_InvalidRateLimitIntFailsFast(t *testing.T) {
	t.Setenv("RATE_LIMIT_RPM", "0")
	_, err := loadAppConfigFromEnv()
	if err == nil {
		t.Fatal("expected error for RATE_LIMIT_RPM=0")
	}
	if !strings.Contains(err.Error(), "RATE_LIMIT_RPM") {
		t.Fatalf("error %q should mention RATE_LIMIT_RPM", err.Error())
	}
}

func TestLoadAppConfigFromEnv_InvalidRateLimitBoolFailsFast(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "maybe")
	_, err := loadAppConfigFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid RATE_LIMIT_ENABLED")
	}
	if !strings.Contains(err.Error(), "RATE_LIMIT_ENABLED") {
		t.Fatalf("error %q should mention RATE_LIMIT_ENABLED", err.Error())
	}
}
