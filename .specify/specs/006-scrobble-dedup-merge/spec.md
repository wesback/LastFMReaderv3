# Feature Specification: Scrobble Deduplication and Merging

**Feature Branch**: `006-scrobble-dedup-merge`  
**Created**: January 7, 2026  
**Status**: Draft  
**Input**: User description: "Add functionality to read exported scrobble data from multiple NDJSON files, deduplicate entries, and write a single consolidated JSON file containing all unique scrobbles. This operation should work seamlessly with both local filesystem and Azure Blob Storage."

## User Scenarios & Testing

### User Story 1 - Basic Scrobble Merge (Priority: P1)

As a Last.fm user with multiple export files, I want to merge all my scrobble data into a single file so that I can analyze my complete listening history without duplicates.

**Why this priority**: This is the core value proposition. Users need a single, deduplicated view of their scrobbles for analysis, backup, and downstream processing.

**Independent Test**: Can be fully tested by running the merge command on a set of NDJSON files and verifying the output contains all unique scrobbles sorted by timestamp. Delivers immediate value by consolidating scattered data.

**Acceptance Scenarios**:

1. **Given** 5 NDJSON files containing 10,000 total scrobbles with 1,000 duplicates, **When** user runs merge command with local storage, **Then** output file contains exactly 9,000 unique scrobbles sorted by timestamp
2. **Given** NDJSON files on Azure Blob Storage, **When** user runs merge with Azure storage backend, **Then** merged file is written to Azure with all unique scrobbles
3. **Given** NDJSON files with overlapping timestamps, **When** merge completes, **Then** output excludes raw field and is valid JSON array format
4. **Given** two identical scrobbles (same artist, normalized title, timestamp), **When** deduplication runs, **Then** only one copy appears in output
5. **Given** merge operation in progress, **When** processing files, **Then** progress indicator shows current file, scrobbles read, and duplicates found

---

### User Story 2 - Handle Data Quality Issues (Priority: P2)

As a user with imperfect export data, I want the merge tool to handle malformed or incomplete records gracefully so that I don't lose all my data due to a few bad records.

**Why this priority**: Real-world data is messy. Users need confidence that the tool won't fail catastrophically on minor issues.

**Independent Test**: Can be tested by creating NDJSON files with intentional errors (invalid JSON, missing fields) and verifying the tool skips bad records while processing good ones, with clear error reporting.

**Acceptance Scenarios**:

1. **Given** NDJSON file with line 50 containing invalid JSON syntax, **When** merge processes the file, **Then** line 50 is skipped with logged error and processing continues with remaining lines
2. **Given** scrobble record missing required field (artist), **When** validation runs, **Then** record is skipped with warning logged including line number and file name
3. **Given** scrobble with zero/negative timestamp (uts: -1), **When** processed, **Then** uses sentinel value (0), logs warning, and includes in output at beginning
4. **Given** merge operation with 100 malformed records out of 50,000, **When** complete, **Then** summary shows 99.8% success rate and references error log file
5. **Given** scrobble missing normalized_title field, **When** generating deduplication key, **Then** falls back to track field with warning logged

---

### User Story 3 - Conflict Resolution and Data Quality (Priority: P2)

As a user with duplicate scrobbles that have different levels of completeness, I want the merge tool to keep the most complete version so that my final dataset has the best quality data.

**Why this priority**: Duplicates often arise from re-exports or API changes. Keeping the most complete record improves data quality.

**Independent Test**: Can be tested by creating duplicate scrobbles with varying completeness (one with album/MBID, one without) and verifying the most complete version is retained.

**Acceptance Scenarios**:

1. **Given** two duplicate scrobbles where one has album field and one doesn't, **When** conflict resolution runs, **Then** version with album is kept
2. **Given** two duplicate scrobbles with equal completeness but one has MusicBrainz ID, **When** conflict resolution runs, **Then** version with MBID is kept
3. **Given** two duplicate scrobbles identical except ingested_at timestamps, **When** conflict resolution runs, **Then** more recently ingested version is kept
4. **Given** 1,000 duplicate scrobbles, **When** merge completes, **Then** verbose mode shows conflict resolution decisions with field comparison scores
5. **Given** scrobbles with same track but different annotations ("Live", "Remastered"), **When** using default strategy, **Then** normalized_title causes them to be treated as duplicates

