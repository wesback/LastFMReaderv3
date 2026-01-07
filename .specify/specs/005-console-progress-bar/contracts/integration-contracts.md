# Integration Contracts

**Feature**: Console Progress Bar  
**Date**: 2026-01-07  
**Status**: Planning

## Integration Points

This document defines how the progress package integrates with existing SpecKit components.

## 1. Last.fm Sync Integration

### Current State

```go
// internal/service/sync.go
func (s *SyncService) FetchRecentScrobbles() ([]models.Scrobble, error) {
    // Current implementation has no progress feedback
    allScrobbles := []models.Scrobble{}
    page := 1
    
    for {
        scrobbles, err := s.client.GetRecentTracks(page)
        if err != nil {
            return nil, err
        }
        
        allScrobbles = append(allScrobbles, scrobbles...)
        
        if len(scrobbles) == 0 {
            break
        }
        page++
    }
    
    return allScrobbles, nil
}
```

### Modified Integration

```go
// internal/service/sync.go
import "internal/progress"

func (s *SyncService) FetchRecentScrobbles(reporter progress.ProgressReporter) ([]models.Scrobble, error) {
    // Get total count first
    totalTracks, err := s.client.GetTotalTrackCount()
    if err != nil {
        return nil, err
    }
    
    // Start progress tracking
    if err := reporter.Start(totalTracks, "Fetching scrobbles"); err != nil {
        s.logger.Warn("Failed to start progress bar", "error", err)
    }
    defer func() {
        if !reporter.IsFinished() {
            reporter.Finish("Fetch complete")
        }
    }()
    
    allScrobbles := []models.Scrobble{}
    page := 1
    
    for {
        // Check for rate limiting
        if s.rateLimiter.IsLimited() {
            reporter.SwitchToSpinner()
            reporter.SetDescription("Rate limited - waiting...")
            waitDuration := s.rateLimiter.Wait()
            time.Sleep(waitDuration)
            reporter.ResumeProgress()
            reporter.SetDescription("Fetching scrobbles")
        }
        
        scrobbles, err := s.client.GetRecentTracks(page)
        if err != nil {
            reporter.FinishWithError(fmt.Sprintf("Fetch failed: %v", err))
            return nil, err
        }
        
        allScrobbles = append(allScrobbles, scrobbles...)
        reporter.Add(len(scrobbles))
        
        if len(scrobbles) == 0 {
            break
        }
        page++
    }
    
    reporter.Finish(fmt.Sprintf("Fetched %d scrobbles", len(allScrobbles)))
    return allScrobbles, nil
}
```

### Constructor Modification

```go
// internal/service/sync.go
type SyncService struct {
    client      *lastfm.Client
    logger      *logging.Logger
    rateLimiter *ratelimit.Limiter
    config      *config.Config
}

func NewSyncService(cfg *config.Config, client *lastfm.Client, logger *logging.Logger) *SyncService {
    return &SyncService{
        client:      client,
        logger:      logger,
        rateLimiter: ratelimit.NewLimiter(cfg),
        config:      cfg,
    }
}
```

### Command Integration

```go
// cmd/lastfm-sync/commands/fetch.go
func runFetch(cmd *cobra.Command, args []string) error {
    // ... existing setup ...
    
    // Create progress reporter
    reporter := progress.NewProgressReporter(cfg, 0, "Initializing")
    
    // Run sync with progress
    scrobbles, err := syncService.FetchRecentScrobbles(reporter)
    if err != nil {
        return fmt.Errorf("sync failed: %w", err)
    }
    
    // ... rest of command ...
}
```

### Contract Guarantees

| Requirement | Implementation | Verification |
|-------------|----------------|--------------|
| Non-blocking errors | Progress failures logged, sync continues | Unit test: `TestSyncWithProgressFailure` |
| Rate limit visibility | Switch to spinner during waits | Integration test: `TestSyncRateLimitDisplay` |
| Accurate counts | Add exact fetch count per page | Unit test: `TestProgressCountAccuracy` |
| Graceful fallback | NoOpProgressBar when terminal non-interactive | Integration test: `TestSyncInPipe` |

## 2. Title Normalization Integration

### Current State

```go
// internal/normalize/normalize.go
func NormalizeScrobbles(scrobbles []models.Scrobble) ([]models.Scrobble, error) {
    normalized := make([]models.Scrobble, len(scrobbles))
    
    for i, s := range scrobbles {
        s.NormalizedTitle = Normalize(s.Track)
        normalized[i] = s
    }
    
    return normalized, nil
}
```

### Modified Integration

