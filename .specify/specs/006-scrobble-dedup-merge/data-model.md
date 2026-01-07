# Data Model: Scrobble Deduplication & Merging

**Feature**: 006-scrobble-dedup-merge  
**Phase**: 1 (Design)  
**Date**: 2026-01-06

## Purpose

Define all data structures, entities, and their relationships for the merge feature. Documents Go struct definitions, validation rules, and state management.

---

## Core Entities

### 1. Scrobble (Existing)

**Location**: `internal/models/scrobble.go`  
**Status**: Already exists (feature 001), reused as-is

```go
// Scrobble represents a single Last.fm scrobble (listened track)
type Scrobble struct {
    Artist              string    `json:"artist"`
    Album               string    `json:"album"`
    Title               string    `json:"title"`
    Timestamp           int64     `json:"timestamp"`           // Unix timestamp (seconds since epoch)
    Duration            int       `json:"duration,omitempty"`  // Track duration in seconds
    MusicBrainzTrackID  string    `json:"mbid,omitempty"`      // MusicBrainz Track ID
    MusicBrainzArtistID string    `json:"artist_mbid,omitempty"`
    MusicBrainzAlbumID  string    `json:"album_mbid,omitempty"`
    NormalizedTitle     string    `json:"normalized_title,omitempty"` // Feature 004
}

// Validate checks if scrobble has required fields
func (s *Scrobble) Validate() error {
    if s.Artist == "" {
        return errors.New("missing required field: artist")
    }
    if s.Title == "" {
        return errors.New("missing required field: title")
    }
    if s.Timestamp <= 0 {
        return errors.New("invalid timestamp: must be positive")
    }
    return nil
}
```

**Validation Rules**:
- `Artist` and `Title` are **required** (non-empty strings)
- `Timestamp` is **required** (positive Unix timestamp)
- `Album` is **optional** (empty for singles/unknown albums)
- All MusicBrainz IDs are **optional**
- `Duration` is **optional** (0 if unknown)

**Notes**:
- `NormalizedTitle` added by feature 004, not used in deduplication (raw `Title` used)
- JSON tags match Last.fm API export format

---

### 2. MergeConfig

**Location**: `internal/merge/config.go` (NEW)  
**Purpose**: Configuration for merge operation

```go
// MergeConfig contains all configuration for a merge operation
type MergeConfig struct {
    // Input configuration
    InputPatterns      []string          `json:"input_patterns"`       // Glob patterns for input files
    InputFiles         []string          `json:"input_files"`          // Resolved input file paths
    Recursive          bool              `json:"recursive"`            // Recursively search subdirectories
    
    // Output configuration
    OutputPath         string            `json:"output_path"`          // Output file path (local or Azure)
    StorageBackend     string            `json:"storage_backend"`      // "local" or "azure"
    AzureConfig        *AzureConfig      `json:"azure_config,omitempty"` // Azure-specific config
    
    // Deduplication configuration
    Strategy           DeduplicationStrategy `json:"strategy"`         // Deduplication strategy
    ConflictResolution ConflictResolution    `json:"conflict_resolution"` // Conflict resolution mode
    
    // Performance configuration
    CheckpointInterval int               `json:"checkpoint_interval"`  // Save checkpoint every N scrobbles
    CheckpointPath     string            `json:"checkpoint_path"`      // Checkpoint file path
    ProgressEnabled    bool              `json:"progress_enabled"`     // Show progress bar
    BufferSize         int               `json:"buffer_size"`          // Scanner buffer size (bytes)
    
    // Resume configuration
    Resume             bool              `json:"resume"`               // Resume from checkpoint
    
    // Logging configuration
    LogLevel           string            `json:"log_level"`            // "debug", "info", "warn", "error"
}

// DeduplicationStrategy defines how duplicates are detected
type DeduplicationStrategy string

const (
    StrategyDefault  DeduplicationStrategy = "default"  // Artist+Album+Title+Timestamp
    StrategyStrict   DeduplicationStrategy = "strict"   // Default + Duration
    StrategyRelaxed  DeduplicationStrategy = "relaxed"  // Artist+Title+Timestamp (no Album)
    StrategyMBID     DeduplicationStrategy = "mbid"     // MusicBrainz Track ID + Timestamp
)

// ConflictResolution defines how duplicate scrobbles are resolved
type ConflictResolution string

const (
    ResolutionCompleteness ConflictResolution = "completeness" // Select most complete metadata
    ResolutionFirst        ConflictResolution = "first"        // Keep first occurrence
    ResolutionLast         ConflictResolution = "last"         // Keep last occurrence
)

// AzureConfig contains Azure Blob Storage configuration
type AzureConfig struct {
    AccountName   string `json:"account_name"`
    ContainerName string `json:"container_name"`
    BlobName      string `json:"blob_name"`
    UseDefaultCredential bool `json:"use_default_credential"` // Use Azure DefaultAzureCredential
}

// Validate checks if config is valid
func (c *MergeConfig) Validate() error {
    if len(c.InputPatterns) == 0 && len(c.InputFiles) == 0 {
        return errors.New("no input patterns or files specified")
    }
    if c.OutputPath == "" {
        return errors.New("output path is required")
    }
    if c.StorageBackend != "local" && c.StorageBackend != "azure" {
        return errors.New("storage backend must be 'local' or 'azure'")
    }
    if c.StorageBackend == "azure" && c.AzureConfig == nil {
        return errors.New("azure_config required when storage_backend is 'azure'")
    }
    if c.CheckpointInterval <= 0 {
        return errors.New("checkpoint_interval must be positive")
    }
    return nil
}
```

