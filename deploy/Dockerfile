FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY templates ./templates
COPY static ./static
COPY docs/recyclable-items.txt ./docs/recyclable-items.txt
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/factorio-upcycle-calc ./cmd/web

FROM alpine:3.21

RUN addgroup -S app && adduser -S -G app app
WORKDIR /app

COPY --from=builder /out/factorio-upcycle-calc /app/factorio-upcycle-calc
COPY templates /app/templates
COPY static /app/static
COPY docs/recyclable-items.txt /app/docs/recyclable-items.txt

RUN mkdir -p /app/docs/cache && chown -R app:app /app

USER app

ENV SERVER_ADDR=:8080
ENV CACHE_DIR=/app/docs/cache
ENV REQUIRE_NON_ROOT=true

EXPOSE 8080

ENTRYPOINT ["/app/factorio-upcycle-calc"]
