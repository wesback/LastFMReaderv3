# Data Model: Configuration Schema

**Date**: 2026-01-06  
**Feature**: 002-containerization-documentation  
**Purpose**: Define configuration entities, validation rules, and relationships

---

## Overview

This feature is documentation-focused, so the "data model" represents the configuration schema that governs environment variables, Docker Compose settings, and Azure Container Instance parameters. These schemas ensure consistent configuration across development and production environments.

---

## Configuration Entities

### 1. Environment Variables

Core configuration loaded from environment or .env files.

#### Required Variables

| Variable | Type | Format | Description | Example |
|----------|------|--------|-------------|---------|
| `LASTFM_API_KEY` | string | 32-char hex | Last.fm API authentication key | `a1b2c3d4e5f6...` |
| `LASTFM_USER` | string | alphanumeric | Last.fm username to fetch scrobbles for | `alice` |

#### Optional Variables

| Variable | Type | Format | Default | Description | Example |
|----------|------|--------|---------|-------------|---------|
| `LASTFM_QPS` | integer | 1-10 | `3` | Rate limit (queries per second) | `3` |
| `LASTFM_TIMEOUT` | duration | Go duration | `10s` | API request timeout | `10s`, `1m` |
| `LASTFM_LOG` | string | enum | `info` | Log level (debug\|info\|warn\|error) | `debug` |
| `LASTFM_STATE` | path | absolute or ~ | `~/.lastfm-sync` | State directory for watermarks | `/data` |

#### Azure Blob Storage Variables (Conditional)

Required when `--output azure`:

| Variable | Type | Format | Description | Example |
|----------|------|--------|-------------|---------|
| `AZURE_STORAGE_ACCOUNT` | string | alphanumeric | Azure Storage account name | `mystorageacct` |
| `AZURE_STORAGE_CONTAINER` | string | lowercase-kebab | Container name for scrobbles | `scrobbles` |
| `AZURE_STORAGE_CONNECTION_STRING` | string | connection string | Full connection string (alternative to account name) | `DefaultEndpoints...` |
| `AZURE_STORAGE_SAS_TOKEN` | string | SAS token | Shared Access Signature (alternative auth) | `?sv=2021-06-08&...` |

---

### 2. Docker Compose Service Configuration

Structure for `docker-compose.yml`.

#### Service Definition

```yaml
version: '3.8'

services:
  lastfm-sync:
    build:
      context: .
      dockerfile: Dockerfile
    image: lastfm-sync:dev
    container_name: lastfm-sync
    volumes:
      - ./data:/data         # Bind mount for output
      - state:/app/.state    # Named volume for watermarks
    env_file:
      - .env                 # Load environment variables
    command: ["--help"]      # Default command (override at runtime)
    restart: "no"            # One-shot execution
```

#### Field Definitions

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| `services.lastfm-sync.build.context` | string | Yes | Build context path | Must be `.` (repo root) |
| `services.lastfm-sync.image` | string | Yes | Image tag | Format: `name:tag` |
| `services.lastfm-sync.volumes` | array | Yes | Volume mounts | At least one for /data |
| `services.lastfm-sync.env_file` | array | Yes | Environment files | Must include `.env` |
| `services.lastfm-sync.command` | array | No | Override CMD | Valid CLI args |

---

### 3. Azure Container Instance Configuration

Parameters for ACI deployment via `aci-params.json`.

#### Parameter Schema

```json
{
  "resourceGroup": "lastfm-sync-rg",
  "location": "eastus",
  "containerName": "lastfm-sync",
  "image": "myregistry.azurecr.io/lastfm-sync:latest",
  "cpu": 0.5,
  "memoryInGb": 1.0,
  "restartPolicy": "OnFailure",
  "environmentVariables": {
    "LASTFM_USER": "alice",
    "LASTFM_QPS": "3",
    "LASTFM_LOG": "info"
  },
  "secureEnvironmentVariables": {
    "LASTFM_API_KEY": "@keyvault(lastfm-api-key)"
  },
  "logAnalytics": {
    "workspaceId": "<workspace-id>",
    "workspaceKey": "<workspace-key>"
  }
}
```

#### Field Definitions

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| `resourceGroup` | string | Yes | Azure resource group name | 1-90 chars, alphanumeric + hyphens |
| `location` | string | Yes | Azure region | Valid Azure region (eastus, westus2, etc.) |
| `containerName` | string | Yes | Container instance name | Lowercase alphanumeric + hyphens |
| `image` | string | Yes | Container image URI | Valid Docker image reference |
| `cpu` | float | Yes | CPU cores | 0.5, 1.0, 2.0, 4.0 (ACI limits) |
| `memoryInGb` | float | Yes | Memory allocation | 0.5-16.0 (based on CPU tier) |
| `restartPolicy` | string | Yes | Restart behavior | `Always`, `OnFailure`, `Never` |
| `environmentVariables` | object | No | Non-sensitive env vars | Key-value pairs |
| `secureEnvironmentVariables` | object | No | Sensitive env vars | Key-value pairs (hidden in portal) |
| `logAnalytics.workspaceId` | string | No | Log Analytics workspace ID | GUID format |
| `logAnalytics.workspaceKey` | string | No | Log Analytics key | Base64 string |

