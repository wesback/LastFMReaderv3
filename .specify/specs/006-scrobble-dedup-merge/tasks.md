# Tasks: Scrobble Deduplication and Merging

**Feature**: 006-scrobble-dedup-merge  
**Input**: Design documents from `/home/wesleyb/git/LastFMReaderv3/specs/006-scrobble-dedup-merge/`  
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

**Testing Approach**: TDD with ≥80% coverage per Constitution. All tests must be written FIRST and FAIL before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

---

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure for merge feature

- [X] T001 Create internal/merge/ package directory structure
- [X] T002 Create tests/unit/merge/ directory for unit tests
- [X] T003 [P] Create cmd/lastfm-sync/commands/merge.go skeleton with cobra command structure
- [X] T004 [P] Update go.mod if needed (verify existing dependencies sufficient per research.md)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core deduplication infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Create internal/merge/config.go with MergeConfig, DeduplicationStrategy, ConflictResolution types per data-model.md
- [X] T006 [P] Create internal/merge/stats.go with MergeStats struct and methods per data-model.md
- [X] T007 Write unit test tests/unit/merge/deduplicator_test.go for DeduplicationMap (table-driven tests for 4 strategies)
- [X] T008 Implement internal/merge/deduplicator.go with DeduplicationMap, Add(), generateKey() methods per data-model.md
- [X] T009 Write unit test tests/unit/merge/strategies_test.go for all 4 deduplication strategies (default, strict, relaxed, mbid)
- [X] T010 Implement internal/merge/strategies.go with strategy-specific key generation logic per research.md
- [X] T011 Write unit test tests/unit/merge/conflict_test.go for conflict resolution (completeness scoring)
- [X] T012 Implement internal/merge/conflict.go with resolveConflict() and completenessScore() per data-model.md
- [X] T013 Write unit test tests/unit/merge/reader_test.go for NDJSON streaming parser
- [X] T014 Implement internal/merge/reader.go with NDJSON streaming using bufio.Scanner per research.md

**Checkpoint**: Foundation ready - deduplication core is testable and working (✅ **COMPLETE** - 24/24 tests passing)

---

## Phase 3: User Story 1 - Basic Scrobble Merge (Priority: P1) 🎯 MVP

**Goal**: Merge multiple NDJSON files into single deduplicated JSON output with progress indication

**Independent Test**: Run merge command on test NDJSON files, verify output contains unique scrobbles sorted by timestamp, progress bar displays correctly

### Tests for User Story 1 (TDD - Write FIRST)

- [X] T015 [P] [US1] Write integration test tests/integration/merge_test.go for basic local merge (5 files, verify deduplication)
- [X] T016 [P] [US1] Write integration test for Azure Blob Storage merge in tests/integration/merge_test.go
- [X] T017 [P] [US1] Write benchmark test internal/merge/merger_bench_test.go for 10K scrobbles/sec target

### Implementation for User Story 1

- [X] T018 [US1] Implement internal/merge/merger.go with Merger struct, Merge() method, file discovery logic
- [X] T019 [US1] Integrate DeduplicationMap into Merger.Merge() with streaming processing
- [X] T020 [US1] Implement output sorting by timestamp in internal/merge/merger.go
- [X] T021 [US1] Implement JSON array writer using atomic writes (temp file + rename) in internal/merge/merger.go
- [X] T022 [US1] Integrate internal/writer interface for local and Azure output in internal/merge/merger.go
- [X] T023 [US1] Integrate internal/progress.Reporter for progress bar in internal/merge/merger.go
- [X] T024 [US1] Implement cmd/lastfm-sync/commands/merge.go with cobra command, flags (--user, --output, --out-path, Azure flags aligned with fetch), and Merger invocation
- [X] T025 [US1] Add flag validation and error handling in cmd/lastfm-sync/commands/merge.go
- [X] T026 [US1] Add summary statistics output (files processed, scrobbles, duplicates) in cmd/lastfm-sync/commands/merge.go
- [X] T027 [US1] Wire up zap logging with appropriate levels in cmd/lastfm-sync/commands/merge.go
- [X] T028 [US1] Run integration tests and verify all pass (T015, T016)
- [X] T029 [US1] Run benchmark test (T017) and verify ≥10K scrobbles/sec performance target