```go
// internal/normalize/normalize.go
import "internal/progress"

func NormalizeScrobbles(scrobbles []models.Scrobble, reporter progress.ProgressReporter) ([]models.Scrobble, error) {
    // Start progress tracking
    if err := reporter.Start(int64(len(scrobbles)), "Normalizing titles"); err != nil {
        logger.Warn("Failed to start progress bar", "error", err)
    }
    defer func() {
        if !reporter.IsFinished() {
            reporter.Finish("Normalization complete")
        }
    }()
    
    normalized := make([]models.Scrobble, len(scrobbles))
    modifiedCount := 0
    
    // Batch updates for performance (100 items per update)
    batchSize := 100
    
    for i, s := range scrobbles {
        originalTitle := s.Track
        s.NormalizedTitle = Normalize(s.Track)
        
        if s.NormalizedTitle != originalTitle {
            modifiedCount++
        }
        
        normalized[i] = s
        
        // Update progress every batch
        if (i+1)%batchSize == 0 || i == len(scrobbles)-1 {
            reporter.Add(batchSize)
        }
    }
    
    reporter.Finish(fmt.Sprintf("Normalized %d titles (%d modified)", len(scrobbles), modifiedCount))
    return normalized, nil
}
```

### Command Integration

```go
// cmd/lastfm-sync/commands/normalize.go (hypothetical new command)
func runNormalize(cmd *cobra.Command, args []string) error {
    // ... load scrobbles ...
    
    // Create progress reporter
    reporter := progress.NewProgressReporter(cfg, int64(len(scrobbles)), "Normalizing")
    
    // Normalize with progress
    normalized, err := normalize.NormalizeScrobbles(scrobbles, reporter)
    if err != nil {
        return fmt.Errorf("normalization failed: %w", err)
    }
    
    // ... save results ...
}
```

### Contract Guarantees

| Requirement | Implementation | Verification |
|-------------|----------------|--------------|
| Batch updates | Update every 100 items | Unit test: `TestNormalizeBatching` |
| Modification count | Track and report changed titles | Unit test: `TestNormalizeModifiedCount` |
| Fast operation | < 1% overhead on normalization time | Benchmark: `BenchmarkNormalizeWithProgress` |
| Memory efficiency | No duplicate allocations | Memory profile in CI |

## 3. File Export Integration

### Current State

```go
// internal/writer/local.go
func (w *LocalWriter) Write(scrobbles []models.Scrobble) error {
    file, err := os.Create(w.outputPath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    encoder := json.NewEncoder(file)
    return encoder.Encode(scrobbles)
}
```

### Modified Integration

```go
// internal/writer/local.go
import "internal/progress"

func (w *LocalWriter) Write(scrobbles []models.Scrobble, reporter progress.ProgressReporter) error {
    // Start progress tracking
    if err := reporter.Start(int64(len(scrobbles)), "Exporting to file"); err != nil {
        w.logger.Warn("Failed to start progress bar", "error", err)
    }
    defer func() {
        if !reporter.IsFinished() {
            reporter.Finish("Export complete")
        }
    }()
    
    file, err := os.Create(w.outputPath)
    if err != nil {
        reporter.FinishWithError(fmt.Sprintf("File creation failed: %v", err))
        return err
    }
    defer file.Close()
    
    // Write with progress updates
    encoder := json.NewEncoder(file)
    batchSize := 100
    
    for i, scrobble := range scrobbles {
        if err := encoder.Encode(scrobble); err != nil {
            reporter.FinishWithError(fmt.Sprintf("Encoding failed: %v", err))
            return err
        }
        
        // Update progress every batch
        if (i+1)%batchSize == 0 || i == len(scrobbles)-1 {
            reporter.Add(min(batchSize, len(scrobbles)-i))
        }
    }
    
    reporter.Finish(fmt.Sprintf("Exported %d scrobbles to %s", len(scrobbles), w.outputPath))
    return nil
}
```

### Azure Writer Integration

