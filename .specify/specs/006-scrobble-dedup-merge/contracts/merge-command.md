# CLI Contract: Merge Command

**Feature**: 006-scrobble-dedup-merge  
**Phase**: 1 (Design)  
**Date**: 2026-01-06

## Purpose

Defines the command-line interface contract for the `lastfm-sync merge` command, including flags, arguments, input/output specifications, exit codes, and usage examples.

---

## Command Structure

```bash
lastfm-sync merge [flags] <input-pattern...>
```

**Description**: Merge multiple NDJSON scrobble files into a single deduplicated JSON output.

---

## Positional Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `input-pattern` | string(s) | Yes | One or more glob patterns or file paths to merge (e.g., `data/*.ndjson`, `/path/to/scrobbles.ndjson`) |

**Examples**:
```bash
# Single pattern
lastfm-sync merge "data/*.ndjson"

# Multiple patterns
lastfm-sync merge "data/2023/*.ndjson" "data/2024/*.ndjson"

# Explicit files
lastfm-sync merge scrobbles-1.ndjson scrobbles-2.ndjson scrobbles-3.ndjson
```

**Glob Pattern Rules**:
- `*` matches any sequence of characters (excluding `/`)
- `**` matches any sequence of characters (including `/`, for recursive search)
- `?` matches any single character
- `[abc]` matches any character in the set
- `{a,b}` matches either `a` or `b`

**Path Resolution**:
- Relative paths resolved from current working directory
- Absolute paths used as-is
- Patterns expanded using Go's `filepath.Glob()` (shell-independent)

---

## Flags

### Output Configuration

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--user` | `-u` | string | (required) | Last.fm username (used for default output filename) |
| `--output` | `-o` | string | `local` | Output destination: `local` or `azure` |
| `--out-path` | | string | `{username}.json` | Output file path (local) or blob name (azure) |

**Default Behavior**:
- Output filename defaults to `{username}.json` where username is from `--user` flag
- Local output: `--output local --out-path ./data/alice.json`
- Azure output: `--output azure` with Azure configuration flags

**Examples**:
```bash
# Local file with default name (alice.json)
lastfm-sync merge --user alice "data/*.ndjson"

# Local file with custom path
lastfm-sync merge --user alice --out-path ./output/merged.json "data/*.ndjson"

# Azure Blob Storage
lastfm-sync merge --user alice --output azure --azure-container scrobbles \
  --azure-account mystorageacct --out-path alice.json "data/*.ndjson"
```

---

### Deduplication Configuration

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--strategy` | - | string | `default` | Deduplication strategy: `default`, `strict`, `relaxed`, or `mbid` |
| `--conflict-resolution` | - | string | `completeness` | Conflict resolution mode: `completeness`, `first`, or `last` |

**Strategy Descriptions**:
- `default`: Match by artist + album + title + timestamp (recommended)
- `strict`: Match by artist + album + title + timestamp + duration (exact match)
- `relaxed`: Match by artist + title + timestamp (ignore album)
- `mbid`: Match by MusicBrainz Track ID + timestamp (authoritative)

**Conflict Resolution Modes**:
- `completeness`: Keep scrobble with most complete metadata (default)
- `first`: Keep first occurrence encountered
- `last`: Keep last occurrence encountered

**Examples**:
```bash
# Strict deduplication (includes duration)
lastfm-sync merge --strategy strict "data/*.ndjson"

# Relaxed deduplication (ignore album differences)
lastfm-sync merge --strategy relaxed "data/*.ndjson"

# Use MusicBrainz IDs for authoritative matching
lastfm-sync merge --strategy mbid "data/*.ndjson"

# Keep first occurrence on conflicts
lastfm-sync merge --conflict-resolution first "data/*.ndjson"
```

---

### Progress & Logging

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--progress` | - | bool | `true` | Show progress bar during merge |
| `--no-progress` | - | bool | `false` | Disable progress bar (useful for scripting) |
| `--log-level` | `-l` | string | `info` | Logging level: `debug`, `info`, `warn`, `error` |

**Examples**:
```bash
# Disable progress bar for scripting
lastfm-sync merge --no-progress "data/*.ndjson"

# Enable debug logging
lastfm-sync merge --log-level debug "data/*.ndjson"
```

---

### Performance Configuration

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--checkpoint-interval` | - | int | `10000` | Save checkpoint every N scrobbles (0 to disable) |
| `--checkpoint-path` | - | string | `.merge-checkpoint-{timestamp}.json` | Checkpoint file path |
| `--buffer-size` | - | int | `131072` | Scanner buffer size in bytes (default 128KB) |

