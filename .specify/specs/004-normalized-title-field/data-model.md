# Data Model: Normalized Title Field

## Overview
This document defines the data structures, interfaces, and relationships for the normalized title feature.

---

## 1. Core Data Structures

### 1.1 Scrobble Model Extension

**File**: `internal/models/scrobble.go`

```go
package models

import (
    "encoding/json"
    "time"
)

// Scrobble represents a single track listen event from Last.fm
type Scrobble struct {
    Username        string          `json:"username"`
    Artist          string          `json:"artist"`
    Track           string          `json:"track"`           // Original title from Last.fm
    NormalizedTitle string          `json:"normalized_title"` // NEW: Cleaned canonical title
    Album           string          `json:"album"`
    UTS             int64           `json:"uts"`
    LocalTime       string          `json:"local_time"`
    MBID            *string         `json:"mbid,omitempty"`
    Source          string          `json:"source"`
    IngestedAt      string          `json:"ingested_at"`
    Raw             json.RawMessage `json:"raw"`
}
```

**Changes**:
- Add `NormalizedTitle string` field
- Populated automatically in `NewScrobble()` constructor
- Original `Track` field remains unchanged
- Backward compatible: new field is additive

**JSON Output Example**:
```json
{
  "username": "johndoe",
  "artist": "Queen",
  "track": "Bohemian Rhapsody - Remastered 2011",
  "normalized_title": "Bohemian Rhapsody",
  "album": "A Night at the Opera",
  "uts": 1704556800,
  "local_time": "2025-01-06T14:30:00Z",
  "source": "lastfm",
  "ingested_at": "2025-01-07T10:00:00Z",
  "raw": {}
}
```

---

## 2. Normalization Package

### 2.1 Package Structure

```
internal/
  normalize/
    normalize.go       # Public API and core logic
    patterns.go        # Pattern definitions and compilation
    config.go          # Configuration loading and validation
    normalize_test.go  # Unit tests
```

### 2.2 Public API

**File**: `internal/normalize/normalize.go`

```go
package normalize

// NormalizeTitle removes common annotations from track titles.
// Returns the original title if normalization would remove everything.
// Thread-safe and performant for concurrent use.
//
// Examples:
//   "Song - Remastered 2011" → "Song"
//   "Song - Live at Venue" → "Song"
//   "Live" → "Live" (preserved)
func NormalizeTitle(title string) string

// SetEnabled enables or disables normalization globally.
// When disabled, NormalizeTitle returns input unchanged.
// Useful for feature flags and A/B testing.
func SetEnabled(enabled bool)

// IsEnabled returns whether normalization is currently enabled.
func IsEnabled() bool

// ValidatePattern checks if a regex pattern is valid.
// Returns error if pattern cannot be compiled.
func ValidatePattern(pattern string) error
```

### 2.3 Configuration Structure

**File**: `internal/normalize/config.go`

```go
package normalize

// Config holds normalization configuration options.
type Config struct {
    // Enabled controls whether normalization is active
    Enabled bool `yaml:"enabled" env:"NORMALIZE_ENABLED" default:"true"`
    
    // MinLength is the minimum length of normalized output.
    // If normalization produces a string shorter than this, 
    // the original title is returned instead.
    MinLength int `yaml:"min_length" env:"NORMALIZE_MIN_LENGTH" default:"2"`
    
    // CustomPatterns allows users to define additional removal patterns
    CustomPatterns []PatternConfig `yaml:"custom_patterns"`
    
    // DisabledPatterns lists built-in patterns to skip
    DisabledPatterns []string `yaml:"disabled_patterns"`
    
    // International enables extended patterns for non-English annotations
    International bool `yaml:"international" env:"NORMALIZE_INTERNATIONAL" default:"true"`
}

// PatternConfig defines a custom normalization pattern.
type PatternConfig struct {
    // Name is a unique identifier for the pattern
    Name string `yaml:"name" validate:"required"`
    
    // Pattern is the regex to match (must be valid Go regex)
    Pattern string `yaml:"pattern" validate:"required"`
    
    // Priority determines application order (lower = earlier)
    // Built-in patterns use priorities 10, 20, 30, 40, 50
    Priority int `yaml:"priority" default:"100"`
    
    // Description explains what the pattern matches
    Description string `yaml:"description"`
}

// LoadConfig loads configuration from file and environment.
// Priority: env vars > config file > defaults
func LoadConfig(configPath string) (*Config, error)

// Validate checks that configuration is valid.
func (c *Config) Validate() error
```

