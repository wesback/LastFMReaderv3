# Quickstart: Normalized Title Field

## Overview
This guide helps you quickly understand and use the normalized title feature in LastFM Reader v3.

## What It Does
The normalized title feature automatically cleans track titles by removing common annotations:
- ✅ Remaster tags: `"Song - Remastered 2011"` → `"Song"`
- ✅ Live recordings: `"Song - Live at Venue"` → `"Song"`
- ✅ Version info: `"Song (Radio Edit)"` → `"Song"`
- ✅ Date stamps: `"Song - 2015"` → `"Song"`
- ✅ Remix tags: `"Song - Remix"` → `"Song"`
- ✅ Featuring info: `"Song (feat. Artist)"` → `"Song"`

## Quick Example

**Input** (from Last.fm):
```json
{
  "track": "Bohemian Rhapsody - Remastered 2011",
  "artist": "Queen"
}
```

**Output** (with normalized_title):
```json
{
  "track": "Bohemian Rhapsody - Remastered 2011",
  "normalized_title": "Bohemian Rhapsody",
  "artist": "Queen",
  "uts": 1704556800,
  "local_time": "2025-01-06T14:30:00Z"
}
```

## Getting Started

### 1. Default Behavior (Enabled by Default)
No configuration needed! The feature works automatically when you sync:

```bash
go run cmd/lastfm-sync/main.go fetch --user yourname
```

Your output will now include both `track` (original) and `normalized_title` (cleaned) fields.

### 2. Disable Normalization (Optional)
If you want to turn it off:

```bash
export NORMALIZE_ENABLED=false
go run cmd/lastfm-sync/main.go fetch --user yourname
```

Or create a config file `normalize.yaml`:
```yaml
normalize:
  enabled: false
```

### 3. View Normalized Titles in Logs
Enable debug logging to see what's being normalized:

```bash
export LOG_LEVEL=debug
go run cmd/lastfm-sync/main.go fetch --user yourname
```

You'll see logs like:
```
DEBUG title normalized original="Song - Live" normalized="Song" patterns=["live"]
```

## Common Use Cases

### Use Case 1: Track Statistics
Get accurate play counts by grouping on `normalized_title`:

```bash
# Example using jq to count unique normalized titles
cat scrobbles.ndjson | jq -r '.normalized_title' | sort | uniq -c | sort -rn | head -10
```

### Use Case 2: Duplicate Detection
Find different versions of the same song:

```bash
# Find all versions of a specific track
cat scrobbles.ndjson | jq -r 'select(.normalized_title == "Bohemian Rhapsody") | .track'
```

Output:
```
Bohemian Rhapsody
Bohemian Rhapsody - Remastered 2011
Bohemian Rhapsody - Live at Wembley
Bohemian Rhapsody (Radio Edit)
```

### Use Case 3: Clean Exports
Use normalized titles for cleaner reports:

```bash
# Export just artist and clean title
cat scrobbles.ndjson | jq -r '[.artist, .normalized_title] | @csv' > clean_export.csv
```

## Configuration Options

### Environment Variables
```bash
# Enable/disable normalization
NORMALIZE_ENABLED=true

# Minimum length for normalized output (default: 2)
NORMALIZE_MIN_LENGTH=2

# Enable international patterns (default: true)
NORMALIZE_INTERNATIONAL=true
```

### Advanced: Custom Patterns
Create `normalize.yaml` to add custom patterns:

```yaml
normalize:
  enabled: true
  min_length: 2
  international: true
  
  # Add custom patterns
  custom_patterns:
    - name: acoustic_version
      pattern: '(?i)\s*[-–—([]?\s*acoustic.*?[)\]]?$'
      priority: 60
      description: "Removes 'Acoustic' annotations"
    
    - name: demo_version
      pattern: '(?i)\s*[-–—([]?\s*demo.*?[)\]]?$'
      priority: 65
      description: "Removes 'Demo' annotations"
  
  # Disable specific built-in patterns
  disabled_patterns:
    - remix  # Keep remix info in titles
```

