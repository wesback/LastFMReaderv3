# Implementation Plan: Add Local Time Field to Last.fm API Response

**Branch**: `003-local-time-field` | **Date**: 2026-01-06 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-local-time-field/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add a human-readable `local_time` field to the Scrobble struct that displays the Unix timestamp (uts) in RFC3339 format (UTC). This enhancement improves JSON output readability while maintaining backward compatibility with existing consumers.

## Technical Context

**Language/Version**: Go 1.24.0+  
**Primary Dependencies**: Go stdlib (time package)  
**Storage**: N/A (field transformation only)  
**Testing**: Go testing package (internal test suite)  
**Target Platform**: Linux (Docker/Azure Container Instances)
**Project Type**: Single (CLI application)  
**Performance Goals**: No measurable impact on sync performance (< 1ms overhead per scrobble)  
**Constraints**: Maintain backward compatibility, no breaking changes to JSON output structure  
**Scale/Scope**: Applied to all scrobbles (potentially millions per sync)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Test-First Development
**Status**: ✅ **PASS** - Will follow TDD
- Unit tests will be written BEFORE implementation in `internal/models/scrobble_test.go`
- Test cases: valid timestamps, edge cases (zero, negative, very large values), JSON marshaling
- Target coverage: >80% for new code
- Integration test: End-to-end sync with new field validation

### II. Code Quality Standards
**Status**: ✅ **PASS** - No complexity concerns
- New time conversion function will be simple (<10 lines, cyclomatic complexity ~2)
- Single field addition to existing struct
- Standard Go formatting enforced
- No linting violations expected

### III. User Experience Consistency
**Status**: ✅ **PASS** - JSON output enhancement only
- Time format: RFC3339 (ISO 8601) - industry standard
- Timezone: UTC (consistent with `ingested_at` field)
- Field naming: `local_time` (snake_case, consistent with existing JSON conventions)
- Empty state: Empty string for invalid/zero timestamps

### IV. Performance Requirements
**Status**: ✅ **PASS** - Minimal performance impact
- `time.Unix()` and `Format()` are stdlib operations (<1ms per call)
- No database queries involved
- No additional API calls required
- Memory impact: ~20 bytes per scrobble for string field

### V. Independent User Story Testing
**Status**: ✅ **PASS** - Independently testable
- Can test JSON marshaling independently
- Can deploy without affecting existing consumers (additive change)
- No dependencies on other features

**Overall Gate Status**: ✅ **PASSED** - Ready for Phase 0

## Project Structure

### Documentation (this feature)

```text
specs/003-local-time-field/
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
├── models/
│   ├── scrobble.go          # ADD: LocalTime field, conversion logic
│   └── scrobble_test.go     # ADD: Unit tests for time conversion
└── [other modules unchanged]

cmd/
└── lastfm-sync/
    └── [unchanged]
```

**Structure Decision**: Single project (CLI application). Changes isolated to `internal/models/scrobble.go` with corresponding tests. No architectural changes required.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

N/A - No constitution violations.

---

## Phase 0: Research & Open Questions

**Objective**: Resolve design decisions and identify best practices for time handling in Go.

### Research Tasks

1. **Time Format Decision**
   - Research: Go time format options and JSON API best practices
   - Question: Confirm RFC3339 (ISO 8601) as preferred format vs custom format
   - Rationale: RFC3339 is industry standard, machine-parseable, human-readable

2. **Timezone Strategy**
   - Research: Timezone handling in distributed systems
   - Question: Confirm UTC as default vs server local time
   - Rationale: UTC aligns with existing `ingested_at` field, avoids timezone confusion

3. **Field Naming**
   - Research: JSON naming conventions in Go projects
   - Question: Confirm `local_time` (snake_case) vs `localTime` (camelCase)
   - Rationale: Existing codebase uses snake_case (`ingested_at`, `max_uts`, etc.)

4. **Edge Case Handling**
   - Research: Best practices for handling invalid timestamps
   - Question: Behavior for zero/negative/future timestamps
   - Options: Empty string, "invalid", omit field entirely
   - Rationale: Empty string maintains JSON structure consistency