**Default Values**:
```go
func DefaultConfig() *MergeConfig {
    return &MergeConfig{
        StorageBackend:     "local",
        Strategy:           StrategyDefault,
        ConflictResolution: ResolutionCompleteness,
        CheckpointInterval: 10000, // Every 10K scrobbles
        ProgressEnabled:    true,
        BufferSize:         128 * 1024, // 128KB
        LogLevel:           "info",
    }
}
```

---

### 3. MergeCheckpoint

**Location**: `internal/merge/checkpoint.go` (NEW)  
**Purpose**: Persistent state for resuming interrupted merges

```go
// MergeCheckpoint represents the saved state of a merge operation
type MergeCheckpoint struct {
    // Metadata
    Version           string                 `json:"version"`          // Checkpoint format version (e.g., "1.0")
    CreatedAt         time.Time              `json:"created_at"`       // Checkpoint creation time
    UpdatedAt         time.Time              `json:"updated_at"`       // Last update time
    
    // Configuration snapshot
    Strategy          DeduplicationStrategy  `json:"strategy"`         // Deduplication strategy used
    ConflictResolution ConflictResolution    `json:"conflict_resolution"`
    InputFiles        []string               `json:"input_files"`      // Ordered list of input files
    OutputPath        string                 `json:"output_path"`      // Output destination
    
    // Progress tracking
    ProcessedFiles    []string               `json:"processed_files"`  // Files fully processed
    CurrentFile       string                 `json:"current_file"`     // File being processed
    CurrentLineNumber int                    `json:"current_line"`     // Line number in current file
    
    // Deduplication state
    DeduplicationMap  map[string]int         `json:"dedup_map"`        // Hash -> index in Scrobbles
    Scrobbles         []*models.Scrobble     `json:"scrobbles"`        // Deduplicated scrobbles
    
    // Statistics
    Stats             MergeStats             `json:"stats"`            // Current statistics
}

// Save writes checkpoint to disk in JSON format
func (c *MergeCheckpoint) Save(path string) error {
    c.UpdatedAt = time.Now()
    
    // Atomic write: tmp file + rename
    tmpPath := path + ".tmp"
    f, err := os.Create(tmpPath)
    if err != nil {
        return fmt.Errorf("create checkpoint file: %w", err)
    }
    defer f.Close()
    
    encoder := json.NewEncoder(f)
    encoder.SetIndent("", "  ") // Pretty-print for debugging
    if err := encoder.Encode(c); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("encode checkpoint: %w", err)
    }
    
    if err := f.Close(); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("close checkpoint file: %w", err)
    }
    
    // Atomic rename
    if err := os.Rename(tmpPath, path); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("rename checkpoint file: %w", err)
    }
    
    return nil
}

// Load reads checkpoint from disk
func LoadCheckpoint(path string) (*MergeCheckpoint, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("open checkpoint: %w", err)
    }
    defer f.Close()
    
    var checkpoint MergeCheckpoint
    decoder := json.NewDecoder(f)
    if err := decoder.Decode(&checkpoint); err != nil {
        return nil, fmt.Errorf("decode checkpoint: %w", err)
    }
    
    // Version check
    if checkpoint.Version != "1.0" {
        return nil, fmt.Errorf("unsupported checkpoint version: %s", checkpoint.Version)
    }
    
    return &checkpoint, nil
}

// Delete removes checkpoint file
func (c *MergeCheckpoint) Delete(path string) error {
    return os.Remove(path)
}
```