---

### User Story 4 - Preview and Validation (Priority: P3)

As a cautious user, I want to preview the merge operation before committing changes so that I can verify the results will be what I expect.

**Why this priority**: Provides safety and confidence before potentially destructive operations. Lower priority as the tool is non-destructive by default.

**Independent Test**: Can be tested by running dry-run mode and verifying no files are modified while statistics and previews are shown.

**Acceptance Scenarios**:

1. **Given** user runs merge with --dry-run flag, **When** operation completes, **Then** no output files are written and preview statistics are displayed
2. **Given** dry-run mode, **When** analyzing files, **Then** shows estimated duplicates, output size, and processing time
3. **Given** dry-run mode, **When** complete, **Then** lists all files that would be processed with size and estimated scrobble count
4. **Given** verbose mode enabled, **When** processing duplicates, **Then** logs show key generation, conflict comparison, and resolution decisions
5. **Given** merge completes, **When** summary displayed, **Then** shows files processed, scrobbles read, duplicates removed, date range, unique artists/tracks, output size, and processing time

---

### User Story 5 - Different Deduplication Strategies (Priority: P3)

As a power user, I want to choose different deduplication strategies based on my needs so that I can handle specific data scenarios (preserving annotations, handling API duplicates, etc.).

**Why this priority**: Provides flexibility for advanced use cases. Most users will be satisfied with default strategy.

**Independent Test**: Can be tested by running merge with different strategies on the same dataset and comparing outputs to verify strategy differences.

**Acceptance Scenarios**:

1. **Given** tracks with different annotations ("Come Together", "Come Together - Remastered"), **When** using default strategy, **Then** treated as duplicates (normalized_title)
2. **Given** same tracks with annotations, **When** using strict strategy, **Then** treated as separate records (preserves track differences)
3. **Given** API-generated duplicate scrobbles within 2 minutes, **When** using relaxed strategy, **Then** 5-minute window groups them as duplicates
4. **Given** scrobbles with varying MBID presence, **When** using mbid strategy, **Then** MusicBrainz IDs are preferred for matching when available
5. **Given** user specifies --dedup-strategy flag, **When** merge runs, **Then** summary indicates which strategy was used

---

### User Story 6 - Long-Running Operations (Priority: P3)

As a user with millions of scrobbles, I want the merge operation to support checkpointing and resume so that I can recover from interruptions without starting over.

**Why this priority**: Important for very large datasets but most users have smaller collections. Nice to have for reliability.

**Independent Test**: Can be tested by running merge on large dataset, interrupting (Ctrl+C), and resuming to verify it continues from checkpoint.

**Acceptance Scenarios**:

1. **Given** merge processing 5 million scrobbles with --checkpoint enabled, **When** interrupted after 60 seconds, **Then** checkpoint file is saved with current progress
2. **Given** checkpoint file exists from previous run, **When** user restarts merge, **Then** prompts to resume from checkpoint
3. **Given** user chooses to resume, **When** merge continues, **Then** starts from last processed file in checkpoint
4. **Given** merge completes successfully, **When** cleanup runs, **Then** checkpoint file is automatically deleted
5. **Given** corrupted checkpoint file, **When** attempting to load, **Then** shows error with suggestion to remove and restart fresh

---

### Edge Cases

