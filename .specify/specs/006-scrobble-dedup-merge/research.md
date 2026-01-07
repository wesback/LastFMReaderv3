# Research: Scrobble Deduplication & Merging

**Feature**: 006-scrobble-dedup-merge  
**Phase**: 0 (Research)  
**Date**: 2026-01-06

## Purpose

Research technical approaches, library choices, and implementation patterns for merging multiple NDJSON scrobble files into a single deduplicated JSON output. Focus on Go best practices for streaming processing, hash-based deduplication, and atomic file operations.

---

## Research Areas

### 1. NDJSON Streaming Parsing in Go

**Question**: What's the most memory-efficient approach to parse large NDJSON files line-by-line?

**Options Evaluated**:

1. **bufio.Scanner** (standard library)
   - ✅ Zero external dependencies
   - ✅ Built-in line splitting with `Scanner.Text()`
   - ✅ Configurable buffer size via `Scanner.Buffer()`
   - ⚠️ Default 64KB buffer limit (can be increased)
   - Pattern: `scanner := bufio.NewScanner(file); scanner.Scan(); json.Unmarshal(scanner.Bytes(), &scrobble)`

2. **encoding/json.Decoder** (standard library)
   - ✅ Stream-based, can decode one object at a time
   - ❌ Expects valid JSON array or object, not NDJSON format
   - ❌ Requires custom delimiter handling for newlines

3. **Third-party NDJSON libraries** (e.g., github.com/ndjson/ndjson-go)
   - ⚠️ Adds dependency for minimal value
   - ✅ Cleaner API for NDJSON-specific parsing
   - ❌ Project has low activity/maintenance

**Decision**: **Use bufio.Scanner + json.Unmarshal**  
**Rationale**: Standard library solution, zero dependencies, proven pattern in Go ecosystem. Scanner handles line splitting, Unmarshal handles JSON parsing. Buffer size can be tuned for performance (e.g., 128KB for large scrobbles). Error handling straightforward with `scanner.Err()`.

**Code Pattern**:
```go
scanner := bufio.NewScanner(file)
buf := make([]byte, 0, 128*1024) // 128KB buffer
scanner.Buffer(buf, 1024*1024)    // 1MB max line size

for scanner.Scan() {
    var scrobble models.Scrobble
    if err := json.Unmarshal(scanner.Bytes(), &scrobble); err != nil {
        // Handle parse error with line number
        continue
    }
    // Process scrobble
}
if err := scanner.Err(); err != nil {
    // Handle scanner error
}
```

---

### 2. In-Memory Hash Map for Deduplication

**Question**: How to efficiently store and lookup deduplication keys (SHA256 hashes) for 1M+ scrobbles while staying under 500MB memory?

**Options Evaluated**:

1. **map[string]*models.Scrobble** (standard library)
   - ✅ Built-in, fast lookups O(1) average
   - ✅ Simple API: `dedupMap[key] = &scrobble`
   - Memory estimate: ~300 bytes/scrobble × 1M = 300MB (within budget)
   - Pattern: Use SHA256 hex string as key (64 chars)

2. **map[[32]byte]*models.Scrobble** (byte array keys)
   - ✅ Slightly more memory-efficient (no string overhead)
   - ❌ Less readable, requires `hex.EncodeToString()` for logging
   - Memory savings: ~24 bytes/scrobble × 1M = 24MB (marginal)

3. **Third-party hash tables** (e.g., github.com/cornelk/hashmap)
   - ⚠️ Lock-free concurrent map (overkill for single-threaded processing)
   - ❌ Adds dependency for minimal benefit

**Decision**: **Use map[string]*models.Scrobble with string keys**  
**Rationale**: Standard library map is sufficient. Memory usage well within 500MB budget. String keys simplify logging/debugging (can print hex hash directly). Storing pointers avoids copying large structs. Concurrent access not needed (single-threaded processing).

**Memory Optimization**:
- Store pointers to avoid struct copying
- Consider `delete(dedupMap, key)` for already-written scrobbles if streaming to output (trade-off: disables checkpoint resume)
- Profile with `pprof` if memory becomes issue

---

### 3. SHA256 Key Generation for Deduplication

**Question**: What fields should be hashed for each deduplication strategy?

**Strategies Defined** (from spec.md FR-DEDUP-002):

