# Research: Containerization Documentation and Configuration Management

**Date**: 2026-01-06  
**Feature**: 002-containerization-documentation  
**Purpose**: Research best practices for Docker, Docker Compose, Azure Container Instances, and configuration management

---

## 1. Docker Multi-Stage Builds for Go

### Decision
Use **golang:1.25-alpine** for build stage and **gcr.io/distroless/static:nonroot** for runtime stage.

### Rationale
- **Build Stage (Alpine)**: Small base image (~5MB), includes build tools, excellent layer caching
- **Runtime Stage (Distroless)**: Minimal attack surface (no shell, no package manager), ~2MB base, runs as non-root by default
- **Go Static Binary**: CGO_ENABLED=0 produces fully static binary compatible with distroless/static
- **Security**: Non-root user, no unnecessary utilities, minimal CVE exposure
- **Size**: Final image ~15-20MB (binary + distroless base)

### Alternatives Considered

| Alternative | Why Rejected |
|-------------|--------------|
| **scratch** | No CA certificates bundle for HTTPS calls to Last.fm API; would require manual cert copying |
| **debian:slim** | 10x larger (~100MB), includes unnecessary utilities, larger attack surface |
| **alpine runtime** | Requires musl libc; distroless is smaller and more secure |
| **Single-stage build** | 10x larger final image (~800MB with build tools), security risk |

### Implementation Notes
- Set `CGO_ENABLED=0` for static compilation
- Use `GOOS=linux GOARCH=amd64` for explicit platform targeting
- Inject version/build time via `-ldflags` at build time
- Copy only binary to runtime stage (no intermediate artifacts)
- Use `VOLUME ["/data"]` for persistent output directory

### Best Practices Applied
1. **Layer Ordering**: COPY go.mod/go.sum before COPY . to cache dependencies
2. **Build Flags**: `-ldflags="-s -w"` strips debug symbols (smaller binary)
3. **Non-Root User**: distroless:nonroot runs as UID 65532 (no privileged execution)
4. **Labels**: OCI image labels for metadata (source repo, description)
5. **Entrypoint vs CMD**: ENTRYPOINT for binary, CMD for default args (overrideable)

---

## 2. Docker Compose for Development

### Decision
Create **minimal docker-compose.yml** with volume mounts for local data persistence and .env file support.

### Rationale
- **Simplicity**: Single-service compose (no databases/dependencies for this CLI tool)
- **Environment Parity**: Development environment matches production container
- **Configuration**: .env file loaded automatically by Docker Compose
- **Volume Mounts**: Persist output and state files between runs
- **Fast Iteration**: No need for image rebuild during config changes

### Alternatives Considered

| Alternative | Why Rejected |
|-------------|--------------|
| **Hot Reload** | CLI tool is stateless with single execution model; not a long-running service |
| **Dev Dockerfile** | Unnecessary complexity; production Dockerfile works for development |
| **Kubernetes/Helm** | Overkill for single CLI tool; adds complexity without benefit |
| **Bare Docker Run** | Harder to document/maintain multi-flag commands vs compose |

### Compose Structure

```yaml
version: '3.8'
services:
  lastfm-sync:
    build:
      context: .
      dockerfile: Dockerfile
    image: lastfm-sync:dev
    volumes:
      - ./data:/data  # Local data directory mapped to container /data
    env_file:
      - .env        # Load environment variables from .env file
    command: fetch --user ${LASTFM_USER}  # Override CMD, use env var
```

### Development Workflow
1. **Initial Setup**: `cp .env.example .env && docker compose build`
2. **Run Fetch**: `docker compose run --rm lastfm-sync fetch --user myuser`
3. **View Logs**: `docker compose logs -f`
4. **Clean Up**: `docker compose down -v`

### Best Practices Applied
1. **Named Volumes**: Use bind mounts for development, named volumes for persistence
2. **env_file**: Separate secrets from compose file
3. **Service Name**: Match binary name for clarity
4. **Build Context**: Root directory to access all source files
5. **Command Override**: Allow easy parameter passing via compose run

---

## 3. Azure Container Instances Deployment

### Decision
Use **Azure CLI script-based deployment** with parameter file for configuration.

### Rationale
- **Scriptable**: Bash script can be version-controlled, tested, and automated
- **Flexible**: Easy to modify parameters without rewriting templates
- **Reproducible**: Script ensures consistent deployments across environments
- **Simple**: Less complex than ARM/Bicep for single container deployment
- **Debuggable**: Clear error messages from Azure CLI

