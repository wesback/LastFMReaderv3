# Tasks: Normalize Command

**Feature**: 007-normalize-command  
**Input**: Design documents from `/.specify/specs/007-normalize-command/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included per constitution requirement (Test-First Development)

**Organization**: Tasks grouped by user story for independent implementation and testing

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story this task belongs to (US1, US2, US3)
- Exact file paths included in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization - prepare for normalize command development

- [X] T001 Create cmd/lastfm-sync/commands/normalize.go following existing command structure from fetch.go and merge.go
- [X] T002 [P] Create tests/integration/normalize_test.go for end-to-end integration tests
- [X] T003 [P] Create tests/unit/commands/normalize_test.go for unit tests
- [X] T004 Register normalize command in cmd/lastfm-sync/main.go root command

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure needed before ANY user story implementation

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Define command-line flags structure in cmd/lastfm-sync/commands/normalize.go (--user, --dry-run, Azure flags)
- [X] T006 Implement argument validation function in cmd/lastfm-sync/commands/normalize.go (FR-018, FR-019)
- [X] T007 Create ProcessingError struct in cmd/lastfm-sync/commands/normalize.go per data-model.md
- [X] T008 Create ProcessingSummary struct in cmd/lastfm-sync/commands/normalize.go per data-model.md
- [X] T009 Implement file discovery for local storage using filepath.Glob with pattern {username}_*.ndjson
- [ ] T010 Implement file discovery for Azure storage using Azure SDK ListBlobs with prefix filter

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Re-normalize User's Scrobble Files (Priority: P1) 🎯 MVP

**Goal**: Implement basic normalize command that processes NDJSON files and updates normalized_title field for local and Azure storage

**Independent Test**: Run `./lastfm-sync normalize --user testuser` on sample NDJSON files and verify normalized_title fields are updated correctly

### Tests for User Story 1 (TDD - Write First)

> **TDD: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T011 [P] [US1] Write unit test for file discovery with various username patterns in tests/unit/commands/normalize_test.go
- [X] T012 [P] [US1] Write unit test for NDJSON line-by-line parsing logic in tests/unit/commands/normalize_test.go
- [X] T013 [P] [US1] Write unit test for normalized_title update decision logic in tests/unit/commands/normalize_test.go
- [X] T014 [P] [US1] Write integration test for end-to-end local file processing in tests/integration/normalize_test.go
- [ ] T015 [P] [US1] Write integration test for Azure storage file processing in tests/integration/normalize_test.go (DEFERRED - Azure support pending)
- [X] T016 [P] [US1] Write integration test for unchanged file handling (normalized_title already correct) in tests/integration/normalize_test.go

### Implementation for User Story 1

- [X] T017 [US1] Implement storage mode detection function in cmd/lastfm-sync/commands/normalize.go (local vs Azure based on flags)
- [X] T018 [US1] Implement file list retrieval using file discovery functions from Phase 2
- [X] T019 [US1] Implement NDJSON file reader using bufio.Scanner for line-by-line processing in cmd/lastfm-sync/commands/normalize.go
- [X] T020 [US1] Implement normalization logic integration calling normalize.Normalize() from internal/normalize package
- [X] T021 [US1] Implement comparison logic to detect if normalized_title changed (FR-016)
- [X] T022 [US1] Implement file writer for updated records using existing writer abstraction from internal/writer
- [X] T023 [US1] Implement per-file error handling with continue-on-error behavior (FR-011)
- [X] T024 [US1] Implement ProcessingError collection during file processing
- [X] T025 [US1] Implement ProcessingSummary generation with counts (total, updated, unchanged, errors) per FR-010
- [X] T026 [US1] Implement summary report output formatting to console per contracts/cli-interface.md
- [X] T027 [US1] Run all US1 tests and verify they pass with implementation

**Checkpoint**: User Story 1 complete - basic normalize command works for local and Azure storage

---

## Phase 4: User Story 2 - Preview Changes Before Applying (Priority: P2)

**Goal**: Add --dry-run flag that previews changes without modifying files

**Independent Test**: Run `./lastfm-sync normalize --user testuser --dry-run` and verify output shows changes but no files are modified

### Tests for User Story 2 (TDD - Write First)

- [X] T028 [P] [US2] Write integration test verifying no files modified in dry-run mode in tests/integration/normalize_test.go
- [X] T029 [P] [US2] Write integration test comparing dry-run output accuracy against actual run in tests/integration/normalize_test.go
- [X] T030 [P] [US2] Write unit test for dry-run flag handling in cmd/lastfm-sync/commands/normalize.go

### Implementation for User Story 2

- [X] T031 [US2] Add --dry-run boolean flag to command definition in cmd/lastfm-sync/commands/normalize.go (FR-008)
- [X] T032 [US2] Pass dry-run flag to file writer abstraction to prevent writes when active
- [X] T033 [US2] Implement dry-run preview output showing current and new normalized_title values (FR-017)
- [X] T034 [US2] Update ProcessingSummary to include dry-run status field
- [X] T035 [US2] Update summary report to display "Dry-run mode: No changes written to storage" when active (FR-013)
- [X] T036 [US2] Run all US2 tests and verify they pass with implementation

**Checkpoint**: User Stories 1 AND 2 both work independently - dry-run safety feature complete

---

## Phase 5: User Story 3 - Monitor Progress and Review Results (Priority: P3)

**Goal**: Add per-file progress display and detailed error reporting

**Independent Test**: Run `./lastfm-sync normalize --user testuser` on 200+ files and verify progress shows current file and summary includes detailed errors

### Tests for User Story 3 (TDD - Write First)

- [ ] T037 [P] [US3] Write integration test for progress bar display validation in tests/integration/normalize_test.go (SKIPPED - output-based testing)
- [X] T038 [P] [US3] Write integration test for error handling with malformed NDJSON files in tests/integration/normalize_test.go
- [X] T039 [P] [US3] Write integration test verifying processing continues after errors in tests/integration/normalize_test.go
- [X] T040 [P] [US3] Write unit test for error summary formatting in tests/unit/commands/normalize_test.go

### Implementation for User Story 3

- [X] T041 [US3] Initialize progress bar using internal/progress.NewFactory with total file count
- [X] T042 [US3] Implement per-file progress updates showing current filename in progress description (FR-009)
- [X] T043 [US3] Implement structured error collection with ProcessingError for each file failure (FR-012)
- [X] T044 [US3] Update summary report to include error list with file path and error type format (e.g., "file.ndjson: parse error")
- [X] T045 [US3] Implement error categorization (parse_error, missing_track_field, permission_denied, read_error, write_error)
- [X] T046 [US3] Add duration tracking to ProcessingSummary
- [X] T047 [US3] Run all US3 tests and verify they pass with implementation

**Checkpoint**: All user stories independently functional - progress and error reporting complete

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final quality improvements affecting all user stories

- [X] T048 [P] Add command help text with description, flags, and examples in cmd/lastfm-sync/commands/normalize.go
- [X] T049 [P] Add edge case unit tests for missing track field, malformed NDJSON, no files found in tests/unit/commands/normalize_test.go
- [X] T050 [P] Add table-driven unit tests for various error scenarios in tests/unit/commands/normalize_test.go
- [ ] T051 [P] Write performance benchmark test targeting 5 seconds per 1000 files (SC-001) in tests/unit/commands/normalize_bench_test.go (DEFERRED - optional)
- [X] T052 Verify test coverage meets 80%+ requirement using `go test -cover`
- [ ] T053 [P] Run golint and go vet, fix any issues
- [ ] T054 [P] Update README.md with normalize command documentation
- [ ] T055 [P] Update docs/troubleshooting.md with error types and solutions
- [ ] T056 Verify cyclomatic complexity <10 per function using complexity analysis tool
- [ ] T057 Run full integration test suite across all user stories
- [ ] T058 Manual testing on Linux, macOS, Windows platforms
- [ ] T059 Run quickstart.md validation checklist

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup (Phase 1) completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational (Phase 2) completion
  - US1 (Phase 3): Can start after Phase 2 - No dependencies on other stories
  - US2 (Phase 4): Can start after Phase 2 - Enhances US1 but independently testable
  - US3 (Phase 5): Can start after Phase 2 - Enhances US1 but independently testable
- **Polish (Phase 6)**: Depends on desired user stories being complete (minimum US1 for MVP)

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational - No dependencies on other stories (MVP)
- **User Story 2 (P2)**: Can start after Foundational - Independently testable (adds --dry-run flag)
- **User Story 3 (P3)**: Can start after Foundational - Independently testable (adds progress/errors)

### Within Each User Story

1. Tests MUST be written FIRST and FAIL before implementation (TDD)
2. Storage mode detection and file discovery (reuses Phase 2 functions)
3. Core processing logic (NDJSON parsing, normalization, comparison)
4. Writer integration and error handling
5. Summary generation and output
6. Run tests and verify they pass

### Parallel Opportunities

- **Phase 1**: T002, T003 can run in parallel (different test files)
- **Phase 2**: No parallelization (sequential setup needed)
- **User Story Tests**: All test tasks marked [P] within a story can run in parallel
- **User Stories**: After Phase 2, US1, US2, US3 can be implemented in parallel by different developers
- **Phase 6**: T048, T049, T050, T051, T053, T054, T055 can run in parallel

---

## Parallel Example: User Story 1

If you have 3 developers available after Phase 2:

**Developer A**:
```bash
# Tests
git checkout -b feature/us1-tests
# T011, T012, T013 (unit tests in parallel branches, merge when done)
# T014, T015, T016 (integration tests)
```

**Developer B**:
```bash
# Implementation - File Processing
git checkout -b feature/us1-core
# T017, T018, T019, T020, T021 (core logic)
```

**Developer C**:
```bash
# Implementation - Output & Summary
git checkout -b feature/us1-output
# T022, T023, T024, T025, T026 (writer, errors, summary)
```

Merge order: A (tests) → B (core) → C (output) → T027 (validate)

---

## Implementation Strategy

### MVP Scope (Minimum Viable Product)

**Includes**: User Story 1 only (Phase 1, 2, 3)
- Basic normalize command
- Local and Azure storage support
- File processing and normalization
- Basic summary output
- 80%+ test coverage

**Estimated Effort**: 16-20 hours (T001-T027 + minimal polish)

**Deliverable**: Administrators can normalize existing scrobble files

### Incremental Delivery

**Release 1 (MVP)**: US1 - Core functionality
**Release 2**: US1 + US2 - Add dry-run safety
**Release 3**: US1 + US2 + US3 - Add progress and error reporting
**Release 4**: All stories + Polish - Production ready

### Task Count Summary

- **Total Tasks**: 59
- **Setup**: 4 tasks
- **Foundational**: 6 tasks
- **User Story 1**: 17 tasks (6 tests + 11 implementation)
- **User Story 2**: 9 tasks (3 tests + 6 implementation)
- **User Story 3**: 11 tasks (4 tests + 7 implementation)
- **Polish**: 12 tasks

**Parallel Opportunities**: 23 tasks marked [P] can run in parallel within constraints

---

## Format Validation

✅ All tasks follow checklist format: `- [ ] [TaskID] [P?] [Story?] Description with file path`
✅ Sequential Task IDs (T001-T059)
✅ [P] markers for parallelizable tasks
✅ [Story] labels for user story tasks (US1, US2, US3)
✅ Exact file paths in descriptions
✅ Tests written before implementation (TDD)
✅ Independent test criteria per user story
✅ Constitution compliance (80%+ coverage, complexity <10, Test-First)
