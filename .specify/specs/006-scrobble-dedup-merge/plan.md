# Implementation Plan: Scrobble Deduplication and Merging

**Branch**: `006-scrobble-dedup-merge` | **Date**: 2026-01-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/006-scrobble-dedup-merge/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

This feature adds a merge command that reads multiple NDJSON scrobble export files, deduplicates entries using configurable strategies (default, strict, relaxed, mbid), applies conflict resolution to preserve the most complete records, and writes a single consolidated JSON array output. The system supports both local filesystem and Azure Blob Storage, includes streaming processing for memory efficiency, atomic writes for corruption prevention, progress tracking, error recovery with checkpointing, and comprehensive error handling. Primary goal is to provide users with a single deduplicated view of their complete listening history for analysis, backup, and downstream processing.

## Technical Context

**Language/Version**: Go 1.24.0+  
**Primary Dependencies**: 
- `github.com/spf13/cobra` (CLI framework)
- `github.com/spf13/viper` (configuration management)
- `go.uber.org/zap` (structured logging)
- `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob` (Azure storage client)
- `github.com/schollz/progressbar/v3` (progress indication)
- `golang.org/x/term` (terminal detection for progress bars)

**Storage**: Azure Blob Storage (optional), Local filesystem (default), In-memory deduplication map (hash map with SHA256 keys)  
**Testing**: Go standard testing package, table-driven tests, integration tests with test fixtures  
**Target Platform**: Linux (primary), macOS, Windows (cross-platform CLI)  
**Project Type**: Single project (CLI tool with internal packages)  
**Performance Goals**: 
- Process ≥10,000 scrobbles/second
- Memory usage <500MB for 1M scrobbles
- Merge 100K scrobbles in <10 seconds

**Constraints**: 
- Memory limited to available RAM (streaming required)
- Must reuse existing storage backend interfaces (`internal/writer`, `internal/watermark` patterns)
- Must integrate with existing progress bar implementation (`internal/progress`)
- Must use existing models (`internal/models.Scrobble`)
- Single-threaded deduplication (mutex-protected for future parallel enhancement)

**Scale/Scope**: 
- Support 100 to 10M scrobbles per merge
- Support 1 to 1000 input files
- CLI command integration into existing `cmd/lastfm-sync/commands`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Test-First Development ✅ **PASS**
- **Requirement**: TDD with red-green-refactor, ≥80% coverage for new code
- **Plan**: 
  - Unit tests for deduplication logic, key generation, conflict resolution, file parsing
  - Integration tests for end-to-end merge workflows (local & Azure)
  - Performance benchmarks for throughput and memory usage
  - Test specifications will be written and approved before implementation
- **Coverage Target**: 80%+ for all new merge command code

### II. Code Quality Standards ✅ **PASS**
- **Requirement**: Consistent linting, complexity <10 cyclomatic/<15 cognitive, type safety
- **Plan**:
  - Go standard formatting (`gofmt`, `go vet`)
  - Complexity managed through small, focused functions
  - Strong typing enforced by Go compiler
  - Code reviews verify adherence before merge
- **Compliance**: Go's built-in tooling ensures quality standards

### III. User Experience Consistency ✅ **PASS**
- **Requirement**: Consistent error messages, loading/empty/error states, accessibility
- **Plan**:
  - CLI follows existing command patterns in `cmd/lastfm-sync/commands`
  - Progress indicators use existing `internal/progress` package
  - Error messages follow format: clear problem + actionable guidance
  - Dry-run mode for preview, verbose mode for debugging
  - Summary statistics in consistent format
- **Compliance**: Reuses existing UX patterns from project

### IV. Performance Requirements ✅ **PASS**
- **Requirement**: Performance budgets, monitoring, optimization practices
- **Plan**:
  - Explicit targets: ≥10K scrobbles/sec, <500MB for 1M scrobbles, <10s for 100K
  - Benchmarks included in test suite
  - Streaming processing prevents memory bloat
  - Performance metrics reported in summary output
- **Compliance**: Performance targets explicitly defined and testable

### V. Independent User Story Testing ✅ **PASS**
- **Requirement**: Each user story independently testable, P1 before P2
- **Plan**:
  - P1: Basic merge (core deduplication + output) → MVP
  - P2: Data quality handling, conflict resolution → Enhancement
  - P3: Preview, strategies, checkpointing → Advanced features
  - Each story has independent acceptance tests
- **Compliance**: Spec prioritizes stories (P1/P2/P3) with independent test definitions

### Gate Status: ✅ **ALL GATES PASSED**

No violations detected. Feature can proceed to Phase 0 research.

## Project Structure

### Documentation (this feature)

```text
specs/006-scrobble-dedup-merge/
├── spec.md              # Feature specification
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── merge-command.md # CLI interface contract
├── checklists/
│   └── requirements.md  # Quality validation checklist
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/lastfm-sync/
├── main.go
└── commands/
    ├── fetch.go         # Existing: fetch scrobbles
    └── merge.go         # NEW: merge command implementation

internal/
├── models/
│   └── scrobble.go      # Existing: Scrobble struct
├── merge/               # NEW: merge package
│   ├── deduplicator.go  # Deduplication map and key generation
│   ├── conflict.go      # Conflict resolution logic
│   ├── reader.go        # NDJSON file reader with streaming
│   ├── merger.go        # Main merge orchestration
│   ├── strategies.go    # Deduplication strategies (default, strict, relaxed, mbid)
│   └── checkpoint.go    # Checkpointing for resume capability
├── writer/              # Existing: storage backend interfaces
│   ├── writer.go        # Writer interface
│   ├── local.go         # Local filesystem writer
│   └── azure.go         # Azure Blob Storage writer
├── progress/            # Existing: progress bar (feature 005)
│   ├── bar.go
│   ├── reporter.go
│   └── factory.go
├── logging/             # Existing: structured logging
│   └── logger.go
└── config/              # Existing: configuration management
    ├── config.go
    └── types.go

tests/
├── unit/
│   └── merge/           # NEW: unit tests for merge package
│       ├── deduplicator_test.go
│       ├── conflict_test.go
│       ├── strategies_test.go
│       └── checkpoint_test.go
└── integration/
    └── merge_test.go    # NEW: end-to-end merge tests
```