- **Empty file set**: What happens when no files match the input pattern? → Error with helpful message suggesting to check pattern and path
- **All duplicates**: What happens when all scrobbles are duplicates? → Output contains only unique set, summary shows 100% duplicate rate
- **Zero duplicates**: What happens when no duplicates exist? → All scrobbles written to output, summary shows 0% duplicate rate
- **Identical timestamps, different tracks**: How are multiple tracks at exact same time handled? → Not duplicates (different keys), both retained with secondary sort by artist/title
- **Missing normalized_title**: What if normalized_title field is empty? → Falls back to track field for key generation with warning
- **Output file already exists**: What happens if merged file name conflicts? → Error requiring manual deletion or use of --output flag for different name
- **Storage quota exceeded**: How does system handle running out of disk space mid-write? → Error caught, temp file preserved, clear message with space requirement
- **Network timeout (Azure)**: How are Azure storage network issues handled? → Exponential backoff retry logic (5 attempts), then graceful failure with resume suggestion
- **Very large memory requirements**: What if dataset exceeds available memory? → Error with memory requirement estimate and suggestion to free space or process in batches
- **Malformed JSON throughout file**: What if most lines in a file are invalid? → Continues processing valid lines, accumulates error count, shows percentage in summary
- **Different usernames in files**: What happens if files contain multiple users? → Each username treated separately in key (no cross-user deduplication), optional warning
- **Conflicting album values in duplicates**: How to handle same track/timestamp but different album names? → Apply conflict resolution (completeness score), keep most complete record

## Requirements

### Functional Requirements

**File Discovery and Reading**
- **FR-001**: System MUST discover all NDJSON files matching a configurable pattern for a given username
- **FR-002**: System MUST read NDJSON files line-by-line for memory efficiency
- **FR-003**: System MUST parse each line as a JSON scrobble object
- **FR-004**: System MUST skip malformed JSON lines with logged error (file, line number, error details)
- **FR-005**: System MUST validate scrobble structure before processing (required fields: username, artist, track, normalized_title, uts)
- **FR-006**: System MUST support both local filesystem and Azure Blob Storage as input sources

**Deduplication Logic**
- **FR-007**: System MUST identify duplicate scrobbles using configurable unique key (default: username + artist + normalized_title + uts)
- **FR-008**: System MUST support multiple deduplication strategies: default (normalized_title), strict (track), relaxed (time windows), mbid (MusicBrainz ID)
- **FR-009**: System MUST apply conflict resolution when duplicates detected (preserve most complete record)
- **FR-010**: System MUST calculate completeness score based on field population (base fields + album + MBID with 2x weight)
- **FR-011**: System MUST prefer scrobble with MusicBrainz ID when completeness scores equal
- **FR-012**: System MUST prefer most recently ingested scrobble (ingested_at) when other factors equal
- **FR-013**: System MUST track and report number of duplicates found and removed
- **FR-014**: System MUST handle edge cases: null timestamps (use sentinel value 0), missing normalized_title (fall back to track)

**Merging and Output**
- **FR-015**: System MUST merge all unique scrobbles into single data structure
- **FR-016**: System MUST sort merged data by timestamp ascending (configurable to descending)
- **FR-017**: System MUST write output as valid JSON array of scrobbles
- **FR-018**: System MUST exclude raw field from output to reduce file size
- **FR-019**: System MUST support pretty-printed JSON (default) and compact JSON output
- **FR-020**: System MUST write to same storage backend as input (local or Azure)
- **FR-021**: System MUST use atomic writes (temporary file + rename) to prevent corruption
- **FR-022**: System MUST verify output file integrity after writing (valid JSON, size > 0, starts with '[', ends with ']')

**Storage Backend Support**
- **FR-023**: System MUST support reading from local filesystem with absolute or relative paths
- **FR-024**: System MUST support reading from Azure Blob Storage with connection string authentication
- **FR-025**: System MUST support Azure managed identity authentication
- **FR-026**: System MUST support writing to local filesystem
- **FR-027**: System MUST support writing to Azure Blob Storage
- **FR-028**: System MUST use consistent storage backend for input and output within single operation

**Progress and Reporting**
- **FR-029**: System MUST display progress while reading files (current file, scrobbles read, duplicates found)
- **FR-030**: System MUST display progress while writing output (records written)
- **FR-031**: System MUST report summary statistics: files processed, scrobbles read, duplicates removed, unique scrobbles, output size, processing time
- **FR-032**: System MUST report date range (earliest to latest scrobble)
- **FR-033**: System MUST report unique artists and tracks count
- **FR-034**: System MUST support verbose mode with detailed debug logging

