# Configuration Reference

Complete reference for all configuration options in LastFMReaderv3.

## Overview

LastFMReaderv3 uses a flexible configuration system with multiple sources. Configuration values are resolved in the following precedence order (highest to lowest):

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Configuration file** (`~/.lastfm/config.yaml`)
4. **Default values** (lowest priority)

This means if you set a value via a CLI flag, it will override any environment variable, config file, or default value.

## Quick Start

The simplest way to get started:

```bash
# Set your API key (required)
export LASTFM_API_KEY="your-api-key-here"

# Run a fetch
lastfm-sync fetch --user yourusername
```

For more complex setups, see the [Environment Variables](#environment-variables) and [CLI Flags](#cli-flags) sections below.

---

## Environment Variables

### Required Variables

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `LASTFM_API_KEY` | string | **Required**. Your Last.fm API key. Get one at https://www.last.fm/api/account/create | `1234567890abcdef1234567890abcdef` |

### Optional Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `LASTFM_QPS` | integer | `3` | Queries per second limit for Last.fm API. Respects Last.fm's rate limits (max: 5 QPS). |
| `LASTFM_TIMEOUT` | duration | `15s` | HTTP timeout for Last.fm API requests. Format: `15s`, `30s`, `1m`, etc. |
| `LASTFM_LOG_LEVEL` | string | `info` | Logging level. Options: `info`, `debug`. |
| `LASTFM_STATE` | string | `~/.lastfm` | Directory for storing state (watermarks, local output files). |
| `LASTFM_CONFIG` | string | none | Path to config file. If not set, searches `./`, `~/.lastfm/`, `/etc/lastfm/` for `config.yaml`. |
| `LASTFM_NO_PROGRESS` | boolean | `false` | Disable console progress bar. Useful for CI/CD pipelines or when redirecting output. |

### Azure Storage Variables

Required only when using `--output azure`:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `AZURE_STORAGE_ACCOUNT` | string | none | Azure Storage account name. Used with `--azure-auth default`, `--azure-auth mi`, or `--azure-auth key`. |
| `AZURE_STORAGE_CONNECTION_STRING` | string | none | **Sensitive**. Full Azure Storage connection string. Used with `--azure-auth connstr`. |
| `AZURE_STORAGE_ACCOUNT_KEY` | string | none | **Sensitive**. Azure Storage account key. Used with `--azure-auth key`. |

> **Security Note**: Never commit `AZURE_STORAGE_CONNECTION_STRING` or `AZURE_STORAGE_ACCOUNT_KEY` to version control. Use environment variables or Azure Key Vault.

---

## CLI Flags

### Fetch Command Flags

The `fetch` command supports the following flags:

#### User and Time Range

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--user` | string | **Yes** | none | Last.fm username to fetch scrobbles for. |
| `--since` | int64 | No | watermark | Unix timestamp (seconds) to start fetching from. If not set, uses watermark from last successful sync. |
| `--until` | int64 | No | now | Unix timestamp (seconds) to fetch until. Defaults to current time. |

#### Pagination

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--page-size` | integer | `200` | Number of scrobbles per API request page. Last.fm max: 200. |
| `--max-pages` | integer | `0` | Maximum pages to fetch. 0 = unlimited (fetch all available). |

#### Output Destination

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `local` | Output destination. Options: `local` (filesystem), `azure` (Azure Blob Storage). |
| `--out-path` | string | `~/.lastfm/{user}.ndjson` | Local output file path. Auto-set based on `--user` if not specified. |

#### Azure Blob Storage Options

Required when `--output azure`:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--azure-container` | string | none | **Required**. Azure Blob Storage container name. |
| `--azure-account` | string | `$AZURE_STORAGE_ACCOUNT` | Storage account name (required for `default`, `mi`, and `key` auth). |
| `--azure-prefix` | string | `lastfm/` | Blob prefix (folder path). Blobs written to `{prefix}dt=YYYY-MM-DD/{user}-YYYYMMDD-HHMMSS.ndjson`. |
| `--azure-auth` | string | `default` | Authentication method. Options: `default`, `mi`, `connstr`, `key`, `sas`. |
| `--azure-account-key` | string | none | **Sensitive**. Storage account key (required for `--azure-auth key`). |
| `--azure-container-url` | string | none | Full container URL with SAS token (for `--azure-auth sas`). |

**Azure Authentication Methods**:

- **`default`** (recommended for Azure VMs/AKS): Uses Azure SDK's DefaultAzureCredential chain (tries Managed Identity, Azure CLI, Environment, etc.).
- **`mi`**: Explicitly uses Managed Identity (Azure VMs, AKS, Container Instances with identity).
- **`connstr`**: Uses connection string from `AZURE_STORAGE_CONNECTION_STRING` environment variable.
- **`key`**: Uses storage account key from `--azure-account-key` flag or `AZURE_STORAGE_ACCOUNT_KEY` environment variable. Requires `--azure-account`.
- **`sas`**: Uses SAS token embedded in `--azure-container-url`.

#### Watermark Storage

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--watermark-store` | string | auto | Watermark storage backend. Options: `file`, `azure`. Auto-selects based on `--output`: `file` for local, `azure` for Azure. |
| `--state-path` | string | `~/.lastfm` | Directory for file-based watermarks (when `--watermark-store file`). |

#### Rate Limiting and Timeouts

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--qps` | integer | `3` | Queries per second for Last.fm API. Max: 5 (Last.fm limit). |
| `--timeout` | duration | `15s` | HTTP timeout for API requests. Format: `15s`, `30s`, `1m`. |

#### Logging and Debugging

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--log-level` | string | `info` | Logging verbosity. Options: `info`, `debug`. |
| `--no-progress` | boolean | `false` | Disable console progress bar. Useful for CI/CD pipelines or when redirecting output. |
| `--dry-run` | boolean | `false` | Preview mode. Fetches data but doesn't write to storage or update watermarks. |

---

## Configuration Precedence Examples

### Example 1: Environment Variable vs CLI Flag

```bash
# Set QPS via environment variable
export LASTFM_QPS=5

# CLI flag overrides the environment variable
lastfm-sync fetch --user alice --qps 2
# Result: Uses QPS=2 (CLI flag wins)
```

### Example 2: Config File vs Default

```yaml
# ~/.lastfm/config.yaml
qps: 5
page_size: 100
```

```bash
# Run without flags
lastfm-sync fetch --user alice
# Result: Uses qps=5, page_size=100 (from config file)

# Run with flag
lastfm-sync fetch --user alice --qps 1
# Result: Uses qps=1 (flag overrides config file), page_size=100 (from config file)
```

### Example 3: Full Precedence Chain

```yaml
# ~/.lastfm/config.yaml
qps: 4
```

```bash
export LASTFM_QPS=5
export LASTFM_API_KEY="mykey"

lastfm-sync fetch --user alice --qps 2
# Result: 
#   qps=2 (CLI flag beats env var and config file)
#   api_key="mykey" (env var, no flag or config file)
#   timeout=15s (default, no flag, env var, or config file)
```

---

## Configuration File

You can create a YAML configuration file at `~/.lastfm/config.yaml` to set default values:

```yaml
# ~/.lastfm/config.yaml
api_key: "your-api-key-here"
qps: 3
timeout: 15s
page_size: 200
output: local
state_path: ~/.lastfm
log_level: info

# Azure defaults (if using Azure Blob Storage)
azure_prefix: lastfm/
azure_auth: default
watermark_store: azure
```

**Search Paths** (in order):
1. Path specified by `LASTFM_CONFIG` environment variable
2. Current directory (`.`)
3. Home directory (`~/.lastfm/`)
4. System directory (`/etc/lastfm/`)

---

## Validation Rules

Configuration values are validated at startup. The following rules apply:

### Required Values

- **`LASTFM_API_KEY`** must be set (via any method: env var, config file, or flag).
- **`--user`** must be provided for the `fetch` command.
- **`--azure-container`** must be provided when `--output azure`.

### Validation Rules

| Field | Rule | Error Message |
|-------|------|---------------|
| `api_key` | Must not be empty | `LASTFM_API_KEY is required` |
| `qps` | Must be > 0 | `qps must be > 0, got {value}` |
| `timeout` | Must be > 0 | `timeout must be > 0, got {value}` |
| `output` | Must be `local` or `azure` | `output must be 'local' or 'azure', got {value}` |
| `watermark_store` | Must be `file` or `azure` | `watermark_store must be 'file' or 'azure', got {value}` |
| `log_level` | Must be `info` or `debug` | `log_level must be 'info' or 'debug', got {value}` |

### Azure-Specific Validation

When `--output azure`:
- Must provide `--azure-container`
- Must provide one of:
  - `--azure-account` (with auth method `default` or `mi`)
  - `AZURE_STORAGE_CONNECTION_STRING` (with auth method `connstr`)
  - `--azure-container-url` with SAS token (with auth method `sas`)

---

## Examples

### Local Development

```bash
export LASTFM_API_KEY="your-api-key"
lastfm-sync fetch --user alice
# Output: ~/.lastfm/alice.ndjson
```

### Custom Output Path

```bash
export LASTFM_API_KEY="your-api-key"
lastfm-sync fetch --user alice --out-path /data/scrobbles/alice.ndjson
```

### Azure Blob Storage with DefaultAzureCredential

```bash
export LASTFM_API_KEY="your-api-key"
lastfm-sync fetch --user alice \
  --output azure \
  --azure-container scrobbles \
  --azure-account mystorageaccount \
  --azure-auth default
# Output: Azure Blob at lastfm/alice/{year}/{month}/{timestamp}.ndjson
```

### Azure with Connection String

```bash
export LASTFM_API_KEY="your-api-key"
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=https;AccountName=..."
lastfm-sync fetch --user alice \
  --output azure \
  --azure-container scrobbles \
  --azure-auth connstr
```

### Azure with Managed Identity (in Container Instance)

```bash
export LASTFM_API_KEY="your-api-key"
lastfm-sync fetch --user alice \
  --output azure \
  --azure-container scrobbles \
  --azure-account mystorageaccount \
  --azure-auth mi
```

### Azure with Account Key

```bash
export LASTFM_API_KEY="your-api-key"
export AZURE_STORAGE_ACCOUNT_KEY="your-account-key"
lastfm-sync fetch --user alice \
  --output azure \
  --azure-container scrobbles \
  --azure-account mystorageaccount \
  --azure-auth key
# Or pass key directly via flag:
# --azure-account-key "your-account-key"
```

### Time Range Fetch

```bash
export LASTFM_API_KEY="your-api-key"
lastfm-sync fetch --user alice \
  --since 1704067200 \
  --until 1735689600
# Fetches scrobbles from 2024-01-01 to 2025-01-01
```

### Dry Run (Preview)

```bash
export LASTFM_API_KEY="your-api-key"
lastfm-sync fetch --user alice --dry-run
# Shows what would be fetched without writing anything
```

### Debug Mode

```bash
export LASTFM_API_KEY="your-api-key"
lastfm-sync fetch --user alice --log-level debug
# Enables verbose logging for troubleshooting
```

---

## Output Format

### NDJSON Structure

Each scrobble is written as a single line of JSON with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `username` | string | Last.fm username |
| `artist` | string | Artist name |
| `track` | string | Original track name from Last.fm (may include annotations) |
| `normalized_title` | string | Clean track title with annotations removed (Live, Remastered, featuring, etc.) |
| `album` | string | Album name |
| `uts` | int64 | Unix timestamp (seconds since epoch) |
| `local_time` | string | Human-readable UTC timestamp (RFC3339 format) |
| `mbid` | string | MusicBrainz ID (omitted if null) |
| `source` | string | Data source (always "lastfm") |
| `ingested_at` | string | UTC timestamp when record was created (RFC3339) |
| `raw` | object | Original Last.fm API response for debugging |

**Example:**
```json
{"username":"alice","artist":"The Beatles","track":"Come Together - Remastered","normalized_title":"Come Together","album":"Abbey Road","uts":1704067200,"local_time":"2024-01-01T00:00:00Z","source":"lastfm","ingested_at":"2024-01-06T14:30:22Z","raw":{}}
```

**Note on normalized_title**: This field is automatically generated by removing common annotations like:
- Remaster/Remastered annotations (e.g., "2009 Remaster", "Remastered 2015")
- Live performance markers (e.g., "Live at Wembley", "Live 1969")
- Version/edit labels (e.g., "Radio Edit", "Extended Version")
- Date/year markers in parentheses or brackets
- Remix labels (e.g., "Dave's Remix")
- Featuring/collaboration markers (e.g., "feat. Artist", "with Orchestra")

This normalization improves data quality for aggregation and matching while preserving the original title in the `track` field.

---

## Security Best Practices

1. **Never commit secrets to version control**:
   - Do not commit `.env` files
   - Do not commit `config.yaml` with API keys
   - Use `.gitignore` to exclude sensitive files

2. **Use environment variables for secrets**:
   - `LASTFM_API_KEY` should always be set via environment variable
   - `AZURE_STORAGE_CONNECTION_STRING` should never be in config files

3. **Azure authentication recommendations**:
   - **Production**: Use Managed Identity (`--azure-auth mi` or `default`)
   - **Development**: Use Azure CLI authentication (included in `default`)
   - **Avoid**: Connection strings in production (use Managed Identity instead)

4. **Config file permissions**:
   ```bash
   chmod 600 ~/.lastfm/config.yaml
   ```

5. **Use Azure Key Vault in production**:
   - Store `LASTFM_API_KEY` in Azure Key Vault
   - Use Managed Identity to access Key Vault
   - Inject secrets as environment variables in Container Instances

---

## Troubleshooting

### "LASTFM_API_KEY is required"

**Cause**: No API key configured.

**Solution**: Set the API key:
```bash
export LASTFM_API_KEY="your-key-here"
```

Or add to config file:
```yaml
# ~/.lastfm/config.yaml
api_key: "your-key-here"
```

### "output must be 'local' or 'azure'"

**Cause**: Invalid `--output` value.

**Solution**: Use `--output local` or `--output azure`:
```bash
lastfm-sync fetch --user alice --output local
```

### "qps must be > 0"

**Cause**: Invalid QPS value (negative or zero).

**Solution**: Set a positive integer:
```bash
lastfm-sync fetch --user alice --qps 3
```

### Azure authentication failures

**Cause**: Missing credentials or wrong auth method.

**Solutions**:
- For `--azure-auth default`: Run `az login` first
- For `--azure-auth connstr`: Set `AZURE_STORAGE_CONNECTION_STRING`
- For `--azure-auth mi`: Ensure Managed Identity is enabled and has Storage Blob Data Contributor role

---

## Related Documentation

- [Docker Setup](./docker.md) - Running in containers
- [Azure Deployment](./azure-deployment.md) - Deploying to Azure Container Instances
- [Security Best Practices](./security.md) - Secure configuration management
- [Troubleshooting](./troubleshooting.md) - Common issues and solutions

---

## Summary Table

### Configuration Precedence

| Priority | Source | Override Behavior |
|----------|--------|-------------------|
| 1 (Highest) | CLI Flags | Overrides all other sources |
| 2 | Environment Variables | Overrides config file and defaults |
| 3 | Config File | Overrides defaults |
| 4 (Lowest) | Defaults | Used if no other source provides value |

### Required Configuration

| What | How | Example |
|------|-----|---------|
| API Key | `export LASTFM_API_KEY="..."` | `export LASTFM_API_KEY="abc123"` |
| Username | `--user <username>` | `--user alice` |

### Optional Configuration (with defaults)

| What | How | Default |
|------|-----|---------|
| Rate Limit | `--qps <number>` or `export LASTFM_QPS=<number>` | `3` |
| Output Mode | `--output <local\|azure>` | `local` |
| Log Level | `--log-level <info\|debug>` | `info` |
| State Path | `--state-path <path>` or `export LASTFM_STATE=<path>` | `~/.lastfm` |

---

**Last Updated**: 2026-01-06  
**Feature**: 002-containerization-documentation