**Checkpoint Lifecycle**:
1. **Create**: Initialize checkpoint at merge start
2. **Update**: Save every `CheckpointInterval` scrobbles
3. **Resume**: Load on `--resume` flag
4. **Delete**: Remove after successful merge completion

**Storage Size Estimate**:
- 1M scrobbles × 300 bytes/scrobble = ~300MB
- DeduplicationMap: 1M keys × 80 bytes = ~80MB
- Total: ~380MB (within 500MB memory budget)

---

### 4. MergeStats

**Location**: `internal/merge/stats.go` (NEW)  
**Purpose**: Track merge operation statistics

```go
// MergeStats tracks statistics for a merge operation
type MergeStats struct {
    // File counts
    TotalFiles      int       `json:"total_files"`       // Total input files discovered
    ProcessedFiles  int       `json:"processed_files"`   // Files fully processed
    
    // Scrobble counts
    TotalScrobbles  int       `json:"total_scrobbles"`   // Total scrobbles read
    UniqueScrobbles int       `json:"unique_scrobbles"`  // Unique scrobbles after deduplication
    Duplicates      int       `json:"duplicates"`        // Duplicate scrobbles removed
    
    // Error counts
    SkippedLines    int       `json:"skipped_lines"`     // Lines with JSON parse errors
    SkippedScrobbles int      `json:"skipped_scrobbles"` // Scrobbles failing validation
    
    // Conflict tracking
    Conflicts       int       `json:"conflicts"`         // Duplicate keys resolved
    ConflictsByStrategy map[string]int `json:"conflicts_by_strategy"` // Conflicts per strategy
    
    // Performance metrics
    StartTime       time.Time `json:"start_time"`        // Merge start time
    EndTime         time.Time `json:"end_time"`          // Merge end time
    Duration        float64   `json:"duration_seconds"`  // Total duration in seconds
    Rate            float64   `json:"rate_per_second"`   // Scrobbles processed per second
}

// Update increments statistics counters
func (s *MergeStats) Update(delta MergeStats) {
    s.TotalScrobbles += delta.TotalScrobbles
    s.UniqueScrobbles = delta.UniqueScrobbles // Replace, not increment
    s.Duplicates += delta.Duplicates
    s.SkippedLines += delta.SkippedLines
    s.SkippedScrobbles += delta.SkippedScrobbles
    s.Conflicts += delta.Conflicts
}

// Finalize calculates derived statistics at merge completion
func (s *MergeStats) Finalize() {
    s.EndTime = time.Now()
    s.Duration = s.EndTime.Sub(s.StartTime).Seconds()
    if s.Duration > 0 {
        s.Rate = float64(s.TotalScrobbles) / s.Duration
    }
}

// String formats stats for console output
func (s *MergeStats) String() string {
    return fmt.Sprintf(
        "Files: %d/%d | Scrobbles: %d total, %d unique, %d duplicates | Errors: %d lines, %d scrobbles | Rate: %.0f/sec",
        s.ProcessedFiles, s.TotalFiles,
        s.TotalScrobbles, s.UniqueScrobbles, s.Duplicates,
        s.SkippedLines, s.SkippedScrobbles,
        s.Rate,
    )
}
```

**Usage**:
```go
stats := &MergeStats{StartTime: time.Now()}
// ... process files ...
stats.TotalScrobbles++
stats.Duplicates++
// ... at end ...
stats.Finalize()
fmt.Println(stats.String())
```

---

### 5. DeduplicationMap

**Location**: `internal/merge/deduplicator.go` (NEW)  
**Purpose**: Hash map for tracking unique scrobbles

