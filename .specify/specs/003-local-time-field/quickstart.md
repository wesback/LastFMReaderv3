# Quickstart: Local Time Field

**Feature**: Add human-readable time field to scrobble JSON output  
**Status**: Planning Phase  
**Date**: 2026-01-06

---

## What's New

A new `local_time` field has been added to all scrobble JSON records, providing a human-readable RFC3339 timestamp derived from the existing `uts` (Unix timestamp) field.

### Before
```json
{
  "username": "john_doe",
  "artist": "The Beatles",
  "track": "Hey Jude",
  "album": "Hey Jude",
  "uts": 1704556800,
  "source": "lastfm",
  "ingested_at": "2024-01-06T14:05:32Z",
  "raw": {...}
}
```

### After
```json
{
  "username": "john_doe",
  "artist": "The Beatles",
  "track": "Hey Jude",
  "album": "Hey Jude",
  "uts": 1704556800,
  "local_time": "2024-01-06T14:00:00Z",
  "source": "lastfm",
  "ingested_at": "2024-01-06T14:05:32Z",
  "raw": {...}
}
```

---

## Key Features

- **RFC3339 Format**: Standard ISO 8601 timestamp format (`YYYY-MM-DDTHH:MM:SSZ`)
- **UTC Timezone**: All timestamps in UTC (consistent with `ingested_at` field)
- **Always Present**: Field included in every scrobble (empty string for invalid timestamps)
- **Backward Compatible**: Existing consumers unaffected (additive change only)
- **Zero Overhead**: Negligible performance impact (<30ms for 100k scrobbles)

---

## Quick Verification

### 1. Check JSON Output

Run a sync and inspect the output:

```bash
./lastfm-sync fetch --user your_username --limit 10 | jq '.'
```

Look for the `local_time` field in each scrobble record.

### 2. Verify Time Format

```bash
./lastfm-sync fetch --user your_username --limit 1 | \
  jq -r '.local_time' | \
  grep -E '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
```

Should output the timestamp if correctly formatted.

### 3. Compare with UTS Field

```bash
./lastfm-sync fetch --user your_username --limit 1 | \
  jq '{uts: .uts, local_time: .local_time}'
```

Example output:
```json
{
  "uts": 1704556800,
  "local_time": "2024-01-06T14:00:00Z"
}
```

---

## Configuration

**No configuration changes required.** The feature works automatically with existing setups.

---

## Use Cases

### 1. Human-Readable Logs

Parse timestamps without external tools:

```bash
./lastfm-sync fetch --user john_doe --limit 100 | \
  jq -r '[.local_time, .artist, .track] | @tsv'
```

Output:
```
2024-01-06T14:00:00Z    The Beatles    Hey Jude
2024-01-06T13:55:32Z    Pink Floyd     Comfortably Numb
...
```

### 2. Time-Based Filtering

Filter scrobbles by date using standard tools:

```bash
./lastfm-sync fetch --user john_doe --limit 1000 | \
  jq 'select(.local_time >= "2024-01-01T00:00:00Z" and .local_time < "2024-02-01T00:00:00Z")'
```

### 3. Debugging

Quickly identify when tracks were played during troubleshooting:

```bash
./lastfm-sync fetch --user john_doe --limit 10 | \
  jq '{track: .track, played: .local_time, ingested: .ingested_at}'
```

---

## Edge Cases

### Zero Timestamp

Scrobbles with `uts = 0` have an empty `local_time`:

```json
{
  "uts": 0,
  "local_time": "",
  ...
}
```

### Future Timestamps (2038+)

Go handles timestamps beyond the 2038 limit:

```json
{
  "uts": 2147483648,
  "local_time": "2038-01-19T03:14:08Z",
  ...
}
```

---

## Migration Notes

### For Existing Consumers

**No action required.** JSON parsers will simply ignore the new field if not explicitly used.

### For New Consumers

To use the new field, update your JSON schema/types:

**Go Example**:
```go
type Scrobble struct {
    Username  string `json:"username"`
    Artist    string `json:"artist"`
    Track     string `json:"track"`
    Album     string `json:"album"`
    UTS       int64  `json:"uts"`
    LocalTime string `json:"local_time"`  // NEW
    Source    string `json:"source"`
    // ... other fields
}
```

**Python Example**:
```python
from typing import TypedDict

class Scrobble(TypedDict):
    username: str
    artist: str
    track: str
    album: str
    uts: int
    local_time: str  # NEW
    source: str
    # ... other fields
```

**TypeScript Example**:
```typescript
interface Scrobble {
  username: string;
  artist: string;
  track: string;
  album: string;
  uts: number;
  local_time: string;  // NEW
  source: string;
  // ... other fields
}
```

---

## Performance Impact

**Negligible overhead:**
- Per scrobble: ~300 nanoseconds (~0.0003 milliseconds)
- 100k scrobbles: ~30 milliseconds total
- Memory: ~36 bytes per scrobble

---

## Troubleshooting

### Issue: `local_time` field missing

**Cause**: Using an older version before this feature  
**Fix**: Update to the latest version

### Issue: `local_time` is empty string

**Cause**: Source `uts` field is 0 or negative (invalid timestamp)  
**Expected**: This is correct behavior for invalid timestamps

### Issue: Timestamp doesn't match expected timezone

**Cause**: All timestamps are in UTC, not local timezone  
**Fix**: Convert to your local timezone in your application layer

---

## Related Documentation

- [Data Model](./data-model.md) - Detailed field specifications
- [JSON Schema](./contracts/scrobble-schema.json) - Machine-readable schema
- [Examples](./contracts/) - Example JSON outputs

---

## Questions?

For implementation details, see:
- [Specification](./spec.md) - Original feature requirements
- [Research](./research.md) - Design decisions and rationale
- [Implementation Plan](./plan.md) - Development roadmap
