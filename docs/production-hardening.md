# Production Hardening Guide

This app now supports production-oriented hardening controls. Use the following baseline:

## 1. Run as non-root

- Set `REQUIRE_NON_ROOT=true`.
- The process will fail fast at startup if running as root.
- The provided `deploy/Dockerfile` runs as `app` user by default.

## 2. Read-only filesystem

- Run container with `read_only: true`.
- Provide a writable volume only for cache: `/app/docs/cache`.
- Mount `/tmp` as tmpfs for temporary files.
- Example is included in `deploy/docker-compose.prod.yml`.

## 3. TLS via reverse proxy

- Terminate TLS at a reverse proxy (Caddy/Nginx/Traefik).
- Forward traffic to app on internal HTTP (`app:8080`).
- Use `deploy/Caddyfile` as reference.
- If app should enforce HTTPS redirects at the app layer, set:
  - `ENFORCE_HTTPS=true`
  - `HTTPS_TRUST_PROXY=true` (when behind a proxy that sets `X-Forwarded-Proto`)

## 4. Operational endpoints

- `GET /healthz` for liveness.
- `GET /readyz` for readiness (checks cache directory writability).
- `GET /metrics` for request/429/wiki/cache metrics.

## 5. Recommended environment

- `SERVER_ADDR=:8080`
- `CACHE_DIR=/app/docs/cache`
- `RATE_LIMIT_ENABLED=true`
- `RATE_LIMIT_RPM=300`
- `RATE_LIMIT_BURST=80`
- `REQUIRE_NON_ROOT=true`
- `ENFORCE_HTTPS=false` (set true only when proxy/TLS setup is in place)