**Error Handling and Recovery**
- **FR-035**: System MUST handle missing or inaccessible input files with clear error messages
- **FR-036**: System MUST handle storage backend errors (network, permissions, quota) with retry logic
- **FR-037**: System MUST handle out-of-memory scenarios gracefully with resource requirement estimates
- **FR-038**: System MUST support checkpointing for long-running operations (optional flag)
- **FR-039**: System MUST support resume from checkpoint after interruption
- **FR-040**: System MUST validate output before replacing existing merged file
- **FR-041**: System MUST provide detailed error messages with actionable guidance

**Command-Line Interface**
- **FR-042**: System MUST accept required --user flag for Last.fm username
- **FR-043**: System MUST accept --output flag (local or azure, default: local)
- **FR-044**: System MUST accept --out-path flag for output file location (default: "{username}.json")
- **FR-045**: System MUST accept input patterns as positional arguments for file matching
- **FR-046**: System MUST accept --strategy flag for deduplication strategy (default, strict, relaxed, mbid)
- **FR-047**: System MUST accept --conflict-resolution flag (completeness, first, last)
- **FR-048**: System MUST accept --verbose flag for detailed logging
- **FR-049**: System MUST accept --checkpoint-interval flag for periodic checkpointing
- **FR-050**: System MUST accept --resume flag to continue from checkpoint file
- **FR-051**: System MUST support 7 Azure configuration flags aligned with fetch command

### Key Entities

- **Scrobble**: Represents a single Last.fm listening event with fields: username (string), artist (string), track (string), normalized_title (string), album (string, optional), uts (int64 Unix timestamp), local_time (string RFC3339), mbid (string, optional), source (string), ingested_at (string RFC3339), raw (object, excluded from output)

- **DeduplicationMap**: In-memory hash map storing unique scrobbles, key is SHA256 hash of unique identifier (username + artist + normalized_title + uts), value is scrobble record pointer, provides O(1) average lookup and insertion

- **ProcessingState**: Tracks merge operation progress with fields: files_processed, files_total, scrobbles_read, duplicates_found, unique_scrobbles, current_file, start_time, bytes_processed, bytes_total

- **MergeConfig**: Configuration for merge operation with fields: username (required), input_pattern, output_filename, storage_backend (local/azure), azure_connection_string, azure_container, base_path, dedup_strategy, sort_order, dry_run, verbose, checkpoint_enabled, exclude_raw, pretty_print

- **MergeStats**: Summary statistics with fields: files_processed, scrobbles_read, duplicates_removed, unique_scrobbles, output_file, output_size_bytes, processing_time, earliest_scrobble, latest_scrobble, unique_artists, unique_tracks

- **CheckpointData**: Serializable state for resume capability with fields: last_processed_file_index, files_completed, deduplication_map_state, processing_statistics, checkpoint_timestamp

## Success Criteria

### Measurable Outcomes

**Performance and Scalability**
- **SC-001**: Users can merge 100,000 scrobbles in under 10 seconds on standard hardware
- **SC-002**: System processes at least 10,000 scrobbles per second
- **SC-003**: Memory usage remains below 500MB when processing 1 million scrobbles
- **SC-004**: System successfully handles datasets from 100 to 10 million scrobbles
- **SC-005**: System successfully processes 1 to 1000 input files

**Data Quality and Accuracy**
- **SC-006**: Duplicate detection accuracy exceeds 99.9% (less than 1 false positive or negative per 1000 duplicates)
- **SC-007**: No data loss occurs during processing (all unique scrobbles retained)
- **SC-008**: Output file contains valid JSON parseable by standard tools
- **SC-009**: Conflict resolution selects most complete scrobble in 100% of cases
- **SC-010**: All output scrobbles sorted correctly by timestamp

**Reliability and Error Handling**
- **SC-011**: Malformed JSON lines are skipped without crashing (100% graceful handling)
- **SC-012**: Azure network failures are retried up to 5 times with exponential backoff
- **SC-013**: Interrupted operations can resume from checkpoint without data loss
- **SC-014**: Atomic writes prevent corrupted output in 100% of cases
- **SC-015**: Output verification detects invalid files before finalization in 100% of cases