---

## Validation Rules

### Environment Variables

1. **LASTFM_API_KEY**
   - Must be non-empty
   - Should be 32 characters (warning if not)
   - Hexadecimal format recommended

2. **LASTFM_USER**
   - Must be non-empty
   - Alphanumeric and underscores allowed
   - Case-sensitive

3. **LASTFM_QPS**
   - Integer range: 1-10
   - Default: 3 (respects Last.fm rate limits)

4. **LASTFM_TIMEOUT**
   - Valid Go duration string (e.g., "10s", "1m30s")
   - Minimum: 1s (enforced by config.Validate())

5. **LASTFM_LOG**
   - Enum: `debug`, `info`, `warn`, `error`
   - Case-insensitive

6. **LASTFM_STATE**
   - Valid file path
   - Directory must be writable
   - Supports `~` expansion

7. **Azure Variables** (when used)
   - `AZURE_STORAGE_ACCOUNT`: 3-24 lowercase alphanumeric
   - `AZURE_STORAGE_CONTAINER`: 3-63 lowercase alphanumeric + hyphens
   - Connection string or SAS token (mutually exclusive with account name + access key)

### Docker Compose

1. **Service Name**
   - Must be `lastfm-sync` (for consistency)

2. **Volumes**
   - At least one bind mount for `/data` (output directory)
   - Optional named volume for state

3. **Environment File**
   - `.env` must exist (created from `.env.example`)

4. **Image Tag**
   - Format: `name:tag`
   - Tag should be `dev` for local development

### Azure Container Instance

1. **Resource Limits**
   - CPU/Memory combinations must be valid per [ACI SKU matrix](https://learn.microsoft.com/en-us/azure/container-instances/container-instances-resource-and-quota-limits)
   - Example: 0.5 CPU supports up to 3.5 GB memory

2. **Restart Policy**
   - `OnFailure` recommended for CLI tools (exit code 0 = success, no restart)
   - `Never` for one-time jobs
   - `Always` for daemons (not applicable here)

3. **Log Analytics**
   - Both workspace ID and key required (or neither)
   - Workspace must exist in same/linked subscription

4. **Secure Environment Variables**
   - Any variable containing secrets (API keys, connection strings)
   - Hidden in Azure Portal after creation
   - Can reference Key Vault: `@keyvault(secret-name)`

---

## Relationships

### Configuration Precedence

```
CLI Flags (highest)
    ↓
Environment Variables
    ↓
Config File (~/.lastfm/config.yaml)
    ↓
Defaults (lowest)
```

### Environment Mapping

| Source | Development | Production |
|--------|-------------|------------|
| **Storage** | .env file | Azure Key Vault |
| **Injection** | Docker Compose | ACI secure-environment-variables |
| **Access** | Plain environment variables | Managed Identity → Key Vault |

### State Management

```
Configuration (load-time)
    ↓
Application Execution (runtime)
    ↓
Watermark Persistence (file or Azure Blob)
```

Configuration is stateless (no state transitions). Watermark state is managed separately by the application (see internal/watermark package).

---

## Configuration Profiles

### Development Profile

```bash
# .env (development)
LASTFM_API_KEY=test_key_32chars_xxxxxxxxxxx
LASTFM_USER=testuser
LASTFM_QPS=1           # Lower rate for testing
LASTFM_LOG=debug       # Verbose logging
LASTFM_STATE=./data    # Local directory
```

**Usage**: `docker compose run lastfm-sync fetch --user testuser`

### Production Profile

```json
// aci-params.json (production)
{
  "environmentVariables": {
    "LASTFM_USER": "alice",
    "LASTFM_QPS": "3",
    "LASTFM_LOG": "info",
    "LASTFM_STATE": "/data"
  },
  "secureEnvironmentVariables": {
    "LASTFM_API_KEY": "@keyvault(lastfm-api-key)"
  }
}
```

**Usage**: `./azure/deploy-aci.sh`

---

## Schema Versioning

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-01-06 | Initial schema definition |

Future changes will bump version and document migrations here.

---

## Validation Implementation

### Application-Level (Go)

Validation occurs in `internal/config/types.go`:

```go
func (c *Config) Validate() error {
    if c.APIKey == "" {
        return fmt.Errorf("LASTFM_API_KEY is required")
    }
    // ... additional checks
}
```

### Docker Compose Validation

Use `docker compose config` to validate syntax:

```bash
docker compose config --quiet  # Exit code 0 = valid
```

### Azure Deployment Validation

Use Azure CLI validation:

```bash
az container create --dry-run --validate ...  # Validates without deploying
```

---

## Security Considerations

1. **Never commit .env files** (in .gitignore)
2. **Use secure-environment-variables in ACI** (hidden in portal)
3. **Prefer Key Vault references** over inline secrets in production
4. **Rotate secrets regularly** (Key Vault supports versioning)
5. **Principle of least privilege**: Grant only required permissions to managed identities

---

## Next Steps

1. Generate JSON Schema files in `contracts/` directory
2. Create validation examples in documentation
3. Generate quickstart.md with configuration examples
4. Update agent context with configuration patterns
