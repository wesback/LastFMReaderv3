# CLI Contract: Normalize Command

**Feature**: 007-normalize-command  
**Date**: 2026-01-08  
**Type**: Command-Line Interface

## Command Signature

```bash
lastfm-sync normalize --user <username> [options]
```

## Required Arguments

| Argument | Type | Description | Validation |
|----------|------|-------------|------------|
| `--user` | string | Username whose files to process | MUST be non-empty (FR-018) |

## Optional Arguments - Storage Selection

### Local Storage (default)
No additional arguments required. Uses current working directory or configured base path.

### Azure Blob Storage
When ANY Azure argument is provided, Azure mode is activated (FR-003).

| Argument | Type | Description | Validation |
|----------|------|-------------|------------|
| `--azure-account` | string | Azure storage account name | Required when Azure mode active (FR-019) |
| `--azure-container` | string | Azure container name | Required when Azure mode active (FR-019) |
| `--azure-prefix` | string | Blob prefix for file discovery | Optional, default: "" |
| `--azure-auth` | string | Authentication method: "key", "sas", "default" | Optional, default: "default" |
| `--azure-account-key` | string | Storage account key (if auth=key) | Required when auth=key |
| `--azure-sas-token` | string | SAS token (if auth=sas) | Required when auth=sas |

**Validation**: Azure mode requires `--azure-account` AND `--azure-container` at minimum (FR-019).

## Optional Arguments - Behavior

| Argument | Type | Description | Default |
|----------|------|-------------|---------|
| `--dry-run` | boolean | Preview changes without writing | false |
| `--log-level` | string | Logging level: debug, info, warn, error | info |

## Output Format

### Progress Display (stdout)

Per-file progress (FR-009):
```
Processing files for user: {username}
Storage: {Local: /path/to/data | Azure: account/container}

Processing: episode_001.ndjson [1/150]
  Current: "Track #1 - Some Title"
  New:     "track 1 some title"
  Status:  {Updated | No change needed}

Processing: episode_002.ndjson [2/150]
  Status:  No change needed

...
```

### Summary Report (stdout)

```
Summary:
  Total files: 150
  Updated: 45
  Unchanged: 105
  Errors: 0
  Duration: 2.3s

{if --dry-run}
Dry-run mode: No changes written to storage
{/if}

{if errors}
Errors encountered:
  - username_042.ndjson: parse error
  - username_099.ndjson: permission denied
{/if}
```

### Error Output (stderr)

Individual file errors logged at DEBUG level, summary errors at INFO level.

## Exit Codes

| Code | Meaning | Scenario |
|------|---------|----------|
| 0 | Success | All files processed (some may have errors but processing completed) |
| 1 | General error | Unexpected error during command execution |
| 2 | Validation error | Missing required arguments or invalid configuration (FR-018, FR-019) |

**Note**: File-level errors (parse errors, missing fields) do NOT cause non-zero exit code per FR-011 (continue processing). Only configuration/validation errors exit early.

## Examples

### Local Storage

```bash
# Basic usage - normalize all files for user
./lastfm-sync normalize --user john_doe

# Dry-run preview
./lastfm-sync normalize --user john_doe --dry-run

# With debug logging
./lastfm-sync normalize --user john_doe --log-level debug
```

### Azure Storage

```bash
# Azure with default authentication (managed identity/Azure CLI)
./lastfm-sync normalize --user jane_doe \
  --azure-account myaccount \
  --azure-container scrobbles

# Azure with account key
./lastfm-sync normalize --user jane_doe \
  --azure-account myaccount \
  --azure-container scrobbles \
  --azure-auth key \
  --azure-account-key "abc123..."

# Azure dry-run
./lastfm-sync normalize --user jane_doe \
  --azure-account myaccount \
  --azure-container scrobbles \
  --dry-run
```

## File Discovery Pattern

Matching pattern: `{username}_*.ndjson` (FR-004)

### Local Storage
- Search in configured base directory
- Example: If base is `/data`, search `/data/john_doe_*.ndjson`
- Recursive search NOT performed (flat directory expected)

### Azure Storage
- List blobs with prefix `{azure-prefix}{username}_`
- Example: If prefix is `lastfm/`, search blobs like `lastfm/john_doe_*.ndjson`
- Blob name extraction: Use blob name as displayed filename

## Alignment with Existing Commands

**Follows patterns from**:
- `fetch` command: Azure argument structure, authentication modes
- `merge` command: Progress display, NDJSON processing, summary format

**Consistency**:
- Same Azure flag names and behavior (FR-014)
- Same logging configuration
- Same progress library usage
- Same error handling patterns

## Non-Functional Contracts

- **Performance**: Process ≥1000 files in under 5 seconds (SC-001)
- **Memory**: Streaming processing, O(1) memory per file
- **Reliability**: Continue on individual file errors (FR-011, SC-005)
- **Safety**: Dry-run produces accurate preview (SC-004)
- **Idempotency**: Running multiple times produces same result (normalized_title stabilizes)
