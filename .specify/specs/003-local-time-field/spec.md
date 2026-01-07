# Feature: Add Local Time Field to Last.fm API Response

## Overview
Enhance the Last.fm API integration by adding a human-readable local time field alongside the existing Unix timestamp (uts) field in JSON output.

## Clarifications

### Session 2026-01-06
- Q: What format should local_time use? → A: RFC3339 (ISO 8601) - `2025-01-06T14:30:00Z`
- Q: Which timezone should be used? → A: UTC (with Z suffix) - consistent with existing `ingested_at` field
- Q: Preferred name for the new field? → A: `local_time` (snake_case, matches API conventions)

## Goals
- Parse Unix timestamp from Last.fm API response
- Convert to human-readable local time format
- Add new field to JSON output without breaking existing structure
- Maintain backward compatibility

## Milestones

### Milestone 1: Analysis and Design
**Objective**: Understand current implementation and design the solution

#### Tasks
1. Locate Last.fm API integration code
   - Identify where uts field is currently processed
   - Review existing JSON marshaling structure
   - Understand current timezone handling (if any)

2. Design the solution
   - Decide on field naming convention (e.g., `local_time`, `localTime`, `timestamp_local`)
   - Choose time format (ISO 8601, RFC3339, custom format)
   - Determine timezone approach:
     - Use server's local timezone?
     - Use UTC with offset?
     - Make timezone configurable?
     - Allow user to specify timezone preference?

3. Document design decisions
   - Create brief design doc explaining choices
   - Consider edge cases (invalid timestamps, null values)

**Deliverables**:
- Design document with field name and format choices
- Code location identified

**Acceptance Criteria**:
- Clear understanding of current implementation
- Design decisions documented with rationale
- Format choice made

**Design Decisions (Resolved)**:
- Time format: RFC3339 (ISO 8601) - e.g., "2025-01-06T14:30:00Z"
- Timezone: UTC (consistent with `ingested_at` field)
- Field naming: `local_time` (snake_case convention)
- Edge cases: Empty string for uts <= 0

---

### Milestone 2: Implementation
**Objective**: Implement the time conversion and add field to JSON output

#### Tasks
1. Add time conversion function
   - Create utility function to convert Unix timestamp to local time
   - Handle edge cases:
     - Zero/negative timestamps
     - Future timestamps
     - Invalid values
   
2. Update JSON structure
   - Add new field to the response struct
   - Use appropriate JSON struct tags
   - Ensure uts field remains unchanged (backward compatibility)
   
3. Add unit tests
   - Test valid timestamp conversion
   - Test edge cases (0, negative, very large values)
   - Test JSON marshaling includes both fields
   - Test timezone handling

**Deliverables**:
- Updated struct with new field
- Time conversion implementation
- Unit tests with >80% coverage

**Acceptance Criteria**:
- Both uts and new local_time field present in JSON
- Time conversion accurate
- All tests passing
- No breaking changes to existing API

---

### Milestone 3: Documentation and Validation
**Objective**: Document the new field and validate end-to-end

#### Tasks
1. Update documentation
   - Update API response documentation
   - Add example JSON output showing both fields
   - Document timezone behavior
   - Update configuration docs if timezone is configurable
   
2. Integration testing
   - Test with actual Last.fm API responses
   - Verify output format in different scenarios
   - Test with scrobbles at different times
   
3. Update example files
   - Update .env.example if new config added
   - Update any API response examples in docs

**Deliverables**:
- Updated documentation
- Integration test results
- Example outputs

**Acceptance Criteria**:
- Documentation clearly explains new field
- Examples show correct format
- End-to-end validation successful

---

## Technical Details

### Proposed JSON Structure
```json
{
  "track": "Song Name",
  "artist": "Artist Name",
  "uts": 1704556800,
  "local_time": "2025-01-06T14:30:00Z"
}
```

### Implementation Approach
```go
type Scrobble struct {
    Track     string `json:"track"`
    Artist    string `json:"artist"`
    UTS       int64  `json:"uts"`
    LocalTime string `json:"local_time"`
}

func convertUnixToLocal(uts int64) string {
    if uts <= 0 {
        return ""
    }
    t := time.Unix(uts, 0)
    // Format as RFC3339 or custom format
    return t.Format(time.RFC3339)
}
```

## Success Metrics
- JSON output includes both uts and local_time fields
- Time conversion is accurate
- No breaking changes to existing consumers
- Documentation updated

## Dependencies
- Go time package (stdlib)
- Existing Last.fm API integration

## Out of Scope
- Changing existing uts field behavior
- Complex timezone configuration UI
- Historical data migration

## Notes
- Keep implementation simple - this is a display enhancement
- Ensure no performance impact on API calls
- Time conversion uses Go stdlib (time.Unix, time.Format)
- Edge case: Return empty string for uts <= 0

