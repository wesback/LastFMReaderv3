# Data Model: Add Local Time Field

**Date**: 2026-01-06  
**Feature**: 003-local-time-field  
**Phase**: 1 - Data Model Design

---

## Overview

This document describes the data model changes required to add a human-readable time field to the Scrobble entity. The change is additive and maintains backward compatibility.

---

## Current Data Model

### Scrobble Entity (Before)

**Location**: `internal/models/scrobble.go`

```go
// Scrobble represents a single track listen event from Last.fm
// matching the NDJSON contract in spec.md
type Scrobble struct {
	Username   string          `json:"username"`
	Artist     string          `json:"artist"`
	Track      string          `json:"track"`
	Album      string          `json:"album"`
	UTS        int64           `json:"uts"`
	MBID       *string         `json:"mbid,omitempty"`
	Source     string          `json:"source"`
	IngestedAt string          `json:"ingested_at"`
	Raw        json.RawMessage `json:"raw"`
}
```

**Field Descriptions**:
- `Username`: Last.fm username who scrobbled the track
- `Artist`: Artist name
- `Track`: Track/song name
- `Album`: Album name
- `UTS`: Unix timestamp (seconds since epoch) when track was played
- `MBID`: MusicBrainz ID (optional, nullable)
- `Source`: Data source, always "lastfm"
- `IngestedAt`: UTC timestamp when record was ingested (RFC3339 format)
- `Raw`: Original Last.fm API response (for debugging/audit)

---

## Updated Data Model

### Scrobble Entity (After)

**Location**: `internal/models/scrobble.go`

```go
// Scrobble represents a single track listen event from Last.fm
// matching the NDJSON contract in spec.md
type Scrobble struct {
	Username   string          `json:"username"`
	Artist     string          `json:"artist"`
	Track      string          `json:"track"`
	Album      string          `json:"album"`
	UTS        int64           `json:"uts"`
	LocalTime  string          `json:"local_time"`     // NEW FIELD
	MBID       *string         `json:"mbid,omitempty"`
	Source     string          `json:"source"`
	IngestedAt string          `json:"ingested_at"`
	Raw        json.RawMessage `json:"raw"`
}
```

**New Field**:
- `LocalTime`: Human-readable UTC timestamp in RFC3339 format, derived from `UTS` field

---

## Field Specification

### LocalTime Field

**Type**: `string`  
**JSON Tag**: `"local_time"`  
**Format**: RFC3339 (ISO 8601)  
**Timezone**: UTC  
**Required**: Yes (always present in JSON output)  
**Source**: Computed from `UTS` field  

#### Value Rules

| Condition | Value | Example |
|-----------|-------|---------|
| `UTS > 0` | RFC3339 formatted timestamp | `"2024-01-06T14:00:00Z"` |
| `UTS <= 0` | Empty string | `""` |
| `UTS` is very large (> 2^31) | Valid RFC3339 (Go handles up to 2^63) | `"2038-01-19T03:14:07Z"` |

#### Computation Logic

```go
func formatTimestamp(uts int64) string {
    if uts <= 0 {
        return ""
    }
    t := time.Unix(uts, 0).UTC()
    return t.Format(time.RFC3339)
}
```

---

## Relationships

### Derived Field Relationship

```
UTS (int64) ──────────> LocalTime (string)
   Unix timestamp           RFC3339 format
   1704556800              "2024-01-06T14:00:00Z"
```

**Dependency**: `LocalTime` is always computed from `UTS`  
**Consistency**: Both fields represent the same moment in time  
**Redundancy**: Intentional - improves human readability without breaking existing consumers  

### Comparison with IngestedAt

| Field | Represents | Format | Timezone | Source |
|-------|-----------|--------|----------|--------|
| `UTS` | When track was played | Unix seconds | N/A (numeric) | Last.fm API |
| `LocalTime` | When track was played | RFC3339 | UTC | Computed from UTS |
| `IngestedAt` | When record was created | RFC3339 | UTC | Current system time |

---

## Validation Rules

### Field-Level Validation

1. **LocalTime Format**:
   - MUST be empty string OR valid RFC3339 format
   - Pattern: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$` (simplified)
   - Full regex: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`

2. **UTS-LocalTime Consistency**:
   - If `UTS <= 0`, then `LocalTime` MUST be `""`
   - If `UTS > 0`, then `LocalTime` MUST be valid RFC3339

3. **JSON Marshaling**:
   - Field MUST always be present (not omitted)
   - Empty string serializes as `"local_time": ""`

### Entity-Level Validation

No change to existing validation. All existing rules remain:
- `Username` must be non-empty
- `Artist` must be non-empty
- `Track` must be non-empty
- `UTS` must be non-negative
- `Source` must be "lastfm"
- `IngestedAt` must be valid RFC3339

---

## State Transitions

N/A - Scrobbles are immutable once created (no state transitions).

---