### Alternatives Considered

| Alternative | Why Rejected |
|-------------|--------------|
| **Azure Portal** | Not reproducible, no version control, manual process prone to errors |
| **ARM Templates** | Verbose JSON syntax, harder to maintain for simple single-container deployment |
| **Bicep** | Better than ARM but overkill for single container; adds tooling dependency |
| **Terraform** | Adds state management complexity; unnecessary for simple ACI deployment |
| **AKS/Kubernetes** | Out of scope (mentioned in feature spec), too complex for single CLI tool |

### Deployment Script Structure

```bash
#!/usr/bin/env bash
# azure/deploy-aci.sh

set -euo pipefail

# Load parameters from aci-params.json
RESOURCE_GROUP=$(jq -r '.resourceGroup' azure/aci-params.json)
CONTAINER_NAME=$(jq -r '.containerName' azure/aci-params.json)
IMAGE=$(jq -r '.image' azure/aci-params.json)
CPU=$(jq -r '.cpu' azure/aci-params.json)
MEMORY=$(jq -r '.memory' azure/aci-params.json)

# Create resource group if not exists
az group create --name "$RESOURCE_GROUP" --location eastus

# Deploy container instance
az container create \
  --resource-group "$RESOURCE_GROUP" \
  --name "$CONTAINER_NAME" \
  --image "$IMAGE" \
  --cpu "$CPU" \
  --memory "$MEMORY" \
  --environment-variables \
    LASTFM_USER="$LASTFM_USER" \
    LASTFM_QPS=3 \
  --secure-environment-variables \
    LASTFM_API_KEY="$LASTFM_API_KEY" \
  --restart-policy OnFailure \
  --log-analytics-workspace "$LOG_ANALYTICS_WORKSPACE_ID" \
  --log-analytics-workspace-key "$LOG_ANALYTICS_WORKSPACE_KEY"
```

### ACI Best Practices Applied
1. **Resource Groups**: Isolate resources for easy cleanup
2. **Restart Policy**: OnFailure for CLI tools (one-shot execution)
3. **Secure Environment Variables**: Separate sensitive data (API keys) from regular config
4. **Logging**: Integrate with Azure Monitor (Log Analytics) for centralized logging
5. **Managed Identity**: Use for production (authenticate to Azure services without keys)

### Networking Considerations
- **Public IP**: Default for simple deployments (acceptable for stateless CLI)
- **VNet Integration**: Document as optional for secure production scenarios
- **Ingress**: Not needed (CLI tool, no web interface)

### Cost Optimization
- **CPU/Memory**: Start with minimal (0.5 CPU, 1GB RAM), scale up if needed
- **Restart Policy**: OnFailure prevents infinite loop costs
- **Cleanup**: Document auto-deletion after successful run (optional)

---

## 4. Environment Variable Management Patterns

### Decision
Use **.env.example as template**, load .env for local development, use native environment variables for production.

### Rationale
- **Industry Standard**: .env pattern widely recognized (Docker Compose, dotenv libraries)
- **Security**: .env in .gitignore prevents secret commits
- **Flexibility**: Template shows all options without exposing values
- **Docker Native**: Docker Compose automatically loads .env files
- **Documentation**: Comments in .env.example explain each variable

### Alternatives Considered

| Alternative | Why Rejected |
|-------------|--------------|
| **Config Files (YAML/TOML)** | Harder to secure (must encrypt secrets), less Docker-native |
| **Command-Line Flags Only** | Poor UX for many parameters, no defaults, harder to document |
| **Hardcoded Defaults** | Inflexible, requires code changes for different environments |
| **Multiple .env Files** | (.env.dev, .env.prod) Complexity without benefit for single-user tool |

### .env.example Structure

```bash
# Last.fm API Configuration
# Required: Obtain from https://www.last.fm/api/account/create
LASTFM_API_KEY=your_api_key_here

# Required: Last.fm username to fetch scrobbles for
LASTFM_USER=your_username

# Optional: Rate limit (requests per second). Default: 3
# LASTFM_QPS=3

# Optional: API request timeout. Default: 10s
# LASTFM_TIMEOUT=10s

# Optional: Log level (debug|info|warn|error). Default: info
# LASTFM_LOG=info

# Optional: State directory for watermarks. Default: ~/.lastfm-sync
# LASTFM_STATE=/data

# Azure Blob Storage (Optional - required if using --output azure)
# AZURE_STORAGE_ACCOUNT=mystorageaccount
# AZURE_STORAGE_CONTAINER=scrobbles
# AZURE_STORAGE_CONNECTION_STRING=DefaultEndpointsProtocol=https;...
```

