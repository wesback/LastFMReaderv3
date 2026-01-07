# Docker Documentation

Complete guide for building and running LastFMReaderv3 in Docker containers.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Multi-Stage Build Architecture](#multi-stage-build-architecture)
- [Building the Image](#building-the-image)
- [Running Containers](#running-containers)
- [Docker Compose](#docker-compose)
- [Volume Management](#volume-management)
- [Environment Variables](#environment-variables)
- [Image Size Optimization](#image-size-optimization)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

---

## Overview

LastFMReaderv3 uses a **multi-stage Dockerfile** to produce a minimal, secure container image:

- **Build Stage**: golang:1.25-alpine (~300MB) - compiles the Go binary
- **Runtime Stage**: gcr.io/distroless/static:nonroot (~2MB) - runs the binary

**Final Image Size**: ~15-20MB (vs ~300MB with full Go image)

**Key Features**:
- ✅ Minimal attack surface (no shell, no package manager)
- ✅ Non-root execution (uid 65532)
- ✅ Static binary (no external dependencies)
- ✅ Layer caching optimization (fast rebuilds)
- ✅ Production-ready security

---

## Quick Start

### Build and Run

```bash
# Build the image
docker build -t lastfm-sync:latest .

# Run with help
docker run --rm lastfm-sync:latest --help

# Fetch scrobbles (local storage)
docker run --rm \
  -v ./data:/data \
  -e LASTFM_API_KEY=your-key \
  lastfm-sync:latest fetch --user alice

# Check output
ls -lh ./data/
cat ./data/alice.ndjson | head -5
```

### With Docker Compose

```bash
# Copy environment template
cp .env.example .env

# Edit .env and add your LASTFM_API_KEY
nano .env

# Start container
docker compose up

# Or run in background
docker compose up -d

# View logs
docker compose logs -f

# Stop and cleanup
docker compose down
```

---

## Multi-Stage Build Architecture

The Dockerfile uses a two-stage build to optimize size and security:

```
┌─────────────────────────────────────┐
│ Stage 1: Build (golang:1.25-alpine) │
│                                     │
│  1. Copy go.mod/go.sum              │
│  2. Download dependencies            │
│  3. Copy source code                │
│  4. Compile static binary           │
│     └→ /out/lastfm-sync             │
└─────────────────────────────────────┘
                ↓
┌─────────────────────────────────────┐
│ Stage 2: Runtime (distroless)       │
│                                     │
│  1. Copy binary from build stage    │
│  2. Set non-root user               │
│  3. Configure entrypoint            │
│     └→ Final image (~15-20MB)       │
└─────────────────────────────────────┘
```

### Why Multi-Stage?

| Aspect | Single-Stage (golang:alpine) | Multi-Stage (distroless) |
|--------|------------------------------|--------------------------|
| Image Size | ~300MB | ~15-20MB |
| Attack Surface | Shell, package manager, build tools | Binary only |
| Security | More potential vulnerabilities | Minimal vulnerability surface |
| Debugging | Can exec into container | Cannot exec (no shell) |
| Use Case | Development | Production |

**Recommendation**: Use multi-stage for production, consider single-stage for development if debugging is needed.

---

## Building the Image

### Basic Build

```bash
docker build -t lastfm-sync:latest .
```

### Build with Custom Tag

```bash
docker build -t lastfm-sync:v1.2.3 .
```

### Build with Version Information

```bash
docker build \
  --build-arg VERSION=v1.2.3 \
  --build-arg BUILD_TIME=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  -t lastfm-sync:v1.2.3 \
  .
```

### Build for ARM Architecture

```bash
# For ARM64 (Apple Silicon, AWS Graviton, Raspberry Pi 4)
docker build \
  --platform linux/arm64 \
  -t lastfm-sync:arm64 \
  .

# Multi-platform build (requires buildx)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t lastfm-sync:multiarch \
  .
```

### Build Performance Tips

1. **Use BuildKit** (faster, better caching):
   ```bash
   DOCKER_BUILDKIT=1 docker build -t lastfm-sync:latest .
   ```

2. **Skip test files** (faster builds):
   The `.dockerignore` file already excludes `*_test.go`

3. **Leverage layer caching**:
   - `go.mod` and `go.sum` are copied first
   - Dependencies downloaded before source code
   - Unchanged layers are reused

4. **Clean build** (no cache):
   ```bash
   docker build --no-cache -t lastfm-sync:latest .
   ```

### Verify Build

```bash
# Check image size
docker images lastfm-sync:latest

# Inspect image layers
docker history lastfm-sync:latest

# Check image metadata
docker inspect lastfm-sync:latest | jq '.[0].Config'

# Verify binary works
docker run --rm lastfm-sync:latest version
```

---

## Running Containers

### Basic Usage

```bash
# Show help
docker run --rm lastfm-sync:latest --help

# Show version
docker run --rm lastfm-sync:latest version
```

### Local Storage Mode

```bash
# Fetch scrobbles to local volume
docker run --rm \
  -v $(pwd)/data:/data \
  -e LASTFM_API_KEY=your-key \
  lastfm-sync:latest \
  fetch --user alice

# View output
cat ./data/alice.ndjson | jq .
```

**Important**: The `-v` flag mounts a host directory into the container at `/data`. Without this, data is lost when the container stops.

### Azure Storage Mode

```bash
# Using DefaultAzureCredential (Azure VM or AKS)
docker run --rm \
  -e LASTFM_API_KEY=your-key \
  -e AZURE_STORAGE_ACCOUNT=mystorageaccount \
  lastfm-sync:latest \
  fetch --user alice --output azure \
    --azure-container scrobbles --azure-auth default

# Using connection string
docker run --rm \
  -e LASTFM_API_KEY=your-key \
  -e AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=https;..." \
  lastfm-sync:latest \
  fetch --user alice --output azure \
    --azure-container scrobbles --azure-auth connstr
```

### Using Environment File

```bash
# Create .env file
cat > .env <<EOF
LASTFM_API_KEY=your-key
LASTFM_QPS=3
LASTFM_LOG_LEVEL=info
EOF

# Run with --env-file
docker run --rm \
  --env-file .env \
  -v $(pwd)/data:/data \
  lastfm-sync:latest \
  fetch --user alice
```

### Interactive Mode (Debugging)

**Note**: The distroless image has no shell, so you cannot exec into it. For debugging, use the build stage or switch to an alpine-based runtime temporarily.

```bash
# Alternative: Use alpine runtime for debugging
# Modify Dockerfile runtime stage to:
# FROM alpine:latest
# RUN apk add --no-cache ca-certificates
# ...

docker run --rm -it \
  --entrypoint /bin/sh \
  lastfm-sync:debug
```

### Named Container (Long-Running)

```bash
# Start container in background
docker run -d \
  --name lastfm-sync-daemon \
  --restart unless-stopped \
  -v $(pwd)/data:/data \
  -e LASTFM_API_KEY=your-key \
  lastfm-sync:latest \
  fetch --user alice

# View logs
docker logs -f lastfm-sync-daemon

# Stop container
docker stop lastfm-sync-daemon

# Remove container
docker rm lastfm-sync-daemon
```

---

## Docker Compose

Docker Compose provides a declarative way to run containers with complex configurations.

### docker-compose.yml Structure

```yaml
services:
  lastfm-sync:
    build: .
    env_file: .env
    volumes:
      - ./data:/data
    command: fetch --user ${LASTFM_USER:-alice}
```

### Usage

```bash
# Start (foreground)
docker compose up

# Start (background)
docker compose up -d

# View logs
docker compose logs -f

# Stop
docker compose down

# Rebuild and start
docker compose up --build

# Stop and remove volumes
docker compose down -v
```

### Customizing Commands

**Option 1**: Override in docker-compose.yml
```yaml
command: fetch --user bob --log-level debug
```

**Option 2**: Override at runtime
```bash
docker compose run --rm lastfm-sync fetch --user charlie --dry-run
```

**Option 3**: Use environment variables
```yaml
command: fetch --user ${LASTFM_USER} --qps ${LASTFM_QPS:-3}
```

### Development Workflow

```bash
# 1. Copy environment template
cp .env.example .env

# 2. Edit configuration
nano .env

# 3. Build and start
docker compose up --build

# 4. Make code changes
# ... edit Go files ...

# 5. Rebuild and restart
docker compose up --build

# 6. View logs
docker compose logs -f

# 7. Cleanup
docker compose down
```

### Compose Profiles (Advanced)

Run different services for dev vs prod:

```yaml
services:
  lastfm-sync-dev:
    profiles: ["dev"]
    build:
      target: build  # Stop at build stage for debugging
    entrypoint: /bin/sh

  lastfm-sync-prod:
    profiles: ["prod"]
    build: .
    restart: unless-stopped
```

```bash
# Run dev profile
docker compose --profile dev up

# Run prod profile
docker compose --profile prod up -d
```

---

## Volume Management

### Understanding Volumes

Docker volumes persist data outside the container lifecycle:

- **Host Mount** (`-v ./data:/data`): Map host directory to container
- **Named Volume** (`-v lastfm-data:/data`): Docker-managed persistent storage
- **Anonymous Volume**: Temporary, deleted with container

### Local Storage Requirements

When using `--output local`, mount a volume at `/data`:

```bash
docker run --rm \
  -v $(pwd)/data:/data \
  -e LASTFM_API_KEY=your-key \
  lastfm-sync:latest \
  fetch --user alice
```

**Output Files**:
- `/data/alice.ndjson` - Scrobbles
- `/data/state/alice.watermark` - Incremental sync state

### Permissions

The container runs as `nonroot:nonroot` (uid 65532):

```bash
# Create data directory with correct permissions
mkdir -p data/state
chmod 777 data  # Or set ownership to uid 65532

# Or use named volume (Docker handles permissions)
docker volume create lastfm-data
docker run --rm \
  -v lastfm-data:/data \
  -e LASTFM_API_KEY=your-key \
  lastfm-sync:latest \
  fetch --user alice
```

### Volume Commands

```bash
# List volumes
docker volume ls

# Inspect volume
docker volume inspect lastfm-data

# Remove volume
docker volume rm lastfm-data

# Remove all unused volumes
docker volume prune
```

---

## Environment Variables

See [docs/configuration.md](configuration.md) for complete reference.

### Required Variables

```bash
LASTFM_API_KEY=your-api-key
```

### Optional Variables

```bash
LASTFM_QPS=3
LASTFM_TIMEOUT=15s
LASTFM_LOG_LEVEL=info
```

### Azure Variables

```bash
AZURE_STORAGE_ACCOUNT=mystorageaccount
AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=..."
```

### Passing Variables

**Method 1**: Inline
```bash
docker run --rm \
  -e LASTFM_API_KEY=your-key \
  -e LASTFM_QPS=5 \
  lastfm-sync:latest fetch --user alice
```

**Method 2**: Environment file
```bash
docker run --rm --env-file .env lastfm-sync:latest fetch --user alice
```

**Method 3**: Docker Compose
```yaml
services:
  lastfm-sync:
    env_file: .env
    environment:
      - LASTFM_QPS=5
```

---

## Image Size Optimization

### Current Optimization

| Component | Size | Purpose |
|-----------|------|---------|
| Base image (distroless) | ~2MB | Runtime only |
| Go binary (stripped) | ~13MB | Application |
| Total | **~15MB** | Final image |

### Optimization Techniques Applied

1. **Multi-stage build**: Only binary in final image (not source code or build tools)
2. **Static linking** (`CGO_ENABLED=0`): No external dependencies
3. **Symbol stripping** (`-ldflags="-s -w"`): Removes debug symbols (~30% reduction)
4. **Distroless base**: Minimal runtime (vs ~7MB for alpine)
5. **.dockerignore**: Excludes unnecessary files from build context

### Further Optimization (Not Recommended)

```bash
# UPX compression (risky - may break binary)
upx --best --lzma /out/lastfm-sync
# Reduces size by 50-70% but slower startup and potential incompatibilities
```

### Image Size Comparison

| Base Image | Size | Shell | Package Manager | Use Case |
|------------|------|-------|-----------------|----------|
| golang:1.25 | ~300MB | ✅ | ✅ | Build only |
| golang:1.25-alpine | ~200MB | ✅ | ✅ | Build only |
| alpine:latest | ~7MB | ✅ | ✅ | Debug/dev runtime |
| distroless/static:nonroot | ~2MB | ❌ | ❌ | **Prod runtime** |
| scratch | ~0MB | ❌ | ❌ | No CA certs (breaks HTTPS) |

---

## Security

### Non-Root Execution

The container runs as `nonroot:nonroot` (uid 65532, gid 65532):

```bash
# Verify user
docker run --rm lastfm-sync:latest id
# (Note: distroless has no `id` command, but USER directive is enforced)

# Alternative verification
docker run --rm lastfm-sync:latest sh -c "whoami"
# (Note: distroless has no shell)
```

**Why non-root?**
- Limits impact of container escape vulnerabilities
- Required by many Kubernetes security policies
- Best practice for production deployments

### Minimal Attack Surface

**Distroless benefits**:
- No shell (`/bin/sh`, `/bin/bash`)
- No package manager (`apt`, `apk`)
- No unnecessary libraries
- Only essential runtime files
- CA certificates included (for HTTPS)

**Comparison with alpine**:

| Feature | alpine:latest | distroless/static:nonroot |
|---------|---------------|---------------------------|
| Shell | ✅ | ❌ |
| Package Manager | ✅ (apk) | ❌ |
| Debugging Tools | ✅ (wget, curl, etc.) | ❌ |
| CVE Surface | Higher | **Minimal** |
| Image Size | 7MB | 2MB |

### Secret Management

**Never build secrets into images**:

❌ **Bad**:
```dockerfile
ENV LASTFM_API_KEY=hardcoded-key
```

✅ **Good**:
```bash
docker run --rm \
  -e LASTFM_API_KEY=your-key \
  lastfm-sync:latest fetch --user alice
```

✅ **Better** (Docker Secrets):
```bash
echo "your-key" | docker secret create lastfm_api_key -
docker service create \
  --secret lastfm_api_key \
  --env LASTFM_API_KEY_FILE=/run/secrets/lastfm_api_key \
  lastfm-sync:latest
```

✅ **Best** (Azure Key Vault + Managed Identity):
See [docs/azure-deployment.md](azure-deployment.md) for production deployment patterns.

### Image Scanning

Scan for vulnerabilities:

```bash
# Docker Scout (built-in)
docker scout cves lastfm-sync:latest

# Trivy
trivy image lastfm-sync:latest

# Grype
grype lastfm-sync:latest
```

### Read-Only Filesystem

Run with read-only root filesystem (extra hardening):

```bash
docker run --rm \
  --read-only \
  -v $(pwd)/data:/data \
  -e LASTFM_API_KEY=your-key \
  lastfm-sync:latest fetch --user alice
```

---

## Troubleshooting

### "Permission denied" when writing to /data

**Cause**: Container runs as uid 65532, host directory owned by different user.

**Solution 1**: Make directory world-writable
```bash
chmod 777 data
```

**Solution 2**: Change ownership to uid 65532
```bash
sudo chown -R 65532:65532 data
```

**Solution 3**: Use named volume
```bash
docker volume create lastfm-data
docker run -v lastfm-data:/data ...
```

### "No such file or directory" when running binary

**Cause**: Missing dynamic library dependencies (if not statically linked).

**Solution**: Verify binary is static
```bash
# Build stage should have CGO_ENABLED=0
docker run --rm -it --entrypoint /bin/sh golang:1.25-alpine
ldd /out/lastfm-sync
# Should output: "not a dynamic executable"
```

### Image build fails with "go.mod not found"

**Cause**: `.dockerignore` is too aggressive or `go.mod` not in context.

**Solution**: Check `.dockerignore` doesn't exclude `go.mod`:
```bash
cat .dockerignore | grep -v "^#" | grep "go.mod"
# Should return nothing
```

### Container exits immediately

**Cause**: No command provided and CMD is `--help`.

**Solution**: Provide a command:
```bash
docker run --rm lastfm-sync:latest fetch --user alice
```

### "LASTFM_API_KEY is required"

**Cause**: Environment variable not passed to container.

**Solution**: Use `-e` flag or `--env-file`:
```bash
docker run --rm \
  -e LASTFM_API_KEY=your-key \
  lastfm-sync:latest fetch --user alice
```

### Cannot exec into container for debugging

**Cause**: Distroless image has no shell.

**Solution**: Use multi-stage build to debug:
```bash
# Build up to build stage only
docker build --target build -t lastfm-sync:debug .

# Exec into build stage
docker run --rm -it --entrypoint /bin/sh lastfm-sync:debug
```

### Image size is larger than expected

**Cause**: Build stage layers included in final image.

**Solution**: Verify multi-stage build:
```bash
docker history lastfm-sync:latest
# Should show only a few layers from distroless stage
```

### Docker Compose "service lastfm-sync didn't complete successfully"

**Cause**: Container exits after fetch completes (expected behavior).

**Solution**: This is normal for one-off commands. Use `--no-log-prefix` to see output:
```bash
docker compose up --no-log-prefix
```

Or run as one-shot:
```bash
docker compose run --rm lastfm-sync fetch --user alice
```

---

## Related Documentation

- [Configuration Reference](./configuration.md) - Environment variables and CLI flags
- [Azure Deployment](./azure-deployment.md) - Deploying to Azure Container Instances
- [Security Best Practices](./security.md) - Secure configuration management
- [Troubleshooting](./troubleshooting.md) - Common issues and solutions

---

**Last Updated**: 2026-01-06  
**Feature**: 002-containerization-documentation
