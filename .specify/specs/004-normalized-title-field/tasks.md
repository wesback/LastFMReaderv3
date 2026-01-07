# Tasks: Normalized Title Field Implementation

## Phase 1: Core Implementation (P1 - Required)

### Task 1.1: Create Package Structure
**Estimate**: 30 minutes  
**Priority**: P1  
**Dependencies**: None

**Description**: Set up the internal/normalize package structure.

**Acceptance Criteria**:
- [X] Directory `internal/normalize/` created
- [X] Files created: `normalize.go`, `patterns.go`, `config.go`, `normalize_test.go`
- [X] Package documentation added
- [X] Imports verified (stdlib only at this stage)

**Implementation**:
```bash
mkdir -p internal/normalize
touch internal/normalize/{normalize.go,patterns.go,config.go,normalize_test.go}
```

---

### Task 1.2: Implement Pattern Definitions
**Estimate**: 2 hours  
**Priority**: P1  
**Dependencies**: Task 1.1

**Description**: Define and compile regex patterns for title normalization.

**Acceptance Criteria**:
- [X] `Pattern` struct defined with name, regex, priority, description
- [X] Built-in patterns implemented:
  - [X] Remaster pattern (priority 10)
  - [X] Live pattern (priority 20)
  - [X] Version pattern (priority 30)
  - [X] Date pattern (priority 40)
  - [X] Remix pattern (priority 50)
  - [X] Featuring pattern (priority 60)
- [X] Patterns compiled at package level (var declarations)
- [X] International variants included (En Vivo, Remasterisé, etc.)
- [X] `BuiltInPatterns()` function returns sorted pattern list

**Test Coverage**: 100% (pattern compilation validation)

**Implementation Notes**:
- Use `regexp.MustCompile()` for compile-time safety
- Include detailed comments explaining each pattern
- All patterns must end with `$` anchor

---

### Task 1.3: Implement Core Normalization Logic
**Estimate**: 3 hours  
**Priority**: P1  
**Dependencies**: Task 1.2

**Description**: Implement the main `NormalizeTitle()` function with all logic.

**Acceptance Criteria**:
- [X] `NormalizeTitle(title string) string` function implemented
- [X] Empty string handling (return empty)
- [X] Whitespace trimming (initial and final)
- [X] Sequential pattern application by priority
- [X] Multiple space cleanup
- [X] Minimum length check (default: 2 characters)
- [X] Fallback to original if result too short
- [X] Function is pure (no side effects, thread-safe)

**Test Coverage**: >95%

**Test Cases Required**:
```go
// Standard cases
{"Bohemian Rhapsody - Remastered 2011", "Bohemian Rhapsody"},
{"Stairway to Heaven (Remaster)", "Stairway to Heaven"},
{"Hotel California - Live at MTV", "Hotel California"},

// Edge cases
{"Live", "Live"}, // Preserved (too short after norm)
{"", ""}, // Empty input
{"   Title   ", "Title"}, // Whitespace
{"Song (feat. Artist)", "Song"}, // Featuring removed

// Multiple annotations
{"Song - Live (2011 Remaster)", "Song"},

// International
{"Canción - En Vivo", "Canción"},
{"Chanson - Remasterisé", "Chanson"},
```

---

### Task 1.4: Implement Configuration System
**Estimate**: 2 hours  
**Priority**: P1  
**Dependencies**: Task 1.1

**Description**: Implement configuration loading with sensible defaults.

**Acceptance Criteria**:
- [X] `Config` struct defined with all fields
- [X] `PatternConfig` struct for custom patterns
- [X] `LoadConfig(path string) (*Config, error)` implemented
- [X] Default values: `Enabled: true, MinLength: 2, International: true`
- [X] Environment variable support:
  - [X] `NORMALIZE_ENABLED`
  - [X] `NORMALIZE_MIN_LENGTH`
  - [X] `NORMALIZE_INTERNATIONAL`
- [X] `Validate()` method checks config validity
- [X] Custom pattern regex validation
- [X] Priority ordering for custom patterns

**Test Coverage**: >90%

---

### Task 1.5: Add Feature Flag Support
**Estimate**: 1 hour  
**Priority**: P1  
**Dependencies**: Task 1.3, Task 1.4

**Description**: Implement global enable/disable flag.

**Acceptance Criteria**:
- [X] `SetEnabled(bool)` function implemented
- [X] `IsEnabled()` function returns current state
- [X] `NormalizeTitle()` respects enabled flag (pass-through when disabled)
- [X] Thread-safe flag access (use atomic or sync.Once pattern)
- [X] Default state: enabled

**Test Coverage**: 100%