**Configuration File Example** (`normalize.yaml`):

```yaml
# normalize.yaml
normalize:
  enabled: true
  min_length: 2
  international: true
  
  # Custom patterns (merged with built-in)
  custom_patterns:
    - name: deluxe_suffix
      pattern: '(?i)\s*[-–—([]?\s*deluxe\s*(edition|version)?.*?[)\]]?$'
      priority: 55
      description: "Removes 'Deluxe Edition' suffixes"
    
    - name: acoustic_version
      pattern: '(?i)\s*[-–—([]?\s*acoustic\s*(version)?.*?[)\]]?$'
      priority: 60
      description: "Removes 'Acoustic Version' annotations"
  
  # Disable specific built-in patterns
  disabled_patterns:
    - remix  # Keep remix annotations in titles
```

### 2.4 Pattern Definitions

**File**: `internal/normalize/patterns.go`

```go
package normalize

import "regexp"

// Pattern represents a compiled normalization pattern.
type Pattern struct {
    Name        string
    Regex       *regexp.Regexp
    Priority    int
    Description string
}

// BuiltInPatterns returns the default set of normalization patterns.
// Patterns are ordered by priority (lower priority = applied first).
func BuiltInPatterns() []Pattern

// Example built-in patterns (not all shown):
var builtInPatterns = []struct {
    name        string
    pattern     string
    priority    int
    description string
}{
    {
        name:     "remaster",
        pattern:  `(?i)\s*[-–—([]?\s*(remaster(ed)?|reissue|remasteris[eé]).*?[)\]]?$`,
        priority: 10,
        description: "Removes remaster and reissue annotations",
    },
    {
        name:     "live",
        pattern:  `(?i)\s*[-–—([]?\s*(live|en vivo|ao vivo|en direct)\s*(from|at|in|@|de)?.*?[)\]]?$`,
        priority: 20,
        description: "Removes live performance annotations",
    },
    {
        name:     "version",
        pattern:  `(?i)\s*[-–—([]?\s*(radio edit|album version|single version|extended|bonus track).*?[)\]]?$`,
        priority: 30,
        description: "Removes version type annotations",
    },
    {
        name:     "date",
        pattern:  `(?i)\s*[-–—([]?\s*\d{4}.*?[)\]]?$`,
        priority: 40,
        description: "Removes year/date annotations",
    },
    {
        name:     "remix",
        pattern:  `(?i)\s*[-–—([]?\s*(remix|mix|remixed by).*?[)\]]?$`,
        priority: 50,
        description: "Removes remix annotations",
    },
}

// CompilePatterns compiles a list of pattern configs.
// Returns error if any pattern fails to compile.
func CompilePatterns(configs []PatternConfig) ([]Pattern, error)

// MergePatterns combines built-in and custom patterns, sorting by priority.
func MergePatterns(builtIn, custom []Pattern, disabled []string) []Pattern
```

---

## 3. Configuration Integration

### 3.1 Main Config Extension

**File**: `internal/config/types.go`

```go
package config

import (
    "fmt"
    "time"
    
    "github.com/yourusername/lastfm-sync/internal/normalize"
)

// Config holds all CLI flags and configuration for lastfm-sync
type Config struct {
    // ... existing fields ...
    
    // Normalization configuration
    Normalize normalize.Config `mapstructure:"normalize"`
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
    // ... existing validation ...
    
    // Validate normalization config
    if err := c.Normalize.Validate(); err != nil {
        return fmt.Errorf("normalize config invalid: %w", err)
    }
    
    return nil
}
```

### 3.2 Environment Variable Mapping

```bash
# .env or environment
NORMALIZE_ENABLED=true
NORMALIZE_MIN_LENGTH=2
NORMALIZE_INTERNATIONAL=true
```

---

## 4. Data Flow

### 4.1 Ingestion Flow

```
Last.fm API Response
        ↓
Parse JSON (internal/lastfm/client.go)
        ↓
Extract track title
        ↓
NewScrobble() constructor
        ↓
normalize.NormalizeTitle(track) ← NEW STEP
        ↓
Scrobble{Track: original, NormalizedTitle: cleaned}
        ↓
Marshal to JSON
        ↓
Write to output (local/Azure)
```

### 4.2 Constructor Update

**File**: `internal/models/scrobble.go`