**Usability and User Experience**
- **SC-016**: Users complete basic merge operation with 3 or fewer command-line flags
- **SC-017**: Progress indication updates smoothly (at least once per second during processing)
- **SC-018**: Error messages include actionable suggestions in 100% of error scenarios
- **SC-019**: Dry-run mode provides accurate preview within 10% of actual results
- **SC-020**: Summary statistics are accurate and displayed within 1 second of completion

**Cross-Platform and Storage Support**
- **SC-021**: Operation completes successfully on Linux, macOS, and Windows
- **SC-022**: Local filesystem and Azure Blob Storage both supported with identical results
- **SC-023**: Azure authentication works with connection strings, managed identity, and SAS tokens

**Code Quality and Testing**
- **SC-024**: Test coverage exceeds 80% for all merge-related code
- **SC-025**: All unit tests pass on supported platforms
- **SC-026**: All integration tests pass for both local and Azure storage
- **SC-027**: Performance benchmarks meet or exceed targets

## Assumptions

### Data Assumptions
- Input files are NDJSON format (one JSON object per line)
- Each scrobble has been normalized (normalized_title field populated) before merge
- Scrobbles from same user are being merged (single username per operation)
- Timestamp (uts) is authoritative source of truth for scrobble time
- MusicBrainz IDs (mbid), when present, are accurate
- Files are encoded in UTF-8

### Operational Assumptions
- Users have read access to input files
- Users have write access to output location
- Sufficient disk space available for output file (approximately size of all input files minus duplicates)
- Network connectivity stable for Azure operations (with retry tolerance)
- Most users have datasets under 10 million scrobbles
- Typical duplicate rate is 5-15% of total scrobbles
- Users run operations from single machine (not distributed)

### Performance Assumptions
- Standard hardware: 4+ cores, 8GB+ RAM, SSD storage
- Azure Blob Storage has reasonable latency (< 500ms per operation)
- Users can tolerate 1-5 minute processing time for 1 million scrobbles
- Memory usage scales linearly with unique scrobble count (not total read count)

## Constraints

### Technical Constraints
- Memory limited by available system RAM (cannot load entire dataset if exceeds memory)
- Go standard library and existing project dependencies only (minimize external dependencies)
- Must integrate with existing LastFMReaderv3 storage backend interfaces
- Must integrate with existing progress bar implementation
- Single-threaded deduplication map (mutex-protected for future parallel enhancement)
- SHA256 hash algorithm for key generation (fixed, not configurable)

### Business Constraints
- Must not require additional Azure services beyond Blob Storage
- Must not require database installation (in-memory only)
- Must be command-line only (no GUI for v1)
- Must complete within reasonable time (users expect minutes, not hours)

### User Experience Constraints
- Command-line interface must be intuitive
- Error messages must be actionable
- Progress indication must be responsive (update at least once per second)
- Must work offline for local filesystem operations

### Data Constraints
- Input files must be valid NDJSON (cannot process arbitrary JSON)
- Output is JSON array only (not NDJSON or other formats in v1)
- Scrobble structure fixed (cannot add custom fields in v1)
- Username is single-value (cannot merge across multiple users in single operation)

## Dependencies

### Required Dependencies
- **Existing project storage backends**: `internal/writer` (local and Azure), `internal/watermark` (if checkpoint uses similar pattern)
- **Existing progress bar**: `internal/progress` package for visual progress indication
- **Existing models**: `internal/models.Scrobble` structure
- **Go standard library**:
  - `encoding/json` for JSON parsing and marshaling
  - `bufio` for efficient line-by-line reading
  - `crypto/sha256` for hash generation
  - `io` and `os` for file operations
  - `sort` for sorting scrobbles
  - `time` for timestamp handling
- **Azure SDK**: `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob` (already in project for Azure operations)