| Strategy | Fields Hashed | Use Case |
|----------|--------------|----------|
| `default` | Artist + Album + Title + Timestamp | Standard deduplication (recommended) |
| `strict` | Artist + Album + Title + Timestamp + Duration | Exact match including duration |
| `relaxed` | Artist + Title + Timestamp (no Album) | Handles album metadata inconsistencies |
| `mbid` | MusicBrainz Track ID (if present) | Authoritative music database IDs |

**Implementation Pattern**:
```go
func GenerateKey(s *models.Scrobble, strategy string) string {
    h := sha256.New()
    
    switch strategy {
    case "strict":
        h.Write([]byte(strings.ToLower(s.Artist)))
        h.Write([]byte(strings.ToLower(s.Album)))
        h.Write([]byte(strings.ToLower(s.Title)))
        h.Write([]byte(fmt.Sprintf("%d", s.Timestamp)))
        h.Write([]byte(fmt.Sprintf("%d", s.Duration)))
    case "relaxed":
        h.Write([]byte(strings.ToLower(s.Artist)))
        h.Write([]byte(strings.ToLower(s.Title)))
        h.Write([]byte(fmt.Sprintf("%d", s.Timestamp)))
    case "mbid":
        if s.MusicBrainzTrackID != "" {
            h.Write([]byte(s.MusicBrainzTrackID))
            h.Write([]byte(fmt.Sprintf("%d", s.Timestamp)))
        } else {
            // Fallback to default strategy
            return GenerateKey(s, "default")
        }
    default: // "default"
        h.Write([]byte(strings.ToLower(s.Artist)))
        h.Write([]byte(strings.ToLower(s.Album)))
        h.Write([]byte(strings.ToLower(s.Title)))
        h.Write([]byte(fmt.Sprintf("%d", s.Timestamp)))
    }
    
    return hex.EncodeToString(h.Sum(nil))
}
```

**Key Decisions**:
- **Case normalization**: `strings.ToLower()` prevents "The Beatles" vs "the beatles" duplicates
- **Field ordering**: Consistent order ensures same hash for same values
- **MBID fallback**: If MusicBrainz ID missing, use default strategy (prevents empty keys)
- **Separator**: No explicit separator needed (SHA256 provides collision resistance)

---

### 4. Conflict Resolution (Completeness Scoring)

**Question**: When duplicate keys are found, which scrobble should be kept?

**Spec Requirement** (FR-DEDUP-004): Select scrobble with most complete metadata.

**Completeness Algorithm**:
```go
func CompletenessScore(s *models.Scrobble) int {
    score := 0
    if s.Artist != "" { score++ }
    if s.Album != "" { score++ }
    if s.Title != "" { score++ }
    if s.Timestamp > 0 { score++ }
    if s.Duration > 0 { score++ }
    if s.MusicBrainzTrackID != "" { score += 2 } // Extra weight for MBID
    if s.MusicBrainzArtistID != "" { score++ }
    if s.MusicBrainzAlbumID != "" { score++ }
    // Add more optional fields as needed
    return score
}

func ResolveConflict(existing, new *models.Scrobble) *models.Scrobble {
    existingScore := CompletenessScore(existing)
    newScore := CompletenessScore(new)
    
    if newScore > existingScore {
        return new
    } else if newScore == existingScore {
        // Tie-breaker: prefer newer timestamp (later discovery assumed more accurate)
        if new.Timestamp >= existing.Timestamp {
            return new
        }
    }
    return existing
}
```

**Design Notes**:
- MusicBrainz IDs weighted higher (authoritative source)
- Tie-breaker: prefer later timestamp (assumption: later exports may have corrections)
- Alternative tie-breaker: prefer first-seen (stable deduplication)
- Log conflicts at DEBUG level for transparency

---

### 5. Atomic File Writes

**Question**: How to ensure output file isn't corrupted if process crashes during write?

**Pattern**: **Temporary File + Atomic Rename**

**Standard Go Pattern**:
```go
// 1. Write to temporary file in same directory
tmpFile, err := os.CreateTemp(filepath.Dir(outputPath), ".merge-*.json.tmp")
if err != nil {
    return err
}
tmpPath := tmpFile.Name()
defer os.Remove(tmpPath) // Cleanup on error

// 2. Write full JSON output
encoder := json.NewEncoder(tmpFile)
encoder.SetIndent("", "  ") // Pretty-print
if err := encoder.Encode(scrobbles); err != nil {
    tmpFile.Close()
    return err
}
if err := tmpFile.Close(); err != nil {
    return err
}

// 3. Atomic rename (OS-level atomic operation on Unix/Linux)
if err := os.Rename(tmpPath, outputPath); err != nil {
    return err
}
```