```go
import "github.com/yourusername/lastfm-sync/internal/normalize"

// NewScrobble creates a Scrobble from API fields with current ingested_at timestamp.
func NewScrobble(username, artist, track, album string, uts int64, mbid *string, raw json.RawMessage) *Scrobble {
    return &Scrobble{
        Username:        username,
        Artist:          artist,
        Track:           track,
        NormalizedTitle: normalize.NormalizeTitle(track), // NEW: Normalize title
        Album:           album,
        UTS:             uts,
        LocalTime:       formatTimestamp(uts),
        MBID:            mbid,
        Source:          "lastfm",
        IngestedAt:      time.Now().UTC().Format(time.RFC3339),
        Raw:             raw,
    }
}
```

---

## 5. Database Schema (Future Consideration)

**Note**: Current implementation writes NDJSON to files/blob storage. 
If migrating to a database in the future:

```sql
CREATE TABLE scrobbles (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    artist TEXT NOT NULL,
    track TEXT NOT NULL,
    normalized_title TEXT NOT NULL,  -- NEW: Indexed for fast lookup
    album TEXT,
    uts BIGINT NOT NULL,
    local_time TIMESTAMP,
    mbid VARCHAR(36),
    source VARCHAR(50),
    ingested_at TIMESTAMP NOT NULL,
    raw JSONB,
    
    -- Indexes for common queries
    INDEX idx_username_uts (username, uts),
    INDEX idx_normalized_title (normalized_title),  -- NEW: For grouping/stats
    INDEX idx_artist_normalized (artist, normalized_title)  -- NEW: For deduplication
);
```

**Benefits of `normalized_title` index**:
- Fast grouping by canonical title
- Efficient "distinct tracks" queries
- Better deduplication detection

---

## 6. Validation Rules

### 6.1 Input Validation

| Field | Validation | Error Handling |
|-------|------------|----------------|
| `title` (input) | Can be empty string | Return empty string |
| `title` (input) | Can contain any UTF-8 | Handled correctly |
| `title` (input) | No max length enforced | Process any length |

### 6.2 Output Validation

| Field | Validation | Guarantee |
|-------|------------|-----------|
| `NormalizedTitle` | Never longer than original | Always `len(normalized) <= len(original)` |
| `NormalizedTitle` | Never empty if input non-empty | Falls back to original if result too short |
| `NormalizedTitle` | Valid UTF-8 | Guaranteed by Go string type |

### 6.3 Pattern Validation

```go
// ValidatePattern ensures a pattern is valid before use
func ValidatePattern(pattern string) error {
    _, err := regexp.Compile(pattern)
    if err != nil {
        return fmt.Errorf("invalid regex pattern: %w", err)
    }
    return nil
}
```

**Custom pattern validation** at config load:
- Must compile as valid regex
- Cannot be empty string
- Name must be unique

---

## 7. Performance Characteristics

### 7.1 Memory Usage

| Component | Memory per Item | Total for 1M Scrobbles |
|-----------|-----------------|------------------------|
| Original `Track` field | ~50 bytes avg | ~50 MB |
| New `NormalizedTitle` field | ~40 bytes avg | ~40 MB |
| **Additional overhead** | **~40 bytes** | **~40 MB** |

**Verdict**: 8% memory increase for 1M scrobbles (acceptable)

### 7.2 Processing Time

| Operation | Target | Expected |
|-----------|--------|----------|
| Pattern compilation (startup) | <10ms | ~500µs |
| Single title normalization | <1ms | ~500-2000ns |
| Batch (1000 titles) | <1s | ~1-2ms |

### 7.3 Concurrency Model

- **Thread-safe**: `regexp.Regexp` is safe for concurrent use
- **No locking needed**: Pre-compiled patterns are read-only
- **Stateless**: `NormalizeTitle()` has no side effects
- **Scalable**: Parallelizable across goroutines

---

## 8. Error Handling

### 8.1 Normalization Errors

**Philosophy**: Never fail, always return a valid title.

```go
func NormalizeTitle(title string) string {
    // No error return - always returns a string
    // Worst case: returns original title unchanged
}
```

**Error scenarios handled gracefully**:
- Empty input → empty output
- Invalid UTF-8 → processed as-is (Go handles gracefully)
- Pattern panic → recovered, original returned (defensive)

### 8.2 Configuration Errors

```go
func LoadConfig(path string) (*Config, error) {
    // Returns error if:
    // - Config file exists but is invalid YAML
    // - Custom pattern fails to compile
    // - Required fields missing
}
```

**Error recovery**:
- Invalid config → log warning, use defaults
- Invalid custom pattern → skip pattern, log error
- Feature flag allows disabling if issues occur

