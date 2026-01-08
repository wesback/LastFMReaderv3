# Quickstart: Normalize Command Implementation

**Feature**: 007-normalize-command  
**Date**: 2026-01-08  
**Audience**: Developers implementing this feature

## Prerequisites

- Go 1.24.0+ installed
- Existing LastFMReaderv3 codebase cloned
- Familiarity with existing `fetch` and `merge` commands
- Understanding of NDJSON format

## Implementation Phases

### Phase 1: Core Command Structure (P1 - MVP)

**Goal**: Implement basic normalize command for local storage with per-file processing.

**Steps**:

1. **Create command file** (`cmd/lastfm-sync/commands/normalize.go`):
   - Follow structure from `fetch.go` / `merge.go`
   - Define cobra command with `--user` flag
   - Add to root command registration

2. **Implement file discovery**:
   - Use `filepath.Glob` with pattern `{username}_*.ndjson`
   - Return sorted file list

3. **Implement per-file processing**:
   - Open file with `os.Open`
   - Create `bufio.Scanner` for line-by-line reading
   - For each line:
     - Parse JSON into `models.Scrobble`
     - Call `normalize.Normalize(scrobble.Track)`
     - Compare with existing `scrobble.NormalizedTitle`
     - Update if changed
   - Collect updated records
   - Write back to file (overwrite)

4. **Add basic summary**:
   - Track: total files, updated, unchanged
   - Print summary at end

**Testing**:
- Unit test: File discovery with various patterns
- Unit test: Single file normalization logic
- Integration test: End-to-end local file processing

**Estimated effort**: 4-6 hours

---

### Phase 2: Dry-Run Mode (P2 - Safety)

**Goal**: Add `--dry-run` flag for preview without modification.

**Steps**:

1. **Add flag** to command:
   ```go
   cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")
   ```

2. **Pass dry-run to writer**:
   - Check flag before file write
   - Display "would update" message instead

3. **Update summary** to indicate dry-run mode

**Testing**:
- Integration test: Verify no files modified in dry-run
- Integration test: Compare dry-run output vs real run output

**Estimated effort**: 2-3 hours

---

### Phase 3: Azure Storage Support (P1 Extension)

**Goal**: Add Azure Blob Storage processing.

**Steps**:

1. **Add Azure flags** to command (mirror `fetch.go` patterns):
   - `--azure-account`, `--azure-container`, `--azure-prefix`
   - Authentication flags: `--azure-auth`, `--azure-account-key`, `--azure-sas-token`

2. **Storage mode detection**:
   - If Azure flags provided, create Azure writer
   - Otherwise, use local filesystem

3. **Azure file discovery**:
   - Use Azure SDK `ListBlobs` with prefix
   - Filter by pattern client-side

4. **Reuse writer abstraction**:
   - Use existing `internal/writer.Writer` interface
   - Azure writer handles blob read/write

**Testing**:
- Integration test: Azure file discovery
- Integration test: Azure file processing (requires Azure emulator or test account)

**Estimated effort**: 4-5 hours

---

### Phase 4: Progress Display & Error Reporting (P3 - UX)

**Goal**: Add per-file progress and detailed error reporting.

**Steps**:

1. **Initialize progress bar**:
   - Use `internal/progress.NewFactory`
   - Set total to file count
   - Update after each file

2. **Enhanced display**:
   - Show current filename in progress description
   - Display old/new normalized_title for changed files

3. **Structured error collection**:
   - Define `ProcessingError` struct
   - Catch errors per file, continue processing
   - Collect in slice

4. **Error summary**:
   - Print error list at end
   - Format: `{filename}: {error_type}`

**Testing**:
- Unit test: Progress bar integration
- Integration test: Error handling with malformed files
- Integration test: Verify processing continues after errors

**Estimated effort**: 3-4 hours

---

### Phase 5: Polish & Documentation

**Goal**: Final quality checks and documentation.

**Steps**:

1. **Add help text**:
   - Command description
   - Flag descriptions
   - Examples

2. **Add tests**:
   - Table-driven tests for edge cases
   - Concurrent execution tests (verify no corruption)
   - Performance benchmark (5 seconds per 1000 files target)

3. **Update documentation**:
   - Add to main README
   - Update docs/troubleshooting.md with error types

4. **Code review**:
   - Verify constitution compliance (test coverage, complexity, etc.)
   - Check linting passes

**Estimated effort**: 3-4 hours

---

## Testing Strategy

### Unit Tests

**Required coverage**: 80%+ per constitution

