# Tasks: Console Progress Bar

**Input**: Design documents from `/specs/005-console-progress-bar/`  
**Prerequisites**: plan.md, spec.md, data-model.md, contracts/, quickstart.md, research.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Go project at repository root
- Package: `internal/progress/`
- Tests alongside implementation: `*_test.go`
- Integration tests: `tests/integration/`
- Benchmarks: `tests/benchmarks/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Add `github.com/schollz/progressbar/v3` dependency to go.mod via `go get github.com/schollz/progressbar/v3`
- [X] T002 Add `golang.org/x/term` dependency to go.mod via `go get golang.org/x/term`
- [X] T003 Create package directory structure `internal/progress/`
- [X] T004 [P] Create package documentation file `internal/progress/doc.go` with package-level comments

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Define ProgressReporter interface in `internal/progress/reporter.go` with Start(), Add(), SetCurrent(), SetDescription(), Finish(), FinishWithError(), FinishWithWarning(), SwitchToSpinner(), ResumeProgress(), IsFinished() methods
- [X] T006 [P] Define ProgressState enum in `internal/progress/state.go` with StateActive, StateSuccess, StateError, StateWarning, StateSpinner constants
- [X] T007 [P] Define Options struct in `internal/progress/options.go` with Width, ShowSpeed, ShowETA, ShowElapsed, ShowPercent, ShowCount, RefreshRate, Style, EnableColors, AutoClear, Writer fields
- [X] T008 [P] Define Style struct in `internal/progress/style.go` with BarStart, BarEnd, Complete, InProgress, Incomplete, SpinnerFrames fields
- [X] T009 Implement DefaultOptions in `internal/progress/options.go` with sensible defaults per data-model.md
- [X] T010 [P] Implement predefined styles (StyleBlocks, StyleArrows, StyleDots, StyleASCII) in `internal/progress/style.go`
- [X] T011 [P] Implement functional options pattern helpers (WithWidth, WithStyle, WithColors, WithSpeed, WithETA, etc.) in `internal/progress/options.go`
- [X] T012 Define TerminalInfo struct in `internal/progress/terminal.go` with Width, Height, SupportsUTF8, SupportsColor, IsInteractive fields
- [X] T013 Implement DetectTerminal() function in `internal/progress/terminal.go` using golang.org/x/term to query terminal capabilities
- [X] T014 Implement TerminalInfo.BestStyle() method in `internal/progress/terminal.go` returning appropriate style based on capabilities
- [X] T015 Implement TerminalInfo.ShouldDisplay() method in `internal/progress/terminal.go` returning whether progress bars should be shown

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Basic Progress Visibility (Priority: P1) 🎯 MVP

**Goal**: Display a visual progress bar during long-running operations showing real-time progress from 0-100%

**Independent Test**: Run Last.fm sync operation with 100+ items and verify progress bar appears, updates smoothly, shows percentage completion

### Implementation for User Story 1

- [X] T016 [P] [US1] Implement NoOpProgressBar struct in `internal/progress/noop.go` implementing ProgressReporter interface with silent no-op methods
- [X] T017 [P] [US1] Implement NewNoOpProgressBar() factory function in `internal/progress/noop.go`
- [X] T018 [P] [US1] Create unit tests for NoOpProgressBar in `internal/progress/noop_test.go` verifying all methods can be called without errors
- [X] T019 [US1] Implement ProgressBar struct in `internal/progress/bar.go` with fields: current, total, description, startTime, style, options, bar, mu, finished, state
- [X] T020 [US1] Implement NewRealProgressBar() factory function in `internal/progress/bar.go` that creates progressbar.ProgressBar instance with configured options
- [X] T021 [US1] Implement ProgressBar.Start() method in `internal/progress/bar.go` initializing progress tracking with total and description
- [X] T022 [US1] Implement ProgressBar.Add() method in `internal/progress/bar.go` with mutex-protected increment and validation (no negative, no exceed total)
- [X] T023 [US1] Implement ProgressBar.SetCurrent() method in `internal/progress/bar.go` with mutex-protected set and validation
- [X] T024 [US1] Implement ProgressBar.SetDescription() method in `internal/progress/bar.go` updating operation description
- [X] T025 [US1] Implement ProgressBar.Finish() method in `internal/progress/bar.go` marking state as StateSuccess and optionally clearing bar
- [X] T026 [US1] Implement ProgressBar.IsFinished() method in `internal/progress/bar.go` returning finished status
- [X] T027 [US1] Implement NewProgressReporter() factory function in `internal/progress/factory.go` that returns NoOpProgressBar if terminal non-interactive or progress disabled, otherwise RealProgressBar
- [X] T028 [US1] Create unit tests for ProgressBar in `internal/progress/bar_test.go` testing state transitions, validation, thread safety with parallel updates
- [X] T029 [US1] Add ProgressConfig struct to `internal/config/types.go` with Enabled, Style, ShowSpeed, ShowETA, ShowCount, ShowPercentage, ShowElapsed, Width, RefreshRate, Colors, AutoClear fields
- [X] T030 [US1] Implement DefaultProgressConfig() in `internal/config/defaults.go` returning default progress configuration
- [X] T031 [US1] Add progress config loading in `internal/config/loader.go` with environment variable overrides (SPECKIT_NO_PROGRESS, SPECKIT_PROGRESS_ASCII, SPECKIT_NO_COLOR, SPECKIT_PROGRESS_REFRESH, SPECKIT_PROGRESS_WIDTH)
- [X] T032 [US1] Update SyncService constructor in `internal/service/sync.go` to accept ProgressReporter parameter
- [X] T033 [US1] Integrate progress reporting in SyncService.Sync() method calling Start(), Add(), Finish() during fetch loop
- [X] T034 [US1] Update fetch command in `cmd/lastfm-sync/commands/fetch.go` to create ProgressReporter and pass to SyncService
- [X] T035 [US1] Create integration test in `tests/integration/progress_lastfm_test.go` running actual Last.fm sync with mock API and verifying progress bar updates

**Checkpoint**: At this point, User Story 1 should be fully functional - progress bars display during Last.fm sync with real-time updates

---

## Phase 4: User Story 5 - Error and Completion States (Priority: P2)

**Goal**: Display clear visual indication of success, error, or warning states when operations complete or fail

**Independent Test**: Run operations that succeed, fail, or encounter warnings and verify appropriate visual states (colors, messages) are displayed

### Implementation for User Story 5

- [ ] T036 [P] [US5] Implement ProgressBar.FinishWithError() method in `internal/progress/bar.go` marking state as StateError, displaying red color, and showing error message
- [ ] T037 [P] [US5] Implement ProgressBar.FinishWithWarning() method in `internal/progress/bar.go` marking state as StateWarning, displaying yellow color, and showing warning message
- [ ] T038 [US5] Add signal handler in `internal/progress/bar.go` catching SIGINT/SIGTERM and calling cleanup method
- [ ] T039 [US5] Implement ProgressBar.Clear() method in `internal/progress/bar.go` clearing progress bar from terminal
- [ ] T040 [P] [US5] Create unit tests in `internal/progress/bar_test.go` for error and warning states verifying correct state transitions and colors
- [ ] T041 [US5] Update SyncService error handling in `internal/service/sync.go` to call reporter.FinishWithError() on failures
- [ ] T042 [US5] Create integration test in `tests/integration/progress_error_test.go` simulating operation failures and verifying error state display

**Checkpoint**: At this point, User Stories 1 AND 5 should both work independently - progress bars show success/error/warning states

---

## Phase 5: User Story 6 - Multi-Operation Sequences (Priority: P2)

**Goal**: Display completed operations stacked above with checkmarks while current operation shows active progress bar

**Independent Test**: Run multi-step workflow (fetch, normalize, export) and verify completed operations remain visible with checkmarks while current operation updates

### Implementation for User Story 6

- [ ] T043 [US6] Implement MultiProgress struct in `internal/progress/multi.go` with bars slice, mu sync.Mutex, writer io.Writer fields
- [ ] T044 [US6] Implement NewMulti() factory function in `internal/progress/multi.go` creating MultiProgress instance
- [ ] T045 [US6] Implement MultiProgress.AddBar() method in `internal/progress/multi.go` adding new progress bar to container and returning it
- [ ] T046 [US6] Implement MultiProgress.RemoveBar() method in `internal/progress/multi.go` removing completed bar from active display
- [ ] T047 [US6] Implement MultiProgress.Wait() method in `internal/progress/multi.go` blocking until all bars complete
- [ ] T048 [US6] Implement MultiProgress.Clear() method in `internal/progress/multi.go` removing all progress displays
- [ ] T049 [P] [US6] Implement completed operation rendering in `internal/progress/multi.go` showing checkmark ✓ and summary message for finished bars
- [ ] T050 [P] [US6] Create unit tests in `internal/progress/multi_test.go` testing multi-bar coordination, stacking behavior, and completion tracking
- [ ] T051 [US6] Create full sync workflow command in `cmd/lastfm-sync/commands/full_sync.go` using MultiProgress for fetch, normalize, export sequence
- [ ] T052 [US6] Create integration test in `tests/integration/progress_multi_test.go` running multi-step workflow and verifying stacking behavior

**Checkpoint**: At this point, User Stories 1, 5, AND 6 should all work independently - multi-operation workflows display correctly

---

## Phase 6: User Story 2 - Time and Speed Information (Priority: P2)

**Goal**: Display estimated time remaining (ETA) and processing speed (items/second) to help users make decisions

**Independent Test**: Run title normalization on 1000+ tracks and verify ETA and speed are displayed and reasonably accurate

### Implementation for User Story 2

- [ ] T053 [US2] Verify schollz/progressbar library is configured to show speed via options in `internal/progress/bar.go`
- [ ] T054 [US2] Verify schollz/progressbar library is configured to show ETA via options in `internal/progress/bar.go`
- [ ] T055 [P] [US2] Update Options struct usage in `internal/progress/bar.go` to enable ShowSpeed and ShowETA based on config
- [ ] T056 [US2] Update NormalizeScrobbles() signature in `internal/normalize/normalize.go` to accept ProgressReporter parameter
- [ ] T057 [US2] Integrate progress reporting in `internal/normalize/normalize.go` with batched updates (every 100 items) calling reporter.Add()
- [ ] T058 [US2] Create integration test in `tests/integration/progress_normalize_test.go` running normalization on 1000+ items and verifying ETA/speed display accuracy within 20%

**Checkpoint**: At this point, User Stories 1, 2, 5, and 6 should all work - progress bars show time and speed information

---

## Phase 7: User Story 3 - Multiple Visual Styles (Priority: P3)

**Goal**: Progress bars adapt to terminal capabilities (Unicode → ASCII fallback, colors, width adjustment)

**Independent Test**: Run SpecKit in various terminal environments and verify appropriate style is used automatically

### Implementation for User Story 3

- [ ] T059 [P] [US3] Implement terminal resize handler in `internal/progress/bar.go` catching SIGWINCH signal and recalculating layout
- [ ] T060 [P] [US3] Implement width adaptation in `internal/progress/bar.go` adjusting bar width based on TerminalInfo.Width
- [ ] T061 [US3] Update NewProgressReporter() in `internal/progress/factory.go` to call DetectTerminal() and select appropriate style via BestStyle()
- [ ] T062 [P] [US3] Create unit tests in `internal/progress/terminal_test.go` testing terminal detection for various capabilities (UTF-8, colors, width)
- [ ] T063 [P] [US3] Create unit tests in `internal/progress/style_test.go` testing style selection based on terminal capabilities
- [ ] T064 [US3] Create integration test in `tests/integration/progress_terminal_test.go` simulating different terminal environments and verifying correct style selection

**Checkpoint**: At this point, User Stories 1, 2, 3, 5, and 6 should all work - progress bars adapt to terminal capabilities

---

## Phase 8: User Story 4 - Configuration Control (Priority: P3)

**Goal**: Allow progress bars to be disabled via environment variables, config file, or command flags

**Independent Test**: Run SpecKit with various configuration options and verify progress bars are enabled/disabled as specified

### Implementation for User Story 4

- [ ] T065 [P] [US4] Add --no-progress flag to fetch command in `cmd/lastfm-sync/commands/fetch.go`
- [ ] T066 [P] [US4] Add --no-progress flag to other commands that use progress bars
- [ ] T067 [US4] Update NewProgressReporter() in `internal/progress/factory.go` to check command flags, environment variables, and config file in precedence order
- [ ] T068 [P] [US4] Create unit tests in `internal/config/config_test.go` testing progress config loading with various overrides
- [ ] T069 [US4] Create integration test in `tests/integration/progress_config_test.go` running operations with different config combinations and verifying progress bar presence/absence

**Checkpoint**: All user stories (1-6) should now be independently functional - progress bars fully configurable

---

## Phase 9: Additional Integration Points

**Purpose**: Extend progress bar support to remaining operations identified in plan.md

- [ ] T070 [P] Update Writer.Write() interface signature in `internal/writer/writer.go` to accept optional ProgressReporter parameter
- [ ] T071 [P] Implement progress reporting in LocalWriter.Write() in `internal/writer/local.go` with batched updates during file write
- [ ] T072 [P] Implement progress reporting in AzureWriter.Write() in `internal/writer/azure.go` with chunk-based updates during upload
- [ ] T073 Update MockWriter in `internal/writer/mock.go` to accept ProgressReporter parameter
- [ ] T074 [P] Create integration test in `tests/integration/progress_export_test.go` running file export with progress tracking

---

## Phase 10: Advanced Features

**Purpose**: Implement rate limiting spinner mode and log message handling

- [ ] T075 Implement ProgressBar.SwitchToSpinner() method in `internal/progress/bar.go` converting to indeterminate spinner using SpinnerFrames
- [ ] T076 Implement ProgressBar.ResumeProgress() method in `internal/progress/bar.go` resuming normal progress bar from spinner
- [ ] T077 Update rate limiter integration in `internal/service/sync.go` to call SwitchToSpinner() during waits and ResumeProgress() after
- [ ] T078 [P] Create unit tests in `internal/progress/bar_test.go` testing spinner mode transitions
- [ ] T079 Update Logger in `internal/logging/logger.go` to detect active progress bar and call Clear() before logging, allowing bar to redraw after
- [ ] T080 Add SetProgressBar() method to Logger in `internal/logging/logger.go` registering active progress bar
- [ ] T081 [P] Create integration test in `tests/integration/progress_logging_test.go` verifying logs don't interfere with progress display

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T082 [P] Create comprehensive example in `examples/progress_demo.go` demonstrating all progress bar features
- [ ] T083 Update documentation in `docs/` adding progress bar usage guide with examples
- [ ] T084 Update README.md with progress bar feature description and configuration options
- [ ] T085 [P] Create benchmark suite in `tests/benchmarks/progress_bench_test.go` testing CPU overhead (target < 1%), memory usage (target < 1MB), update rate
- [ ] T086 Run benchmarks and verify performance targets met (SC-002: 10+ FPS, SC-006: < 1% CPU overhead)
- [ ] T087 [P] Add performance validation to CI pipeline in `.github/workflows/` running benchmarks on every PR
- [ ] T088 Code review and refactoring for consistency across all progress bar code
- [ ] T089 Security review ensuring no sensitive data leaked in progress descriptions
- [ ] T090 Validate against quickstart.md ensuring all documented patterns work correctly
- [ ] T091 Update CHANGELOG.md documenting new progress bar feature

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-8)**: All depend on Foundational phase completion
  - User Story 1 (P1 - MVP): Can start after Foundational - **HIGHEST PRIORITY**
  - User Story 5 (P2): Can start after US1 or in parallel (different methods on same struct)
  - User Story 6 (P2): Can start after US1 (requires working ProgressBar)
  - User Story 2 (P2): Can start after US1 or in parallel (configuration only)
  - User Story 3 (P3): Can start after US1 or in parallel (terminal detection + factory logic)
  - User Story 4 (P3): Can start after US1 or in parallel (configuration layer only)
- **Additional Integration (Phase 9)**: Can start after US1 complete (extends to more operations)
- **Advanced Features (Phase 10)**: Can start after US1 complete (adds spinner mode)
- **Polish (Phase 11)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1) - Basic Progress**: Foundation only - **START HERE FOR MVP**
- **User Story 5 (P2) - Error States**: Requires US1 ProgressBar struct - extends methods
- **User Story 6 (P2) - Multi-Operation**: Requires US1 ProgressBar working - adds MultiProgress coordinator
- **User Story 2 (P2) - Time/Speed**: Requires US1 - configuration options only
- **User Story 3 (P3) - Styles**: Requires US1 - terminal detection + factory selection
- **User Story 4 (P3) - Configuration**: Requires US1 - config layer only

### Within Each User Story

- Foundational types and interfaces first (reporter.go, state.go, options.go, style.go)
- Terminal detection next (terminal.go)
- Core implementation (bar.go, noop.go)
- Factory logic (factory.go)
- Configuration integration (config/)
- Service integration (service/, commands/)
- Tests last (verifying implementation)

### Parallel Opportunities

#### Phase 1: Setup
- Tasks T001-T004 can all run in parallel (different files)

#### Phase 2: Foundational
- T006 (state.go), T007 (options.go), T008 (style.go) can run in parallel
- T010 (predefined styles), T011 (option helpers) can run in parallel after T007-T008
- T012-T015 (terminal detection) can run in parallel with options work

#### Phase 3: User Story 1
- T016-T018 (NoOpProgressBar) can run in parallel with T019-T026 (RealProgressBar)
- T029-T031 (config integration) can run in parallel with T019-T026
- T028 (tests) can run after T019-T026 complete

#### Cross-Story Parallelization (after Foundational)
Once Phase 2 is complete, multiple developers can work in parallel:
- **Developer A**: User Story 1 (T016-T035) - MVP implementation
- **Developer B**: User Story 3 (T059-T064) - Terminal adaptation (once T016-T027 from US1 done)
- **Developer C**: User Story 4 (T065-T069) - Configuration (once T027 from US1 done)

#### Within User Story 5
- T036 (error), T037 (warning) can run in parallel
- T040 (tests) can run after T036-T037

#### Within User Story 6
- T049 (rendering), T050 (tests) can run in parallel with T043-T048

#### Phase 9: Integration Points
- T070-T072 (Writer implementations) can all run in parallel (different files)

#### Phase 11: Polish
- T082 (examples), T083 (docs), T085 (benchmarks) can all run in parallel

---

## Parallel Example: User Story 1

```bash
# After Phase 2 (Foundational) is complete:

# Launch NoOp implementation:
Task: "[US1] Implement NoOpProgressBar struct in internal/progress/noop.go"
Task: "[US1] Implement NewNoOpProgressBar() factory in internal/progress/noop.go"
Task: "[US1] Create unit tests for NoOpProgressBar in internal/progress/noop_test.go"

# Launch RealProgressBar implementation in parallel:
Task: "[US1] Implement ProgressBar struct in internal/progress/bar.go"
Task: "[US1] Implement NewRealProgressBar() factory in internal/progress/bar.go"
Task: "[US1] Implement ProgressBar.Start() method in internal/progress/bar.go"
# ... all bar.go methods ...

# Launch config integration in parallel:
Task: "[US1] Add ProgressConfig struct to internal/config/types.go"
Task: "[US1] Implement DefaultProgressConfig() in internal/config/defaults.go"
Task: "[US1] Add progress config loading in internal/config/loader.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. ✅ Complete Phase 1: Setup (T001-T004)
2. ✅ Complete Phase 2: Foundational (T005-T015) - **CRITICAL - blocks all stories**
3. 🎯 Complete Phase 3: User Story 1 (T016-T035) - **FOCUS HERE**
4. **STOP and VALIDATE**: Test User Story 1 independently with Last.fm sync
5. Deploy/demo basic progress bars

