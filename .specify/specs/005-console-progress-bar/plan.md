# Implementation Plan: Console Progress Bar

**Branch**: `005-console-progress-bar` | **Date**: 2026-01-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/005-console-progress-bar/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add visual progress indication to long-running console operations in SpecKit (Last.fm data sync, title normalization, file operations) using the `github.com/schollz/progressbar/v3` library. Progress bars will display percentage completion, ETA, speed, and adapt to terminal capabilities with support for multi-operation workflows and graceful error handling.

## Technical Context

**Language/Version**: Go 1.24.0+  
**Primary Dependencies**: `github.com/schollz/progressbar/v3`, `golang.org/x/term`  
**Storage**: N/A (UI component only)  
**Testing**: Go standard testing package (`testing`), table-driven tests  
**Target Platform**: Linux, macOS, Windows (cross-platform CLI)  
**Project Type**: Single project (CLI enhancement)  
**Performance Goals**: < 1% CPU overhead, 10+ FPS update rate, < 1MB memory  
**Constraints**: < 100ms initial display latency, 200ms terminal resize response, no blocking of operation  
**Scale/Scope**: 5 integration points (Last.fm sync, title normalization, file I/O, API rate limiting, database operations)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Test-First Development ✅
- **Status**: PASS
- **Plan**: Unit tests will be written for progress bar state management, terminal detection, multi-bar coordination, and error handling before implementation. Integration tests will verify progress display in actual sync operations.
- **Coverage Target**: 80% minimum (constitution requirement met)

### Code Quality Standards ✅
- **Status**: PASS
- **Plan**: Go standard formatting (`gofmt`), linting (`golint`/`staticcheck`), and complexity analysis will be enforced. All functions will maintain cyclomatic complexity < 10.
- **Tools**: Pre-commit hooks for `go fmt` and `go vet`

### User Experience Consistency ✅
- **Status**: PASS  
- **Plan**: Progress bars will follow consistent visual patterns across all operations. Error states, completion states, and loading indicators defined in spec. Terminal capability detection ensures graceful degradation.
- **Standards**: ANSI color usage, Unicode fallback, consistent message formatting

### Performance Requirements ✅
- **Status**: PASS
- **Plan**: Performance metrics defined in spec (< 1% CPU overhead, 10 FPS, < 1MB memory). Benchmarks will be created to verify these targets.
- **Monitoring**: Performance tests in CI/CD, benchmark suite for update throttling

### Independent User Story Testing ✅
- **Status**: PASS
- **Plan**: User stories are prioritized (P1-P3) with independent test definitions. P1 (Basic Progress Visibility) can be tested and deployed independently before P2/P3 features.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── progress/              # New package for progress bar functionality
│   ├── bar.go            # Core progress bar implementation
│   ├── bar_test.go       # Unit tests for progress bar
│   ├── options.go        # Configuration options
│   ├── style.go          # Visual style definitions
│   ├── terminal.go       # Terminal capability detection
│   ├── terminal_test.go  # Terminal detection tests
│   ├── multi.go          # Multi-progress bar coordinator
│   └── multi_test.go     # Multi-bar tests
├── service/
│   └── sync.go           # Updated to use progress bars
├── lastfm/
│   └── client.go         # Updated to report progress
└── normalize/
    └── normalize.go      # Updated to report progress

cmd/
└── lastfm-sync/
    └── commands/
        └── fetch.go      # Updated to display progress bars

tests/
├── integration/
│   └── progress_integration_test.go  # E2E progress display tests
└── benchmarks/
    └── progress_bench_test.go        # Performance benchmarks