---

## 9. Testing Data Model

### 9.1 Test Fixtures

**File**: `internal/normalize/testdata/fixtures.go`

```go
package normalize_test

// TestCase represents a normalization test case
type TestCase struct {
    Input       string
    Expected    string
    Description string
    Tags        []string // e.g., "edge-case", "international", "remaster"
}

// StandardTestSuite returns the comprehensive test suite
func StandardTestSuite() []TestCase {
    return []TestCase{
        {
            Input:       "Bohemian Rhapsody - Remastered 2011",
            Expected:    "Bohemian Rhapsody",
            Description: "Remaster with year",
            Tags:        []string{"remaster", "date"},
        },
        {
            Input:       "Live",
            Expected:    "Live",
            Description: "Short title that matches pattern (preserved)",
            Tags:        []string{"edge-case", "false-positive"},
        },
        // ... more test cases
    }
}
```

### 9.2 Benchmark Data

```go
// BenchmarkData provides realistic title samples for performance testing
func BenchmarkData() []string {
    return []string{
        "Normal Title Without Annotations",                    // No match (fast path)
        "Song - Remastered",                                  // Single pattern
        "Song - Live at Venue (2011 Remaster)",              // Multiple patterns
        strings.Repeat("Very Long Title ", 20) + " - Live",  // Stress test
        // ... more samples
    }
}
```

---

## 10. Migration and Rollout

### 10.1 Backward Compatibility

**Guarantee**: Existing consumers are not broken.

- ✅ New field is optional (consumer can ignore)
- ✅ Existing fields unchanged
- ✅ JSON schema remains valid
- ✅ Old versions of LastFM Reader can still parse output (unknown fields ignored)

### 10.2 Feature Flag Rollout

**Phase 1**: Internal testing
```bash
NORMALIZE_ENABLED=true  # Enable for internal testing
```

**Phase 2**: Gradual rollout
```bash
NORMALIZE_ENABLED=true  # Enable for all users
```

**Phase 3**: Always-on (default)
```go
// No config needed, enabled by default
```

**Rollback plan**:
```bash
NORMALIZE_ENABLED=false  # Instant rollback if issues found
```

### 10.3 Data Migration

**Note**: No migration needed! This is an additive feature.

- Existing NDJSON files: still valid, just missing `normalized_title`
- New files: include `normalized_title`
- Mixed data: handled gracefully by JSON parsers

---

## 11. Monitoring and Observability

### 11.1 Metrics to Track

```go
// Proposed metrics (implementation in internal/logging/)
type NormalizationMetrics struct {
    // Counter: Total titles processed
    TotalProcessed int64
    
    // Counter: Titles that were modified
    TitlesNormalized int64
    
    // Histogram: Processing time per title
    ProcessingTimeNs []int64
    
    // Counter: Patterns matched (by pattern name)
    PatternMatches map[string]int64
    
    // Counter: Fallbacks (returned original due to min length)
    Fallbacks int64
}
```

### 11.2 Logging

```go
// Debug log when normalization occurs
log.Debug("title normalized",
    "original", originalTitle,
    "normalized", normalizedTitle,
    "patterns_matched", matchedPatterns,
)

// Warn log for unexpected cases
log.Warn("normalization fallback",
    "reason", "min_length_violation",
    "original", originalTitle,
)
```

---

## Summary

### New Data Structures

1. **Scrobble.NormalizedTitle** - New field in existing model
2. **normalize.Config** - Configuration structure
3. **normalize.Pattern** - Pattern definition
4. **normalize.PatternConfig** - User-defined patterns

### Key Relationships

```
Config (global)
  └── normalize.Config
       └── []PatternConfig (custom patterns)

Scrobble
  ├── Track (original)
  └── NormalizedTitle (derived from Track via normalize.NormalizeTitle())

Pattern
  ├── Built-in patterns (hardcoded)
  └── Custom patterns (from config)
```

### File Changes

- ✏️ `internal/models/scrobble.go` - Add field, update constructor
- ➕ `internal/normalize/normalize.go` - New package
- ➕ `internal/normalize/patterns.go` - Pattern definitions
- ➕ `internal/normalize/config.go` - Configuration
- ➕ `internal/normalize/normalize_test.go` - Tests
- ✏️ `internal/config/types.go` - Add normalize config section

### Backward Compatibility

✅ **Fully backward compatible**
- Additive change only
- No breaking changes
- Feature flag for safety