```go
// DeduplicationMap tracks unique scrobbles by hash key
type DeduplicationMap struct {
    // Map from hash key to scrobble pointer
    data      map[string]*models.Scrobble
    
    // Strategy used for key generation
    strategy  DeduplicationStrategy
    
    // Conflict resolution mode
    resolution ConflictResolution
    
    // Statistics
    conflicts int
}

// NewDeduplicationMap creates a new deduplication map
func NewDeduplicationMap(strategy DeduplicationStrategy, resolution ConflictResolution) *DeduplicationMap {
    return &DeduplicationMap{
        data:       make(map[string]*models.Scrobble),
        strategy:   strategy,
        resolution: resolution,
        conflicts:  0,
    }
}

// Add attempts to add a scrobble to the map
// Returns true if added (new), false if duplicate (existing kept or replaced)
func (dm *DeduplicationMap) Add(scrobble *models.Scrobble) bool {
    key := dm.generateKey(scrobble)
    
    existing, exists := dm.data[key]
    if !exists {
        // New scrobble
        dm.data[key] = scrobble
        return true
    }
    
    // Duplicate found - resolve conflict
    dm.conflicts++
    winner := dm.resolveConflict(existing, scrobble)
    dm.data[key] = winner
    return false // Duplicate
}

// Get retrieves a scrobble by key
func (dm *DeduplicationMap) Get(key string) (*models.Scrobble, bool) {
    scrobble, exists := dm.data[key]
    return scrobble, exists
}

// All returns all unique scrobbles as a slice
func (dm *DeduplicationMap) All() []*models.Scrobble {
    scrobbles := make([]*models.Scrobble, 0, len(dm.data))
    for _, scrobble := range dm.data {
        scrobbles = append(scrobbles, scrobble)
    }
    return scrobbles
}

// Size returns the number of unique scrobbles
func (dm *DeduplicationMap) Size() int {
    return len(dm.data)
}

// Conflicts returns the number of conflicts resolved
func (dm *DeduplicationMap) Conflicts() int {
    return dm.conflicts
}

// generateKey creates a hash key for a scrobble based on strategy
func (dm *DeduplicationMap) generateKey(s *models.Scrobble) string {
    h := sha256.New()
    
    switch dm.strategy {
    case StrategyStrict:
        h.Write([]byte(strings.ToLower(s.Artist)))
        h.Write([]byte(strings.ToLower(s.Album)))
        h.Write([]byte(strings.ToLower(s.Title)))
        h.Write([]byte(fmt.Sprintf("%d", s.Timestamp)))
        h.Write([]byte(fmt.Sprintf("%d", s.Duration)))
        
    case StrategyRelaxed:
        h.Write([]byte(strings.ToLower(s.Artist)))
        h.Write([]byte(strings.ToLower(s.Title)))
        h.Write([]byte(fmt.Sprintf("%d", s.Timestamp)))
        
    case StrategyMBID:
        if s.MusicBrainzTrackID != "" {
            h.Write([]byte(s.MusicBrainzTrackID))
            h.Write([]byte(fmt.Sprintf("%d", s.Timestamp)))
        } else {
            // Fallback to default
            return dm.generateKeyDefault(s)
        }
        
    default: // StrategyDefault
        return dm.generateKeyDefault(s)
    }
    
    return hex.EncodeToString(h.Sum(nil))
}

// generateKeyDefault generates default strategy key
func (dm *DeduplicationMap) generateKeyDefault(s *models.Scrobble) string {
    h := sha256.New()
    h.Write([]byte(strings.ToLower(s.Artist)))
    h.Write([]byte(strings.ToLower(s.Album)))
    h.Write([]byte(strings.ToLower(s.Title)))
    h.Write([]byte(fmt.Sprintf("%d", s.Timestamp)))
    return hex.EncodeToString(h.Sum(nil))
}

// resolveConflict selects which scrobble to keep when duplicate key found
func (dm *DeduplicationMap) resolveConflict(existing, new *models.Scrobble) *models.Scrobble {
    switch dm.resolution {
    case ResolutionFirst:
        return existing
    case ResolutionLast:
        return new
    case ResolutionCompleteness:
        existingScore := completenessScore(existing)
        newScore := completenessScore(new)
        if newScore > existingScore {
            return new
        } else if newScore == existingScore {
            // Tie-breaker: prefer later timestamp
            if new.Timestamp >= existing.Timestamp {
                return new
            }
        }
        return existing
    default:
        return existing
    }
}

// completenessScore calculates metadata completeness score
func completenessScore(s *models.Scrobble) int {
    score := 0
    if s.Artist != "" { score++ }
    if s.Album != "" { score++ }
    if s.Title != "" { score++ }
    if s.Timestamp > 0 { score++ }
    if s.Duration > 0 { score++ }
    if s.MusicBrainzTrackID != "" { score += 2 } // Extra weight
    if s.MusicBrainzArtistID != "" { score++ }
    if s.MusicBrainzAlbumID != "" { score++ }
    return score
}
```