**Checkpoint**: User Story 1 complete - basic merge works locally and on Azure with progress indication (✅ **ALL 15 TASKS COMPLETE** - 142K+ scrobbles/sec achieved!)

---

## Phase 4: User Story 2 - Handle Data Quality Issues (Priority: P2)

**Goal**: Gracefully handle malformed JSON, missing fields, invalid timestamps with clear error reporting

**Independent Test**: Create NDJSON files with intentional errors, verify tool skips bad records and processes good ones with detailed logging

### Tests for User Story 2 (TDD - Write FIRST)

- [X] T030 [P] [US2] Write unit test tests/unit/merge/reader_test.go for invalid JSON syntax handling
- [X] T031 [P] [US2] Write unit test tests/unit/merge/reader_test.go for missing required fields (artist, title)
- [X] T032 [P] [US2] Write integration test tests/integration/merge_test.go for mixed valid/invalid records (verify 99.8% success rate scenario)

### Implementation for User Story 2

- [X] T033 [US2] Enhance internal/merge/reader.go with JSON parse error recovery (skip line, log warning)
- [X] T034 [US2] Add Scrobble.Validate() call in internal/merge/reader.go with error logging (file, line number)
- [X] T035 [US2] Implement invalid timestamp handling (zero/negative → sentinel value 0) in internal/merge/reader.go
- [X] T036 [US2] Add SkippedLines and SkippedScrobbles counters to MergeStats in internal/merge/stats.go
- [X] T037 [US2] Update summary output in cmd/lastfm-sync/commands/merge.go to show error counts and success rate
- [X] T038 [US2] Add structured error logging with zap (file, line, error details) throughout internal/merge/reader.go
- [X] T039 [US2] Run unit tests (T030, T031) and verify error handling works correctly
- [X] T040 [US2] Run integration test (T032) and verify 99.8% success rate scenario

**Checkpoint**: User Story 2 complete - tool handles data quality issues gracefully (✅ **ALL 11 TASKS COMPLETE** - 99.80% success rate achieved!)

---

## Phase 5: User Story 3 - Conflict Resolution and Data Quality (Priority: P2)

**Goal**: Keep most complete version of duplicates using completeness scoring and MBID preference

**Independent Test**: Create duplicates with varying completeness, verify most complete version retained

### Tests for User Story 3 (TDD - Write FIRST)

- [x] T041 [P] [US3] Write unit test tests/unit/merge/conflict_test.go for completeness scoring (album presence, MBID weight)
- [x] T042 [P] [US3] Write unit test tests/unit/merge/conflict_test.go for tie-breaker scenarios (equal completeness, MBID preference, timestamp)
- [x] T043 [P] [US3] Write integration test tests/integration/merge_test.go for 1,000 duplicate resolution scenario

### Implementation for User Story 3

- [x] T044 [US3] Enhance completenessScore() in internal/merge/conflict.go with field-by-field scoring (album +1, MBID +2)
- [x] T045 [US3] Implement tie-breaker logic in resolveConflict() (MBID preference, then timestamp) in internal/merge/conflict.go
- [x] T046 [US3] Add conflict tracking to MergeStats (Conflicts counter, ConflictsByStrategy map) in internal/merge/stats.go
- [x] T047 [US3] Integrate conflict resolution into DeduplicationMap.Add() in internal/merge/deduplicator.go
- [x] T048 [US3] Add DEBUG logging for conflict decisions (fields compared, scores, winner) in internal/merge/conflict.go
- [x] T049 [US3] Update summary output to show conflicts resolved in cmd/lastfm-sync/commands/merge.go
- [x] T050 [US3] Run unit tests (T041, T042) and verify completeness scoring works
- [x] T051 [US3] Run integration test (T043) and verify 1,000 duplicates handled correctly