## Migration Impact

### Backward Compatibility

**No breaking changes**:
- Existing fields unchanged
- JSON structure remains compatible
- Existing consumers will ignore `local_time` field (standard JSON behavior)
- New consumers can opt-in to using `local_time`

### Data Migration

**Not required**:
- Historical data (already written) does not need updating
- New field computed at struct creation time
- Only applies to newly created Scrobble instances

### Consumer Impact

| Consumer Type | Impact | Action Required |
|---------------|--------|-----------------|
| JSON parsers (strict schema) | None - new field ignored | None |
| JSON parsers (loose schema) | New field available | Update to use `local_time` if desired |
| Downstream systems | None - additive change | None |
| Tests | Update expected JSON | Verify tests include new field |

---

## Examples

### Example 1: Valid Timestamp

**Input**:
```go
uts := int64(1704556800)  // 2024-01-06 14:00:00 UTC
```

**Output**:
```go
Scrobble{
    Username:   "john_doe",
    Artist:     "The Beatles",
    Track:      "Hey Jude",
    Album:      "Hey Jude",
    UTS:        1704556800,
    LocalTime:  "2024-01-06T14:00:00Z",  // NEW
    MBID:       nil,
    Source:     "lastfm",
    IngestedAt: "2024-01-06T14:05:00Z",
    Raw:        {...},
}
```

**JSON**:
```json
{
  "username": "john_doe",
  "artist": "The Beatles",
  "track": "Hey Jude",
  "album": "Hey Jude",
  "uts": 1704556800,
  "local_time": "2024-01-06T14:00:00Z",
  "source": "lastfm",
  "ingested_at": "2024-01-06T14:05:00Z",
  "raw": {}
}
```

### Example 2: Zero Timestamp (Edge Case)

**Input**:
```go
uts := int64(0)
```

**Output**:
```go
Scrobble{
    Username:   "jane_doe",
    Artist:     "Unknown",
    Track:      "Test Track",
    Album:      "",
    UTS:        0,
    LocalTime:  "",  // Empty string for invalid timestamp
    MBID:       nil,
    Source:     "lastfm",
    IngestedAt: "2024-01-06T15:00:00Z",
    Raw:        {...},
}
```

**JSON**:
```json
{
  "username": "jane_doe",
  "artist": "Unknown",
  "track": "Test Track",
  "album": "",
  "uts": 0,
  "local_time": "",
  "source": "lastfm",
  "ingested_at": "2024-01-06T15:00:00Z",
  "raw": {}
}
```

### Example 3: Future Timestamp (2038+)

**Input**:
```go
uts := int64(2147483648)  // 2038-01-19 03:14:08 UTC (beyond int32 max)
```

**Output**:
```json
{
  "username": "future_user",
  "artist": "Future Band",
  "track": "Future Song",
  "album": "Future Album",
  "uts": 2147483648,
  "local_time": "2038-01-19T03:14:08Z",
  "source": "lastfm",
  "ingested_at": "2024-01-06T15:00:00Z",
  "raw": {}
}
```

---

## Testing Requirements

### Unit Tests

**Location**: `internal/models/scrobble_test.go`

**Test Cases**:
1. `TestNewScrobble_ValidTimestamp` - Verify correct RFC3339 format
2. `TestNewScrobble_ZeroTimestamp` - Verify empty string
3. `TestNewScrobble_NegativeTimestamp` - Verify empty string
4. `TestNewScrobble_LargeTimestamp` - Verify year 2038+ handling
5. `TestScrobble_MarshalJSON` - Verify JSON includes `local_time`
6. `TestFormatTimestamp` - Unit test for helper function

### Integration Tests

**Scenario**: Full sync with new field validation
- Fetch scrobbles from Last.fm API
- Verify all scrobbles have `local_time` field
- Verify `local_time` matches `uts` conversion
- Verify JSON output is valid

---

## Performance Considerations

### Memory Impact

**Per Scrobble**:
- String field: ~20 characters
- Go string overhead: 16 bytes (header)
- Total: ~36 bytes per scrobble

**Scale**:
- 1,000 scrobbles: 36 KB
- 10,000 scrobbles: 360 KB
- 100,000 scrobbles: 3.6 MB

**Verdict**: Negligible impact

### CPU Impact

**Per Scrobble**:
- `time.Unix()`: ~100 ns
- `Format(RFC3339)`: ~200 ns
- Total: ~300 ns per scrobble

**Scale**:
- 100,000 scrobbles: ~30 milliseconds total

**Verdict**: Negligible impact (<0.1% of typical sync time)

---

## Summary

- **Change Type**: Additive (new field)
- **Breaking Changes**: None
- **Migration Required**: No
- **Performance Impact**: Negligible
- **Testing Coverage Target**: >80% for new code
- **Backward Compatibility**: Full

**Status**: ✅ Ready for contract generation (Phase 1.2)
