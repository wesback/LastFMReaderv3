# lastfm-sync: Last.fm Scrobble Sync CLI

![Test Status](https://img.shields.io/badge/tests-105%20passing-brightgreen)
![Go Version](https://img.shields.io/badge/go-1.24%2B-blue)

**lastfm-sync** is a production-ready CLI tool for exporting Last.fm scrobble history with incremental sync capabilities. It supports local NDJSON output and Azure Blob Storage with crash-safe resumption, rate limit compliance, and comprehensive error handling.

## Features

✅ **Local & Cloud Storage**: Write to local files or Azure Blob Storage with time-partitioned paths  
✅ **Incremental Sync**: Automatic watermarking to resume from last sync point  
✅ **Rate Limit Compliant**: 3 QPS throttling with Retry-After header support  
✅ **Production Ready**: Exponential backoff, timeout handling, structured logging  
✅ **Dry-Run Mode**: Test configurations without consuming API quota  
✅ **Azure Integration**: DefaultAzureCredential, managed identity, SAS tokens, connection strings, account keys  
✅ **Secret Redaction**: Automatic credential masking in logs  
✅ **NDJSON Output**: Newline-delimited JSON for easy streaming and processing  
✅ **Console Progress Bar**: Real-time progress tracking with ETA, speed, and scrobble count  
✅ **Title Normalization**: Automatic removal of annotations (Live, Remastered, Featuring) for better data quality

## Quick Start

### Installation

**From Source:**
```bash
git clone https://github.com/lastfm-reader/lastfm-sync.git
cd lastfm-sync
make build
./dist/lastfm-sync --help
```

**Using Docker:**
```bash
docker build -t lastfm-sync .
docker run --rm lastfm-sync --help
```

**Using Docker Compose:**
```bash
cp .env.example .env  # Edit and add your LASTFM_API_KEY
docker compose up

# Or use the helper script
./scripts/dev-up.sh
```

**Using Go Install:**
```bash
go install github.com/lastfm-reader/lastfm-sync/cmd/lastfm-sync@latest
```

> **📦 Container Documentation**: See [docs/docker.md](docs/docker.md) for complete Docker and Docker Compose documentation.
>
> **☁️ Azure Deployment**: See [docs/azure-deployment.md](docs/azure-deployment.md) for deploying to Azure Container Instances.

### Prerequisites

- Go 1.24+ (for building from source)
- Last.fm API key ([get one here](https://www.last.fm/api/account/create))
- Azure Storage account (optional, for cloud storage)

### Basic Usage

**Fetch scrobbles to local file:**
```bash
export LASTFM_API_KEY="your-api-key"
lastfm-sync fetch --user alice --output local --out-path ./alice-scrobbles.ndjson
```

**Incremental sync (continues from last position):**
```bash
# First run fetches all history
lastfm-sync fetch --user alice --output local

# Subsequent runs only fetch new scrobbles
lastfm-sync fetch --user alice --output local
```

**Fetch specific time range:**
```bash
# Unix timestamps
lastfm-sync fetch --user alice --since 1704067200 --until 1735689600
```

**Dry-run mode (preview without side effects):**
```bash
lastfm-sync fetch --user alice --dry-run
```

## Configuration

> **📘 Complete Configuration Documentation**: See [docs/configuration.md](docs/configuration.md) for comprehensive configuration reference including all environment variables, CLI flags, configuration precedence, validation rules, and examples.

### Quick Configuration Reference

**Required:**
```bash
export LASTFM_API_KEY="your-api-key"  # Get one at https://www.last.fm/api/account/create
```

**Optional (with defaults):**
```bash
export LASTFM_QPS="3"                    # Queries per second (default: 3)
export LASTFM_TIMEOUT="15s"              # HTTP timeout (default: 15s)
export LASTFM_LOG_LEVEL="info"           # Log level: info or debug
export LASTFM_STATE="~/.lastfm"          # State directory for watermarks
```

**Azure Storage (optional):**
```bash
export AZURE_STORAGE_ACCOUNT="mystorageaccount"
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=https;..."
```

### Using .env Files

Copy the example environment file and edit it:

```bash
cp .env.example .env
nano .env  # Add your LASTFM_API_KEY
```

### Configuration Precedence

Configuration values are resolved in this order (highest to lowest):
1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Config file** (`~/.lastfm/config.yaml`)
4. **Defaults** (lowest priority)

### Command-Line Flags

```bash
lastfm-sync fetch [flags]

Flags:
  -u, --user string              Last.fm username (REQUIRED)
      --api-key string           Last.fm API key (or LASTFM_API_KEY env var)
      --since int                Fetch from this unix timestamp (default: use watermark)
      --until int                Fetch until this unix timestamp (default: now)
      --page-size int            Page size 1-200 (default: 200)
      --max-pages int            Max pages to fetch, 0=unlimited (default: 0)
      
  Output Options:
      --output string            Output destination: local or azure (default: "local")
      --out-path string          Local output path (default: ~/.lastfm/{user}.ndjson)
      
  Azure Options:
      --azure-container string   Azure container name
      --azure-prefix string      Azure blob prefix (default: "lastfm/")
      --azure-auth string        Auth method: default, mi, connstr, key, sas (default: "default")
      --azure-account string     Azure storage account name
      --azure-account-key string Azure storage account key (for key auth)
      --azure-container-url string  Full container URL (for SAS)
      
  Watermark Options:
      --watermark-store string   Watermark store: file or azure (default: auto-detect)
      --state-path string        State directory (default: ~/.lastfm/state)
      
  Performance Options:
      --qps int                  Queries per second (default: 3)
      --timeout duration         Request timeout (default: 15s)
      
  Other Options:
      --dry-run                  Preview only, no API calls or writes
      --no-progress              Disable progress bar (useful for CI/CD)
      --log-level string         Log level: info or debug (default: "info")
  -h, --help                     Help for fetch
```

## Usage Examples

### Local Storage

**Basic fetch:**
```bash
lastfm-sync fetch --user alice
# Output: ~/.lastfm/alice.ndjson
# Watermark: ~/.lastfm/state/alice.watermark
```

**Custom output path:**
```bash
lastfm-sync fetch --user alice --out-path /data/scrobbles/alice.ndjson
```

**Fetch last 30 days:**
```bash
SINCE=$(date -d '30 days ago' +%s)
lastfm-sync fetch --user alice --since $SINCE
```

### Azure Blob Storage

**Using DefaultAzureCredential (recommended for Azure VMs/AKS):**
```bash
lastfm-sync fetch \
  --user alice \
  --output azure \
  --azure-container scrobbles \
  --azure-account mystorageaccount \
  --azure-auth default
# Output: dt=2026-01-06/alice-20260106-143022.ndjson
# Watermark: alice.watermark
```

**Using connection string:**
```bash
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=https;AccountName=..."
lastfm-sync fetch \
  --user alice \
  --output azure \
  --azure-container scrobbles \
  --azure-auth connstr
```

**Using SAS token:**
```bash
lastfm-sync fetch \
  --user alice \
  --output azure \
  --azure-container-url "https://account.blob.core.windows.net/scrobbles?sv=2021..." \
  --azure-auth sas
```

**Using managed identity:**
```bash
lastfm-sync fetch \
  --user alice \
  --output azure \
  --azure-container scrobbles \
  --azure-account mystorageaccount \
  --azure-auth mi
```

**Using account key:**
```bash
lastfm-sync fetch \
  --user alice \
  --output azure \
  --azure-container scrobbles \
  --azure-account mystorageaccount \
  --azure-account-key "your-account-key" \
  --azure-auth key
```

### Advanced Usage

**Debug logging:**
```bash
lastfm-sync fetch --user alice --log-level debug
# Shows detailed fetch.page.params, rate limiting, watermark updates
```

**High-volume sync (adjust rate limit):**
```bash
lastfm-sync fetch --user alice --qps 5 --timeout 30s --max-pages 100
```

**Dry-run preview:**
```bash
lastfm-sync fetch --user alice --output azure --azure-container test --dry-run
# Shows what would be fetched and written without consuming resources
```

### Merge Command

The `merge` command consolidates multiple NDJSON scrobble files into a single deduplicated output for a specific user. This is useful for:
- Combining exports from different time periods
- Merging data from multiple sources
- Deduplicating scrobble history
- Creating unified user archives

**Basic merge (outputs to `{username}.json`):**
```bash
lastfm-sync merge --user alice file1.ndjson file2.ndjson file3.ndjson
# Output: alice.json (in current directory)
```

**With explicit output path:**
```bash
lastfm-sync merge --user alice --out-path /data/archives/alice.json exports/*.ndjson
```

**With glob patterns:**
```bash
lastfm-sync merge --user alice exports/*.ndjson
```

**Azure Blob Storage (both input and output):**
```bash
# Auto-discover from Azure (no file patterns needed!)
# Finds: lastfm/dt=*/alice-*.ndjson
# Outputs: merged/alice.json
lastfm-sync merge --user alice \
  --azure-container lastfmdata \
  --azure-account myaccount

# With custom output prefix
# Outputs: archives/2026/alice.json
lastfm-sync merge --user alice \
  --azure-container lastfmdata \
  --azure-account myaccount \
  --azure-prefix "archives/2026/"

# Using connection string for authentication
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=..."
lastfm-sync merge --user alice \
  --azure-container lastfmdata \
  --azure-auth connstr
```

**Deduplication strategies:**
```bash
# Default: Artist+Album+Track+Timestamp
lastfm-sync merge --user alice *.ndjson --strategy default

# Strict: Includes duration (if available)
lastfm-sync merge --user alice *.ndjson --strategy strict

# Relaxed: Ignores album differences
lastfm-sync merge --user alice *.ndjson --strategy relaxed

# MBID: Uses MusicBrainz IDs when available
lastfm-sync merge --user alice *.ndjson --strategy mbid
```

**Conflict resolution:**
```bash
# Completeness: Keep most complete metadata (default)
lastfm-sync merge --user alice *.ndjson --conflict-resolution completeness

# First: Keep first occurrence
lastfm-sync merge --user alice *.ndjson --conflict-resolution first

# Last: Keep last occurrence
lastfm-sync merge --user alice *.ndjson --conflict-resolution last
```

**Dry-run preview:**
```bash
lastfm-sync merge --user alice *.ndjson --dry-run
# Shows statistics without writing output:
# - Total/unique scrobbles
# - Duplicates removed
# - Date range
# - Unique artists/tracks
# - Estimated output size
```

**Checkpointing (large datasets):**
```bash
# Save checkpoint every 10,000 scrobbles
lastfm-sync merge --user alice large-export-*.ndjson \
  --checkpoint-interval 10000 \
  --checkpoint-path .merge-checkpoint.json

# Resume from checkpoint after interruption
lastfm-sync merge --user alice large-export-*.ndjson --resume
```

**Verbose logging:**
```bash
lastfm-sync merge --user alice *.ndjson --verbose
# Shows DEBUG-level logs including:
# - Conflict resolution decisions
# - Deduplication keys
# - File processing progress
```

### Normalize Command

The `normalize` command updates the `normalized_title` field in existing NDJSON scrobble files. It removes common annotations like "(Live)", "- Remastered", "(feat. Artist)" to create clean, consistent track titles for better matching and analytics.

**Key Features:**
- ✅ **In-place updates**: Atomic file replacement with backup safety
- ✅ **Dry-run mode**: Preview changes without modifying files
- ✅ **Pattern detection**: Automatically identifies Remastered, Live, Featuring annotations
- ✅ **Error handling**: Continues processing even if individual files fail
- ✅ **Idempotency**: Second run shows 0 changes (safe to re-run)
- ✅ **Local & Azure**: Supports both local filesystem and Azure Blob Storage

**Basic normalization (local files):**
```bash
# Process all files for user 'alice'
lastfm-sync normalize --user alice

# Finds: alice_*.ndjson in current directory
# Updates: normalized_title field for all scrobbles
```

**Preview changes before applying (dry-run):**
```bash
lastfm-sync normalize --user alice --dry-run
# Shows what would be changed without modifying files:
# ✓ alice_001.ndjson (updated 15/50 scrobbles)
# ✓ alice_002.ndjson (updated 23/75 scrobbles)
# ✓ alice_003.ndjson (no changes needed)
```

**Azure Blob Storage:**
```bash
# Using DefaultAzureCredential (managed identity or az login)
lastfm-sync normalize --user alice \
  --azure-container scrobbles \
  --azure-account mystorageaccount \
  --azure-auth default

# Using connection string
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=..."
lastfm-sync normalize --user alice \
  --azure-container scrobbles \
  --azure-auth connstr

# With custom prefix
lastfm-sync normalize --user alice \
  --azure-container scrobbles \
  --azure-account mystorageaccount \
  --azure-prefix "archives/2026/"
```

**Common use cases:**

```bash
# After initial fetch - normalize all data
lastfm-sync fetch --user alice
lastfm-sync normalize --user alice

# Re-normalize after pattern updates (idempotent - safe to re-run)
lastfm-sync normalize --user alice

# Test on specific directory
cd /data/exports/alice
lastfm-sync normalize --user alice

# Verbose output for troubleshooting
lastfm-sync normalize --user alice --log-level debug
```

**Normalization patterns:**

The normalize command removes these common annotations:
- **Remastered versions**: `"Hey Jude - Remastered 2009"` → `"Hey Jude"`
- **Live recordings**: `"Bohemian Rhapsody - Live at Wembley"` → `"Bohemian Rhapsody"`
- **Featured artists**: `"Empire State of Mind (feat. Alicia Keys)"` → `"Empire State of Mind"`
- **Deluxe editions**: `"Song Title - Deluxe Edition"` → `"Song Title"`
- **Bonus tracks**: `"Track Name - Bonus Track"` → `"Track Name"`
- **Radio edits**: `"Song - Radio Edit"` → `"Song"`

**Output format:**

```bash
$ lastfm-sync normalize --user alice

Processing 3 files for user 'alice'...
✓ alice_001.ndjson (updated 15/50 scrobbles)
✓ alice_002.ndjson (updated 23/75 scrobbles)
✓ alice_003.ndjson (no changes needed)

Summary:
  Total files: 3
  Updated: 2
  Unchanged: 1
  Errors: 0
  Total scrobbles processed: 175
  Normalized titles updated: 38
  Duration: 0.15s
```

**Error handling:**

The command continues processing even if individual files fail:

```bash
$ lastfm-sync normalize --user alice

Processing 4 files for user 'alice'...
✓ alice_001.ndjson (updated 15/50 scrobbles)
✗ alice_002.ndjson (parse_error: invalid JSON at line 42)
✓ alice_003.ndjson (no changes needed)
✗ alice_004.ndjson (permission_denied: cannot write file)

Summary:
  Total files: 4
  Updated: 1
  Unchanged: 1
  Errors: 2
  
Errors:
  alice_002.ndjson: parse_error
  alice_004.ndjson: permission_denied

See docs/troubleshooting.md for error resolution guidance.
```

**Performance:**

- Streams files line-by-line (memory efficient for large files)
- Atomic writes with temp files (crash-safe)
- Typically processes 1000+ files in under 5 seconds (<5ms per file)

## Output Format

### NDJSON Structure

Each line is a JSON object representing one scrobble:

```json
{"username":"alice","artist":"Radiohead","track":"Paranoid Android","normalized_title":"Paranoid Android","album":"OK Computer","uts":1704067200,"local_time":"2024-01-01T00:00:00Z","mbid":"abc123","source":"lastfm","ingested_at":"2024-01-06T14:30:22Z","raw":{}}
{"username":"alice","artist":"The Beatles","track":"Come Together - Remastered","normalized_title":"Come Together","album":"Abbey Road","uts":1704070800,"local_time":"2024-01-01T01:00:00Z","source":"lastfm","ingested_at":"2024-01-06T14:30:22Z","raw":{}}
```

**Fields:**
- `username` (string): Last.fm username
- `artist` (string): Artist name
- `track` (string): Track name (original from Last.fm, may include annotations)
- `normalized_title` (string): Clean track title with annotations removed (Live, Remastered, featuring, etc.)
- `album` (string): Album name
- `uts` (int64): Unix timestamp (seconds since epoch)
- `local_time` (string): Human-readable UTC timestamp in RFC3339 format (derived from uts)
- `mbid` (string, optional): MusicBrainz ID (omitted if null)
- `source` (string): Data source (always "lastfm")
- `ingested_at` (string): UTC timestamp when record was created (RFC3339 format)
- `raw` (object): Original Last.fm API response for debugging

### Azure Blob Path Structure

**Data blobs:** `{prefix}dt=YYYY-MM-DD/{username}-YYYYMMDD-HHMMSS.ndjson`

Example: `lastfm/dt=2026-01-06/alice-20260106-143022.ndjson`

**Watermark blobs:** `{prefix}{username}.watermark`

Example: `lastfm/alice.watermark`

## Incremental Sync

The tool uses watermarks to track the last successfully synced scrobble timestamp. On subsequent runs:

1. Loads existing watermark (max_uts from previous sync)
2. Fetches only scrobbles with `uts > watermark`
3. Updates watermark after successful write

**Watermark storage:**
- **Local mode**: `~/.lastfm/state/{username}.watermark`
- **Azure mode**: `{prefix}{username}.watermark` blob
- **Auto-selection**: Uses same storage as output (override with `--watermark-store`)

**Reset sync:** Delete the watermark file to refetch all history

## Rate Limiting & Retry

**Rate Limiting:**
- Default: 3 queries per second (Last.fm API limit)
- Configurable via `--qps` flag
- Token bucket algorithm with burst allowance

**Retry Logic:**
- HTTP 429 (rate limited): Respects `Retry-After` header, exponential backoff
- HTTP 5xx (server error): Exponential backoff (1s, 2s, 4s, 8s, 16s)
- Max retries: 5 attempts before giving up
- Timeout: 15s per request (configurable)

**Backoff Strategy:**
1. Check for `Retry-After` header (preferred)
2. Fall back to exponential backoff if header missing
3. Wait before retry: min(retry-after, exponential_backoff)

## Troubleshooting

### API Key Errors

**Error:** `Invalid API key`

**Solution:**
```bash
# Verify key is set
echo $LASTFM_API_KEY

# Or pass directly
lastfm-sync fetch --user alice --api-key YOUR_KEY
```

### Rate Limit Errors

**Error:** `max retries (5) exceeded: rate limited (429)`

**Solution:**
- Reduce QPS: `--qps 2`
- Increase timeout: `--timeout 30s`
- Check if multiple instances are running

### Azure Authentication Errors

**Error:** `no Azure Storage account name provided`

**Solution:**
```bash
# Ensure account name is provided
lastfm-sync fetch --user alice --output azure \
  --azure-container scrobbles \
  --azure-account YOUR_ACCOUNT_NAME
```

**Error:** `failed to get token`

**Solution for DefaultAzureCredential:**
```bash
# Login with Azure CLI
az login

# Or use connection string instead
--azure-auth connstr
```

### Watermark Issues

**Problem:** Sync keeps refetching old scrobbles

**Solution:**
```bash
# Check watermark file
cat ~/.lastfm/state/alice.watermark

# If corrupt, delete and resync
rm ~/.lastfm/state/alice.watermark
lastfm-sync fetch --user alice
```

### Debug Mode

Enable debug logging to see detailed operation:

```bash
lastfm-sync fetch --user alice --log-level debug 2>&1 | tee debug.log
```

Look for:
- `fetch.page.params`: API request details
- `fetch.write.details`: Write operation info
- `watermark.update.start`: Watermark changes
- `rate.limit`: Rate limiting events

## FAQ

**Q: Can I sync multiple users?**  
A: Yes, run the command once per user. Each user has separate watermarks.

**Q: What happens if sync is interrupted?**  
A: The watermark is updated after each successful page write. Restart will continue from the last successful page.

**Q: Can I sync to both local and Azure simultaneously?**  
A: No, choose one output per run. Run twice with different `--output` flags if needed.

**Q: How do I migrate from local to Azure storage?**  
A: Copy the watermark file content to Azure, then switch `--output` to azure. Or delete watermark and refetch.

**Q: Does it support pagination of millions of scrobbles?**  
A: Yes, tested with 100K+ scrobbles. Use `--max-pages` for testing to limit API calls.

**Q: What's the performance?**  
A: At 3 QPS with 200 items/page: ~600 scrobbles/minute, 36K/hour. 100K scrobbles ≈ 3 hours.

**Q: Can I run this in a cron job?**  
A: Yes, perfect for scheduled syncs. Dry-run first to verify configuration.

**Q: Are secrets logged?**  
A: No, all secrets (API keys, connection strings, SAS tokens) are automatically redacted to `****last4` in logs.

## Development

### Prerequisites

- Go 1.24+
- Make
- Docker (optional)

### Build Commands

```bash
make deps          # Install dependencies
make test          # Run tests (105 tests)
make test-coverage # Generate coverage report
make build         # Build binary to dist/lastfm-sync
make docker        # Build Docker image
make clean         # Clean build artifacts
```

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Specific package
go test ./internal/lastfm -v

# Debug output
go test ./internal/service -v -run TestSync
```

## Related Documentation

- [Configuration Reference](docs/configuration.md) - Complete environment variables and CLI flags reference
- [Docker Setup](docs/docker.md) - Building and running containers with Docker and Docker Compose
- [Azure Deployment](docs/azure-deployment.md) - Deploying to Azure Container Instances with managed identity
- [Security Best Practices](docs/security.md) - Secure configuration and secret management
- [Troubleshooting Guide](docs/troubleshooting.md) - Common issues and solutions

### Project Structure

```
├── cmd/lastfm-sync/          # CLI entry point
│   ├── main.go
│   └── commands/             # Cobra commands
│       ├── fetch.go          # fetch command implementation
│       └── fetch_test.go
├── internal/
│   ├── config/               # Configuration management
│   ├── lastfm/               # Last.fm API client
│   ├── logging/              # Structured logging with redaction
│   ├── models/               # Data models (Scrobble)
│   ├── ratelimit/            # Rate limiting & retry logic
│   ├── service/              # Sync service orchestration
│   ├── watermark/            # Watermark persistence
│   └── writer/               # Output writers (local, Azure)
├── .specify/specs/001-lastfm-scrobble-cli/  # Complete specification
│   ├── spec.md               # Requirements document
│   ├── plan.md               # Technical architecture
│   └── tasks.md              # Implementation checklist (83 tasks)
└── Dockerfile                # Multi-stage Docker build
```

## Dependencies

- **spf13/cobra**: CLI framework
- **spf13/viper**: Configuration management
- **uber-go/zap**: Structured logging
- **cenkalti/backoff/v4**: Exponential backoff
- **golang.org/x/time/rate**: Rate limiting
- **Azure SDK for Go**: Azure Blob Storage support
  - github.com/Azure/azure-sdk-for-go/sdk/storage/azblob
  - github.com/Azure/azure-sdk-for-go/sdk/azidentity

## License

Apache License 2.0 - See LICENSE file

## Contributing

See `.specify/specs/001-lastfm-scrobble-cli/` for complete specification and implementation plan.
