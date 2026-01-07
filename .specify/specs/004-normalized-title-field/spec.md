# Feature: Add Normalized Title Field

## Overview
Add a `normalized_title` field to track data that removes common suffixes and annotations like "Live", "Remastered", date annotations, and other metadata, providing a clean canonical title for better matching and grouping.

## Goals
- Extract clean, canonical track titles from Last.fm data
- Remove common suffixes (Live, Remastered, Remix, etc.)
- Remove date/year annotations in various formats
- Add new field without breaking existing data structure
- Enable better track matching and deduplication

## Clarifications

### Session 2026-01-07
- Q: Should normalization package be located in `pkg/normalize/` (public) or `internal/normalize/` (internal-only)? → A: Use `internal/normalize/` for better encapsulation and consistency with existing codebase structure
- Q: Which YAML library should be used for optional configuration file support? → A: `gopkg.in/yaml.v3` - Most popular, mature, well-maintained YAML library for Go
- Q: Should "feat./ft./featuring" annotations be kept or removed during normalization? → A: Remove "feat./ft./featuring" - Treat them as metadata annotations like "Live" or "Remastered"
- Q: What logging strategy should be used for normalization operations? → A: DEBUG level when changed - Log only when title is modified, at DEBUG level for troubleshooting

## Milestones

### Milestone 1: Pattern Research and Analysis
**Objective**: Identify all common title patterns that need normalization

#### Tasks
1. Analyze existing Last.fm data
   - Sample track titles from Last.fm API responses
   - Identify common patterns and suffixes
   - Document edge cases and ambiguous patterns
   - Create categorized list of patterns to remove

2. Research title normalization patterns
   - Common patterns to identify:
     - Remaster annotations: "Remastered", "Remaster", "- Remastered", "(Remastered)"
     - Date formats: "- 2001", "(2004)", "- 2001 Remaster", "2015 Remastered Version"
     - Live recordings: "Live", "Live from", "Live at", "- Live"
     - Versions: "Radio Edit", "Album Version", "Single Version", "Extended Version"
     - Remixes: "Remix", "- Remix", "(Remix)"
     - Featuring artists: "feat.", "ft.", "featuring"
     - Deluxe/Special: "Deluxe Edition", "Bonus Track"
   - Document pattern variations (parentheses, brackets, dashes)

3. Define normalization rules
   - Create prioritized list of patterns to remove
   - Define rule precedence (order of operations)
   - Document ambiguous cases and how to handle them
   - Consider international characters and unicode

**Deliverables**:
- `docs/title-normalization-patterns.md` with pattern catalog
- List of normalization rules with examples
- Edge case documentation

**Acceptance Criteria**:
- Comprehensive list of patterns documented
- Rules cover 90%+ of common cases
- Edge cases identified and documented

---

### Milestone 2: Design and Algorithm Development
**Objective**: Design the normalization algorithm and approach

#### Tasks
1. Design normalization algorithm
   - Choose approach:
     - Regex-based pattern matching
     - Rule-based string processing
     - Combination approach
   - Define processing order
   - Handle whitespace normalization

2. Design data structure
   - Add `normalized_title` field to structs
   - Decide on JSON field naming
   - Ensure backward compatibility

3. Create test dataset
   - Compile test cases with expected outputs
   - Include edge cases:
     - Titles that are legitimately "Live" or "Remastered" (bands/albums)
     - Multiple annotations in one title
     - Non-English titles
     - Special characters
   - Document expected behavior

**Deliverables**:
- Algorithm design document
- Test dataset with input/expected output pairs
- Updated struct definitions

**Acceptance Criteria**:
- Algorithm handles all documented patterns
- Test dataset covers common and edge cases
- Design reviewed and validated

**Example Test Cases**:
```
Input: "Bohemian Rhapsody - Remastered 2011"
Output: "Bohemian Rhapsody"

Input: "Stairway to Heaven (Remaster)"
Output: "Stairway to Heaven"

Input: "Hotel California - Live from MTV Unplugged"
Output: "Hotel California"

Input: "Sweet Child O' Mine - 2022 Remastered Version"
Output: "Sweet Child O' Mine"

Input: "Wonderwall (Radio Edit)"
Output: "Wonderwall"

Input: "Thriller - Single Version"
Output: "Thriller"

Input: "Live (The Band Name)" 
Output: "Live" (no change - it's the actual title)
```

---

### Milestone 3: Core Implementation
**Objective**: Implement the title normalization function

#### Tasks
1. Implement normalization function
   - Create `normalizeTitle(title string) string` function
   - Implement pattern removal in correct order:
     1. Trim leading/trailing whitespace
     2. Remove parenthetical annotations
     3. Remove bracketed annotations
     4. Remove dash-separated suffixes
     5. Remove date patterns
     6. Final whitespace normalization
   - Handle empty/nil inputs gracefully

