# Data Model: Normalize Command

**Feature**: 007-normalize-command  
**Date**: 2026-01-08  
**Phase**: 1 (Design & Contracts)

## Entities

### Scrobble Record

Existing model from `internal/models/scrobble.go` - **REUSED, NOT MODIFIED**

```go
type Scrobble struct {
    Artist          string `json:"artist"`
    Track           string `json:"track"`
    Album           string `json:"album,omitempty"`
    AlbumArtist     string `json:"album_artist,omitempty"`
    Timestamp       int64  `json:"timestamp"`
    MusicBrainzID   string `json:"mbid,omitempty"`
    NormalizedTitle string `json:"normalized_title"` // Field updated by normalize command
    // ... other fields
}
```

**Relationships**: None (scrobble records are independent)

**Validation Rules**:
- `track` field MUST be present (FR-005 requirement)
- `timestamp` used for ordering (informational only)
- All other fields preserved unchanged (FR-006)

**State Transitions**:
- Initial: `normalized_title` may be empty, outdated, or current
- After normalization: `normalized_title` = Normalize(track)
- No state machine - single transformation per file

---

### Processing Error

New struct for structured error reporting (FR-012)

```go
type ProcessingError struct {
    FilePath  string
    ErrorType string // "parse_error", "missing_track_field", "permission_denied", "read_error", "write_error"
    Details   error
}
```

**Validation Rules**:
- `FilePath` MUST be the relative or absolute path to the failed file
- `ErrorType` MUST be one of the defined error type strings
- `Details` captures underlying error for logging

**Usage**: Collected during processing, reported in summary

---

### Processing Summary

New struct for summary report output (FR-010)

```go
type ProcessingSummary struct {
    TotalFiles     int
    UpdatedFiles   int
    UnchangedFiles int
    ErrorCount     int
    Errors         []ProcessingError
    DryRun         bool
    Duration       time.Duration
}
```

**Validation Rules**:
- `TotalFiles` = `UpdatedFiles` + `UnchangedFiles` + `ErrorCount` (SC-006)
- All counts ≥ 0
- `Errors` slice length MUST equal `ErrorCount`

**Display Format**:
```
Processing files for user: {username}
Storage: {Local|Azure}

Processing: {filename}
  Current: "{old_normalized_title}"
  New:     "{new_normalized_title}"
  Status:  {Would update|Updated|No change needed|Error: {error_type}}

Summary:
  Total files: {TotalFiles}
  Updated: {UpdatedFiles}
  Unchanged: {UnchangedFiles}
  Errors: {ErrorCount}
  Duration: {Duration}

{if DryRun}Dry-run mode: No changes written to storage{/if}

{if ErrorCount > 0}
Errors encountered:
  - {FilePath}: {ErrorType}
  ...
{/if}
```

---

### File Metadata

Ephemeral struct for file discovery and processing (not persisted)

```go
type FileMetadata struct {
    Path            string // Full path (local) or blob name (Azure)
    Size            int64  // File size in bytes
    Storage         string // "local" or "azure"
}
```

**Usage**: 
- Built during file discovery phase
- Passed to processing pipeline
- Not included in output or persisted

---

## Data Flow

```
1. File Discovery
   Input: username, storage config
   Output: []FileMetadata

2. Per-File Processing
   Input: FileMetadata
   Process: Read NDJSON → Parse lines → Normalize → Update records → Write/dry-run
   Output: ProcessingResult (updated/unchanged/error)

3. Summary Generation
   Input: []ProcessingResult
   Output: ProcessingSummary

4. Display
   Input: ProcessingSummary
   Output: Console text (formatted per template above)
```

---

## Memory Considerations

**Streaming Processing**:
- Files processed one at a time, not loaded into memory simultaneously
- Within each file, records processed line-by-line (NDJSON streaming)
- Maximum memory per file: O(largest_single_record) + buffer overhead

**Error Collection**:
- Errors accumulated in slice during processing
- Worst case: O(total_files) if all files error
- Acceptable given typical error rates << 10% per SC-005

**Summary**:
- Single ProcessingSummary struct held in memory
- Negligible compared to file processing memory

---

## Reuse of Existing Infrastructure

**NO NEW DATA MODELS NEEDED FOR**:
- File storage abstraction (`internal/writer.Writer` interface)
- Configuration (`internal/config` types)
- Normalization logic (`internal/normalize.Normalize` function)
- Progress display (`internal/progress.Reporter` interface)
- Logging (`internal/logging.Logger` interface)

All existing models and abstractions are sufficient. Only new structs are ProcessingError and ProcessingSummary for command-specific reporting.