**MVP Scope**: Tasks T001-T035 deliver immediate value - users see progress during sync operations

### Incremental Delivery

1. Setup + Foundational (T001-T015) → Foundation ready
2. **User Story 1** (T016-T035) → Test independently → **Deploy/Demo (MVP!)**
3. **User Story 5** (T036-T042) → Add error/warning states → Deploy/Demo
4. **User Story 6** (T043-T052) → Add multi-operation support → Deploy/Demo
5. **User Story 2** (T053-T058) → Add time/speed info → Deploy/Demo
6. **User Story 3** (T059-T064) → Add terminal adaptation → Deploy/Demo
7. **User Story 4** (T065-T069) → Add configuration → Deploy/Demo
8. Polish + Integration (T070-T091) → Production ready

Each story adds value without breaking previous stories.

### Parallel Team Strategy

With multiple developers after Foundational phase (T015):

1. **Team completes Setup + Foundational together** (T001-T015)
2. Once Foundational is done:
   - **Developer A**: User Story 1 Core (T016-T026) → Most critical path
   - **Developer B**: User Story 1 Config (T029-T031) → Can work in parallel
   - **Developer C**: User Story 1 Integration (T032-T035) → After T026 done
3. After US1 complete:
   - **Developer A**: User Story 5 (T036-T042)
   - **Developer B**: User Story 6 (T043-T052)
   - **Developer C**: User Story 2 (T053-T058)
