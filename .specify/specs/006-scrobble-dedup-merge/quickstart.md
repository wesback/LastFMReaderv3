# Developer Quickstart: Scrobble Deduplication & Merging

**Feature**: 006-scrobble-dedup-merge  
**Phase**: 1 (Design)  
**Date**: 2026-01-06

## Purpose

Get developers up to speed quickly with building, testing, and running the merge feature. This guide covers project setup, code structure, testing strategies, and common development workflows.

---

## Prerequisites

- **Go**: 1.24.0 or later
- **Git**: For cloning repository
- **Make**: For build automation (optional)
- **Azure CLI**: For Azure Blob Storage testing (optional)

**Verify Installation**:
```bash
go version  # Should show 1.24.0 or later
git --version
make --version  # Optional
az --version    # Optional, for Azure testing
```

---

## Quick Start (5 Minutes)

### 1. Clone & Build

```bash
# Clone repository
git clone https://github.com/lastfm-reader/lastfm-sync.git
cd lastfm-sync

# Install dependencies
go mod download

# Build binary
go build -o bin/lastfm-sync ./cmd/lastfm-sync

# Verify installation
./bin/lastfm-sync --version
```

### 2. Run Example Merge

```bash
# Create test data
echo '{"artist":"The Beatles","album":"Abbey Road","title":"Come Together","timestamp":1735689600}' > test1.ndjson
echo '{"artist":"Pink Floyd","album":"The Dark Side of the Moon","title":"Time","timestamp":1735689700}' > test2.ndjson

# Run merge
./bin/lastfm-sync merge test1.ndjson test2.ndjson -o merged.json

# View output
cat merged.json
```

**Expected Output**:
```json
[
  {
    "artist": "The Beatles",
    "album": "Abbey Road",
    "title": "Come Together",
    "timestamp": 1735689600
  },
  {
    "artist": "Pink Floyd",
    "album": "The Dark Side of the Moon",
    "title": "Time",
    "timestamp": 1735689700
  }
]
```

---

## Project Structure

```
lastfm-sync/
├── cmd/
│   └── lastfm-sync/
│       ├── main.go           # Entry point
│       └── commands/
│           ├── fetch.go      # Existing fetch command
│           └── merge.go      # NEW: merge command implementation
├── internal/
│   ├── merge/                # NEW: merge package
│   │   ├── merger.go         # Main merge orchestration
│   │   ├── deduplicator.go   # Deduplication logic
│   │   ├── conflict.go       # Conflict resolution
│   │   ├── strategies.go     # Deduplication strategies
│   │   ├── checkpoint.go     # Checkpointing
│   │   ├── reader.go         # NDJSON file reader
│   │   └── config.go         # Configuration types
│   ├── models/
│   │   └── scrobble.go       # Existing Scrobble struct
│   ├── writer/               # Existing writer interfaces
│   ├── progress/             # Existing progress bars
│   ├── logging/              # Existing logging
│   └── config/               # Existing config management
└── tests/
    ├── unit/
    │   └── merge/            # NEW: unit tests
    └── integration/
        └── merge_test.go     # NEW: integration tests
```

---

## Development Workflow

### Create Feature Branch

```bash
git checkout -b 006-scrobble-dedup-merge
```

### Implement Core Logic

**Step 1**: Create `internal/merge/deduplicator.go`
```go
package merge

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "strings"
    "github.com/lastfm-reader/lastfm-sync/internal/models"
)

type DeduplicationMap struct {
    data      map[string]*models.Scrobble
    strategy  string
    conflicts int
}

func NewDeduplicationMap(strategy string) *DeduplicationMap {
    return &DeduplicationMap{
        data:     make(map[string]*models.Scrobble),
        strategy: strategy,
    }
}

func (dm *DeduplicationMap) Add(scrobble *models.Scrobble) bool {
    key := dm.generateKey(scrobble)
    
    if _, exists := dm.data[key]; exists {
        dm.conflicts++
        return false // Duplicate
    }
    
    dm.data[key] = scrobble
    return true // New
}

func (dm *DeduplicationMap) generateKey(s *models.Scrobble) string {
    h := sha256.New()
    h.Write([]byte(strings.ToLower(s.Artist)))
    h.Write([]byte(strings.ToLower(s.Album)))
    h.Write([]byte(strings.ToLower(s.Title)))
    h.Write([]byte(fmt.Sprintf("%d", s.Timestamp)))
    return hex.EncodeToString(h.Sum(nil))
}
```