**Structure Decision**: Single project structure (Option 1) selected. This is a CLI tool that extends the existing `lastfm-sync` command with a new `merge` subcommand. The new `internal/merge` package encapsulates all merge-specific logic while reusing existing infrastructure (models, writer, progress, logging, config). Tests follow the existing pattern with unit tests colocated with packages and integration tests in a separate directory.

## Complexity Tracking

**No Constitution violations** - all 5 gates passed. Optional complexity notes:

**Complexity Assessment**: MODERATE

- **High Complexity Areas**: Deduplication strategies (4 algorithms), conflict resolution (completeness scoring), checkpoint state management, memory-efficient streaming (<500MB for 1M records)
- **Medium Complexity**: NDJSON parsing, SHA256 key generation, progress reporting integration
- **Reused (Low)**: Storage backends, progress bars, logging, configuration, Scrobble model

**Cyclomatic Complexity Targets**: All functions <10 per Constitution. Hotspots: `deduplicator.go` (strategy selection), `merger.go` (orchestration), `conflict.go` (resolution logic). Mitigation: Strategy pattern, separate conflict resolution function, independent unit tests per strategy.

---

## Phase 0: Research

**Status**: ✅ COMPLETE

**Artifacts**:
- [research.md](research.md) - Technical research and technology decisions

**Key Decisions**:
1. **NDJSON Parsing**: Use `bufio.Scanner` + `encoding/json.Unmarshal` (standard library, zero dependencies)
2. **Deduplication Map**: Use `map[string]*models.Scrobble` with SHA256 hex keys (300MB for 1M scrobbles)
3. **Atomic Writes**: Use `os.CreateTemp()` + `os.Rename()` pattern (atomic on Unix/Linux)
4. **Checkpoint Format**: JSON with pretty-printing (human-readable, 380MB for 1M scrobbles)
5. **Progress Integration**: Reuse `internal/progress.Reporter` interface (consistent UX)
6. **Deduplication Strategies**: 4 strategies (default/strict/relaxed/mbid) with consistent hash key generation
7. **Conflict Resolution**: 3 modes (completeness/first/last) with completeness scoring algorithm
8. **Error Recovery**: Log warnings for parse/validation errors, skip line, continue processing

**Research Outcomes**:
- All core functionality achievable with Go standard library + existing dependencies
- Memory budget (500MB for 1M scrobbles) feasible with pointer-based map storage
- Performance target (10K scrobbles/sec) achievable with streaming + efficient hashing
- No new external dependencies required

---

## Phase 1: Design

**Status**: ✅ COMPLETE

**Artifacts**:
- [data-model.md](data-model.md) - Entity definitions, relationships, validation rules
- [contracts/merge-command.md](contracts/merge-command.md) - CLI interface specification
- [quickstart.md](quickstart.md) - Developer guide and testing strategies
- [.github/copilot-instructions.md](../../.github/copilot-instructions.md) - Updated agent context

**Key Entities**:
1. **MergeConfig**: Configuration for merge operation (input patterns, output path, strategy, etc.)
2. **DeduplicationMap**: Hash map for tracking unique scrobbles (SHA256 keys → scrobble pointers)
3. **MergeCheckpoint**: Persistent state for resume capability (version, progress, stats)
4. **MergeStats**: Statistics tracking (files, scrobbles, duplicates, errors, performance)
5. **MergeResult**: Return value from merge operation (output path, stats, warnings, success flag)

**CLI Contract**:
- **Command**: `lastfm-sync merge [flags] <input-pattern...>`
- **Key Flags**: `--output`, `--strategy`, `--conflict-resolution`, `--checkpoint-interval`, `--resume`
- **Exit Codes**: 0 (success), 1 (general), 2 (input), 3 (resume), 4 (write), 5 (validation)
- **Input Format**: NDJSON (one JSON object per line)
- **Output Format**: JSON array (pretty-printed, sorted by timestamp)

**Testing Strategy**:
- **Unit Tests**: ≥80% coverage per Constitution, table-driven tests for strategies
- **Integration Tests**: End-to-end with temporary files, validate output correctness
- **Benchmark Tests**: Verify 10K scrobbles/sec, <500MB memory for 1M scrobbles

**Constitution Re-check**: All 5 gates remain PASSED after design phase.

---

## Phase 2: Implementation Planning

**Status**: ⏳ NOT STARTED (use `/speckit.tasks` command)

**Next Steps**:
1. Run `/speckit.tasks` to generate implementation tasks
2. Begin TDD cycle: Write test → Implement → Verify
3. Start with `internal/merge/deduplicator.go` (core deduplication logic)
4. Progress to CLI command `cmd/lastfm-sync/commands/merge.go`
5. Complete integration tests in `tests/integration/merge_test.go`

---

**Implementation Plan Complete** ✅  
Ready for `/speckit.tasks` to generate detailed task breakdown.