**Key Properties**:
- ✅ `os.Rename()` is atomic on Unix/Linux when src/dst on same filesystem
- ✅ Temporary file in same directory ensures same filesystem
- ✅ `defer os.Remove()` cleans up temp file on error
- ✅ Existing file (if any) replaced atomically
- ⚠️ Windows: `os.Rename()` not atomic if destination exists (acceptable trade-off)

**Azure Blob Storage**: Use Azure SDK's atomic write features (see internal/writer/azure.go for existing patterns).

---

### 6. Checkpoint Format for Resume Capability

**Question**: What state must be saved to resume interrupted merge operations?

**Spec Requirement** (FR-MERGE-005): Save progress every N scrobbles to checkpoint file.

**Checkpoint Data Structure**:
```go
type MergeCheckpoint struct {
    Version           string              `json:"version"`          // Checkpoint format version
    Strategy          string              `json:"strategy"`         // Deduplication strategy
    InputFiles        []string            `json:"input_files"`      // Ordered list of input files
    ProcessedFiles    []string            `json:"processed_files"`  // Files fully processed
    CurrentFile       string              `json:"current_file"`     // File being processed
    CurrentLineNumber int                 `json:"current_line"`     // Line number in current file
    DeduplicationMap  map[string]int      `json:"dedup_map"`        // Key -> index in Scrobbles array
    Scrobbles         []*models.Scrobble  `json:"scrobbles"`        // Deduplicated scrobbles so far
    TotalProcessed    int                 `json:"total_processed"`  // Total scrobbles read
    Duplicates        int                 `json:"duplicates"`       // Duplicate count
    CreatedAt         time.Time           `json:"created_at"`       // Checkpoint creation time
}
```

**Checkpoint File Lifecycle**:
1. **Initialize**: Create checkpoint file at merge start
2. **Update**: Save progress every 10,000 scrobbles (configurable)
3. **Resume**: On `--resume` flag, load checkpoint and skip processed files
4. **Cleanup**: Delete checkpoint file after successful merge completion

**Storage Location**:
- Local: `.merge-checkpoint-{timestamp}.json` in current directory
- Azure: Not supported (ephemeral environment, checkpointing for local development only)

**Serialization**: JSON format for human readability and easy debugging.

**Alternative Considered**: Binary format (gob encoding) for speed - rejected for debugging complexity.

---

### 7. Progress Bar Integration

**Question**: How to integrate with existing `internal/progress` package (feature 005)?

**Existing API** (from [internal/progress/reporter.go](internal/progress/reporter.go)):
```go
type Reporter interface {
    Start()
    Update(current, total int64, message string)
    Finish(message string)
}
```

**Integration Pattern**:
```go
// During merge initialization
progressBar := progress.NewBar(progress.Options{
    Total:       totalEstimatedScrobbles, // Sum of file sizes / avg scrobble size
    Description: "Merging scrobbles",
    ShowRate:    true,
})
progressBar.Start()

// In processing loop
for each scrobble {
    // Process scrobble
    processedCount++
    
    if processedCount % 100 == 0 { // Update every 100 scrobbles
        progressBar.Update(
            int64(processedCount),
            int64(totalEstimatedScrobbles),
            fmt.Sprintf("Processed %d files, %d duplicates", filesProcessed, duplicateCount),
        )
    }
}

progressBar.Finish(fmt.Sprintf("Merged %d scrobbles (%d duplicates removed)", uniqueCount, duplicateCount))
```

**Total Estimation Strategy**:
- Count total lines across all input files before processing (fast pre-scan)
- Or estimate based on average file size (less accurate but faster startup)

---

### 8. Error Recovery Strategies

**Question**: How to handle malformed NDJSON lines without aborting entire merge?

**Error Categories**:

| Error Type | Example | Recovery Strategy |
|------------|---------|-------------------|
| Invalid JSON syntax | `{"artist": "Test"` (unclosed brace) | Log warning, skip line, continue processing |
| Missing required fields | `{"album": "Test"}` (no artist/title) | Log warning, skip scrobble, continue |
| Invalid timestamp | `{"timestamp": -1}` | Log warning, use current time, continue |
| File read error | Permission denied | Abort file, continue with remaining files |
| Out of memory | Heap exhaustion | Save checkpoint, abort with error |