**Step 2**: Write Unit Test
```go
// internal/merge/deduplicator_test.go
package merge

import (
    "testing"
    "github.com/lastfm-reader/lastfm-sync/internal/models"
)

func TestDeduplicationMap_Add(t *testing.T) {
    dm := NewDeduplicationMap("default")
    
    scrobble := &models.Scrobble{
        Artist:    "The Beatles",
        Album:     "Abbey Road",
        Title:     "Come Together",
        Timestamp: 1735689600,
    }
    
    // First add should succeed
    if !dm.Add(scrobble) {
        t.Error("Expected first add to succeed")
    }
    
    // Second add (duplicate) should fail
    if dm.Add(scrobble) {
        t.Error("Expected duplicate add to fail")
    }
    
    // Check conflict count
    if dm.conflicts != 1 {
        t.Errorf("Expected 1 conflict, got %d", dm.conflicts)
    }
}
```

**Step 3**: Run Tests
```bash
# Run unit tests
go test ./internal/merge/...

# Run with coverage
go test -cover ./internal/merge/...

# Generate coverage report
go test -coverprofile=coverage.out ./internal/merge/...
go tool cover -html=coverage.out -o coverage.html
```

### Implement CLI Command

**Step 4**: Create `cmd/lastfm-sync/commands/merge.go`
```go
package commands

import (
    "github.com/spf13/cobra"
    "github.com/lastfm-reader/lastfm-sync/internal/merge"
)

var mergeCmd = &cobra.Command{
    Use:   "merge [flags] <input-pattern...>",
    Short: "Merge NDJSON scrobble files into deduplicated JSON",
    Args:  cobra.MinimumNArgs(1),
    RunE:  runMerge,
}

func init() {
    rootCmd.AddCommand(mergeCmd)
    
    mergeCmd.Flags().StringP("output", "o", "merged-scrobbles.json", "Output file")
    mergeCmd.Flags().String("strategy", "default", "Deduplication strategy")
    // ... more flags
}

func runMerge(cmd *cobra.Command, args []string) error {
    // Parse flags
    output, _ := cmd.Flags().GetString("output")
    strategy, _ := cmd.Flags().GetString("strategy")
    
    // Create config
    config := &merge.MergeConfig{
        InputPatterns: args,
        OutputPath:    output,
        Strategy:      strategy,
    }
    
    // Run merge
    merger := merge.NewMerger(config)
    result, err := merger.Merge()
    if err != nil {
        return err
    }
    
    // Print stats
    fmt.Println(result.Stats.String())
    return nil
}
```

**Step 5**: Manual Testing
```bash
# Rebuild binary
go build -o bin/lastfm-sync ./cmd/lastfm-sync

# Test merge command
./bin/lastfm-sync merge --help
./bin/lastfm-sync merge test1.ndjson test2.ndjson -o output.json
```

---

## Testing Strategy

### Unit Tests

**Location**: `internal/merge/*_test.go`

**Coverage Requirements**: ≥80% per Constitution

**Run Unit Tests**:
```bash
# All unit tests
go test ./internal/merge/...

# Specific test
go test ./internal/merge -run TestDeduplicationMap_Add

# With verbose output
go test -v ./internal/merge/...

# With coverage
go test -cover ./internal/merge/...
```

