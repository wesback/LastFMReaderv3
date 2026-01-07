# Implementation Tasks: Add Local Time Field to Last.fm API Response

**Branch**: `003-local-time-field`  
**Date**: 2026-01-07  
**Feature**: Add human-readable `local_time` field to Scrobble JSON output

---

## Overview

This document breaks down the implementation into executable tasks following Test-Driven Development (TDD) principles. The feature adds a single field to the existing Scrobble struct with minimal complexity and zero breaking changes.

**Key Characteristics**:
- Single user story (no complex dependencies)
- TDD approach: Tests written before implementation
- Isolated changes to `internal/models/scrobble.go`
- Target: >80% test coverage for new code
- Backward compatible (additive change only)

---

## User Stories

### User Story 1: Human-Readable Time Field
**Priority**: P1  
**As a**: Developer consuming scrobble JSON data  
**I want**: A human-readable time field alongside the Unix timestamp  
**So that**: I can quickly understand when tracks were played without manual conversion

**Acceptance Criteria**:
1. JSON output includes `local_time` field in RFC3339 format (UTC)
2. `local_time` field always present (empty string for invalid timestamps)
3. Existing `uts` field unchanged
4. No breaking changes to JSON structure
5. Time conversion accurate (matches `uts` value)

**Independent Test**:
Run a sync and verify:
- All scrobbles contain `local_time` field
- Format is RFC3339: `YYYY-MM-DDTHH:MM:SSZ`
- Value matches corresponding `uts` field when converted
- Edge cases handled: uts=0 returns empty string

---

## Task Phases

### Phase 1: Setup & Preparation

**Objective**: Prepare development environment and verify current implementation

- [x] T001 Review current Scrobble struct in internal/models/scrobble.go
- [x] T002 Review NewScrobble constructor in internal/models/scrobble.go
- [x] T003 Verify test file exists at internal/models/scrobble_test.go

**Deliverables**: Understanding of current code structure

---

### Phase 2: User Story 1 - Human-Readable Time Field

**Objective**: Implement `local_time` field with TDD approach

#### Tests (Write First - TDD)

- [x] T004 [P] [US1] Write test for formatTimestamp with valid timestamp in internal/models/scrobble_test.go
- [x] T005 [P] [US1] Write test for formatTimestamp with zero timestamp in internal/models/scrobble_test.go
- [x] T006 [P] [US1] Write test for formatTimestamp with negative timestamp in internal/models/scrobble_test.go
- [x] T007 [P] [US1] Write test for formatTimestamp with large timestamp (2038+) in internal/models/scrobble_test.go
- [x] T008 [US1] Write test for NewScrobble with LocalTime field population in internal/models/scrobble_test.go
- [x] T009 [US1] Write test for JSON marshaling includes local_time field in internal/models/scrobble_test.go

**Test Cases Summary**:
```go
// T004: TestFormatTimestamp_Valid
// Input: uts = 1704556800
// Expected: "2024-01-06T14:00:00Z"

// T005: TestFormatTimestamp_Zero
// Input: uts = 0
// Expected: ""

// T006: TestFormatTimestamp_Negative
// Input: uts = -1000
// Expected: ""

// T007: TestFormatTimestamp_Large
// Input: uts = 2147483648 (beyond 2038)
// Expected: "2038-01-19T03:14:08Z"

// T008: TestNewScrobble_LocalTime
// Verify LocalTime field populated correctly

// T009: TestScrobble_MarshalJSON_LocalTime
// Verify JSON output contains local_time field
```

**Verification**: Run `go test ./internal/models/scrobble_test.go` - all tests should FAIL (Red phase)

#### Implementation

- [x] T010 [US1] Add LocalTime field to Scrobble struct in internal/models/scrobble.go
- [x] T011 [US1] Implement formatTimestamp helper function in internal/models/scrobble.go
- [x] T012 [US1] Update NewScrobble to populate LocalTime field in internal/models/scrobble.go

**Implementation Details**:

