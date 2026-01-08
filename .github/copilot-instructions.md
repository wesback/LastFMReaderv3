# LastFMReaderv3 Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-06

## Active Technologies
- Go 1.24.0+ + `github.com/schollz/progressbar/v3`, `golang.org/x/term` (005-console-progress-bar)
- N/A (UI component only) (005-console-progress-bar)
- Go 1.24.0+ (alpine-based Docker build) (002-containerization-documentation)
- Go 1.24.0+ + `crypto/sha256`, `bufio.Scanner`, `encoding/json` (006-scrobble-dedup-merge)
- Local filesystem and Azure Blob Storage (via existing `writer` abstraction) (007-normalize-command)

## Project Structure

```text
src/
tests/
cmd/lastfm-sync/commands/
  - merge.go (006-scrobble-dedup-merge)
internal/merge/
  - deduplicator.go (006-scrobble-dedup-merge)
  - conflict.go (006-scrobble-dedup-merge)
  - reader.go (006-scrobble-dedup-merge)
  - merger.go (006-scrobble-dedup-merge)
  - strategies.go (006-scrobble-dedup-merge)
  - checkpoint.go (006-scrobble-dedup-merge)
  - config.go (006-scrobble-dedup-merge)
tests/integration/merge_test.go (006-scrobble-dedup-merge)
```

## Commands

# Add commands for Go 1.24.0+ (alpine-based Docker build)

# Merge command (006-scrobble-dedup-merge)
lastfm-sync merge [flags] <input-pattern...>
  --output, -o: Output file path (default: merged-scrobbles.json)
  --strategy: Deduplication strategy (default|strict|relaxed|mbid)
  --conflict-resolution: Conflict resolution mode (completeness|first|last)
  --checkpoint-interval: Save checkpoint every N scrobbles (default: 10000)
  --resume: Resume from checkpoint file

## Code Style

Go 1.24.0+ (alpine-based Docker build): Follow standard conventions
Go 1.24.0+ (006-scrobble-dedup-merge): Follow standard Go conventions
  - Use `bufio.Scanner` for NDJSON streaming
  - Store pointers in maps: `map[string]*models.Scrobble`
  - SHA256 keys as hex strings (64 chars)
  - Cyclomatic complexity <10 per function
  - 80%+ test coverage required
  - Table-driven tests for strategy variations

## Recent Changes
- 007-normalize-command: Added Go 1.24.0+
- 006-scrobble-dedup-merge: Added merge command for deduplicating and merging multiple NDJSON scrobble files. Uses in-memory hash map with SHA256 keys. Supports 4 deduplication strategies (default/strict/relaxed/mbid) and 3 conflict resolution modes (completeness/first/last). Includes checkpointing for resume capability. Performance targets: ≥10K scrobbles/sec, <500MB for 1M records. Reuses existing internal/writer, internal/progress, internal/models packages.
- 005-console-progress-bar: Added Go 1.24.0+ + `github.com/schollz/progressbar/v3`, `golang.org/x/term`

<!-- MANUAL ADDITIONS START -->
If you notice any systemic issues please add the needed requirements to this file or to the constitution if that is more appropriate.
<!-- MANUAL ADDITIONS END -->
