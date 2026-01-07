# Build variables
VERSION ?= v1.0.0-dev
BUILD_TIME ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
ORG ?= lastfm-reader

# Paths
DIST_DIR := dist
BIN_NAME := lastfm-sync

.PHONY: help deps lint test test-coverage build build-all docker docker-build clean

help:
	@echo "Available targets:"
	@echo "  make deps           - download and tidy dependencies"
	@echo "  make lint           - run linters (golangci-lint, go vet)"
	@echo "  make test           - run unit tests with race detector"
	@echo "  make test-coverage  - run tests + generate coverage report"
	@echo "  make build          - build native binary"
	@echo "  make build-all      - cross-compile for multiple platforms"
	@echo "  make docker         - build Docker image (local)"
	@echo "  make clean          - remove build artifacts"

deps:
	@echo "=== Downloading dependencies ==="
	go mod download
	go mod tidy

lint:
	@echo "=== Running linters ==="
	golangci-lint run ./... || true
	go vet ./... || true

test:
	@echo "=== Running tests ==="
	go test -race -cover -v ./...

test-coverage:
	@echo "=== Running tests with coverage ==="
	go test -race -coverprofile=coverage.out -v ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

build: | $(DIST_DIR)
	@echo "=== Building $(BIN_NAME) ==="
	CGO_ENABLED=0 go build \
		-ldflags="-s -w \
			-X github.com/lastfm-reader/lastfm-sync/internal/version.Version=$(VERSION) \
			-X github.com/lastfm-reader/lastfm-sync/internal/version.BuildTime=$(BUILD_TIME) \
			-X github.com/lastfm-reader/lastfm-sync/internal/version.GitCommit=$(GIT_COMMIT)" \
		-o $(DIST_DIR)/$(BIN_NAME) \
		./cmd/lastfm-sync
	@echo "Binary: $(DIST_DIR)/$(BIN_NAME)"

build-all: | $(DIST_DIR)
	@echo "=== Cross-compiling for multiple platforms ==="
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			echo "Building $$os/$$arch"; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build \
				-ldflags="-s -w" \
				-o $(DIST_DIR)/$(BIN_NAME)-$$os-$$arch \
				./cmd/lastfm-sync; \
		done; \
	done

docker: build
	@echo "=== Building Docker image ==="
	docker build -t ghcr.io/$(ORG)/$(BIN_NAME):$(VERSION) .
	docker tag ghcr.io/$(ORG)/$(BIN_NAME):$(VERSION) ghcr.io/$(ORG)/$(BIN_NAME):latest

docker-build:
	@echo "=== Building Docker image (without binary rebuild) ==="
	docker build -t ghcr.io/$(ORG)/$(BIN_NAME):$(VERSION) .

clean:
	@echo "=== Cleaning build artifacts ==="
	rm -rf $(DIST_DIR) coverage.* *.profraw

$(DIST_DIR):
	mkdir -p $(DIST_DIR)