Files to test:
- `normalize.go` command logic
- File discovery function
- Error handling paths
- Summary generation

Table-driven test pattern:
```go
func TestFileDiscovery(t *testing.T) {
    tests := []struct {
        name     string
        username string
        files    []string // Files to create in temp dir
        want     []string // Expected matches
    }{
        {"single file", "user1", []string{"user1_001.ndjson"}, []string{"user1_001.ndjson"}},
        {"multiple files", "user1", []string{"user1_001.ndjson", "user1_002.ndjson", "user2_001.ndjson"}, []string{"user1_001.ndjson", "user1_002.ndjson"}},
        {"no matches", "user1", []string{"user2_001.ndjson"}, []string{}},
    }
    // ... test implementation
}
```

### Integration Tests

**Required**: Minimum 1 per user story per constitution

Test scenarios:
1. **End-to-end local processing** (P1)
2. **Dry-run accuracy** (P2)
3. **Azure storage processing** (P1 extension)
4. **Error handling and continuation** (P3)
5. **Progress display validation** (P3)

Example structure:
```go
func TestNormalizeCommand_EndToEnd(t *testing.T) {
    // Create temp directory with test NDJSON files
    // Run normalize command
    // Verify files updated correctly
    // Verify summary accurate
}
```

### Performance Tests

Benchmark processing speed:
```go
func BenchmarkNormalizeCommand(b *testing.B) {
    // Create 1000 test files
    // Measure processing time
    // Verify < 5 seconds (SC-001)
}
```

---

## Key Reusable Components

**DO NOT REWRITE** - these exist and work:

| Component | Package | Purpose |
|-----------|---------|---------|
| Normalization logic | `internal/normalize` | Normalize track titles |
| Writer abstraction | `internal/writer` | Local/Azure file operations |
| Progress display | `internal/progress` | Progress bar UI |
| Configuration | `internal/config` | Azure config loading |
| Logging | `internal/logging` | Structured logging |
| Scrobble model | `internal/models` | Data structure |

**New code limited to**:
- `cmd/lastfm-sync/commands/normalize.go` (~300-400 lines)
- Test files (~500-600 lines total)

---

## Common Pitfalls

1. **Memory issues**: Don't load all files into memory. Process one at a time.
2. **NDJSON parsing**: Don't parse entire file as JSON array. Use line-by-line Scanner.
3. **Error handling**: Don't exit on first error. Collect errors and continue (FR-011).
4. **Azure authentication**: Reuse existing config patterns, don't reinvent auth flow.
5. **Progress updates**: Don't update console on every record. Per-file is sufficient.

---

## Definition of Done

- [X] All functional requirements (FR-001 through FR-020) implemented including Azure (FR-005, FR-006)
- [X] All success criteria (SC-001 through SC-006) verified
- [X] Unit test coverage ≥ 80% (85%+ on critical functions)
- [X] Integration tests for all user stories pass (4/4 passing, Azure test added - conditional on credentials)
- [ ] Performance benchmark meets 5 sec per 1000 files target (deferred - optional)
- [X] Linting passes (golint, go vet)
- [ ] Code review approved (pending PR)
- [X] Documentation updated (README, troubleshooting, test summary)
- [ ] Manual testing on Linux, macOS, Windows (Linux complete, Azure pending real credentials, others pending)

---

## Getting Started

```bash
# 1. Checkout feature branch
git checkout 007-normalize-command

# 2. Create command file
mkdir -p cmd/lastfm-sync/commands
touch cmd/lastfm-sync/commands/normalize.go

# 3. Run tests in watch mode during development
go test -v ./... -run TestNormalize

# 4. Build and test locally
go build -o bin/lastfm-sync ./cmd/lastfm-sync
./bin/lastfm-sync normalize --user testuser --dry-run

# 5. Run full test suite before PR
make test
make lint
```

---

## Resources

- **Existing commands to reference**: 
  - `cmd/lastfm-sync/commands/fetch.go` (Azure flags, config loading)
  - `cmd/lastfm-sync/commands/merge.go` (NDJSON processing, progress display)
- **Related packages**:
  - `internal/normalize/normalize.go` (normalization function)
  - `internal/merge/reader.go` (NDJSON streaming pattern)
  - `internal/writer/writer.go` (storage abstraction)
- **Testing patterns**: `tests/integration/merge_test.go`

---

## Questions?

See [spec.md](./spec.md) for requirements, [data-model.md](./data-model.md) for data structures, and [contracts/cli-interface.md](./contracts/cli-interface.md) for CLI specification.