---

### 6. MergeResult

**Location**: `internal/merge/merger.go` (NEW)  
**Purpose**: Return value from merge operation

```go
// MergeResult contains the outcome of a merge operation
type MergeResult struct {
    // Output location
    OutputPath string `json:"output_path"`
    
    // Statistics
    Stats MergeStats `json:"stats"`
    
    // Errors encountered (non-fatal)
    Warnings []MergeWarning `json:"warnings,omitempty"`
    
    // Success flag
    Success bool `json:"success"`
}

// MergeWarning represents a non-fatal error during merge
type MergeWarning struct {
    File    string `json:"file"`              // File where warning occurred
    Line    int    `json:"line,omitempty"`    // Line number (if applicable)
    Message string `json:"message"`           // Warning message
    Type    string `json:"type"`              // "parse_error", "validation_error", etc.
}

// AddWarning adds a warning to the result
func (r *MergeResult) AddWarning(warning MergeWarning) {
    r.Warnings = append(r.Warnings, warning)
}
```

---

## Entity Relationships

```
┌─────────────────┐
│   MergeConfig   │──┐
└─────────────────┘  │
                     │
                     ▼
           ┌─────────────────┐
           │     Merger      │
           └─────────────────┘
                 │     │
        ┌────────┘     └────────┐
        ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│DeduplicationMap │     │ MergeCheckpoint │
└─────────────────┘     └─────────────────┘
        │                       │
        │  hash keys            │  state
        ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│    Scrobble     │────▶│   MergeStats    │
└─────────────────┘     └─────────────────┘
        │
        │ written to
        ▼
┌─────────────────┐
│  Writer (I/F)   │
└─────────────────┘
    │         │
    ▼         ▼
 Local      Azure
```

**Relationships**:
- `Merger` uses `MergeConfig` for configuration
- `Merger` owns `DeduplicationMap` for tracking unique scrobbles
- `Merger` creates `MergeCheckpoint` periodically for resume capability
- `DeduplicationMap` stores pointers to `Scrobble` instances
- `Merger` uses `Writer` interface (from internal/writer) for output
- `MergeStats` tracks statistics throughout operation
- `MergeResult` aggregates final stats and warnings

---

## State Machine

### Merge Operation States

```
┌──────────┐
│   INIT   │ Initialize config, create dedup map, setup progress bar
└─────┬────┘
      │
      ▼
┌──────────┐
│ DISCOVER │ Discover input files (glob patterns), validate existence
└─────┬────┘
      │
      ▼
┌──────────┐
│  RESUME? │ Check for checkpoint file (if --resume flag)
└─────┬────┘
      │
  ┌───┴───┐
  │       │
  ▼       ▼
LOAD    FRESH
CKPT    START
  │       │
  └───┬───┘
      │
      ▼
┌──────────┐
│ PROCESS  │ Read files, parse NDJSON, deduplicate, track progress
└─────┬────┘
      │
      ├─every N scrobbles─┐
      │                   ▼
      │             ┌──────────┐
      │             │CHECKPOINT│ Save state to disk
      │             └─────┬────┘
      │                   │
      │◀──────────────────┘
      │
      ▼
┌──────────┐
│  MERGE   │ Sort scrobbles, write to output (atomic)
└─────┬────┘
      │
      ▼
┌──────────┐
│ CLEANUP  │ Delete checkpoint, close files, finalize stats
└─────┬────┘
      │
      ▼
┌──────────┐
│   DONE   │ Return MergeResult
└──────────┘
```

