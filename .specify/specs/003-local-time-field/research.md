# Research: Add Local Time Field to Last.fm API Response

**Date**: 2026-01-06  
**Feature**: 003-local-time-field  
**Phase**: 0 - Research & Decision Making

---

## Overview

This research document resolves all design decisions for adding a human-readable time field to the Scrobble JSON output. All unknowns from the Technical Context have been investigated and decisions documented with rationale.

---

## Decision 1: Time Format

### Decision
**Use RFC3339 (ISO 8601) format**: `2024-01-06T14:00:00Z`

### Research Findings

**Go Time Package Options**:
- `time.RFC3339`: `"2006-01-02T15:04:05Z07:00"` - ISO 8601 compliant
- `time.RFC3339Nano`: Includes nanoseconds (excessive for our use case)
- `time.RFC1123`: `"Mon, 02 Jan 2006 15:04:05 MST"` - Less machine-readable
- Custom format: `"2006-01-02 15:04:05"` - Requires documentation

**Industry Best Practices**:
- JSON APIs: RFC3339 is de facto standard (used by GitHub, Google Cloud, AWS)
- ISO 8601: International standard for date/time representation
- Machine-parseable: All major languages have RFC3339 parsers
- Human-readable: Clear date and time separation with timezone indicator

### Rationale
1. **Consistency**: Existing `ingested_at` field uses `time.RFC3339`
2. **Interoperability**: RFC3339 is universally supported
3. **Standards Compliance**: ISO 8601 is the international standard
4. **No Documentation Needed**: Developers recognize the format immediately

### Code Example
```go
t := time.Unix(uts, 0).UTC()
localTime := t.Format(time.RFC3339)
// Output: "2024-01-06T14:00:00Z"
```

### Alternatives Considered
- **Custom format** (`"2006-01-02 15:04:05"`): Rejected - requires parsing documentation
- **RFC1123**: Rejected - verbose, less machine-readable
- **Unix milliseconds**: Rejected - defeats purpose of human-readable field

---

## Decision 2: Timezone Strategy

### Decision
**Use UTC timezone for all timestamps**

### Research Findings

**Current Implementation**:
- `ingested_at` field uses UTC: `time.Now().UTC().Format(time.RFC3339)`
- Last.fm API provides Unix timestamps (timezone-agnostic)
- No user timezone configuration exists in the codebase

**Timezone Options**:
1. **UTC**: Universal reference, no ambiguity
2. **Server Local Time**: Varies by deployment (Docker, Azure ACI)
3. **User's Local Time**: Requires new configuration mechanism

**Distributed Systems Best Practice**:
- Store and transmit in UTC
- Convert to local time at display layer (client-side)
- Avoids daylight saving time complications
- Prevents timezone-related bugs in data processing

### Rationale
1. **Consistency**: Aligns with existing `ingested_at` field (UTC)
2. **Simplicity**: No new configuration required
3. **Correctness**: Server timezone is deployment-dependent (Docker uses UTC, Azure ACI may differ)
4. **Best Practice**: UTC is standard for APIs and data storage
5. **Client Flexibility**: Consumers can convert to their timezone if needed

### Code Example
```go
t := time.Unix(uts, 0).UTC()  // Force UTC
localTime := t.Format(time.RFC3339)
```

### Alternatives Considered
- **Server Local Time**: Rejected - deployment-dependent, inconsistent
- **Configurable Timezone**: Rejected - adds complexity, minimal value for CLI tool
- **User Timezone Config**: Rejected - out of scope, belongs in display layer

---

## Decision 3: Field Naming Convention

### Decision
**Field name**: `local_time` (snake_case)  
**JSON tag**: `"local_time"`

### Research Findings

**Current Codebase Convention**:
```go
// Existing Scrobble struct uses snake_case
type Scrobble struct {
    Username   string `json:"username"`
    Artist     string `json:"artist"`
    Track      string `json:"track"`
    Album      string `json:"album"`
    UTS        int64  `json:"uts"`
    MBID       *string `json:"mbid,omitempty"`
    Source     string `json:"source"`
    IngestedAt string `json:"ingested_at"`  // snake_case
    Raw        json.RawMessage `json:"raw"`
}
```

**Naming Options**:
1. `local_time` - Matches `ingested_at` convention
2. `localTime` - camelCase (common in JavaScript APIs)
3. `timestamp_local` - Alternative ordering
4. `time_formatted` - Describes transformation

### Rationale
1. **Consistency**: 100% of existing fields use snake_case
2. **Convention**: Go struct field names are PascalCase, JSON tags are snake_case
3. **Clarity**: "local_time" clearly indicates a time representation
4. **Readability**: Separates "local" and "time" for scanning

### Code Example
```go
type Scrobble struct {
    // ... existing fields
    UTS       int64  `json:"uts"`
    LocalTime string `json:"local_time"`  // New field
    // ... remaining fields
}
```

### Alternatives Considered
- **`localTime`**: Rejected - breaks snake_case convention
- **`timestamp_local`**: Rejected - less intuitive ordering
- **`time_formatted`**: Rejected - doesn't clarify format or timezone

---

## Decision 4: Edge Case Handling

### Decision
**Invalid/zero timestamps**: Return empty string `""`

### Research Findings

**Edge Cases**:
1. **Zero timestamp** (`uts = 0`): Represents Unix epoch (1970-01-01), but often indicates "not set"
2. **Negative timestamp** (`uts < 0`): Pre-epoch dates (unlikely from Last.fm)
3. **Very large timestamp** (`uts > 2147483647`): Year 2038 problem (int32 max)
4. **Future timestamp**: Valid but may indicate data error