**Examples**:
```bash
# Checkpoint every 50K scrobbles
lastfm-sync merge --checkpoint-interval 50000 "data/*.ndjson"

# Custom checkpoint path
lastfm-sync merge --checkpoint-path /tmp/merge.checkpoint "data/*.ndjson"

# Disable checkpointing (faster but no resume)
lastfm-sync merge --checkpoint-interval 0 "data/*.ndjson"
```

---

### Resume Configuration

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--resume` | - | bool | `false` | Resume from existing checkpoint file |

**Example**:
```bash
# Resume interrupted merge
lastfm-sync merge --resume --checkpoint-path .merge-checkpoint-20260106.json "data/*.ndjson"
```

**Resume Behavior**:
1. Load checkpoint file specified by `--checkpoint-path`
2. Validate checkpoint configuration matches current flags
3. Skip already-processed files
4. Resume from `current_line` in `current_file`
5. Continue with remaining files

**Resume Validation**:
- Strategy must match checkpoint
- Conflict resolution mode must match
- Input files must match (order-sensitive)
- Output path must match

**Error Handling**:
- If checkpoint file not found: Exit with error code 2
- If checkpoint config mismatch: Exit with error code 3
- If checkpoint version unsupported: Exit with error code 3

---

### File Discovery

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--recursive` | `-r` | bool | `false` | Recursively search subdirectories (when pattern includes `**`) |

**Example**:
```bash
# Recursively find all NDJSON files in data/ and subdirectories
lastfm-sync merge -r "data/**/*.ndjson"
```

**Note**: Without `-r` flag, `**` patterns treated as `*` (non-recursive).

---

### Azure Configuration

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--azure-account` | - | string | - | Azure Storage account name (overrides URL) |
| `--azure-container` | - | string | - | Azure Blob container name (overrides URL) |
| `--azure-blob` | - | string | - | Azure Blob name (overrides URL) |
| `--azure-use-default-credential` | - | bool | `true` | Use Azure DefaultAzureCredential (managed identity, Azure CLI, etc.) |

**Examples**:
```bash
# Using Azure Blob URL (recommended)
lastfm-sync merge -s azure -o "az://mystorageacct/scrobbles/merged.json" "data/*.ndjson"

# Using explicit flags
lastfm-sync merge -s azure \
  --azure-account mystorageacct \
  --azure-container scrobbles \
  --azure-blob merged.json \
  "data/*.ndjson"
```

**Authentication**:
- Default: Use `DefaultAzureCredential` (tries managed identity, Azure CLI, environment variables)
- Fallback: Set `AZURE_STORAGE_ACCOUNT_KEY` environment variable

---

## Exit Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | Success | Merge completed successfully |
| 1 | General Error | Unspecified error (invalid flags, configuration error, etc.) |
| 2 | Input Error | Input files not found, invalid glob pattern, or empty input |
| 3 | Resume Error | Checkpoint file not found, corrupted, or config mismatch |
| 4 | Write Error | Failed to write output file (permissions, disk full, network error) |
| 5 | Validation Error | All input files failed validation (no valid scrobbles) |

**Exit Code Examples**:
```bash
# Success
$ lastfm-sync merge "data/*.ndjson"
$ echo $?
0