**State Transitions**:
- `INIT → DISCOVER`: Always
- `DISCOVER → RESUME?`: Always
- `RESUME? → LOAD CKPT`: If `--resume` flag and checkpoint exists
- `RESUME? → FRESH START`: Otherwise
- `PROCESS → CHECKPOINT`: Every `CheckpointInterval` scrobbles
- `CHECKPOINT → PROCESS`: After successful checkpoint save
- `PROCESS → MERGE`: After all files processed
- `MERGE → CLEANUP`: After successful write
- `CLEANUP → DONE`: Always

**Error Handling**:
- Parse errors: Log warning, skip line, continue
- Validation errors: Log warning, skip scrobble, continue
- File read errors: Log error, skip file, continue with remaining files
- Write errors: Fatal, abort merge, preserve checkpoint
- Checkpoint save errors: Log warning, continue (next checkpoint will retry)

---

## Validation Rules Summary

| Entity | Field | Rule |
|--------|-------|------|
| `Scrobble` | `Artist` | Required, non-empty string |
| `Scrobble` | `Title` | Required, non-empty string |
| `Scrobble` | `Timestamp` | Required, positive integer (Unix timestamp) |
| `Scrobble` | `Album` | Optional |
| `Scrobble` | `Duration` | Optional, non-negative if present |
| `MergeConfig` | `InputPatterns` or `InputFiles` | At least one required |
| `MergeConfig` | `OutputPath` | Required, non-empty string |
| `MergeConfig` | `StorageBackend` | Must be "local" or "azure" |
| `MergeConfig` | `CheckpointInterval` | Must be positive |
| `MergeCheckpoint` | `Version` | Must be "1.0" (supported version) |

---

## JSON Schemas

### Scrobble JSON Example

```json
{
  "artist": "The Beatles",
  "album": "Abbey Road",
  "title": "Come Together",
  "timestamp": 1735689600,
  "duration": 259,
  "mbid": "f3d8e9a0-1234-5678-9abc-def012345678",
  "artist_mbid": "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d",
  "album_mbid": "df7d1c7f-1234-5678-9abc-def012345678"
}
```

### MergeCheckpoint JSON Example

```json
{
  "version": "1.0",
  "created_at": "2026-01-06T10:00:00Z",
  "updated_at": "2026-01-06T10:15:30Z",
  "strategy": "default",
  "conflict_resolution": "completeness",
  "input_files": [
    "/data/scrobbles-2023.ndjson",
    "/data/scrobbles-2024.ndjson"
  ],
  "output_path": "/data/merged.json",
  "processed_files": ["/data/scrobbles-2023.ndjson"],
  "current_file": "/data/scrobbles-2024.ndjson",
  "current_line": 50000,
  "dedup_map": {
    "a1b2c3...": 0,
    "d4e5f6...": 1
  },
  "scrobbles": [
    { "artist": "...", "title": "...", "timestamp": 123456 }
  ],
  "stats": {
    "total_files": 2,
    "processed_files": 1,
    "total_scrobbles": 150000,
    "unique_scrobbles": 145000,
    "duplicates": 5000
  }
}
```

---

## Performance Considerations

### Memory Usage

| Entity | Size per Instance | Count (1M scrobbles) | Total |
|--------|-------------------|----------------------|-------|
| `Scrobble` | ~300 bytes | 1,000,000 | ~300 MB |
| `DeduplicationMap` key | ~80 bytes | 1,000,000 | ~80 MB |
| `MergeCheckpoint` (serialized) | ~380 MB | 1 | ~380 MB |

**Total**: ~380 MB for 1M scrobbles (within 500 MB budget)

### Optimization Strategies

1. **Store pointers in map**: `map[string]*Scrobble` instead of `map[string]Scrobble` to avoid struct copying
2. **Streaming output** (future): Write scrobbles incrementally instead of buffering all in memory
3. **Checkpoint compression** (future): gzip checkpoint files to reduce disk usage
4. **Scanner buffer tuning**: Use 128KB buffer for better throughput with large NDJSON files

---

## Next Steps

1. Generate [contracts/merge-command.md](contracts/merge-command.md) - CLI interface specification
2. Generate [quickstart.md](quickstart.md) - Developer guide
3. Update `.github/copilot-instructions.md` - Add merge feature context

---

**Data Model Complete** ✅  
All entities, relationships, and validation rules documented. Ready for contract definition.