### Optional Dependencies
- **Testing**: `github.com/stretchr/testify` for test assertions (if already in project)
- **Logging**: Existing project logger (`internal/logging`)
- **Configuration**: Existing config package (`internal/config`)

### Integration Points
- Must use existing `writer.Writer` interface for output
- Must use existing storage backend patterns for consistency
- Must use existing progress reporter interface
- Should use existing logging patterns for consistency
- Command should integrate into `cmd/lastfm-sync/commands` structure

## Out of Scope (Future Enhancements)

### Explicitly Excluded from v1
- **Incremental merges**: Only processing new/changed files since last merge
- **Scheduled automatic merges**: Cron-like scheduling or background daemon mode
- **Merge validation reports**: Detailed quality assurance reports (CSV/JSON)
- **Export to additional formats**: CSV, Parquet, SQLite, Excel
- **Web UI**: Browser-based monitoring or configuration
- **Distributed processing**: Splitting work across multiple workers/machines
- **Fuzzy matching**: ML-based duplicate detection for typos/variations
- **Advanced analytics**: Listening pattern detection, data quality scoring
- **Real-time streaming**: Processing scrobbles as they arrive
- **Collaborative features**: Sharing datasets, community normalization
- **Compression support**: Gzip output (can be added in v1.1)
- **Multiple username support**: Processing multiple users in single operation
- **Custom deduplication keys**: User-defined key formulas beyond 4 strategies
- **Conflict reports**: Detailed CSV/JSON of all conflict resolutions
- **Automatic backups**: Creating timestamped backups of existing merged files
- **Prompt on overwrite**: Interactive confirmation (error-only in v1)

### Deferred to Later Versions
- **v1.1**: Incremental merges, merge validation reports, compression, conflict reports
- **v2.0**: Web UI, advanced analytics, additional export formats
- **v3.0**: Real-time streaming, distributed processing, ML-based matching

## Open Questions and Decisions

### Resolved Decisions

**Q1: Output file overwrite behavior?**
- **Decision**: Error and require manual deletion (safe default)
- **Rationale**: Prevents accidental data loss; users can delete or use --output flag for different name
- **Future**: Add --force flag in v1.1 if requested

**Q2: Checkpoint storage location?**
- **Decision**: Same directory as output file (`.speckit-merge-checkpoint.json`)
- **Rationale**: Intuitive, keeps related files together, survives reboots, easy to find
- **Future**: Make configurable if users need different location

**Q3: Very large dataset handling?**
- **Decision**: Fail with clear error and memory requirement estimate for v1
- **Rationale**: Most users have < 1M scrobbles (< 500MB); clear error better than slow/complex implementation
- **Future**: Add disk-based or external database approach in v2 based on user feedback

**Q4: Conflict resolution for equal completeness?**
- **Decision**: Keep most recently ingested (newer ingested_at timestamp)
- **Rationale**: More recent data likely more accurate; ingested_at exists for this purpose
- **Tiebreaker**: First encountered if ingested_at also equal

**Q5: Timestamp normalization?**
- **Decision**: Preserve original uts and local_time as-is
- **Rationale**: uts is authoritative; regenerating local_time requires timezone assumptions
- **Future**: Users can regenerate themselves if needed

**Q6: Progress persistence across sessions?**
- **Decision**: Progress only in current session for v1
- **Rationale**: Most merges complete in minutes; web dashboard is significant scope addition
- **Future**: Add web dashboard in v2 if demand exists

**Q7: Multiple usernames in single merge?**
- **Decision**: Require single username for v1
- **Rationale**: Simpler implementation/testing; most use cases single-user
- **Future**: Add multi-user support based on demand

**Q8: Include raw field option?**
- **Decision**: Always exclude for v1
- **Rationale**: Raw field is debug data, significantly increases size
- **Future**: Add --include-raw flag in v1.1 if requested

**Q9: Default concurrency?**
- **Decision**: Sequential (concurrency=1) with opt-in parallel via --concurrency flag
- **Rationale**: Sequential is safest and most predictable; easier to debug
- **Future**: May change default based on real-world usage patterns