**Unit Test Structure**:
```go
func TestFunctionName_Scenario(t *testing.T) {
    // Arrange: Setup test data
    input := /* ... */
    
    // Act: Execute function
    result := FunctionName(input)
    
    // Assert: Verify outcome
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

**Use Table-Driven Tests**:
```go
func TestGenerateKey_Strategies(t *testing.T) {
    scrobble := &models.Scrobble{
        Artist:    "The Beatles",
        Album:     "Abbey Road",
        Title:     "Come Together",
        Timestamp: 1735689600,
    }
    
    tests := []struct {
        name     string
        strategy string
        wantLen  int // SHA256 hex length
    }{
        {"default", "default", 64},
        {"strict", "strict", 64},
        {"relaxed", "relaxed", 64},
        {"mbid", "mbid", 64},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dm := NewDeduplicationMap(tt.strategy)
            key := dm.generateKey(scrobble)
            if len(key) != tt.wantLen {
                t.Errorf("Expected key length %d, got %d", tt.wantLen, len(key))
            }
        })
    }
}
```

---

### Integration Tests

**Location**: `tests/integration/merge_test.go`

**Run Integration Tests**:
```bash
# Run all integration tests
go test ./tests/integration/...

# Run with short flag (skip slow tests)
go test -short ./tests/integration/...
```

**Integration Test Structure**:
```go
func TestMerge_EndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Create temporary directory
    tmpDir := t.TempDir()
    
    // Create test files
    input1 := filepath.Join(tmpDir, "test1.ndjson")
    writeTestFile(t, input1, /* data */)
    
    input2 := filepath.Join(tmpDir, "test2.ndjson")
    writeTestFile(t, input2, /* data */)
    
    // Create config
    config := &merge.MergeConfig{
        InputPatterns: []string{filepath.Join(tmpDir, "*.ndjson")},
        OutputPath:    filepath.Join(tmpDir, "merged.json"),
        Strategy:      "default",
    }
    
    // Run merge
    merger := merge.NewMerger(config)
    result, err := merger.Merge()
    
    // Assert success
    if err != nil {
        t.Fatalf("Merge failed: %v", err)
    }
    
    // Verify output file exists
    if _, err := os.Stat(config.OutputPath); err != nil {
        t.Fatalf("Output file not created: %v", err)
    }
    
    // Verify statistics
    if result.Stats.UniqueScrobbles != expectedCount {
        t.Errorf("Expected %d unique scrobbles, got %d", 
            expectedCount, result.Stats.UniqueScrobbles)
    }
}
```

---

### Benchmark Tests

**Location**: `internal/merge/deduplicator_bench_test.go`

**Run Benchmarks**:
```bash
# Run all benchmarks
go test -bench=. ./internal/merge/...

# Run specific benchmark
go test -bench=BenchmarkDeduplication ./internal/merge/...

# With memory stats
go test -bench=. -benchmem ./internal/merge/...

# Multiple iterations for accuracy
go test -bench=. -benchtime=10s ./internal/merge/...
```

**Benchmark Structure**:
```go
func BenchmarkDeduplication(b *testing.B) {
    // Generate test data
    scrobbles := generateTestScrobbles(10000)
    
    b.ResetTimer() // Don't count setup time
    
    for i := 0; i < b.N; i++ {
        dm := NewDeduplicationMap("default")
        for _, s := range scrobbles {
            dm.Add(s)
        }
    }
}

func BenchmarkGenerateKey(b *testing.B) {
    scrobble := &models.Scrobble{
        Artist:    "The Beatles",
        Title:     "Come Together",
        Timestamp: 1735689600,
    }
    
    dm := NewDeduplicationMap("default")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = dm.generateKey(scrobble)
    }
}
```

**Performance Targets**:
- Deduplication: ≥10,000 scrobbles/sec (SC-PERF-001)
- Memory: <500MB for 1M scrobbles (SC-PERF-002)

---

## Debugging Tips

### Enable Debug Logging

```bash
./bin/lastfm-sync merge --log-level debug "data/*.ndjson"
```

### Use `go run` for Rapid Iteration

```bash
# No need to rebuild binary
go run ./cmd/lastfm-sync merge "data/*.ndjson"
```

### Print Deduplication Keys

```go
// In deduplicator.go
func (dm *DeduplicationMap) generateKey(s *models.Scrobble) string {
    // ... generate key ...
    
    // Temporary debug logging
    fmt.Printf("DEBUG: Key=%s Artist=%s Title=%s\n", key, s.Artist, s.Title)
    
    return key
}
```

### Profile Memory Usage

```bash
# Generate memory profile
go test -memprofile=mem.prof -bench=BenchmarkDeduplication ./internal/merge/...

