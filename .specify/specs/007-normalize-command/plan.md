# Implementation Plan: Normalize Command

**Branch**: `007-normalize-command` | **Date**: 2026-01-08 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/.specify/specs/007-normalize-command/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add a `normalize` command to the Last.fm sync CLI that processes existing NDJSON scrobble files and updates the `normalized_title` field by reapplying the current normalization logic. The command supports both local filesystem and Azure Blob Storage, provides dry-run preview mode, displays per-file progress, and generates comprehensive summary reports. This enables retroactive application of normalization improvements to historical data without re-fetching from Last.fm.

## Technical Context

**Language/Version**: Go 1.24.0+  
**Primary Dependencies**: 
- `github.com/spf13/cobra` (CLI framework, existing)
- `go.uber.org/zap` (structured logging, existing)
- `github.com/schollz/progressbar/v3` (progress display, existing)
- Reuse existing internal packages: `config`, `writer`, `normalize`, `progress`, `logging`, `models`

**Storage**: Local filesystem and Azure Blob Storage (via existing `writer` abstraction)  
**Testing**: Go testing with table-driven tests, integration tests for end-to-end workflows  
**Target Platform**: Linux (primary), macOS, Windows (via Go cross-compilation)  
**Project Type**: Single CLI application with command structure  
**Performance Goals**: Process ≥1000 files in under 5 seconds (5ms per file)  
**Constraints**: 
- Streaming NDJSON processing to minimize memory footprint
- Per-file progress updates without performance degradation
- Graceful error handling with continuation for remaining files

**Scale/Scope**: Handle users with 100-1000 NDJSON files, file sizes from 1KB to 100MB+

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| **I. Test-First Development** | ✅ PASS | Unit tests will be written first for normalize logic. Integration tests will validate end-to-end file processing. Target: 80%+ coverage. |
| **II. Code Quality Standards** | ✅ PASS | Go standard linting (golint, go vet) enforced. Cyclomatic complexity <10 per function. Type-safe with explicit error handling. |
| **III. UX Consistency** | ✅ PASS | Command follows existing patterns (fetch, merge). Progress display matches existing progress bar implementation. Error messages follow existing format. |
| **IV. Performance Requirements** | ✅ PASS | Performance target defined: 5 seconds per 1000 files. Streaming NDJSON processing prevents memory issues. Per-file metrics tracked. |
| **V. Independent User Story Testing** | ✅ PASS | Three prioritized user stories (P1: core, P2: dry-run, P3: progress). Each independently testable. P1 provides standalone value. |

**Gate Result**: ✅ **PASSED** - All constitution principles satisfied. No violations to justify.

## Project Structure

## Project Structure

### Documentation (this feature)

```text
/.specify/specs/007-normalize-command/
├── spec.md              # Feature specification (completed)
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (to be generated)
├── data-model.md        # Phase 1 output (to be generated)
├── quickstart.md        # Phase 1 output (to be generated)
├── contracts/           # Phase 1 output (CLI contract/interface)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/lastfm-sync/commands/
  normalize.go           # New normalize command implementation

internal/normalize/
  normalize.go           # Existing normalization logic (reused)
  patterns.go            # Existing pattern definitions (reused)
  config.go              # Existing configuration (reused)

internal/models/
  scrobble.go            # Existing scrobble model (reused)

internal/writer/
  writer.go              # Existing storage abstraction (reused)
  local.go               # Local filesystem writer (reused)
  azure.go               # Azure Blob Storage writer (reused)

internal/progress/
  bar.go                 # Existing progress bar (reused)
  factory.go             # Progress factory (reused)

internal/config/
  config.go              # Existing configuration loader (reused)
  azure.go               # Azure config (reused)

internal/logging/
  logger.go              # Existing logger (reused)

tests/
  integration/
    normalize_test.go    # New integration tests for normalize command
  unit/
    commands/
      normalize_test.go  # New unit tests for normalize command logic
```

**Structure Decision**: Single Go project with established cmd/internal organization. Normalize command follows existing patterns from fetch.go and merge.go. Maximum code reuse from existing packages (normalize, writer, progress, config). New code limited to cmd/lastfm-sync/commands/normalize.go and tests.

## Complexity Tracking

> **No violations to track** - All constitution gates passed without need for complexity justification.

---

## Phase 1 Completion

### Constitution Re-Check (Post-Design)

| Principle | Status | Design Validation |
|-----------|--------|-------------------|
| **I. Test-First Development** | ✅ PASS | Quickstart defines clear test strategy. Unit tests for file discovery, parsing, normalization. Integration tests for end-to-end workflows. Performance benchmarks for SC-001 target. |
| **II. Code Quality Standards** | ✅ PASS | Design limits new code to ~300-400 lines in single file. Reuses existing packages. No complex abstractions needed. Standard Go patterns throughout. |
| **III. UX Consistency** | ✅ PASS | CLI contract follows existing fetch/merge patterns. Progress display uses existing library. Error messages follow established format. |
| **IV. Performance Requirements** | ✅ PASS | Research validates streaming NDJSON approach. Buffio.Scanner provides O(1) memory per file. Performance target of 5 sec/1000 files achievable with design. |
| **V. Independent User Story Testing** | ✅ PASS | Quickstart phases map to prioritized user stories. P1 (core) independently testable. P2 (dry-run) adds safety without breaking P1. P3 (progress) enhances UX orthogonally. |

**Final Gate Result**: ✅ **PASSED** - Design maintains constitution compliance. Ready for task breakdown.

---

## Artifacts Generated

### Phase 0: Research
- ✅ [research.md](./research.md) - All technical unknowns resolved, decisions documented

### Phase 1: Design & Contracts
- ✅ [data-model.md](./data-model.md) - Entity definitions, data flow, reuse analysis
- ✅ [contracts/cli-interface.md](./contracts/cli-interface.md) - Complete CLI specification
- ✅ [quickstart.md](./quickstart.md) - Implementation guide with phases, testing strategy
- ✅ Agent context updated (GitHub Copilot instructions)

### Next Phase
- ⏳ [tasks.md](./tasks.md) - Created by `/speckit.tasks` command (not part of `/speckit.plan`)

---

## Summary

**Planning Complete**: All research and design artifacts generated. Feature is ready for task breakdown.

**Key Decisions**:
1. NDJSON-only format for streaming efficiency
2. Filename pattern matching for user file discovery
3. Per-file progress display for visibility
4. Structured error reporting with file path and type
5. No concurrent execution locking (admin coordination)

**Architecture Approach**:
- Maximum reuse of existing packages (normalize, writer, progress, config)
- Single new command file (~300-400 lines)
- Streaming processing for memory efficiency
- Consistent with fetch/merge command patterns

**Next Step**: Run `/speckit.tasks` to generate implementation task breakdown.