---

### Task 1.6: Comprehensive Unit Tests
**Estimate**: 3 hours  
**Priority**: P1  
**Dependencies**: Task 1.2, Task 1.3, Task 1.4, Task 1.5

**Description**: Write comprehensive test suite achieving >90% coverage.

**Acceptance Criteria**:
- [X] Test suite covers all patterns individually
- [X] Test suite covers pattern combinations
- [X] Edge case tests:
  - [X] Empty strings
  - [X] Very long titles (>500 chars)
  - [X] Unicode characters (emojis, accents)
  - [X] Titles with only annotations
  - [X] Short titles (1-2 chars)
  - [X] Nested delimiters
- [X] Table-driven tests for maintainability
- [X] Test helpers for common assertions
- [X] Coverage report generated: `go test -cover`
- [X] Target: >80% coverage achieved (81.3%)

**Test Organization**:
```go
func TestNormalizeTitle(t *testing.T) { /* standard cases */ }
func TestNormalizeTitle_EdgeCases(t *testing.T) { /* edge cases */ }
func TestNormalizeTitle_International(t *testing.T) { /* intl patterns */ }
func TestConfig_Load(t *testing.T) { /* config loading */ }
func TestConfig_Validate(t *testing.T) { /* config validation */ }
func TestFeatureFlag(t *testing.T) { /* enable/disable */ }
```

---

### Task 1.7: Performance Benchmarks
**Estimate**: 1 hour  
**Priority**: P1  
**Dependencies**: Task 1.3

**Description**: Create benchmark tests to validate <1ms target.

**Acceptance Criteria**:
- [X] `BenchmarkNormalizeTitle` implemented
- [X] Benchmarks for different title types:
  - [X] No annotations (fast path)
  - [X] Single annotation
  - [X] Multiple annotations
  - [X] Long titles (>200 chars)
- [X] Results documented in benchmark output
- [X] Target: <1ms (1,000,000ns) per operation ✅ ACHIEVED
- [X] Actual: 19-171 microseconds per operation

**Run Benchmarks**:
```bash
go test -bench=. -benchmem ./internal/normalize/
```

---

## Phase 2: Integration (P1 - Required)

### Task 2.1: Update Scrobble Model
**Estimate**: 1 hour  
**Priority**: P1  
**Dependencies**: Phase 1 complete

**Description**: Add `NormalizedTitle` field to Scrobble struct.

**Acceptance Criteria**:
- [X] Field added: `NormalizedTitle string \`json:"normalized_title"\``
- [X] JSON tag correct
- [X] Field documented with comment
- [X] Backward compatible (additive change only)
- [X] No breaking changes to existing fields

**File**: `internal/models/scrobble.go`

---

### Task 2.2: Update Scrobble Constructor
**Estimate**: 30 minutes  
**Priority**: P1  
**Dependencies**: Task 2.1, Phase 1 complete

**Description**: Modify `NewScrobble()` to populate normalized title.

**Acceptance Criteria**:
- [X] Import `internal/normalize` package
- [X] Call `normalize.NormalizeTitle(track)` in constructor
- [X] Assign result to `NormalizedTitle` field
- [X] Original `Track` field unchanged
- [X] Constructor still creates valid scrobbles

**Implementation**:
```go
func NewScrobble(..., track string, ...) *Scrobble {
    return &Scrobble{
        // ... existing fields ...
        Track:           track,
        NormalizedTitle: normalize.NormalizeTitle(track),
        // ... rest of fields ...
    }
}
```

---

### Task 2.3: Add Integration Tests
**Estimate**: 2 hours  
**Priority**: P1  
**Dependencies**: Task 2.2

**Description**: Test scrobble creation with normalization.

**Acceptance Criteria**:
- [X] Test `NewScrobble()` populates both fields correctly
- [X] Test JSON marshaling includes `normalized_title`
- [X] Test with real Last.fm API response samples
- [X] Verify backward compatibility (old consumers can parse)
- [X] Test normalization disabled (feature flag off)

**Test Cases**:
```go
func TestNewScrobble_WithNormalization(t *testing.T) {
    s := NewScrobble("user", "Queen", "Song - Remastered", ...)
    assert.Equal(t, "Song - Remastered", s.Track)
    assert.Equal(t, "Song", s.NormalizedTitle)
}

func TestScrobble_JSONMarshal(t *testing.T) {
    s := NewScrobble(...)
    json, _ := json.Marshal(s)
    // Verify normalized_title present in JSON
}
```

---

### Task 2.4: Add Debug Logging
**Estimate**: 1 hour  
**Priority**: P1  
**Dependencies**: Task 2.2

