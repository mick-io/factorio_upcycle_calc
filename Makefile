APP_NAME := factorio-upcycle-calc
CMD := ./cmd/web
BIN_DIR := ./bin
CONTAINER_ENGINE ?= podman
CONTAINER_ARGS ?=
CONTAINER_IMAGE ?= localhost/$(APP_NAME):prod
CONTAINER_NAME ?= factorio-upcycle-app
CONTAINER_DOCKERFILE ?= deploy/Dockerfile
CONTAINER_RUN_ARGS ?= --network=host

.PHONY: help setup install-air dev run build test fmt tidy clean container-build container-run

help:
	@echo "Available targets:"
	@echo "  make setup        - install tooling and tidy modules"
	@echo "  make install-air  - install air live-reload tool"
	@echo "  make dev          - run app with air"
	@echo "  make run          - run app directly with go run"
	@echo "  make build        - build binary"
	@echo "  make test         - run tests"
	@echo "  make fmt          - format Go files"
	@echo "  make tidy         - tidy go modules"
	@echo "  make clean        - remove build artifacts"
	@echo "  make container-build - build production container image"
	@echo "  make container-run   - run production container image"

setup: install-air tidy

install-air:
	go install github.com/air-verse/air@latest

dev:
	@if command -v air >/dev/null 2>&1; then \
		air -c .air.toml; \
	elif [ -x "$$(go env GOPATH)/bin/air" ]; then \
		"$$(go env GOPATH)/bin/air" -c .air.toml; \
	else \
		echo "air is not installed; run \`make install-air\`"; \
		exit 1; \
	fi

run:
	go run $(CMD)

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(CMD)

test:
	go test ./...

fmt:
	gofmt -w $$(find . -type f -name '*.go' -not -path './tmp/*' -not -path './bin/*')

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR) tmp

container-build:
	$(CONTAINER_ENGINE) $(CONTAINER_ARGS) build -f $(CONTAINER_DOCKERFILE) -t $(CONTAINER_IMAGE) .

container-run:
	$(CONTAINER_ENGINE) $(CONTAINER_ARGS) run -d --replace --name $(CONTAINER_NAME) $(CONTAINER_RUN_ARGS) $(CONTAINER_IMAGE)