**T010**: Add field after UTS:
```go
type Scrobble struct {
    Username   string          `json:"username"`
    Artist     string          `json:"artist"`
    Track      string          `json:"track"`
    Album      string          `json:"album"`
    UTS        int64           `json:"uts"`
    LocalTime  string          `json:"local_time"`  // NEW
    MBID       *string         `json:"mbid,omitempty"`
    Source     string          `json:"source"`
    IngestedAt string          `json:"ingested_at"`
    Raw        json.RawMessage `json:"raw"`
}
```

**T011**: Add helper function:
```go
// formatTimestamp converts Unix timestamp to RFC3339 UTC format.
// Returns empty string for invalid timestamps (uts <= 0).
func formatTimestamp(uts int64) string {
    if uts <= 0 {
        return ""
    }
    t := time.Unix(uts, 0).UTC()
    return t.Format(time.RFC3339)
}
```

**T012**: Update constructor:
```go
func NewScrobble(username, artist, track, album string, uts int64, mbid *string, raw json.RawMessage) *Scrobble {
    return &Scrobble{
        Username:   username,
        Artist:     artist,
        Track:      track,
        Album:      album,
        UTS:        uts,
        LocalTime:  formatTimestamp(uts),  // NEW LINE
        MBID:       mbid,
        Source:     "lastfm",
        IngestedAt: time.Now().UTC().Format(time.RFC3339),
        Raw:        raw,
    }
}
```

**Verification**: Run `go test ./internal/models/scrobble_test.go` - all tests should PASS (Green phase)

#### Integration & Validation

- [x] T013 [US1] Run full test suite to verify no regressions with `go test ./...`
- [x] T014 [US1] Perform manual sync test to verify JSON output in cmd/lastfm-sync
- [x] T015 [US1] Verify backward compatibility - existing JSON consumers unaffected

**Manual Test**:
```bash
# Build and run sync
go build -o lastfm-sync ./cmd/lastfm-sync
./lastfm-sync fetch --user test_user --limit 10 | jq '.'

# Verify each scrobble has local_time field
./lastfm-sync fetch --user test_user --limit 1 | jq '{uts, local_time}'

# Expected output:
# {
#   "uts": 1704556800,
#   "local_time": "2024-01-06T14:00:00Z"
# }
```

---

### Phase 3: Documentation & Polish

**Objective**: Update documentation and examples

- [x] T016 [P] Update README.md with local_time field example
- [x] T017 [P] Add before/after JSON examples to documentation
- [x] T018 Verify all documentation references are correct
- [x] T019 Run final code quality checks with `go fmt` and `go vet`

**Documentation Updates**:
- README.md: Add example showing new field
- Any existing JSON examples: Add `local_time` field
- Note RFC3339 format and UTC timezone

---

## Dependencies & Execution Order

### Dependency Graph

```
Phase 1 (Setup)
    │
    ├─> Phase 2 (User Story 1)
    │       ├─> T004-T009 (Tests - Parallel)
    │       │       │
    │       │       └─> T010-T012 (Implementation - Sequential)
    │       │               │
    │       │               └─> T013-T015 (Integration - Sequential)
    │       │
    │       └─> Phase 3 (Documentation)
    │               └─> T016-T019 (Parallel except T018-T019)
```

### Execution Strategy

**Critical Path**: T001 → T002 → T003 → T004-T009 → T010 → T011 → T012 → T013 → T014 → T015 → T018 → T019

**Parallel Opportunities**:
1. **Test Writing** (T004-T007): Can write unit tests in parallel (different test cases)
2. **Documentation** (T016-T017): Can update docs in parallel after implementation complete

**Blockers**:
- T010-T012 BLOCKED by T004-T009 (TDD: tests first)
- T013-T015 BLOCKED by T012 (integration after implementation)
- Phase 3 BLOCKED by Phase 2 completion

---

## Parallel Execution Examples