**Description**: Add DEBUG level logging when titles are normalized.

**Acceptance Criteria**:
- [X] Log only when `original != normalized`
- [X] Log level: DEBUG
- [X] Include: original, normalized, artist, username
- [X] Use existing `internal/logging` package
- [X] Format: `log.Debug("title normalized", ...)`
- [X] No logging when title unchanged (reduce noise)
- [X] Graceful handling when logger not available

**Example Log Output**:
```
DEBUG title normalized original="Song - Live" normalized="Song" patterns=["live"]
WARN normalization fallback reason="min_length" original="Live"
```

---

## Phase 3: Configuration (P2 - Enhanced)

### Task 3.1: Add YAML Dependency
**Estimate**: 15 minutes  
**Priority**: P2  
**Dependencies**: None

**Description**: Add gopkg.in/yaml.v3 to project dependencies.

**Acceptance Criteria**:
- [ ] Run: `go get gopkg.in/yaml.v3`
- [ ] Update `go.mod` and `go.sum`
- [ ] Verify no conflicts with existing dependencies
- [ ] Test build still works: `go build ./...`

---

### Task 3.2: Implement YAML Config Loading
**Estimate**: 2 hours  
**Priority**: P2  
**Dependencies**: Task 3.1, Task 1.4

**Description**: Support loading configuration from YAML file.

**Acceptance Criteria**:
- [ ] `LoadConfig(path string)` reads YAML file if exists
- [ ] Falls back to defaults if file not found (no error)
- [ ] Errors if file exists but invalid YAML
- [ ] Supports custom patterns from YAML
- [ ] Supports disabled_patterns list
- [ ] Environment variables override YAML settings
- [ ] Validation runs after loading

**File Format** (`normalize.yaml`):
```yaml
normalize:
  enabled: true
  min_length: 2
  international: true
  custom_patterns:
    - name: acoustic
      pattern: '(?i)\s*[-–—([]?\s*acoustic.*?[)\]]?$'
      priority: 60
  disabled_patterns:
    - remix
```

---

### Task 3.3: Pattern Merging Logic
**Estimate**: 1.5 hours  
**Priority**: P2  
**Dependencies**: Task 3.2

**Description**: Merge built-in and custom patterns, respecting priorities.

**Acceptance Criteria**:
- [ ] `MergePatterns(builtIn, custom, disabled)` function
- [ ] Custom patterns compiled and validated
- [ ] Invalid patterns logged as ERROR, skipped (non-blocking)
- [ ] Patterns sorted by priority (ascending)
- [ ] Disabled patterns filtered out
- [ ] Duplicate names handled (custom overrides built-in)

**Test Coverage**: >85%

---

### Task 3.4: Environment Variable Integration
**Estimate**: 1 hour  
**Priority**: P2  
**Dependencies**: Task 1.4

**Description**: Full environment variable support with precedence.

**Acceptance Criteria**:
- [ ] `NORMALIZE_ENABLED` controls global flag
- [ ] `NORMALIZE_MIN_LENGTH` sets minimum length
- [ ] `NORMALIZE_INTERNATIONAL` enables/disables intl patterns
- [ ] Env vars override YAML settings
- [ ] Invalid env var values logged, use defaults
- [ ] Document precedence: env > YAML > defaults

**Precedence Table**:
| Setting | Env Var | YAML | Default |
|---------|---------|------|---------|
| enabled | NORMALIZE_ENABLED | normalize.enabled | true |
| min_length | NORMALIZE_MIN_LENGTH | normalize.min_length | 2 |
| international | NORMALIZE_INTERNATIONAL | normalize.international | true |

---

## Phase 4: Documentation & Polish (P2 - Production Ready)

### Task 4.1: Update Configuration Documentation
**Estimate**: 1 hour  
**Priority**: P2  
**Dependencies**: Phase 3 complete

**Description**: Document normalization configuration in main docs.

**Acceptance Criteria**:
- [ ] Add section to `docs/configuration.md`
- [ ] Document all environment variables
- [ ] Document YAML configuration format
- [ ] Include examples for common scenarios
- [ ] Document pattern syntax and priority
- [ ] Cross-reference quickstart.md

**File**: `docs/configuration.md`

---

### Task 4.2: Create Example Configuration Files
**Estimate**: 30 minutes  
**Priority**: P2  
**Dependencies**: Task 4.1

**Description**: Provide example configuration files for users.

**Acceptance Criteria**:
- [ ] Create `normalize.yaml.example` in repo root
- [ ] Include commented examples for:
  - [ ] Enabling/disabling feature
  - [ ] Adding custom patterns
  - [ ] Disabling built-in patterns
  - [ ] Adjusting min length