## Built-in Patterns

| Pattern | Priority | Example Match | Result |
|---------|----------|---------------|--------|
| Remaster | 10 | `"Song - Remastered 2011"` | `"Song"` |
| Live | 20 | `"Song - Live at Venue"` | `"Song"` |
| Version | 30 | `"Song (Radio Edit)"` | `"Song"` |
| Date | 40 | `"Song - 2015"` | `"Song"` |
| Remix | 50 | `"Song - DJ Remix"` | `"Song"` |
| Featuring | 60 | `"Song (feat. Artist)"` | `"Song"` |

Lower priority numbers are applied first.

## Edge Cases Handled

### Band Named "Live"
```json
{
  "track": "Live",
  "normalized_title": "Live"
}
```
✅ Short titles are preserved to avoid false positives.

### Multiple Annotations
```json
{
  "track": "Song - Live from London (2015 Remaster)",
  "normalized_title": "Song"
}
```
✅ All annotations removed sequentially.

### International Titles
```json
{
  "track": "Canción - En Vivo",
  "normalized_title": "Canción"
}
```
✅ Spanish "En Vivo" (Live) is recognized when `NORMALIZE_INTERNATIONAL=true`.

### Nothing to Normalize
```json
{
  "track": "Song Title",
  "normalized_title": "Song Title"
}
```
✅ Titles without annotations pass through unchanged.

## Troubleshooting

### Issue: Normalization too aggressive
**Problem**: Legitimate parts of titles are being removed.

**Solution**: Adjust minimum length or disable specific patterns:
```yaml
normalize:
  min_length: 5  # Require at least 5 chars after normalization
  disabled_patterns:
    - live  # Stop removing "Live" if causing issues
```

### Issue: Normalization not working
**Problem**: `normalized_title` equals `track` for everything.

**Check**:
1. Is normalization enabled? `echo $NORMALIZE_ENABLED`
2. Do titles actually contain patterns? Check with debug logs:
   ```bash
   LOG_LEVEL=debug go run cmd/lastfm-sync/main.go fetch --user yourname 2>&1 | grep normalized
   ```

### Issue: Custom pattern not matching
**Problem**: Added a custom pattern but it's not working.

**Debug**:
1. Validate regex syntax:
   ```bash
   # Test pattern with Go
   echo 'package main; import "regexp"; import "fmt"; func main() { r := regexp.MustCompile(`YOUR_PATTERN`); fmt.Println(r.MatchString("Test String")) }' > /tmp/test.go
   go run /tmp/test.go
   ```

2. Check pattern priority - lower numbers run first
3. Ensure pattern ends with `$` to match suffix only

## Performance Notes

- **Speed**: ~500-2000ns per title (< 1ms target met ✅)
- **Memory**: ~40 bytes additional per scrobble
- **Impact**: Negligible on sync performance

## Next Steps

- **For statistics**: Query by `normalized_title` instead of `track`
- **For debugging**: Enable DEBUG logs to see normalization in action
- **For customization**: Create `normalize.yaml` with your patterns
- **For analysis**: Export `normalized_title` field for clean datasets

## Related Documentation

- [Configuration Guide](../../docs/configuration.md) - Full configuration options
- [Data Model](data-model.md) - Technical implementation details
- [Research](research.md) - Pattern analysis and decisions
- [Spec](spec.md) - Complete feature specification

## FAQ

**Q: Does this modify my Last.fm data?**  
A: No. Original titles are preserved in the `track` field. `normalized_title` is a computed field.

**Q: Can I use both fields?**  
A: Yes! Use `track` for exact matches, `normalized_title` for grouping/statistics.

**Q: Will this break my existing scripts?**  
A: No. The `track` field is unchanged. `normalized_title` is an additional field.

**Q: Can I normalize artist names too?**  
A: Not yet. This feature only normalizes track titles. Artist normalization is planned for a future release.

**Q: What if I want to keep some annotations?**  
A: Use the `disabled_patterns` config to skip specific pattern types.