### Phase 2: Tests (Parallel)
```bash
# Developer A: T004-T005
# Write valid and zero timestamp tests

# Developer B: T006-T007
# Write negative and large timestamp tests

# Developer C: T008-T009
# Write NewScrobble and JSON marshaling tests
```

### Phase 3: Documentation (Parallel)
```bash
# Developer A: T016
# Update README.md

# Developer B: T017
# Update documentation examples
```

---

## Test Coverage Requirements

**Target**: >80% coverage for new code

**Coverage Areas**:
- `formatTimestamp` function: 100% (4 test cases cover all branches)
- `NewScrobble` LocalTime population: Covered by T008
- JSON marshaling: Covered by T009

**Coverage Command**:
```bash
go test ./internal/models -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Implementation Strategy

### MVP Scope (Minimum Viable Product)

**User Story 1 (Phase 2)** constitutes the complete MVP:
- Core functionality: `local_time` field addition
- Tests: Full test coverage
- Validation: Manual and automated tests pass

**Incremental Delivery**:
1. ✅ **Phase 1**: Setup (understand current code)
2. ✅ **Phase 2**: Implement feature with tests (deployable)
3. ✅ **Phase 3**: Polish documentation (nice-to-have)

**Deployment Strategy**:
- Deploy after Phase 2 completion
- Phase 3 can follow in subsequent release if needed

---

## Task Summary

### Task Count by Phase

| Phase | Task Count | Parallelizable | Estimated Time |
|-------|-----------|----------------|----------------|
| Phase 1: Setup | 3 | 3 | 15 minutes |
| Phase 2: User Story 1 | 12 | 6 | 2 hours |
| Phase 3: Documentation | 4 | 2 | 30 minutes |
| **Total** | **19** | **11** | **~3 hours** |

### Task Count by Type

| Type | Count | Notes |
|------|-------|-------|
| Test Tasks | 6 | T004-T009 (TDD) |
| Implementation Tasks | 3 | T010-T012 |
| Integration Tasks | 3 | T013-T015 |
| Documentation Tasks | 4 | T016-T019 |
| Setup Tasks | 3 | T001-T003 |

### Parallel Opportunities

**11 of 19 tasks (58%)** can be executed in parallel within their phase:
- Phase 1: All 3 tasks parallelizable (review tasks)
- Phase 2: 6 test tasks parallelizable
- Phase 3: 2 documentation tasks parallelizable

---

## Success Criteria

### Phase Completion

**Phase 1 Complete When**:
- Current implementation understood
- Test file verified

**Phase 2 Complete When**:
- All 6 tests pass ✅
- `local_time` field present in JSON ✅
- Manual sync test successful ✅
- No regressions in test suite ✅

**Phase 3 Complete When**:
- Documentation updated ✅
- Examples include new field ✅
- Code quality checks pass ✅

### Feature Complete When

- [ ] All 19 tasks completed
- [ ] Test coverage >80%
- [ ] All tests passing (unit + integration)
- [ ] Manual verification successful
- [ ] Documentation updated
- [ ] Backward compatibility verified
- [ ] Constitution compliance maintained

---

## Risk Mitigation

### Low-Risk Areas
- **Complexity**: Single field addition, minimal code changes
- **Dependencies**: No external dependencies (stdlib only)
- **Scope**: Well-defined, no feature creep

### Mitigation Strategies
- **TDD**: Tests written first catch issues early
- **Incremental**: Small changes, easy to review and rollback
- **Backward Compatible**: Additive change only, no breaking changes

---

## Next Steps

1. **Start Phase 1**: Review current code (T001-T003)
2. **Begin TDD**: Write tests first (T004-T009)
3. **Implement**: Add field and logic (T010-T012)
4. **Validate**: Run integration tests (T013-T015)
5. **Polish**: Update documentation (T016-T019)

**Estimated Total Time**: ~3 hours for complete implementation

---

**Format Validation**: ✅ All tasks follow checklist format with Task IDs, [P] markers, [Story] labels, and file paths