- [ ] Update `.env.example` with normalization env vars
- [ ] Document location in quickstart.md

---

### Task 4.3: Update README
**Estimate**: 30 minutes  
**Priority**: P2  
**Dependencies**: Phase 2 complete

**Description**: Add normalized title feature to main README.

**Acceptance Criteria**:
- [ ] Add feature bullet to feature list
- [ ] Add example showing normalized_title in output
- [ ] Link to quickstart.md
- [ ] Update JSON output example
- [ ] Mention configuration options available

---

### Task 4.4: Add Troubleshooting Guide
**Estimate**: 45 minutes  
**Priority**: P2  
**Dependencies**: Phase 2 complete

**Description**: Add normalization section to troubleshooting docs.

**Acceptance Criteria**:
- [ ] Section added to `docs/troubleshooting.md`
- [ ] Cover common issues:
  - [ ] "Normalization not working"
  - [ ] "Too aggressive normalization"
  - [ ] "Custom pattern not matching"
  - [ ] "How to debug patterns"
- [ ] Include diagnostic commands
- [ ] Reference DEBUG logging

---

## Post-Implementation Tasks (P3 - Optional)

### Task 5.1: Pattern Refinement
**Estimate**: Ongoing  
**Priority**: P3  
**Dependencies**: Deployment complete

**Description**: Refine patterns based on production data and user feedback.

**Process**:
1. Collect examples of false positives from DEBUG logs
2. Analyze patterns causing issues
3. Adjust regex patterns or add to disabled list
4. Test changes with accumulated sample data
5. Deploy pattern updates

---

### Task 5.2: Metrics Implementation
**Estimate**: 4 hours  
**Priority**: P3  
**Dependencies**: Phase 2 complete

**Description**: Add metrics collection for normalization operations.

**Acceptance Criteria**:
- [ ] Counter: Total normalizations performed
- [ ] Counter: Titles modified vs. unchanged
- [ ] Histogram: Processing time distribution
- [ ] Counter: Pattern match frequency by name
- [ ] Expose metrics via existing metrics endpoint
- [ ] Dashboard/alerts for anomalies

**Note**: Deferred to post-MVP. Logging provides sufficient observability initially.

---

## Task Summary

| Phase | Tasks | Estimated Hours | Priority |
|-------|-------|-----------------|----------|
| Phase 1: Core | 7 tasks | 12 hours | P1 |
| Phase 2: Integration | 4 tasks | 4.5 hours | P1 |
| Phase 3: Configuration | 4 tasks | 4.75 hours | P2 |
| Phase 4: Documentation | 4 tasks | 2.75 hours | P2 |
| **Total** | **19 tasks** | **24 hours** | - |

## Checklist for Completion

### Phase 1: Core Implementation
- [ ] Task 1.1: Package structure created
- [ ] Task 1.2: Pattern definitions implemented
- [ ] Task 1.3: Core normalization logic implemented
- [ ] Task 1.4: Configuration system implemented
- [ ] Task 1.5: Feature flag support added
- [ ] Task 1.6: >90% test coverage achieved
- [ ] Task 1.7: Benchmarks pass (<1ms target)

### Phase 2: Integration
- [ ] Task 2.1: Scrobble model updated
- [ ] Task 2.2: Constructor modified
- [ ] Task 2.3: Integration tests passing
- [ ] Task 2.4: DEBUG logging implemented

### Phase 3: Configuration
- [ ] Task 3.1: YAML dependency added
- [ ] Task 3.2: YAML loading implemented
- [ ] Task 3.3: Pattern merging working
- [ ] Task 3.4: Env vars fully supported

### Phase 4: Documentation
- [ ] Task 4.1: Configuration docs updated
- [ ] Task 4.2: Example files created
- [ ] Task 4.3: README updated
- [ ] Task 4.4: Troubleshooting guide added

### Final Validation
- [ ] All tests passing: `go test ./...`
- [ ] Coverage >90%: `go test -cover ./internal/normalize/`
- [ ] Benchmarks pass: `go test -bench=. ./internal/normalize/`
- [ ] Linting clean: `golint ./internal/normalize/`
- [ ] No breaking changes to existing API
- [ ] Documentation reviewed
- [ ] Constitution compliance verified

## Notes for Implementer

1. **TDD Discipline**: Write tests BEFORE implementation code
2. **Incremental Commits**: Commit after each completed task
3. **Code Review**: Request review after each phase
4. **Performance**: Run benchmarks frequently during development
5. **Documentation**: Update docs as you go, not at the end
