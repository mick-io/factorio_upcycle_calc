# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/factorio-upcycle-calc ./cmd/web

FROM alpine:3.21 AS runtime

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app -g 10001 \
    && adduser -S app -u 10001 -G app

WORKDIR /app

COPY --from=builder --chown=app:app /out/factorio-upcycle-calc /app/factorio-upcycle-calc
COPY --chown=app:app templates /app/templates
COPY --chown=app:app static /app/static
COPY --chown=app:app docs/recyclable-items.txt /app/docs/recyclable-items.txt

RUN mkdir -p /app/docs/cache && chown -R app:app /app

USER app:app

ENV SERVER_ADDR=:8080 \
    CACHE_DIR=/app/docs/cache \
    REQUIRE_NON_ROOT=true

EXPOSE 8080/tcp
STOPSIGNAL SIGTERM

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- "http://127.0.0.1:8080/healthz" >/dev/null || exit 1

ENTRYPOINT ["/app/factorio-upcycle-calc"]