5. **Performance Impact**
   - Research: Benchmark time.Unix() and Format() operations
   - Validate: <1ms overhead acceptable for current sync scale
   - Mitigation: None expected needed

### Deliverable

`research.md` documenting:
- Time format choice (RFC3339) and justification
- Timezone strategy (UTC) and alignment with existing fields
- Field naming convention (`local_time`) and consistency rationale
- Edge case handling strategy (empty string for invalid)
- Performance validation results

---

## Phase 1: Design & Contracts

**Prerequisites**: `research.md` complete

### 1.1 Data Model

**Objective**: Document the Scrobble struct enhancement

**Output**: `data-model.md`

**Content**:
- Current Scrobble struct (before)
- Updated Scrobble struct (after) with `LocalTime` field
- Field types and JSON tags
- Relationship to existing fields (derived from `UTS`)
- Validation rules: Empty string if `UTS <= 0`

### 1.2 API Contracts

**Objective**: Define JSON output contract

**Output**: `contracts/scrobble-schema.json` (JSON Schema)

**Schema Elements**:
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "username": { "type": "string" },
    "artist": { "type": "string" },
    "track": { "type": "string" },
    "album": { "type": "string" },
    "uts": { "type": "integer" },
    "local_time": { 
      "type": "string", 
      "format": "date-time",
      "description": "RFC3339 formatted timestamp in UTC derived from uts field"
    },
    "mbid": { "type": ["string", "null"] },
    "source": { "type": "string" },
    "ingested_at": { "type": "string", "format": "date-time" },
    "raw": { "type": "object" }
  },
  "required": ["username", "artist", "track", "album", "uts", "local_time", "source", "ingested_at", "raw"]
}
```

### 1.3 Example Output

**Output**: `contracts/scrobble-example.json`

```json
{
  "username": "testuser",
  "artist": "The Beatles",
  "track": "Hey Jude",
  "album": "Hey Jude",
  "uts": 1704556800,
  "local_time": "2024-01-06T14:00:00Z",
  "mbid": null,
  "source": "lastfm",
  "ingested_at": "2024-01-06T14:05:00Z",
  "raw": { }
}
```

### 1.4 Quickstart

**Output**: `quickstart.md`

**Content**:
- Brief feature description
- Before/after JSON output comparison
- How to verify the new field in existing syncs
- No configuration changes required

### 1.5 Agent Context Update

**Action**: Run `.specify/scripts/bash/update-agent-context.sh copilot`
- Updates `.github/copilot-instructions.md`
- No new technologies (uses existing Go stdlib)

---

## Phase 2: Tasks Breakdown (NOT executed by /speckit.plan)

**Note**: Phase 2 task breakdown is generated by the `/speckit.tasks` command, NOT by this planning phase.

The task breakdown will follow Test-First Development:
1. Write test cases for time conversion
2. Write test cases for JSON marshaling
3. Implement LocalTime field and conversion logic
4. Run tests (should pass)
5. Update documentation

---

## Execution Notes

- **Simplicity**: This is an additive change with minimal complexity
- **Backward Compatibility**: Existing consumers ignore unknown fields
- **No Configuration**: No new environment variables or CLI flags
- **Performance**: Negligible overhead (<1ms per scrobble)
- **Testing Strategy**: Unit tests (conversion logic) + integration test (full sync)

## Success Criteria

- [ ] `research.md` complete with all decisions documented
- [ ] `data-model.md` defines updated Scrobble struct
- [ ] `contracts/` contains JSON schema and example
- [ ] `quickstart.md` provides before/after comparison
- [ ] Constitution check re-evaluated (should remain PASS)
- [ ] Agent context updated with feature details

## Next Steps

1. Execute Phase 0 research (generate `research.md`)
2. Execute Phase 1 design (generate `data-model.md`, `contracts/`, `quickstart.md`)
3. Update agent context
4. Report completion of `/speckit.plan` command
5. User runs `/speckit.tasks` for detailed implementation breakdown