**Options for Invalid Values**:
1. **Empty string** (`""`): Indicates missing/invalid data
2. **Omit field** (`omitempty`): Changes JSON structure conditionally
3. **Null value** (`null`): Requires pointer type change
4. **Error string** (`"invalid"`): Not machine-parseable

### Rationale
1. **JSON Structure Consistency**: Field always present (required in schema)
2. **Type Safety**: Remains a string type (no nullable pointer needed)
3. **Client Handling**: Empty string is easily checked (`if localTime != ""`)
4. **Backward Compatibility**: Doesn't require schema changes
5. **Simplicity**: Single conditional in conversion function

### Code Example
```go
func formatTimestamp(uts int64) string {
    if uts <= 0 {
        return ""  // Invalid or not set
    }
    t := time.Unix(uts, 0).UTC()
    return t.Format(time.RFC3339)
}
```

### Alternatives Considered
- **`omitempty`**: Rejected - inconsistent JSON structure
- **`null`**: Rejected - requires pointer type, complicates marshaling
- **`"invalid"` or `"N/A"`**: Rejected - not standard, breaks parsing
- **Epoch string**: Rejected - misleading (zero doesn't mean 1970)

### Future Timestamp Handling
**Decision**: Allow future timestamps (format normally)
- **Rationale**: Last.fm may have clock skew, legitimate use cases exist
- **Validation**: If validation needed, implement in business logic layer, not formatting

---

## Decision 5: Performance Impact

### Decision
**No optimization needed** - stdlib operations are sufficiently fast

### Research Findings

**Performance Characteristics**:
```go
// Benchmark: time.Unix() and Format()
func BenchmarkTimeConversion(b *testing.B) {
    uts := int64(1704556800)
    for i := 0; i < b.N; i++ {
        t := time.Unix(uts, 0).UTC()
        _ = t.Format(time.RFC3339)
    }
}
// Typical result: ~200-300 ns/op on modern hardware
```

**Scale Analysis**:
- Typical sync: 1,000-10,000 scrobbles
- Worst case: 100,000 scrobbles (heavy user, full history)
- Overhead per scrobble: ~300 nanoseconds = 0.0003 milliseconds
- Total overhead for 100k scrobbles: ~30 milliseconds

**Memory Impact**:
- RFC3339 string length: ~20 characters
- Memory per scrobble: 20 bytes (string) + 16 bytes (string header) = 36 bytes
- 100k scrobbles: 3.6 MB additional memory (negligible)

### Rationale
1. **Negligible Overhead**: <0.001% of typical sync time
2. **stdlib Efficiency**: `time` package is highly optimized
3. **No Caching Needed**: Formatting is faster than cache lookup overhead
4. **Memory Acceptable**: 3.6 MB for 100k records is trivial for modern systems

### Alternatives Considered
- **Lazy computation** (compute on demand): Rejected - adds complexity, no benefit
- **Caching**: Rejected - overhead exceeds benefit
- **Pre-formatting in Last.fm client**: Rejected - violates separation of concerns

---

## Decision 6: Implementation Location

### Decision
**Add to Scrobble struct** with a computed field during creation

### Research Findings

**Options**:
1. **Struct field** (selected): Add `LocalTime string` to Scrobble
2. **Custom MarshalJSON**: Compute during JSON marshaling
3. **Separate function**: Return transformed object
4. **At write time**: Compute in writer layer

**Current Pattern**:
```go
// Existing: IngestedAt is set during struct creation
func NewScrobble(...) *Scrobble {
    return &Scrobble{
        // ...
        IngestedAt: time.Now().UTC().Format(time.RFC3339),
        // ...
    }
}
```

### Rationale
1. **Consistency**: Mirrors `IngestedAt` pattern (computed at creation)
2. **Simplicity**: Single location for time conversion
3. **Testability**: Easy to unit test in `NewScrobble()`
4. **Reusability**: Field available for all consumers (JSON, logging, etc.)
5. **Performance**: Computed once at creation, not repeatedly during marshaling

### Code Example
```go
func NewScrobble(username, artist, track, album string, uts int64, mbid *string, raw json.RawMessage) *Scrobble {
    return &Scrobble{
        Username:   username,
        Artist:     artist,
        Track:      track,
        Album:      album,
        UTS:        uts,
        LocalTime:  formatTimestamp(uts),  // NEW
        MBID:       mbid,
        Source:     "lastfm",
        IngestedAt: time.Now().UTC().Format(time.RFC3339),
        Raw:        raw,
    }
}

func formatTimestamp(uts int64) string {
    if uts <= 0 {
        return ""
    }
    return time.Unix(uts, 0).UTC().Format(time.RFC3339)
}
```

### Alternatives Considered
- **Custom MarshalJSON**: Rejected - complicates marshaling, computed multiple times
- **Writer layer**: Rejected - violates single responsibility, hard to test
- **Separate transformation function**: Rejected - requires extra step for consumers

---

## Summary of Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Time Format** | RFC3339 | Industry standard, consistent with `ingested_at` |
| **Timezone** | UTC | Consistent, deployment-agnostic, best practice |
| **Field Name** | `local_time` | Matches existing snake_case convention |
| **Edge Cases** | Empty string for `uts <= 0` | Maintains JSON structure, type-safe |
| **Performance** | No optimization needed | <30ms overhead for 100k scrobbles |
| **Implementation** | Computed field in `NewScrobble()` | Consistent with `IngestedAt` pattern |

---

## Open Questions Resolved

All questions from [spec.md](./spec.md) have been resolved:

1. ✅ **Time format preference**: RFC3339 (ISO 8601)
2. ✅ **Timezone handling**: UTC (matches `ingested_at`)
3. ✅ **Field naming**: `local_time` (snake_case convention)

---

## Implementation Readiness

All technical unknowns resolved. Ready to proceed to Phase 1 (Design & Contracts).

**Phase 0 Complete**: ✅