# Analyze with pprof
go tool pprof mem.prof
> top10
> list DeduplicationMap.Add
```

### Profile CPU Usage

```bash
# Generate CPU profile
go test -cpuprofile=cpu.prof -bench=BenchmarkDeduplication ./internal/merge/...

# Visualize with pprof
go tool pprof -http=:8080 cpu.prof
```

---

## Common Development Tasks

### Add New Deduplication Strategy

**1. Update `DeduplicationStrategy` enum**:
```go
// internal/merge/config.go
const (
    StrategyDefault  DeduplicationStrategy = "default"
    StrategyStrict   DeduplicationStrategy = "strict"
    StrategyRelaxed  DeduplicationStrategy = "relaxed"
    StrategyMBID     DeduplicationStrategy = "mbid"
    StrategyCustom   DeduplicationStrategy = "custom" // NEW
)
```

**2. Implement key generation logic**:
```go
// internal/merge/deduplicator.go
func (dm *DeduplicationMap) generateKey(s *models.Scrobble) string {
    // ...
    case StrategyCustom:
        // Custom logic here
        h.Write([]byte(strings.ToLower(s.NormalizedTitle)))
        h.Write([]byte(fmt.Sprintf("%d", s.Timestamp)))
    // ...
}
```

**3. Add unit tests**:
```go
func TestGenerateKey_Custom(t *testing.T) {
    // ... test implementation
}
```

**4. Update documentation**:
- [spec.md](spec.md): Add to FR-DEDUP-002
- [contracts/merge-command.md](contracts/merge-command.md): Document flag usage

---

### Add New Conflict Resolution Mode

**1. Update `ConflictResolution` enum**:
```go
// internal/merge/config.go
const (
    ResolutionCompleteness ConflictResolution = "completeness"
    ResolutionFirst        ConflictResolution = "first"
    ResolutionLast         ConflictResolution = "last"
    ResolutionNewest       ConflictResolution = "newest" // NEW
)
```

**2. Implement resolution logic**:
```go
// internal/merge/conflict.go
func (dm *DeduplicationMap) resolveConflict(existing, new *models.Scrobble) *models.Scrobble {
    // ...
    case ResolutionNewest:
        if new.Timestamp > existing.Timestamp {
            return new
        }
        return existing
    // ...
}
```

**3. Add tests and documentation** (same as above)

---

### Test Azure Blob Storage Integration

**1. Set up Azure credentials**:
```bash
# Option 1: Azure CLI login
az login

# Option 2: Service principal (for CI/CD)
export AZURE_CLIENT_ID=<client-id>
export AZURE_CLIENT_SECRET=<client-secret>
export AZURE_TENANT_ID=<tenant-id>
```

**2. Create test storage account**:
```bash
az storage account create \
  --name testlastfmsync \
  --resource-group test-rg \
  --location eastus \
  --sku Standard_LRS

az storage container create \
  --name scrobbles \
  --account-name testlastfmsync
```

**3. Run merge with Azure**:
```bash
./bin/lastfm-sync merge \
  -s azure \
  -o "az://testlastfmsync/scrobbles/merged.json" \
  "data/*.ndjson"
```

**4. Verify output**:
```bash
az storage blob download \
  --account-name testlastfmsync \
  --container-name scrobbles \
  --name merged.json \
  --file local-copy.json

cat local-copy.json
```

---

## Makefile Shortcuts

**Add to `Makefile`**:
```makefile
# Build merge command
.PHONY: build-merge
build-merge:
	go build -o bin/lastfm-sync ./cmd/lastfm-sync