4. Continue with remaining stories in parallel

---

## Task Summary

- **Total Tasks**: 91
- **Setup**: 4 tasks
- **Foundational**: 11 tasks (blocking)
- **User Story 1 (P1 - MVP)**: 20 tasks ⭐
- **User Story 5 (P2)**: 7 tasks
- **User Story 6 (P2)**: 10 tasks
- **User Story 2 (P2)**: 6 tasks
- **User Story 3 (P3)**: 6 tasks
- **User Story 4 (P3)**: 5 tasks
- **Additional Integration**: 5 tasks
- **Advanced Features**: 7 tasks
- **Polish**: 10 tasks

**Parallel Opportunities**: 35+ tasks marked [P] can run concurrently

**MVP Scope** (Phases 1-3): 35 tasks → ~16-20 hours estimated implementation time

**Full Feature** (All phases): 91 tasks → ~40-50 hours estimated (matches research.md timeline when accounting for learning/polish)

---

## Validation Checklist

Before marking feature complete, verify:

- [ ] **SC-001**: Progress bar appears within 100ms of operation start
- [ ] **SC-002**: Updates at 10+ FPS without flickering
- [ ] **SC-003**: Works on Linux, macOS, Windows terminals
- [ ] **SC-004**: Can be disabled via env var, config, or flag
- [ ] **SC-005**: ETA within 20% accuracy after 10% progress
- [ ] **SC-006**: < 1% CPU overhead verified via benchmarks
- [ ] **SC-007**: ASCII fallback works on limited terminals
- [ ] **SC-008**: Terminal resize handled within 200ms
- [ ] **SC-009**: Ctrl+C interruption cleans up properly
- [ ] **SC-011**: Multi-operation workflows display correctly
- [ ] All user stories independently testable
- [ ] No shared state between stories
- [ ] Quickstart.md examples work correctly
- [ ] All integration tests pass
- [ ] Performance benchmarks pass targets

---

## Notes

- **[P] tasks** = different files, no dependencies
- **[Story] label** maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- **Tests are NOT requested** in spec.md - implementation-focused approach
- Focus on User Story 1 (MVP) first before adding additional features
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