### Configuration Precedence

Documented order (highest to lowest priority):
1. **Command-line flags** (--user, --api-key, etc.)
2. **Environment variables** (LASTFM_USER, LASTFM_API_KEY, etc.)
3. **Config file** (~/.lastfm/config.yaml - if implemented)
4. **Defaults** (hardcoded in config/defaults.go)

### Security Best Practices
1. **Never commit .env**: Always in .gitignore
2. **Example file only**: .env.example has placeholders, no real values
3. **Sensitive marker**: Clearly label sensitive variables (API keys, connection strings)
4. **Production secrets**: Use Azure Key Vault, not .env files
5. **Validation**: Fail fast on missing required variables

---

## 5. Azure Key Vault Integration for Containers

### Decision
Use **Azure Key Vault with Managed Identity** for production secret management, document environment variable fallback for development.

### Rationale
- **Security**: Secrets never in container environment variables (ACI supports Key Vault references)
- **Auditing**: Azure Key Vault logs all secret access
- **Rotation**: Centralized secret rotation without container redeployment
- **Compliance**: Meets enterprise security requirements
- **Managed Identity**: No credentials needed in container (Azure handles authentication)

### Alternatives Considered

| Alternative | Why Rejected |
|-------------|--------------|
| **Environment Variables Only** | Secrets visible in Azure Portal, no audit trail, manual rotation |
| **Container Registry Secrets** | Limited to registry auth, not for application secrets |
| **Azure App Configuration** | Better for non-secret config; Key Vault specialized for secrets |
| **Secrets in Code** | Unacceptable security risk, violates best practices |

### Key Vault Integration Pattern

#### Setup (One-Time)
```bash
# Create Key Vault
az keyvault create --name lastfm-sync-kv --resource-group lastfm-sync-rg

# Store secrets
az keyvault secret set --vault-name lastfm-sync-kv --name lastfm-api-key --value "$LASTFM_API_KEY"

# Enable Managed Identity for ACI (during container create)
az container create \
  --resource-group lastfm-sync-rg \
  --name lastfm-sync \
  --assign-identity \
  --secrets \
    LASTFM_API_KEY=@Microsoft.KeyVault(SecretUri=https://lastfm-sync-kv.vault.azure.net/secrets/lastfm-api-key/)
```

#### Application Code (No Changes Needed)
Application reads from environment variables as usual. ACI injects Key Vault values transparently.

### Key Vault Best Practices
1. **Managed Identity**: Preferred over service principals (no credential management)
2. **RBAC**: Grant "Key Vault Secrets User" role to container identity
3. **Secret Versioning**: Use latest version by default, pin for stability
4. **Access Policies**: Restrict to specific secrets (principle of least privilege)
5. **Soft Delete**: Enable for secret recovery (30-90 day retention)

### Development vs Production

| Environment | Secret Source | Rationale |
|-------------|---------------|-----------|
| **Development** | .env file | Fast iteration, no Azure dependencies |
| **CI/CD** | GitHub Secrets | Secure pipeline execution |
| **Production** | Azure Key Vault | Enterprise security, compliance, auditing |

### Secret Rotation Strategy
1. **Create new version** in Key Vault
2. **Test with canary** deployment
3. **Update all containers** (ACI auto-fetches latest)
4. **Deprecate old version** after 30 days
5. **Monitor logs** for authentication failures

---

## Summary of Decisions

| Component | Decision | Key Benefit |
|-----------|----------|-------------|
| **Docker Build** | golang:alpine → distroless:static | Security + minimal size |
| **Docker Compose** | Minimal single-service with volumes | Simple local development |
| **ACI Deployment** | Azure CLI script + parameter file | Reproducible, scriptable |
| **Configuration** | .env.example template + env vars | Industry standard, secure |
| **Secret Management** | Azure Key Vault + Managed Identity | Enterprise-grade security |

---

## Open Questions (Resolved)

All research questions have been answered:
- ✅ Optimal Docker base images selected
- ✅ Docker Compose patterns defined
- ✅ Azure deployment strategy chosen
- ✅ Configuration management approach established
- ✅ Secret management patterns documented

---

## Next Steps

1. Generate data-model.md (configuration schema)
2. Generate contracts/ (env-schema.json, compose-schema.yaml, aci-params-schema.json)
3. Generate quickstart.md (5-minute getting started)
4. Update agent context
5. Proceed to task breakdown (Phase 2)