**Implementation**:
```go
lineNumber := 0
for scanner.Scan() {
    lineNumber++
    
    var scrobble models.Scrobble
    if err := json.Unmarshal(scanner.Bytes(), &scrobble); err != nil {
        logger.Warn("Invalid JSON on line",
            zap.String("file", currentFile),
            zap.Int("line", lineNumber),
            zap.Error(err),
        )
        stats.SkippedLines++
        continue // Skip malformed line
    }
    
    if err := scrobble.Validate(); err != nil {
        logger.Warn("Invalid scrobble on line",
            zap.String("file", currentFile),
            zap.Int("line", lineNumber),
            zap.Error(err),
        )
        stats.SkippedScrobbles++
        continue // Skip invalid scrobble
    }
    
    // Process valid scrobble
}
```

**Logging Strategy**: Use structured logging (zap) with file/line context for debugging.

---

## Technology Recommendations

| Area | Technology | Rationale |
|------|------------|-----------|
| NDJSON Parsing | `bufio.Scanner` + `encoding/json` | Standard library, proven pattern |
| Deduplication Map | `map[string]*models.Scrobble` | Built-in, sufficient performance |
| Hash Algorithm | `crypto/sha256` | Standard library, collision-resistant |
| Atomic Writes | `os.CreateTemp()` + `os.Rename()` | Standard library, atomic on Unix/Linux |
| Checkpoint Format | JSON with `encoding/json` | Human-readable, easy debugging |
| Progress Reporting | `internal/progress` (existing) | Already integrated, consistent UX |
| Structured Logging | `go.uber.org/zap` (existing) | High-performance, structured |
| CLI Framework | `github.com/spf13/cobra` (existing) | Consistent with fetch command |

**Zero New Dependencies**: All core functionality uses Go standard library + existing project dependencies.

---

## Performance Benchmarks (Target)

From spec.md (SC-PERF-001, SC-PERF-002):

| Metric | Target | Measurement Strategy |
|--------|--------|---------------------|
| Processing Rate | ≥ 10,000 scrobbles/sec | Benchmark test with 100K synthetic scrobbles |
| Memory Usage | < 500MB for 1M scrobbles | Measure with `runtime.MemStats` and `pprof` |
| Startup Time | < 1 second for file discovery | Time from command invocation to first scrobble processed |

**Benchmark Test Structure**:
```go
func BenchmarkMerge(b *testing.B) {
    // Generate 100K synthetic scrobbles
    testData := generateTestScrobbles(100_000)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        merger := NewMerger(/* ... */)
        merger.Merge(testData)
    }
}

func TestMemoryUsage(t *testing.T) {
    // Generate 1M scrobbles
    testData := generateTestScrobbles(1_000_000)
    
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    before := m.Alloc
    
    merger := NewMerger(/* ... */)
    merger.Merge(testData)
    
    runtime.ReadMemStats(&m)
    after := m.Alloc
    
    used := (after - before) / 1024 / 1024 // MB
    assert.Less(t, used, 500, "Memory usage exceeds 500MB")
}
```

---

## Open Questions

| Question | Impact | Research Plan |
|----------|--------|---------------|
| Should checkpoint files support compression (gzip)? | Medium - 10x smaller checkpoints but slower I/O | Prototype both, benchmark with 1M scrobbles |
| Should deduplication map use consistent hashing for distributed processing? | Low - single-machine processing sufficient for MVP | Defer to future feature if needed |
| How to handle timezone differences in timestamps? | Low - Last.fm API returns Unix timestamps (UTC) | Document assumption, validate in tests |
| Should we support streaming output (write scrobbles as discovered)? | High - reduces memory usage but prevents checkpoint resume | Trade-off: analyze use cases, decide in Phase 1 |

**Decision for MVP**: No compression, no distributed processing, assume UTC timestamps, in-memory processing with checkpoint support.

---

## Next Steps (Phase 1)

1. Generate [data-model.md](data-model.md) with Go struct definitions
2. Create [contracts/merge-command.md](contracts/merge-command.md) with CLI interface specification
3. Write [quickstart.md](quickstart.md) with developer guide
4. Update `.github/copilot-instructions.md` with merge feature context

---

**Research Phase Complete** ✅  
All technical decisions documented. Ready for Phase 1 (Design).