```go
// internal/writer/azure.go
func (w *AzureWriter) Write(scrobbles []models.Scrobble, reporter progress.ProgressReporter) error {
    if err := reporter.Start(int64(len(scrobbles)), "Uploading to Azure"); err != nil {
        w.logger.Warn("Failed to start progress bar", "error", err)
    }
    defer func() {
        if !reporter.IsFinished() {
            reporter.Finish("Upload complete")
        }
    }()
    
    // Create blob
    data, err := json.Marshal(scrobbles)
    if err != nil {
        reporter.FinishWithError(fmt.Sprintf("Serialization failed: %v", err))
        return err
    }
    
    // Upload with progress simulation (Azure SDK doesn't provide byte-level progress)
    reporter.SetDescription("Uploading to Azure Blob Storage")
    
    // Simulate chunked upload for large files
    chunkSize := 4 * 1024 * 1024 // 4MB chunks
    totalBytes := len(data)
    chunksTotal := (totalBytes + chunkSize - 1) / chunkSize
    
    for i := 0; i < chunksTotal; i++ {
        start := i * chunkSize
        end := min(start+chunkSize, totalBytes)
        
        // Upload chunk (actual Azure SDK call here)
        if err := w.uploadChunk(data[start:end]); err != nil {
            reporter.FinishWithError(fmt.Sprintf("Upload failed: %v", err))
            return err
        }
        
        reporter.Add(1)
    }
    
    reporter.Finish(fmt.Sprintf("Uploaded %d scrobbles to Azure", len(scrobbles)))
    return nil
}
```

### Contract Guarantees

| Requirement | Implementation | Verification |
|-------------|----------------|--------------|
| File I/O efficiency | Batch writes, minimal overhead | Benchmark: `BenchmarkWriteWithProgress` |
| Azure upload visibility | Chunk-based progress updates | Integration test: `TestAzureUploadProgress` |
| Error recovery | Graceful progress cleanup on errors | Unit test: `TestWriteErrorHandling` |
| Large file support | Progress updates for multi-GB files | Integration test: `TestLargeFileProgress` |

## 4. Configuration Integration

### Config Structure

```go
// internal/config/types.go
type Config struct {
    // ... existing fields ...
    
    Progress ProgressConfig `yaml:"progress" json:"progress"`
}

type ProgressConfig struct {
    Enabled         bool          `yaml:"enabled" json:"enabled"`
    Style           string        `yaml:"style" json:"style"`
    ShowSpeed       bool          `yaml:"show_speed" json:"show_speed"`
    ShowETA         bool          `yaml:"show_eta" json:"show_eta"`
    ShowCount       bool          `yaml:"show_count" json:"show_count"`
    ShowPercentage  bool          `yaml:"show_percentage" json:"show_percentage"`
    ShowElapsed     bool          `yaml:"show_elapsed" json:"show_elapsed"`
    Width           int           `yaml:"width" json:"width"`
    RefreshRate     time.Duration `yaml:"refresh_rate" json:"refresh_rate"`
    Colors          bool          `yaml:"colors" json:"colors"`
    AutoClear       bool          `yaml:"auto_clear" json:"auto_clear"`
}
```

### Default Values

```go
// internal/config/defaults.go
func DefaultProgressConfig() ProgressConfig {
    return ProgressConfig{
        Enabled:        true,
        Style:          "blocks",
        ShowSpeed:      true,
        ShowETA:        true,
        ShowCount:      true,
        ShowPercentage: true,
        ShowElapsed:    false,
        Width:          0, // Auto-detect
        RefreshRate:    100 * time.Millisecond,
        Colors:         true,
        AutoClear:      true,
    }
}
```

### Environment Variable Overrides

```go
// internal/config/loader.go
func (l *Loader) loadProgressConfig() ProgressConfig {
    cfg := DefaultProgressConfig()
    
    // Override from environment
    if val := os.Getenv("SPECKIT_NO_PROGRESS"); val != "" {
        cfg.Enabled = val != "true" && val != "1"
    }
    
    if val := os.Getenv("SPECKIT_PROGRESS_ASCII"); val == "true" || val == "1" {
        cfg.Style = "ascii"
    }
    
    if val := os.Getenv("SPECKIT_NO_COLOR"); val == "true" || val == "1" {
        cfg.Colors = false
    }
    
    if val := os.Getenv("SPECKIT_PROGRESS_REFRESH"); val != "" {
        if ms, err := strconv.Atoi(val); err == nil {
            cfg.RefreshRate = time.Duration(ms) * time.Millisecond
        }
    }
    
    if val := os.Getenv("SPECKIT_PROGRESS_WIDTH"); val != "" {
        if width, err := strconv.Atoi(val); err == nil {
            cfg.Width = width
        }
    }
    
    return cfg
}
```

### Contract Guarantees

| Requirement | Implementation | Verification |
|-------------|----------------|--------------|
| Config validation | Invalid values use defaults | Unit test: `TestProgressConfigDefaults` |
| Env precedence | Environment overrides config file | Unit test: `TestProgressEnvOverride` |
| Backward compatibility | Missing config fields use defaults | Unit test: `TestProgressConfigMigration` |
| Type safety | Validated at load time | Unit test: `TestProgressConfigValidation` |

## 5. Logging Integration