**Q10: Statistics in output file?**
- **Decision**: Console output only for v1
- **Rationale**: Pure array is simpler for downstream processing; stats visible during operation
- **Future**: Add separate stats file (merged-scrobbles-stats.json) as optional in v1.1

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| ---- | ------ | ----------- | ---------- |
| Out of memory with very large datasets | High | Medium | Streaming processing, efficient data structures, checkpoint support, clear memory limits documentation |
| Azure API rate limiting/throttling | Medium | High | Exponential backoff, respect Retry-After headers, batch operations where possible |
| Data corruption during write | High | Low | Atomic writes with temp files, pre-write verification, post-write validation |
| Slow performance on large datasets | Medium | Medium | Parallel processing option, efficient algorithms, profiling and optimization |
| Duplicate detection false positives | High | Low | Comprehensive test suite, configurable strategies, conflict reports (future) |
| Network failures during Azure operations | Medium | High | Retry logic with backoff (5 attempts), checkpointing, graceful degradation |
| Inconsistent timestamp formats | Low | Low | Robust timestamp parsing, validation, handle edge cases |
| Storage quota exceeded mid-operation | Medium | Low | Pre-flight space check estimate, incremental writes with validation |
| Malformed data causing crashes | Medium | Medium | Extensive validation, defensive programming, skip invalid data gracefully |
| Missing normalized_title field | Medium | Medium | Fallback to track field, log warnings, suggest running normalization first |
| Race conditions in parallel processing | High | Medium | Proper mutex locking, thread-safe data structures, race detector testing |
| Checkpoint file corruption | Low | Low | Validate checkpoint on load, versioned format, provide recovery instructions |

## Timeline and Effort Estimate

### Development Phases

**Phase 1: Core Implementation (14-18 hours)**
- File discovery and pattern matching: 2 hours
- NDJSON streaming reader: 2 hours
- Deduplication map and key generation: 3 hours
- Conflict resolution logic: 3 hours
- Storage backend integration (reuse existing): 2 hours
- Output writing with atomic operations: 2 hours
- Basic CLI with core flags: 2 hours

**Phase 2: Enhancement and Polish (10-12 hours)**
- Progress tracking integration: 3 hours
- Comprehensive error handling: 3 hours
- Checkpointing mechanism: 3 hours
- Configuration file and env vars: 2 hours
- Dry-run mode: 1 hour

**Phase 3: Testing (12-15 hours)**
- Unit tests (deduplication, parsing, sorting, key generation): 6 hours
- Integration tests (end-to-end scenarios, both storages): 5 hours
- Performance testing and benchmarks: 3 hours
- Manual cross-platform testing: 2 hours

**Phase 4: Documentation (5-7 hours)**
- User guide and README updates: 3 hours
- Code documentation (godoc): 1 hour
- Configuration reference: 1 hour
- Examples and common scenarios: 2 hours

**Total Estimate**: 41-52 hours (~6-7 working days)

### Critical Path
1. Data structures (Scrobble, DeduplicationMap, Config)
2. File reading and parsing (NDJSON streaming)
3. Deduplication logic (key generation, conflict resolution)
4. Output writing (JSON array, atomic operations)
5. Storage backend integration (local + Azure)
6. CLI interface and flags
7. Comprehensive testing
8. Documentation

### Milestones
- **Milestone 1** (Day 2): Basic merge working for local files with default strategy
- **Milestone 2** (Day 4): Azure storage support, multiple strategies, error handling
- **Milestone 3** (Day 5): Checkpointing, progress tracking, polish
- **Milestone 4** (Day 7): All tests passing, documentation complete, ready for release

## Related Documents

- **Data Schema**: See internal/models/scrobble.go for Scrobble structure
- **Storage Backends**: See internal/writer package for existing Writer interface
- **Progress Bars**: See internal/progress package for existing progress implementation (feature 005-console-progress-bar)
- **Normalization**: See internal/normalize package for title normalization (feature 004-normalized-title-field)
- **Configuration**: See internal/config package for configuration patterns
- **Docker**: See docs/docker.md for containerization (feature 002-containerization-documentation)