# Input error (pattern matches no files)
$ lastfm-sync merge "nonexistent/*.ndjson"
Error: No input files found matching pattern: nonexistent/*.ndjson
$ echo $?
2

# Resume error (checkpoint not found)
$ lastfm-sync merge --resume --checkpoint-path missing.json "data/*.ndjson"
Error: Checkpoint file not found: missing.json
$ echo $?
3

# Write error (permission denied)
$ lastfm-sync merge -o /root/merged.json "data/*.ndjson"
Error: Failed to write output: permission denied
$ echo $?
4
```

---

## Input Specification

### Input File Format

**Format**: NDJSON (Newline-Delimited JSON)
- One JSON object per line
- Each object represents a single scrobble
- No enclosing array brackets

**Example Input**:
```ndjson
{"artist":"The Beatles","album":"Abbey Road","title":"Come Together","timestamp":1735689600,"duration":259}
{"artist":"Pink Floyd","album":"The Dark Side of the Moon","title":"Time","timestamp":1735689700,"duration":413}
{"artist":"Led Zeppelin","album":"IV","title":"Stairway to Heaven","timestamp":1735689800,"duration":482}
```

**Required Fields**:
- `artist` (string): Artist name
- `title` (string): Track title
- `timestamp` (integer): Unix timestamp (seconds since epoch)

**Optional Fields**:
- `album` (string): Album name
- `duration` (integer): Track duration in seconds
- `mbid` (string): MusicBrainz Track ID
- `artist_mbid` (string): MusicBrainz Artist ID
- `album_mbid` (string): MusicBrainz Album ID

**Validation**:
- Lines with invalid JSON: Logged at WARN level, skipped
- Scrobbles missing required fields: Logged at WARN level, skipped
- Scrobbles with invalid timestamps: Logged at WARN level, skipped

---

### File Discovery Rules

**Pattern Matching Order**:
1. Expand glob patterns using `filepath.Glob()`
2. Sort file paths lexicographically (ensures deterministic order)
3. Validate files exist and are readable
4. Filter out non-NDJSON files (optional: check `.ndjson` or `.json` extension)

**Error Handling**:
- If no files match patterns: Exit with code 2
- If some files match but are unreadable: Log warning, skip file, continue
- If all files are unreadable: Exit with code 2

**Example Discovery**:
```bash
# Input: "data/202[3-4]/*.ndjson"
# Discovered files (sorted):
#   data/2023/jan.ndjson
#   data/2023/feb.ndjson
#   data/2024/jan.ndjson
#   data/2024/feb.ndjson
```

---

## Output Specification

### Output File Format

**Format**: JSON array
- Single array containing all deduplicated scrobbles
- Pretty-printed with 2-space indentation
- UTF-8 encoding

**Example Output**:
```json
[
  {
    "artist": "The Beatles",
    "album": "Abbey Road",
    "title": "Come Together",
    "timestamp": 1735689600,
    "duration": 259,
    "mbid": "f3d8e9a0-1234-5678-9abc-def012345678"
  },
  {
    "artist": "Pink Floyd",
    "album": "The Dark Side of the Moon",
    "title": "Time",
    "timestamp": 1735689700,
    "duration": 413
  }
]
```

**Sorting**: Scrobbles sorted by `timestamp` (ascending, oldest first)

**Atomic Write**:
- Output written to temporary file first
- Atomic rename on success (prevents partial writes)
- Temporary file deleted on error

---

### Standard Output (stdout)

**Success Output**:
```
Discovering input files...
Found 3 files matching patterns
Processing: data/2023.ndjson [===================] 100%
Processing: data/2024.ndjson [===================] 100%

Merge Summary:
  Files processed: 3/3
  Total scrobbles: 150,000
  Unique scrobbles: 145,000
  Duplicates removed: 5,000
  Skipped (errors): 12 lines, 8 scrobbles
  Conflicts resolved: 142
  Processing rate: 10,234 scrobbles/sec
  Output: /path/to/merged.json

Merge completed successfully!
```

**Progress Bar Format** (when `--progress` enabled):
```
Processing: data/2024.ndjson [=====>        ] 45% | 67,500/150,000 | 10.2K/sec | ETA: 8s | Duplicates: 3,421
```

---

### Standard Error (stderr)

**Warning Messages** (non-fatal):
```
WARN: Invalid JSON on line 1234 in file data/corrupted.ndjson: unexpected end of JSON input
WARN: Missing required field 'artist' in file data/incomplete.ndjson:5678
WARN: Skipping unreadable file: data/locked.ndjson (permission denied)
```

**Error Messages** (fatal):
```
ERROR: No input files found matching pattern: nonexistent/*.ndjson
ERROR: Failed to write output to /root/merged.json: permission denied
ERROR: Checkpoint config mismatch: strategy 'strict' != checkpoint strategy 'default'
```

**Debug Messages** (when `--log-level debug`):
```
DEBUG: Generated deduplication key: a1b2c3d4e5f6... (strategy: default)
DEBUG: Conflict resolved: keeping new scrobble (completeness score: 8 vs 6)
DEBUG: Checkpoint saved: 50,000 scrobbles processed
```

---

## Usage Examples

### Basic Usage

```bash
# Merge all NDJSON files in current directory
lastfm-sync merge "*.ndjson"

# Merge files from specific directory
lastfm-sync merge "data/*.ndjson" -o merged.json

# Merge multiple directories
lastfm-sync merge "data/2023/*.ndjson" "data/2024/*.ndjson"
```

### Advanced Deduplication

```bash
# Strict matching (includes duration)
lastfm-sync merge --strategy strict "data/*.ndjson"

# Relaxed matching (ignore album)
lastfm-sync merge --strategy relaxed "exports/*.ndjson"

# Use MusicBrainz IDs
lastfm-sync merge --strategy mbid "mb-exports/*.ndjson"

# Keep first occurrence on conflicts
lastfm-sync merge --conflict-resolution first "data/*.ndjson"
```

### Azure Blob Storage

```bash
# Write to Azure Blob
lastfm-sync merge \
  -s azure \
  -o "az://mystorageacct/scrobbles/merged.json" \
  "data/*.ndjson"

# Read from local, write to Azure
lastfm-sync merge \
  -s azure \
  --azure-account mystorageacct \
  --azure-container scrobbles \
  --azure-blob merged-2024.json \
  "exports/2024-*.ndjson"
```

### Resume & Checkpointing

```bash
# Enable checkpointing with custom interval
lastfm-sync merge \
  --checkpoint-interval 50000 \
  --checkpoint-path merge.checkpoint \
  "large-dataset/*.ndjson"

# Resume interrupted merge
lastfm-sync merge \
  --resume \
  --checkpoint-path merge.checkpoint \
  "large-dataset/*.ndjson"
```

### Scripting & Automation

```bash
# Disable progress bar for cron job
lastfm-sync merge --no-progress "data/*.ndjson" 2>/var/log/merge.log

# Capture exit code
lastfm-sync merge "data/*.ndjson"
if [ $? -eq 0 ]; then
  echo "Merge succeeded"
  rm -f .merge-checkpoint-*.json
else
  echo "Merge failed"
  exit 1
fi

# JSON output for parsing (future: --output-format json)
lastfm-sync merge --log-level error "data/*.ndjson" > /dev/null
echo $?
```

---

## Flag Validation Rules

| Flag | Validation Rule | Error Message |
|------|-----------------|---------------|
| `--user` | Non-empty string | "user flag is required" |
| `--output` | Must be "local" or "azure" | "output must be 'local' or 'azure'" |
| `--out-path` | Non-empty string | "out-path is required" |
| `--strategy` | Must be "default", "strict", "relaxed", or "mbid" | "invalid strategy: {value}" |
| `--conflict-resolution` | Must be "completeness", "first", or "last" | "invalid conflict resolution: {value}" |
| `--checkpoint-interval` | Non-negative integer | "checkpoint interval must be >= 0" |
| `--buffer-size` | Positive integer | "buffer size must be > 0" |
| `--log-level` | Must be "debug", "info", "warn", or "error" | "invalid log level: {value}" |
| `--resume` | If true, checkpoint file must exist | "checkpoint file not found: {path}" |
| Azure flags | If `--output azure`, container and account required | "azure output requires --azure-container and --azure-account" |

**Validation Order**:
1. Parse flags with cobra
2. Validate flag values (types, enums, ranges)
3. Validate flag combinations (e.g., `--resume` requires `--checkpoint-path`)
4. Validate input patterns (non-empty, well-formed)
5. Discover input files (glob expansion)
6. Validate input files (readable, NDJSON format)

---

## Compatibility

### Cobra Integration

**Command Registration**:
```go
var mergeCmd = &cobra.Command{
    Use:   "merge [flags] <input-pattern...>",
    Short: "Merge multiple NDJSON scrobble files into deduplicated JSON",
    Long:  `...`,
    Args:  cobra.MinimumNArgs(1),
    RunE:  runMerge,
}

func init() {
    rootCmd.AddCommand(mergeCmd)
    
    // Output flags
    mergeCmd.Flags().StringP("output", "o", "merged-scrobbles.json", "Output file path")
    mergeCmd.Flags().StringP("storage", "s", "local", "Storage backend (local or azure)")
    
    // Deduplication flags
    mergeCmd.Flags().String("strategy", "default", "Deduplication strategy")
    mergeCmd.Flags().String("conflict-resolution", "completeness", "Conflict resolution mode")
    
    // ... more flags ...
}
```

### Viper Integration

**Configuration Precedence**:
1. Command-line flags (highest priority)
2. Environment variables (e.g., `LASTFM_SYNC_MERGE_STRATEGY`)
3. Config file (e.g., `~/.lastfm-sync.yaml`)
4. Default values (lowest priority)

**Environment Variable Mapping**:
- `--strategy` → `LASTFM_SYNC_MERGE_STRATEGY`
- `--output` → `LASTFM_SYNC_MERGE_OUTPUT`
- `--azure-account` → `AZURE_STORAGE_ACCOUNT_NAME`

---

## Future Extensions

**Planned Features** (not in MVP):
- `--output-format`: Support `ndjson`, `csv`, `parquet` output formats
- `--sort-by`: Sort output by custom field (e.g., `artist`, `album`, `title`)
- `--filter`: Apply filters (e.g., `--filter "duration > 300"`)
- `--stats-output`: Write statistics to separate JSON file
- `--dry-run`: Show what would be merged without writing output
- `--parallel`: Process files in parallel (multi-threaded)

---

**CLI Contract Complete** ✅  
All flags, arguments, exit codes, and examples documented. Ready for quickstart guide.