### Progress + Logging Coexistence

```go
// internal/logging/logger.go
import "internal/progress"

type Logger struct {
    // ... existing fields ...
    progressActive bool
    progressBar    progress.ProgressReporter
}

// SetProgressBar registers active progress bar
func (l *Logger) SetProgressBar(bar progress.ProgressReporter) {
    l.progressActive = bar != nil && !bar.IsFinished()
    l.progressBar = bar
}

// Info logs info message
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
    // If progress active, temporarily clear bar before logging
    if l.progressActive && l.progressBar != nil {
        l.progressBar.Clear()
    }
    
    l.logger.Info(msg, keysAndValues...)
    
    // Progress bar automatically redraws on next update
}
```

### Contract Guarantees

| Requirement | Implementation | Verification |
|-------------|----------------|--------------|
| No interference | Logs clear progress, bars redraw | Integration test: `TestProgressWithLogs` |
| Log visibility | All logs visible even with progress | Manual test: Verify log output |
| Minimal overhead | Logging performance unchanged | Benchmark: `BenchmarkLogWithProgress` |
| Thread safety | Concurrent log/progress calls safe | Race detector in CI |

## 6. Multi-Operation Workflow

### Full Sync Workflow Integration

```go
// cmd/lastfm-sync/commands/full_sync.go
func runFullSync(cmd *cobra.Command, args []string) error {
    cfg := config.Load()
    
    // Create multi-progress container
    multi := progress.NewMulti(
        progress.WithStyle(getStyleFromConfig(cfg)),
        progress.WithColors(cfg.Progress.Colors),
    )
    
    // Phase 1: Fetch
    fetchBar := multi.AddBar(0, "Fetching scrobbles from Last.fm")
    scrobbles, err := syncService.FetchRecentScrobbles(fetchBar)
    if err != nil {
        fetchBar.FinishWithError(fmt.Sprintf("Fetch failed: %v", err))
        return err
    }
    fetchBar.Finish(fmt.Sprintf("✓ Fetched %d scrobbles", len(scrobbles)))
    
    // Phase 2: Normalize
    normalizeBar := multi.AddBar(int64(len(scrobbles)), "Normalizing track titles")
    normalized, err := normalize.NormalizeScrobbles(scrobbles, normalizeBar)
    if err != nil {
        normalizeBar.FinishWithError(fmt.Sprintf("Normalization failed: %v", err))
        return err
    }
    normalizeBar.Finish("✓ Normalization complete")
    
    // Phase 3: Export
    exportBar := multi.AddBar(int64(len(normalized)), "Exporting to file")
    err = writer.Write(normalized, exportBar)
    if err != nil {
        exportBar.FinishWithError(fmt.Sprintf("Export failed: %v", err))
        return err
    }
    exportBar.Finish(fmt.Sprintf("✓ Exported to %s", cfg.OutputPath))
    
    // Wait for all bars to complete
    multi.Wait()
    
    fmt.Println("\n✓ Full sync completed successfully")
    return nil
}
```

### Contract Guarantees

| Requirement | Implementation | Verification |
|-------------|----------------|--------------|
| Sequential steps | Bars added in order, completed marked | Integration test: `TestMultiStepWorkflow` |
| Stack display | Completed bars show with checkmarks | Manual test: Visual verification |
| Error propagation | Errors stop workflow, mark failed step | Integration test: `TestWorkflowErrorHandling` |
| Clean completion | All bars cleaned up on success/failure | Integration test: `TestWorkflowCleanup` |

## 7. Testing Contracts

### Mock Implementation

```go
// internal/progress/mock.go
type MockProgressBar struct {
    StartCalled         bool
    StartTotal          int64
    StartDescription    string
    Adds                int
    SetCurrentCalled    bool
    SetCurrentValue     int64
    FinishCalled        bool
    FinishMessage       string
    FinishWithErrorCalled bool
    ErrorMessage        string
    IsFinishedValue     bool
}

func (m *MockProgressBar) Start(total int64, desc string) error {
    m.StartCalled = true
    m.StartTotal = total
    m.StartDescription = desc
    return nil
}

func (m *MockProgressBar) Add(n int) error {
    m.Adds += n
    return nil
}

// ... other methods ...
```

### Test Patterns

```go
// Example test using mock
func TestSyncServiceWithProgress(t *testing.T) {
    // Setup
    mock := progress.NewMockProgressBar()
    service := NewSyncService(cfg, client, logger)
    
    // Execute
    scrobbles, err := service.FetchRecentScrobbles(mock)
    
    // Verify
    assert.NoError(t, err)
    assert.True(t, mock.StartCalled)
    assert.Equal(t, 1000, mock.Adds) // Assuming 1000 scrobbles fetched
    assert.True(t, mock.FinishCalled)
}
```