2. Create pattern matching utilities
   - Build regex patterns for common suffixes
   - Create helper functions for:
     - Date pattern matching
     - Parenthetical content removal
     - Dash-separated suffix removal
   - Make patterns configurable if needed

3. Integrate into existing data structures
   - Add `NormalizedTitle` field to track struct
   - Update JSON marshaling
   - Populate field when processing Last.fm responses
   - Ensure original title remains unchanged

**Deliverables**:
- `internal/normalize/normalize.go` with normalization logic
- Updated track structs
- Pattern matching utilities

**Acceptance Criteria**:
- Function handles all test cases correctly
- Original title field unchanged
- Performance acceptable (< 1ms per title)
- Code is maintainable and well-commented

**Implementation Sketch**:
```go
package normalize

import (
    "regexp"
    "strings"
)

var (
    // Regex patterns for common suffixes
    remasterPattern = regexp.MustCompile(`(?i)\s*[-–—([]?\s*(remaster(ed)?|reissue).*?[)\]]?$`)
    livePattern     = regexp.MustCompile(`(?i)\s*[-–—([]?\s*live\s*(from|at|in|@)?.*?[)\]]?$`)
    versionPattern  = regexp.MustCompile(`(?i)\s*[-–—([]?\s*(radio edit|album version|single version|extended|bonus track).*?[)\]]?$`)
    datePattern     = regexp.MustCompile(`(?i)\s*[-–—([]?\s*\d{4}.*?[)\]]?$`)
    remixPattern    = regexp.MustCompile(`(?i)\s*[-–—([]?\s*remix.*?[)\]]?$`)
)

func NormalizeTitle(title string) string {
    if title == "" {
        return title
    }
    
    normalized := strings.TrimSpace(title)
    
    // Apply patterns in order
    normalized = remasterPattern.ReplaceAllString(normalized, "")
    normalized = livePattern.ReplaceAllString(normalized, "")
    normalized = versionPattern.ReplaceAllString(normalized, "")
    normalized = datePattern.ReplaceAllString(normalized, "")
    normalized = remixPattern.ReplaceAllString(normalized, "")
    
    // Clean up multiple spaces and trim
    normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
    normalized = strings.TrimSpace(normalized)
    
    // If we removed everything, return original
    if normalized == "" {
        return title
    }
    
    return normalized
}
```

---

### Milestone 4: Testing and Validation
**Objective**: Comprehensive testing of normalization logic

#### Tasks
1. Unit testing
   - Test each pattern type individually
   - Test combinations of patterns
   - Test edge cases:
     - Empty strings
     - Titles with only annotations
     - Unicode characters
     - Very long titles
     - Titles that should NOT be normalized
   - Test performance with large datasets

2. Integration testing
   - Test with real Last.fm API responses
   - Validate against sample of user's actual data
   - Check for false positives (over-normalization)
   - Check for false negatives (under-normalization)

3. Create benchmarks
   - Benchmark normalization performance
   - Ensure no significant performance impact
   - Document performance characteristics

**Deliverables**:
- `internal/normalize/normalize_test.go` with comprehensive tests
- Benchmark results
- Integration test results

**Acceptance Criteria**:
- >90% test coverage
- All test cases pass
- No performance regressions
- Edge cases handled correctly

---

### Milestone 5: Configuration and Extensibility
**Objective**: Make normalization configurable and extensible

#### Tasks
1. Add configuration options
   - Create config structure for normalization rules
   - Allow enabling/disabling specific patterns
   - Support custom patterns via config
   - Add configuration to .env.example

2. Create pattern management
   - Support loading custom patterns from config file
   - Allow pattern priority ordering
   - Document pattern syntax

3. Add feature flag
   - Make normalization optional via config
   - Default to enabled
   - Document in configuration.md

**Deliverables**:
- Configuration options in config structs
- Pattern configuration file format
- Updated .env.example
- Configuration documentation

**Acceptance Criteria**:
- Normalization can be disabled if needed
- Custom patterns can be added without code changes
- Configuration well-documented

**Example Configuration**:
```yaml
# normalization.yaml
normalization:
  enabled: true
  patterns:
    - name: remaster
      regex: '(?i)\s*[-–—([]?\s*(remaster(ed)?|reissue).*?[)\]]?$'
      priority: 1
    - name: live
      regex: '(?i)\s*[-–—([]?\s*live\s*(from|at|in|@)?.*?[)\]]?$'
      priority: 2
    # Custom patterns can be added here
```

---