✅ **ALL 11 TASKS COMPLETE** - 100% completeness retention achieved!
**Checkpoint**: User Story 3 complete - conflict resolution preserves best data quality

---

## Phase 6: User Story 4 - Preview and Validation (Priority: P3)

**Goal**: Provide --dry-run mode for previewing merge without writing files, verbose logging for debugging

**Independent Test**: Run with --dry-run, verify no files modified while statistics displayed

### Tests for User Story 4 (TDD - Write FIRST)

- [x] T052 [P] [US4] Write unit test tests/unit/merge/merger_test.go for dry-run mode (verify no output written)
- [x] T053 [P] [US4] Write integration test tests/integration/merge_test.go for dry-run preview statistics

### Implementation for User Story 4

- [x] T054 [US4] Add DryRun bool field to MergeConfig in internal/merge/config.go
- [x] T055 [US4] Add --dry-run flag to cobra command in cmd/lastfm-sync/commands/merge.go
- [x] T056 [US4] Implement dry-run logic in internal/merge/merger.go (skip output write, show preview stats)
- [x] T057 [US4] Add estimated output size calculation in internal/merge/merger.go
- [x] T058 [US4] Add date range tracking (earliest/latest timestamp) to MergeStats in internal/merge/stats.go
- [x] T059 [US4] Add unique artists/tracks count to MergeStats in internal/merge/stats.go
- [x] T060 [US4] Enhance summary output with date range, unique counts, output size in cmd/lastfm-sync/commands/merge.go
- [x] T061 [US4] Add --verbose flag for DEBUG level logging in cmd/lastfm-sync/commands/merge.go
- [x] T062 [US4] Run unit test (T052) and verify dry-run doesn't write files
- [x] T063 [US4] Run integration test (T053) and verify preview statistics accurate

✅ **ALL 12 TASKS COMPLETE** - Dry-run mode with preview statistics working!
**Checkpoint**: User Story 4 complete - dry-run and verbose modes work

---

## Phase 7: User Story 5 - Different Deduplication Strategies (Priority: P3)

**Goal**: Support --strategy flag with 4 options (default, strict, relaxed, mbid) for different use cases

**Independent Test**: Run merge with each strategy on same dataset, verify different outputs

### Tests for User Story 5 (TDD - Write FIRST)

- [x] T064 [P] [US5] Write integration test tests/integration/merge_test.go comparing default vs strict strategy (annotations)
- [x] T065 [P] [US5] Write integration test tests/integration/merge_test.go for relaxed strategy (time window grouping)
- [x] T066 [P] [US5] Write integration test tests/integration/merge_test.go for mbid strategy (MusicBrainz ID matching)

### Implementation for User Story 5

- [x] T067 [US5] Add --strategy flag to cobra command with validation (default|strict|relaxed|mbid) in cmd/lastfm-sync/commands/merge.go
- [x] T068 [US5] Pass strategy to MergeConfig and DeduplicationMap in cmd/lastfm-sync/commands/merge.go
- [x] T069 [US5] Implement strict strategy key generation (includes Duration) in internal/merge/strategies.go
- [x] T070 [US5] Implement relaxed strategy key generation (excludes Album) in internal/merge/strategies.go
- [x] T071 [US5] Implement mbid strategy key generation (MusicBrainz ID + fallback) in internal/merge/strategies.go
- [x] T072 [US5] Add strategy indicator to summary output in cmd/lastfm-sync/commands/merge.go
- [x] T073 [US5] Run integration tests (T064, T065, T066) and verify strategy differences

✅ **ALL 10 TASKS COMPLETE** - All 4 deduplication strategies verified!
**Checkpoint**: User Story 5 complete - all 4 deduplication strategies work correctly