### Contract Guarantees

| Requirement | Implementation | Verification |
|-------------|----------------|--------------|
| Testable operations | All services accept ProgressReporter | Unit tests: 100% coverage of progress paths |
| Mock completeness | Mock implements full ProgressReporter | Unit test: `TestMockImplementsInterface` |
| Assertion helpers | Mock tracks all method calls | Unit tests: All tests use mock |
| No test pollution | Tests don't output progress to terminal | CI: All tests run with NoOpProgressBar |

## 8. Performance Contracts

### CPU Usage

```go
// Benchmark contract
func BenchmarkProgressBarOverhead(b *testing.B) {
    bar := progress.NewRealProgressBar(int64(b.N), "Benchmark")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        bar.Add(1)
    }
    
    // Target: < 100ns per Add() call
}
```

### Memory Usage

```go
// Memory benchmark contract
func BenchmarkProgressBarMemory(b *testing.B) {
    var bars []*progress.ProgressBar
    
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        bars = append(bars, progress.NewRealProgressBar(1000, "Test"))
    }
    
    // Target: < 1024 bytes per bar
}
```

### Contract Guarantees

| Metric | Target | Enforcement |
|--------|--------|-------------|
| Add() latency | < 100ns | Benchmark in CI fails if exceeded |
| Memory per bar | < 1024 bytes | Benchmark tracks allocations |
| CPU overhead | < 1% of operation time | Integration benchmarks |
| Refresh rate impact | No dropped frames | Manual testing with 60 FPS |

## Implementation Checklist

### Phase 1: Core Integration
- [ ] Add `ProgressReporter` parameter to `SyncService.FetchRecentScrobbles()`
- [ ] Add `ProgressReporter` parameter to `normalize.NormalizeScrobbles()`
- [ ] Add `ProgressReporter` parameter to `Writer.Write()` interface
- [ ] Update all Writer implementations (LocalWriter, AzureWriter, MockWriter)
- [ ] Add `ProgressConfig` to `Config` struct
- [ ] Add progress config loading with defaults and env overrides

### Phase 2: Command Integration
- [ ] Update `fetch.go` command to create and pass ProgressReporter
- [ ] Create multi-operation workflow in `full-sync` command
- [ ] Add progress support to watermark operations
- [ ] Add progress support to Azure upload operations

### Phase 3: Testing Integration
- [ ] Create MockProgressBar implementation
- [ ] Update all unit tests to use MockProgressBar
- [ ] Add integration tests for progress + logging coexistence
- [ ] Add integration tests for multi-operation workflows
- [ ] Add benchmarks for performance contracts

### Phase 4: Configuration Integration
- [ ] Add progress config section to example config files
- [ ] Document all environment variables
- [ ] Add config validation tests
- [ ] Test backward compatibility with existing configs

## Breaking Changes

### Public API Changes

**NONE** - This is additive. All integration changes are backward compatible:

- New `ProgressReporter` parameters are **added to end** of function signatures
- Existing tests continue to work (will pass NoOpProgressBar)
- Configuration has sensible defaults (enabled by default)

### Migration Path

For external callers (if any):

```go
// Old (still works if we add default parameter)
scrobbles, err := service.FetchRecentScrobbles()

// New (recommended)
reporter := progress.NewProgressReporter(cfg, 0, "Fetching")
scrobbles, err := service.FetchRecentScrobbles(reporter)
```

If backward compatibility required, use functional options or separate methods:

```go
// Option 1: Separate method
func (s *SyncService) FetchRecentScrobbles() ([]models.Scrobble, error) {
    return s.FetchRecentScrobblesWithProgress(progress.NewNoOpProgressBar())
}

func (s *SyncService) FetchRecentScrobblesWithProgress(reporter progress.ProgressReporter) ([]models.Scrobble, error) {
    // ... implementation ...
}
```

## Validation Criteria

Integration complete when:

1. ✅ All services accept ProgressReporter parameter
2. ✅ All commands create appropriate progress reporters
3. ✅ All tests pass with MockProgressBar
4. ✅ Configuration loads and validates correctly
5. ✅ Performance benchmarks pass targets
6. ✅ Integration tests verify multi-operation workflows
7. ✅ Manual testing confirms visual appearance
8. ✅ Documentation updated with integration examples

## Next Steps

1. ✅ Contracts defined
2. → Create quickstart guide
3. → Update agent context
4. → Begin TDD implementation with progress package tests