```

**Structure Decision**: Single project structure (Option 1). Progress bar functionality added as new `internal/progress` package to maintain separation of concerns. Existing packages (`service`, `lastfm`, `normalize`) updated to integrate progress reporting.

## Phase 1: Design & Contracts ✅

*Complete this phase before writing any code.*

### Artifacts Generated

1. **Data Model** (`data-model.md`) ✅
   - Defined 6 core entities: ProgressBar, Options, Style, TerminalInfo, MultiProgress, ProgressReporter interface
   - Documented all relationships and state transitions
   - Included complete validation rules and memory estimates

2. **API Contracts** (`contracts/`) ✅
   - `progress-api.md`: Complete public API specification with all interfaces, factory functions, options, error types
   - `integration-contracts.md`: Integration patterns for Last.fm sync, title normalization, file export, configuration, logging, multi-operation workflows, and testing
   - Documented all error conditions, thread safety guarantees, and performance contracts

3. **Integration Points** ✅
   - Defined integration with SyncService, normalize package, Writer interface (local + Azure), Config loader
   - Documented backward-compatible API changes with migration examples
   - Specified configuration schema (YAML + environment variables)

4. **Quickstart Guide** (`quickstart.md`) ✅
   - 5-minute integration guide with import and basic usage
   - 5 common patterns: paginated fetch, known counts, rate limiting, multi-step, file processing
   - Configuration reference, testing examples, troubleshooting guide
   - Quick reference card for common operations

5. **Agent Context Update** ✅
   - Ran: `.specify/scripts/bash/update-agent-context.sh copilot`
   - Added Go 1.24.0+, github.com/schollz/progressbar/v3, golang.org/x/term to GitHub Copilot context
   - Updated `.github/copilot-instructions.md` with feature 005 information

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**No violations** - Constitution check passed all 5 principles. No complexity justifications required.

---

## Phase 2: Final Report ✅

### Constitution Re-Check

All principles verified after Phase 1 completion:

| Principle | Status | Notes |
|-----------|--------|-------|
| Test-First Development | ✅ PASS | Mock implementation planned, TDD approach documented in all contracts |
| Code Quality Standards | ✅ PASS | Professional abstractions (ProgressReporter interface), clear error handling contracts |
| UX Consistency | ✅ PASS | Graceful fallbacks, terminal detection, configurable display elements |
| Performance Requirements | ✅ PASS | < 1% CPU overhead enforced via benchmarks, batched updates, throttled refresh |
| Independent User Story Testing | ✅ PASS | Each user story testable independently via mock, no shared state dependencies |

**Verdict**: No constitution violations introduced during design phase. All contracts maintain project standards.

### Completion Summary

#### Feature Ready for Implementation

- **Feature**: Console Progress Bar (005-console-progress-bar)
- **Branch**: `005-console-progress-bar`
- **Status**: ✅ Planning Complete - Ready for TDD Implementation

#### Artifacts Delivered

| Artifact | Location | Status | Lines |
|----------|----------|--------|-------|
| Feature Specification | `spec.md` | ✅ Complete | 285 |
| Implementation Plan | `plan.md` | ✅ Complete | 200+ |
| Phase 0 Research | `research.md` | ✅ Complete | 2,000+ |
| Phase 1 Data Model | `data-model.md` | ✅ Complete | 3,000+ |
| API Contract | `contracts/progress-api.md` | ✅ Complete | 450+ |
| Integration Contract | `contracts/integration-contracts.md` | ✅ Complete | 800+ |
| Quickstart Guide | `quickstart.md` | ✅ Complete | 500+ |
| Agent Context Update | `.github/copilot-instructions.md` | ✅ Updated | — |

**Total Documentation**: ~7,000+ lines of comprehensive planning and design

#### Key Design Decisions

1. **Library Choice**: `github.com/schollz/progressbar/v3` selected over alternatives
   - Rationale: MIT licensed, actively maintained, 50KB, excellent feature set
   - Terminal detection via `golang.org/x/term` (official Go stdlib extension)

2. **Architecture Pattern**: ProgressReporter interface with 3 implementations
   - `RealProgressBar`: Full-featured progress bar with schollz/progressbar
   - `NoOpProgressBar`: Silent implementation for non-interactive terminals
   - `MockProgressBar`: Test double for unit testing

3. **Integration Strategy**: Non-breaking, backward-compatible changes
   - Add ProgressReporter parameter to existing functions
   - Graceful fallback to NoOp when progress disabled/unavailable
   - Configuration via YAML + environment variable overrides

4. **Performance Guarantees**: < 1% CPU overhead, < 1MB memory
   - Enforced via CI benchmarks
   - Batched updates (100-1000 items)
   - Throttled refresh (10-60 FPS)

#### Implementation Roadmap

**Next Steps** (in order):

1. **Run `/speckit.tasks`** to generate implementation task breakdown
2. **Set up Go module dependencies**: `go get github.com/schollz/progressbar/v3 golang.org/x/term`
3. **TDD Cycle 1**: Implement `ProgressReporter` interface + `NoOpProgressBar` with tests
4. **TDD Cycle 2**: Implement `RealProgressBar` with terminal detection
5. **TDD Cycle 3**: Implement `MultiProgress` for multi-step workflows
6. **TDD Cycle 4**: Integrate with SyncService, normalize, Writer interfaces
7. **TDD Cycle 5**: Add configuration loading and environment variable support
8. **Integration Testing**: E2E tests with actual terminal output
9. **Performance Validation**: Run benchmarks, verify < 1% overhead target
10. **Documentation**: Update user-facing docs with progress bar examples

**Estimated Implementation Time**: 22 hours (see research.md timeline)

#### Success Criteria Checklist

From spec.md, implementation complete when:

- [ ] Last.fm sync displays progress bar during fetch operations
- [ ] Title normalization shows progress for batch operations
- [ ] File export operations display progress
- [ ] Progress bars adapt to terminal capabilities (UTF-8 → ASCII fallback)
- [ ] Non-interactive terminals (pipes, redirects) silently disable progress
- [ ] Rate limiting switches to spinner mode with clear messaging
- [ ] Multi-step workflows show stacked completed operations
- [ ] Configuration via `config.yaml` and environment variables works
- [ ] All unit tests pass with MockProgressBar
- [ ] Performance benchmarks confirm < 1% CPU overhead
- [ ] Integration tests verify E2E behavior

#### Developer Resources

- **Feature Spec**: [spec.md](spec.md) - User stories, requirements, success criteria
- **This Plan**: [plan.md](plan.md) - Architecture, decisions, structure
- **Research**: [research.md](research.md) - Library evaluation, best practices
- **Data Model**: [data-model.md](data-model.md) - Entity definitions, relationships
- **API Contract**: [contracts/progress-api.md](contracts/progress-api.md) - Public interfaces
- **Integration**: [contracts/integration-contracts.md](contracts/integration-contracts.md) - Service integration patterns
- **Quickstart**: [quickstart.md](quickstart.md) - 5-minute integration guide

---

**✅ Planning phase complete.** Ready to begin test-first implementation following TDD methodology.

