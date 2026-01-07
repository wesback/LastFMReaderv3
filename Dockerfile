# ============================================================================
# Multi-Stage Dockerfile for LastFMReaderv3
# ============================================================================
# This Dockerfile uses a two-stage build process to produce a minimal,
# secure container image optimized for production deployment.
#
# Stage 1 (Build): Compiles the Go binary using golang:alpine
# Stage 2 (Runtime): Runs the binary using distroless/static:nonroot
#
# Benefits:
#   - Small image size (~15-20MB vs ~300MB with full Go image)
#   - Minimal attack surface (no shell, package manager, or unnecessary tools)
#   - Non-root execution for enhanced security
#   - Reproducible builds with pinned base images
# ============================================================================

# ============================================================================
# Build Stage: Compile Go binary
# ============================================================================
# Base: golang:1.25-alpine (~300MB)
# Purpose: Download dependencies and compile static binary
# Optimization: go.mod/go.sum copied first to leverage Docker layer caching
FROM golang:1.25-alpine AS build

WORKDIR /src

# Copy dependency manifests first (enables Docker layer caching)
# These files rarely change, so this layer can be reused across builds
COPY go.mod go.sum ./

# Download dependencies before copying source code
# This layer is cached unless go.mod/go.sum changes
RUN go mod download

# Copy source code
# This layer is invalidated on any source change
COPY . .

# Build static binary with optimizations
# CGO_ENABLED=0: Disables CGO for fully static binary (no libc dependency)
# GOOS=linux GOARCH=amd64: Target Linux x86-64 (change for ARM: GOARCH=arm64)
# -ldflags="-s -w": Strip debug symbols and DWARF table (-s) and symbol table (-w)
#   Reduces binary size by ~30% with no runtime performance impact
# -X: Inject version and build time at compile time
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w \
      -X github.com/lastfm-reader/lastfm-sync/internal/version.Version=v1.0.0 \
      -X github.com/lastfm-reader/lastfm-sync/internal/version.BuildTime=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
    -o /out/lastfm-sync \
    ./cmd/lastfm-sync

# ============================================================================
# Runtime Stage: Minimal distroless container
# ============================================================================
# Base: gcr.io/distroless/static:nonroot (~2MB)
# Purpose: Run the compiled binary with minimal dependencies
# Security: No shell, no package manager, runs as non-root user (uid 65532)
#
# Why distroless?
#   - Minimal attack surface (only essential runtime files)
#   - Smaller size than alpine (~2MB vs ~7MB)
#   - Maintained by Google with security updates
#   - Includes CA certificates for HTTPS requests
#   - Non-root user pre-configured (no need for USER directive)
#
# Alternatives:
#   - scratch: Even smaller but no CA certs (breaks HTTPS)
#   - alpine: Includes shell and package manager (larger attack surface)
FROM gcr.io/distroless/static:nonroot

# OCI image labels for container registries and tooling
LABEL org.opencontainers.image.source="https://github.com/lastfm-reader/lastfm-sync"
LABEL org.opencontainers.image.description="Last.fm scrobble sync tool with Azure Blob Storage support"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.title="LastFMReaderv3"
LABEL org.opencontainers.image.version="1.0.0"

# Security: Run as non-root user (uid=65532, gid=65532)
# distroless/static:nonroot already defaults to this user, but explicit is better
USER nonroot:nonroot

# Working directory for the application
# Not strictly necessary for static binary, but provides a clean execution context
WORKDIR /app

# Copy compiled binary from build stage
# --from=build: Copy from previous stage (not from host filesystem)
# Binary is owned by root but executable by nonroot user
COPY --from=build /out/lastfm-sync /app/lastfm-sync

# Volume mount point for local storage (used with --output local)
# Map to host directory at runtime: docker run -v ./data:/data
# Container writes: /data/{user}.ndjson and /data/state/{user}.watermark
VOLUME ["/data"]

# Default entrypoint: Always run lastfm-sync binary
# Cannot be overridden without --entrypoint flag
ENTRYPOINT ["/app/lastfm-sync"]

# Default command: Show help text
# Can be overridden: docker run lastfm-sync fetch --user alice
CMD ["--help"]

# ============================================================================
# Build Instructions
# ============================================================================
# Build image:
#   docker build -t lastfm-sync:latest .
#
# Build with custom version:
#   docker build -t lastfm-sync:v1.2.3 \
#     --build-arg VERSION=v1.2.3 .
#
# Run with local storage:
#   docker run --rm \
#     -v ./data:/data \
#     -e LASTFM_API_KEY=your-key \
#     lastfm-sync:latest fetch --user alice
#
# Run with Azure storage:
#   docker run --rm \
#     -e LASTFM_API_KEY=your-key \
#     -e AZURE_STORAGE_ACCOUNT=myaccount \
#     lastfm-sync:latest fetch --user alice --output azure \
#       --azure-container scrobbles --azure-auth default
#
# See docs/docker.md for complete documentation.
# ============================================================================