---

## Phase 8: User Story 6 - Long-Running Operations (Priority: P3)

**Goal**: Support checkpointing and resume for large datasets to recover from interruptions

**Independent Test**: Run merge on large dataset, interrupt (Ctrl+C), resume and verify continues from checkpoint

### Tests for User Story 6 (TDD - Write FIRST)

- [x] T074 [P] [US6] Write unit test tests/unit/merge/checkpoint_test.go for checkpoint save/load round-trip
- [x] T075 [P] [US6] Write unit test tests/unit/merge/checkpoint_test.go for checkpoint version validation
- [x] T076 [P] [US6] Write integration test tests/integration/merge_test.go for resume from checkpoint (simulate interruption)

### Implementation for User Story 6

- [x] T077 [US6] Implement MergeCheckpoint struct in internal/merge/checkpoint.go per data-model.md
- [x] T078 [US6] Implement Save() method with atomic write (temp + rename) in internal/merge/checkpoint.go
- [x] T079 [US6] Implement LoadCheckpoint() with version validation in internal/merge/checkpoint.go
- [x] T080 [US6] Add CheckpointInterval field to MergeConfig in internal/merge/config.go
- [x] T081 [US6] Add --checkpoint-interval and --checkpoint-path flags in cmd/lastfm-sync/commands/merge.go
- [x] T082 [US6] Add --resume flag for loading checkpoint in cmd/lastfm-sync/commands/merge.go
- [ ] T083 [US6] Integrate checkpoint saving every N scrobbles in internal/merge/merger.go
- [ ] T084 [US6] Implement resume logic (skip processed files, resume from current line) in internal/merge/merger.go
- [ ] T085 [US6] Add checkpoint deletion on successful completion in internal/merge/merger.go
- [x] T086 [US6] Add checkpoint config validation (strategy, input files match) in internal/merge/checkpoint.go
- [x] T087 [US6] Run unit tests (T074, T075) and verify checkpoint serialization works
- [ ] T088 [US6] Run integration test (T076) and verify resume from checkpoint works

**Checkpoint**: User Story 6 complete - checkpointing enables reliable large dataset processing

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories, documentation, and final validation

- [ ] T089 [P] Add exit code handling (0=success, 1=general, 2=input, 3=resume, 4=write, 5=validation) in cmd/lastfm-sync/commands/merge.go
- [ ] T090 [P] Verify all error messages follow format: clear problem + actionable guidance
- [x] T091 [P] Add --conflict-resolution flag (completeness|first|last) in cmd/lastfm-sync/commands/merge.go
- [x] T092 [P] Update README.md with merge command examples and usage
- [ ] T093 [P] Verify contracts/merge-command.md examples all work correctly
- [x] T094 Run full test suite and verify ≥80% coverage per Constitution
- [x] T095 Run benchmark suite and verify performance targets (≥10K scrobbles/sec, <500MB for 1M)
- [x] T096 Test against real Last.fm export data (various sizes: 1K, 10K, 100K scrobbles)
- [x] T097 [P] Code cleanup and refactoring (verify cyclomatic complexity <10 per function)
- [ ] T098 Validate quickstart.md examples and verify all commands work
- [ ] T099 Final integration test with Azure Blob Storage (end-to-end)
- [x] T100 Update CHANGELOG.md with feature 006 release notes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies - can start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 completion - BLOCKS all user stories
- **Phase 3+ (User Stories)**: All depend on Phase 2 completion
  - User stories can proceed in parallel (if multiple developers)
  - Or sequentially in priority order: US1 → US2 → US3 → US4 → US5 → US6
- **Phase 9 (Polish)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: No dependencies on other stories - MVP delivery
- **User Story 2 (P2)**: Independent, but enhances US1 error handling
- **User Story 3 (P2)**: Independent, but enhances US1 deduplication quality
- **User Story 4 (P3)**: Independent, adds preview/validation to US1
- **User Story 5 (P3)**: Independent, extends US1 with strategy options
- **User Story 6 (P3)**: Independent, adds resume capability to US1