# Run merge unit tests
.PHONY: test-merge
test-merge:
	go test -v -cover ./internal/merge/...

# Run merge integration tests
.PHONY: test-merge-integration
test-merge-integration:
	go test -v ./tests/integration/merge_test.go

# Run merge benchmarks
.PHONY: bench-merge
bench-merge:
	go test -bench=. -benchmem ./internal/merge/...

# Generate test data
.PHONY: generate-test-data
generate-test-data:
	@echo '{"artist":"Test1","title":"Song1","timestamp":1000}' > test1.ndjson
	@echo '{"artist":"Test2","title":"Song2","timestamp":2000}' > test2.ndjson
	@echo "Test data generated: test1.ndjson, test2.ndjson"

# Clean test artifacts
.PHONY: clean-merge
clean-merge:
	rm -f merged.json .merge-checkpoint-*.json
	rm -f coverage.out coverage.html
	rm -f *.prof
```

**Usage**:
```bash
make build-merge
make test-merge
make test-merge-integration
make bench-merge
make generate-test-data
make clean-merge
```

---

## CI/CD Integration

### GitHub Actions Workflow

**Create `.github/workflows/merge-tests.yml`**:
```yaml
name: Merge Tests

on:
  push:
    paths:
      - 'internal/merge/**'
      - 'cmd/lastfm-sync/commands/merge.go'
      - 'tests/integration/merge_test.go'
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Run unit tests
        run: go test -v -cover ./internal/merge/...
      
      - name: Run integration tests
        run: go test -v ./tests/integration/merge_test.go
      
      - name: Check coverage
        run: |
          go test -coverprofile=coverage.out ./internal/merge/...
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $coverage%"
          if (( $(echo "$coverage < 80" | bc -l) )); then
            echo "Coverage below 80% threshold"
            exit 1
          fi
      
      - name: Run benchmarks
        run: go test -bench=. -benchmem ./internal/merge/...
```

---

## Troubleshooting

### "No input files found"

**Problem**: Glob pattern doesn't match any files

**Solution**:
```bash
# Check pattern matches files
ls data/*.ndjson

# Use absolute paths
./bin/lastfm-sync merge "$(pwd)/data/*.ndjson"

# Enable debug logging
./bin/lastfm-sync merge --log-level debug "data/*.ndjson"
```

### "Out of memory" Error

**Problem**: Processing too many scrobbles at once

**Solution**:
```bash
# Enable checkpointing (allows resume)
./bin/lastfm-sync merge --checkpoint-interval 10000 "data/*.ndjson"

# Process files in smaller batches
./bin/lastfm-sync merge "data/batch1/*.ndjson" -o merged1.json
./bin/lastfm-sync merge "data/batch2/*.ndjson" -o merged2.json
./bin/lastfm-sync merge merged1.json merged2.json -o final.json
```

### Tests Failing with "SHA256 mismatch"

**Problem**: Case sensitivity in deduplication keys

**Solution**: Ensure `strings.ToLower()` applied consistently:
```go
// ✓ Correct
h.Write([]byte(strings.ToLower(s.Artist)))

// ✗ Incorrect
h.Write([]byte(s.Artist))
```

### Progress Bar Not Updating

**Problem**: Progress bar frozen or not showing

**Solution**:
```bash
# Check if stderr is a TTY
if [ -t 2 ]; then echo "stderr is TTY"; fi

# Disable progress bar for non-TTY environments
./bin/lastfm-sync merge --no-progress "data/*.ndjson"
```

---

## Next Steps

1. **Implement Core Logic**: Start with `internal/merge/deduplicator.go`
2. **Write Unit Tests**: Achieve ≥80% coverage per Constitution
3. **Implement CLI Command**: Add `cmd/lastfm-sync/commands/merge.go`
4. **Integration Testing**: Test end-to-end with real NDJSON files
5. **Performance Tuning**: Run benchmarks, optimize hot paths
6. **Documentation**: Update README, add examples

---

**Quickstart Guide Complete** ✅  
Developers can now build, test, and extend the merge feature. Ready for implementation!
