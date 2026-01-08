# Research: Normalize Command

**Feature**: 007-normalize-command  
**Date**: 2026-01-08  
**Phase**: 0 (Outline & Research)

## Research Areas

### 1. File Discovery Pattern Implementation

**Decision**: Use filepath.Glob for local storage, Azure SDK list operations for Azure storage

**Rationale**: 
- `filepath.Glob` is Go standard library, handles pattern `{username}_*.ndjson` efficiently
- Azure SDK's `ListBlobs` with prefix filter provides equivalent functionality
- Both approaches already used in existing fetch/merge commands, proven pattern
- No additional dependencies needed

**Alternatives Considered**:
- **filepath.Walk**: More flexible but overkill for single-pattern matching, slower for large directories
- **Manual directory reading**: More control but reinvents stdlib functionality
- **Third-party glob library**: Unnecessary complexity when stdlib sufficient

**Implementation Notes**:
- Local: `filepath.Glob(filepath.Join(baseDir, username+"_*.ndjson"))`
- Azure: List blobs with prefix filter, client-side pattern validation if needed
- Reuse existing reader/writer abstraction from fetch/merge commands

---

### 2. NDJSON Streaming Best Practices

**Decision**: Use `bufio.Scanner` with line-by-line processing

**Rationale**:
- NDJSON format is one JSON object per line, perfectly suited for Scanner
- Scanner handles line buffering automatically, minimal memory per file
- Already used in existing merge command's reader implementation
- Memory footprint: O(1) per record, not O(n) for entire file
- Enables processing files > available RAM

**Alternatives Considered**:
- **Load entire file into memory**: Simple but fails for large files, violates SC-001 performance target
- **io.Reader with manual buffering**: More control but reinvents Scanner functionality
- **Streaming JSON parser**: Complex, unnecessary when NDJSON guarantees line-delimited structure

**Implementation Notes**:
- Reuse existing `internal/merge/reader.go` patterns
- Error handling: Log malformed lines, continue processing (FR-011)
- Buffer size: Use Scanner's default (64KB), sufficient for typical scrobble records

---

### 3. Progress Display Without Performance Degradation

**Decision**: Per-file progress updates using existing progress bar library, batch console writes

**Rationale**:
- Existing `internal/progress` package provides proven implementation
- Per-file updates satisfy FR-009 and clarification decision
- Performance target (5 seconds per 1000 files) = ~200 files/sec = ~5ms per file
- Console I/O batched to avoid blocking file processing
- Progress library handles terminal width detection and updates efficiently

**Alternatives Considered**:
- **Batch updates (every N files)**: Rejected per clarification session - user chose per-file
- **Percentage-based**: Less informative for troubleshooting specific file issues
- **No progress display**: Poor UX for long-running operations

**Implementation Notes**:
- Initialize progress bar with total file count
- Update after each file processed: `bar.Add(1)` 
- Display current filename in progress bar description
- Progress library already handles efficient console updates (doesn't print every increment)

---

### 4. Dry-Run Implementation Pattern

**Decision**: Mode flag throughout call chain, write path check before flush

**Rationale**:
- Simple boolean flag passed to writer abstraction
- Writer layer (local/Azure) checks dry-run flag before actual write operations
- Allows full processing logic to execute, validating correctness without side effects
- Matches existing patterns in codebase (merge command has similar flag handling)

**Alternatives Considered**:
- **Mock writer interface**: More complex, harder to ensure identical behavior
- **Separate code paths for dry-run vs. real**: High duplication risk, violates DRY
- **Transaction with rollback**: Overkill for file operations, doesn't work well with Azure

**Implementation Notes**:
- Add `dryRun bool` to normalize command struct
- Pass to writer factory
- Writer interface: `Write()` method no-ops when dry-run active
- Summary report indicates dry-run mode status (FR-013)

---

### 5. Error Reporting with File Path and Type

**Decision**: Structured error type with file path, operation, and error message

**Rationale**:
- Satisfies clarification requirement: "file path and error type"
- Structured errors enable consistent formatting in summary
- Allows aggregation by error type for analysis
- Follows Go error handling best practices (wrap errors with context)

**Alternatives Considered**:
- **Simple string concatenation**: Less structured, harder to parse programmatically
- **Error codes**: Overkill for CLI tool, string descriptions more user-friendly
- **Full stack traces in summary**: Too verbose, belongs in debug logs not summary

**Implementation Notes**:
```go
type ProcessingError struct {
    FilePath string
    ErrorType string  // "parse_error", "missing_field", "permission_denied", etc.
    Details error
}
```
- Collect errors in slice during processing
- Format in summary: `{FilePath}: {ErrorType}` (per clarification)
- Full details to debug log level

---

### 6. Concurrent Execution Safety

**Decision**: No locking mechanism implemented, documented admin responsibility

**Rationale**:
- Clarification session determined: allow concurrent operations (Option B)
- Normalization is idempotent - running twice produces same result
- Read-modify-write race window minimal per file
- Complexity cost of distributed locking (especially Azure) not justified
- Worst case: duplicate work, not data corruption

**Alternatives Considered**:
- **File locking**: Doesn't work across local/Azure storage, complex edge cases
- **Distributed locks (Azure Lease)**: High complexity, failure modes (stale locks)
- **Operation queue**: Overkill for admin tool with manual coordination

**Implementation Notes**:
- Document in CLI help text: "Does not prevent concurrent execution. Administrators should coordinate."
- Add warning in dry-run output if concurrent execution detected (advisory only)
- No blocking or error on concurrent execution

---

## Summary

All research complete. Key decisions:
1. **File discovery**: filepath.Glob (local), Azure List with prefix (cloud)
2. **Streaming**: bufio.Scanner line-by-line processing
3. **Progress**: Per-file updates with existing progress library
4. **Dry-run**: Boolean flag with write-path check
5. **Error reporting**: Structured ProcessingError type
6. **Concurrency**: No locking, admin coordination

All decisions align with existing codebase patterns. Maximum code reuse achieved. No new dependencies required. Ready for Phase 1 design.