### Within Each User Story

1. **Tests FIRST** (TDD red-green-refactor)
2. Models/data structures
3. Core logic implementation
4. CLI integration
5. Verify tests pass
6. Independent test validation

### Parallel Opportunities

**Phase 1 (Setup)**: T001, T002, T003, T004 can all run in parallel

**Phase 2 (Foundational)**: 
- T005, T006 can run in parallel (different files)
- T007/T008 sequential (test → implementation)
- T009/T010 sequential (test → implementation)
- T011/T012 sequential (test → implementation)
- T013/T014 sequential (test → implementation)

**User Story Tests**: All test tasks within a story marked [P] can run in parallel

**User Stories**: After Phase 2, all user stories can be implemented in parallel by different team members:
- Team Member A: US1 (MVP)
- Team Member B: US2 (Error handling)
- Team Member C: US3 (Conflict resolution)
- Team Member D: US4 (Preview mode)
- Team Member E: US5 (Strategies)
- Team Member F: US6 (Checkpointing)

**Phase 9 (Polish)**: All tasks marked [P] can run in parallel

---

## MVP Scope Recommendation

**Minimum Viable Product**: User Story 1 only

**Rationale**: US1 delivers core value (merge + deduplicate + output) with progress indication. Users can immediately consolidate their scrobble data. US2-US6 are enhancements that improve reliability, flexibility, and user experience but aren't required for basic functionality.

**MVP Task Count**: 29 tasks (T001-T029)

**MVP Delivery**:
1. Phase 1: Setup (4 tasks)
2. Phase 2: Foundational (10 tasks)
3. Phase 3: User Story 1 (15 tasks)

**Post-MVP Increments**:
- **Increment 2**: Add US2 (error handling) + US3 (conflict resolution) - 22 tasks
- **Increment 3**: Add US4 (preview) + US5 (strategies) - 21 tasks
- **Increment 4**: Add US6 (checkpointing) - 12 tasks
- **Final**: Polish (12 tasks)

---

## Task Statistics

**Total Tasks**: 100
- **Phase 1 (Setup)**: 4 tasks
- **Phase 2 (Foundational)**: 10 tasks (BLOCKING)
- **Phase 3 (US1 - MVP)**: 15 tasks
- **Phase 4 (US2)**: 11 tasks
- **Phase 5 (US3)**: 11 tasks
- **Phase 6 (US4)**: 12 tasks
- **Phase 7 (US5)**: 10 tasks
- **Phase 8 (US6)**: 15 tasks
- **Phase 9 (Polish)**: 12 tasks

**Parallelization Opportunities**: 35 tasks marked [P] can run in parallel within their phase

**Test Tasks**: 21 (all TDD - written before implementation)
**Implementation Tasks**: 67
**Polish Tasks**: 12

**Coverage Target**: ≥80% per Constitution (verify with T094)
**Performance Targets**: ≥10K scrobbles/sec, <500MB for 1M scrobbles (verify with T095)

---

## Implementation Strategy

1. **TDD Approach**: Every feature starts with failing tests (red-green-refactor)
2. **Incremental Delivery**: MVP (US1) → Enhancements (US2-US3) → Advanced (US4-US6)
3. **Independent Testing**: Each user story has acceptance tests that verify functionality in isolation
4. **Parallel Execution**: Foundation complete → 6 user stories can proceed in parallel
5. **Quality Gates**: 
   - 80% test coverage before merge
   - All benchmarks pass performance targets
   - Cyclomatic complexity <10 per function
   - All integration tests pass

---

**Tasks Generated**: ✅ Ready for implementation  
**Next Step**: Begin Phase 1 (Setup) or jump directly to MVP (T001-T029)