### Milestone 6: Documentation and Examples
**Objective**: Document the feature comprehensively

#### Tasks
1. Update API documentation
   - Document normalized_title field
   - Provide examples of normalization
   - Explain when normalization is applied
   - Document configuration options

2. Create usage examples
   - Show before/after examples
   - Document common use cases
   - Provide troubleshooting guide

3. Update configuration documentation
   - Add normalization section to docs/configuration.md
   - Document all patterns and their behavior
   - Explain how to add custom patterns

**Deliverables**:
- Updated `docs/configuration.md`
- `docs/title-normalization.md` with detailed explanation
- Example outputs in documentation

**Acceptance Criteria**:
- Feature fully documented
- Examples clear and helpful
- Configuration options explained

---

## Technical Details

### Updated JSON Structure
```json
{
  "track": "Bohemian Rhapsody - Remastered 2011",
  "normalized_title": "Bohemian Rhapsody",
  "artist": "Queen",
  "uts": 1704556800,
  "local_time": "2025-01-06T14:30:00Z"
}
```

### Pattern Priority Order
1. Remove remaster annotations (most common)
2. Remove live annotations
3. Remove version annotations (radio edit, etc.)
4. Remove date patterns
5. Remove remix annotations
6. Remove featuring artist annotations
7. Clean up whitespace

### Edge Cases to Handle
- Titles where the annotation is part of the actual name (e.g., band named "Live")
- Multiple nested parentheses/brackets
- Mixed annotation types in one title
- Non-English characters and accents
- Very short titles (single words)
- Titles that are entirely annotations

## Success Metrics
- 95%+ of titles correctly normalized
- < 1% false positive rate (incorrect normalization)
- No performance impact on API response time
- Zero breaking changes to existing API

## Dependencies
- Go regexp package (stdlib)
- Existing track data structures
- `gopkg.in/yaml.v3` - YAML parsing for optional configuration file

## Out of Scope (Future Enhancements)
- Machine learning-based normalization
- Artist name normalization
- Album name normalization
- Fuzzy matching using normalized titles
- Automatic duplicate detection

## Performance Considerations
- Regex compilation: compile patterns once at startup
- Caching: consider caching normalized titles for frequently accessed tracks
- Batch processing: optimize for bulk normalization if needed

## Observability

### Logging Strategy
- **DEBUG level**: Log when title is modified by normalization
  - Include: original title, normalized title, matched patterns
  - Example: `log.Debug("title normalized", "original", "Song - Live", "normalized", "Song", "patterns", ["live"])`
- **No logging**: When title unchanged (pass-through)
- **WARN level**: When normalization fallback occurs (result too short, returned original)
  - Example: `log.Warn("normalization fallback", "reason", "min_length", "original", "Live")`
- **ERROR level**: Configuration errors (invalid custom patterns)

### Metrics (Future Enhancement)
- Counter: Total normalizations performed
- Counter: Titles modified vs. unchanged
- Histogram: Processing time per title
- Counter: Pattern match frequency (by pattern name)

## Risk Mitigation
- **Over-normalization**: Keep original title always available
- **False positives**: Extensive testing with real data
- **Performance**: Benchmark before deployment
- **Configuration errors**: Validate patterns at startup

## Testing Strategy
```go
// Example test cases
func TestNormalizeTitle(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"Bohemian Rhapsody - Remastered 2011", "Bohemian Rhapsody"},
        {"Stairway to Heaven (Remaster)", "Stairway to Heaven"},
        {"Hotel California - Live from MTV", "Hotel California"},
        {"Sweet Child O' Mine - 2022 Remastered Version", "Sweet Child O' Mine"},
        {"Wonderwall (Radio Edit)", "Wonderwall"},
        {"Live", "Live"}, // Band name, don't change
        {"", ""}, // Empty string
        {"   Title   ", "Title"}, // Whitespace
        {"Title (feat. Artist)", "Title"}, // Remove features
    }
    
    for _, tt := range tests {
        got := NormalizeTitle(tt.input)
        if got != tt.expected {
            t.Errorf("NormalizeTitle(%q) = %q, want %q", 
                tt.input, got, tt.expected)
        }
    }
}
```

## Open Questions
1. **Custom patterns**: Should users be able to add patterns via UI or only config file?
   - Recommendation: Config file only for now, UI in future version
   
2. **Caching**: Should normalized titles be cached in database?
   - Recommendation: Yes, if titles are accessed frequently

## Notes
- This feature will significantly improve track grouping and statistics
- Consider using normalized_title as the primary display field with original as tooltip
- Future: Use normalized titles for duplicate detection and "now playing" matching
- Pattern refinement will be ongoing based on user feedback
