# Console Progress Bar - Developer Quickstart

**Feature**: Console Progress Bar (005)  
**Package**: `internal/progress`  
**Status**: Planning Complete - Ready for Implementation

## Overview

This guide helps developers quickly integrate console progress bars into SpecKit operations. The progress package provides thread-safe, terminal-aware progress tracking with automatic fallbacks for non-interactive environments.

## Quick Integration (5 minutes)

### 1. Import the package

```go
import "github.com/yourusername/lastfmreader/internal/progress"
```

### 2. Create a progress reporter

```go
// In your command or service function
reporter := progress.NewProgressReporter(cfg, total, "Operation description")
defer reporter.Finish("Complete")
```

### 3. Update progress in your loop

```go
for _, item := range items {
    // Do work
    processItem(item)
    
    // Update progress
    reporter.Add(1)
}
```

That's it! The reporter automatically handles:
- Terminal detection (disables in pipes/redirects)
- Unicode/ASCII fallback based on terminal capabilities
- Configuration from config file or environment variables
- Thread-safe concurrent updates
- Graceful error handling

## Common Patterns

### Pattern 1: Paginated API Fetch

```go
func (s *SyncService) FetchScrobbles(reporter progress.ProgressReporter) ([]Scrobble, error) {
    // Start with unknown total (will update as we go)
    reporter.Start(0, "Fetching scrobbles")
    defer reporter.Finish("Fetch complete")
    
    var allScrobbles []Scrobble
    page := 1
    
    for {
        scrobbles, err := s.client.GetPage(page)
        if err != nil {
            reporter.FinishWithError(fmt.Sprintf("Failed: %v", err))
            return nil, err
        }
        
        if len(scrobbles) == 0 {
            break // No more pages
        }
        
        allScrobbles = append(allScrobbles, scrobbles...)
        reporter.Add(len(scrobbles))
        page++
    }
    
    return allScrobbles, nil
}
```

### Pattern 2: Known Item Count

```go
func NormalizeScrobbles(items []Scrobble, reporter progress.ProgressReporter) error {
    reporter.Start(int64(len(items)), "Normalizing titles")
    defer reporter.Finish("Normalization complete")
    
    for i, item := range items {
        item.NormalizedTitle = normalize(item.Title)
        reporter.Add(1)
        
        // Optional: Update description dynamically
        if i%100 == 0 {
            reporter.SetDescription(fmt.Sprintf("Normalizing titles (%d/%d)", i, len(items)))
        }
    }
    
    return nil
}
```

### Pattern 3: Rate Limiting with Spinner

```go
func FetchWithRateLimit(reporter progress.ProgressReporter) error {
    reporter.Start(totalPages, "Fetching data")
    defer reporter.Finish("Fetch complete")
    
    for page := 1; page <= totalPages; page++ {
        // Check for rate limit
        if rateLimited() {
            reporter.SwitchToSpinner()
            reporter.SetDescription("Rate limited - waiting 60s...")
            time.Sleep(60 * time.Second)
            reporter.ResumeProgress()
            reporter.SetDescription("Fetching data")
        }
        
        fetchPage(page)
        reporter.Add(1)
    }
    
    return nil
}
```

### Pattern 4: Multi-Step Workflow

```go
func FullSync(cfg *config.Config) error {
    multi := progress.NewMulti()
    
    // Step 1: Fetch
    fetchBar := multi.AddBar(0, "1/3 Fetching scrobbles")
    scrobbles, err := fetchScrobbles(fetchBar)
    if err != nil {
        fetchBar.FinishWithError("Fetch failed")
        return err
    }
    fetchBar.Finish(fmt.Sprintf("✓ Fetched %d scrobbles", len(scrobbles)))
    
    // Step 2: Normalize
    normalizeBar := multi.AddBar(int64(len(scrobbles)), "2/3 Normalizing titles")
    if err := normalize(scrobbles, normalizeBar); err != nil {
        normalizeBar.FinishWithError("Normalization failed")
        return err
    }
    normalizeBar.Finish("✓ Normalization complete")
    
    // Step 3: Export
    exportBar := multi.AddBar(int64(len(scrobbles)), "3/3 Exporting")
    if err := export(scrobbles, exportBar); err != nil {
        exportBar.FinishWithError("Export failed")
        return err
    }
    exportBar.Finish("✓ Export complete")
    
    multi.Wait()
    return nil
}
```

### Pattern 5: File Processing

```go
func ExportToFile(scrobbles []Scrobble, reporter progress.ProgressReporter) error {
    reporter.Start(int64(len(scrobbles)), "Exporting to file")
    defer reporter.Finish("Export complete")
    
    file, err := os.Create(outputPath)
    if err != nil {
        reporter.FinishWithError(fmt.Sprintf("File error: %v", err))
        return err
    }
    defer file.Close()
    
    encoder := json.NewEncoder(file)
    
    // Batch updates for performance
    batchSize := 100
    for i, scrobble := range scrobbles {
        if err := encoder.Encode(scrobble); err != nil {
            reporter.FinishWithError(fmt.Sprintf("Encode error: %v", err))
            return err
        }
        
        // Update every batch
        if (i+1)%batchSize == 0 {
            reporter.Add(batchSize)
        }
    }
    
    // Final update for remainder
    remainder := len(scrobbles) % batchSize
    if remainder > 0 {
        reporter.Add(remainder)
    }
    
    return nil
}
```

## Configuration

### Config File (`config.yaml`)

```yaml
progress:
  enabled: true              # Enable/disable all progress bars
  style: blocks              # blocks|arrows|dots|ascii
  show_speed: true           # Show items/sec
  show_eta: true             # Show estimated time remaining
  show_count: true           # Show current/total count
  show_percentage: true      # Show percentage complete
  show_elapsed: false        # Show elapsed time
  width: 0                   # Bar width (0=auto)
  refresh_rate: 100ms        # Update frequency
  colors: true               # Enable ANSI colors
  auto_clear: true           # Clear bar on completion
```

### Environment Variables

Override config with environment variables:

```bash
# Disable all progress bars
export SPECKIT_NO_PROGRESS=true

# Force ASCII mode (for older terminals)
export SPECKIT_PROGRESS_ASCII=true

# Disable colors
export SPECKIT_NO_COLOR=true

# Set refresh rate (milliseconds)
export SPECKIT_PROGRESS_REFRESH=50

# Set bar width
export SPECKIT_PROGRESS_WIDTH=80
```

## Testing

### Unit Tests with Mock

```go
func TestFetchWithProgress(t *testing.T) {
    // Create mock progress reporter
    mock := progress.NewMockProgressBar()
    
    // Run operation
    scrobbles, err := syncService.FetchScrobbles(mock)
    
    // Verify progress updates
    assert.NoError(t, err)
    assert.True(t, mock.StartCalled)
    assert.Equal(t, len(scrobbles), mock.Adds)
    assert.True(t, mock.FinishCalled)
    assert.Contains(t, mock.FinishMessage, "complete")
}
```

### Integration Tests

```go
func TestProgressInPipeline(t *testing.T) {
    // Progress should auto-disable when output is redirected
    cfg := testConfig()
    
    // Simulate non-interactive terminal
    reporter := progress.NewProgressReporter(cfg, 100, "Test")
    
    // Should be NoOp implementation
    _, ok := reporter.(*progress.NoOpProgressBar)
    assert.True(t, ok, "Expected NoOp in non-interactive mode")
}
```

## Visual Styles

Progress bars automatically adapt to terminal capabilities:

### Blocks Style (Default)
```
Fetching scrobbles [████████████████░░░░] 80% | 800/1000 | 120/s | ETA: 2s
```

### Arrows Style
```
Normalizing titles [===============>     ] 75% | 750/1000 | 200/s | ETA: 1s
```

### Dots Style
```
Exporting files    [●●●●●●●●●●●●●●●○○○○○] 70% | 700/1000 | 50/s | ETA: 5s
```

### ASCII Style (Automatic fallback)
```
Processing items   [##########          ] 50% | 500/1000 | 100/s | ETA: 5s
```

### Spinner Mode (Rate Limiting)
```
⠋ Rate limited - waiting 60s...
```

## Common Mistakes

### ❌ Don't forget to call Finish()

```go
// BAD: Progress bar left hanging
func processItems(reporter progress.ProgressReporter) {
    reporter.Start(100, "Processing")
    for i := 0; i < 100; i++ {
        process(i)
        reporter.Add(1)
    }
    // Forgot to call Finish()!
}

// GOOD: Always finish
func processItems(reporter progress.ProgressReporter) {
    reporter.Start(100, "Processing")
    defer reporter.Finish("Complete") // ✓ Always finishes
    
    for i := 0; i < 100; i++ {
        process(i)
        reporter.Add(1)
    }
}
```

### ❌ Don't update too frequently

```go
// BAD: Updates every item (performance impact)
for i := 0; i < 1000000; i++ {
    process(i)
    reporter.Add(1) // 1 million updates!
}

// GOOD: Batch updates
batchSize := 100
for i := 0; i < 1000000; i++ {
    process(i)
    if i%batchSize == 0 {
        reporter.Add(batchSize) // Only 10,000 updates
    }
}
```

### ❌ Don't ignore errors from progress operations

```go
// BAD: Ignoring errors
reporter.Start(100, "Processing")
reporter.Add(1)

// GOOD: Handle errors (even if just logging)
if err := reporter.Start(100, "Processing"); err != nil {
    logger.Warn("Failed to start progress", "error", err)
}
if err := reporter.Add(1); err != nil {
    logger.Debug("Progress update failed", "error", err)
}
```

### ❌ Don't create progress reporters in tight loops

```go
// BAD: Creating new reporter each iteration
for _, item := range items {
    reporter := progress.NewProgressReporter(cfg, 1, "Processing")
    processItem(item, reporter)
    reporter.Finish("Done")
}

// GOOD: Create once, reuse
reporter := progress.NewProgressReporter(cfg, int64(len(items)), "Processing")
defer reporter.Finish("Complete")

for _, item := range items {
    processItem(item)
    reporter.Add(1)
}
```

## Performance Tips

1. **Batch updates** for large datasets (update every 100-1000 items)
2. **Use appropriate refresh rate** (100ms is good default)
3. **Disable in CI/CD** (use `SPECKIT_NO_PROGRESS=true`)
4. **Test with mock** (avoid terminal output in unit tests)
5. **Profile if needed** (progress overhead should be < 1%)

## Troubleshooting

### Progress bar not showing?

Check these common issues:

1. **Output redirected?** Progress auto-disables in pipes/redirects
   ```bash
   # Won't show progress (output redirected)
   ./speckit sync > output.txt
   
   # Will show progress
   ./speckit sync
   ```

2. **Progress disabled in config?**
   ```yaml
   progress:
     enabled: false  # ← Check this
   ```

3. **Environment variable set?**
   ```bash
   # Check if disabled
   echo $SPECKIT_NO_PROGRESS
   ```

4. **Terminal not interactive?**
   ```go
   // Debug terminal detection
   info, _ := progress.DetectTerminal()
   fmt.Printf("Interactive: %v\n", info.IsInteractive)
   ```

### Progress bar looks broken?

1. **Try ASCII mode** for older terminals:
   ```bash
   export SPECKIT_PROGRESS_ASCII=true
   ```

2. **Disable colors** if ANSI codes cause issues:
   ```bash
   export SPECKIT_NO_COLOR=true
   ```

3. **Check terminal size**:
   ```bash
   echo $COLUMNS  # Should be > 40
   ```

### Progress updates slow?

1. **Reduce refresh rate** if CPU usage too high:
   ```bash
   export SPECKIT_PROGRESS_REFRESH=200  # 200ms instead of 100ms
   ```

2. **Batch updates** more aggressively (every 1000 items)

3. **Check for mutex contention** if many goroutines updating

## Next Steps

1. **Read the full API contract**: [contracts/progress-api.md](contracts/progress-api.md)
2. **Review integration patterns**: [contracts/integration-contracts.md](contracts/integration-contracts.md)
3. **Check implementation plan**: [plan.md](plan.md)
4. **Start TDD implementation**: Write tests first for each component

## Quick Reference Card

```go
// Basic usage
reporter := progress.NewProgressReporter(cfg, total, "Description")
defer reporter.Finish("Complete")
for _, item := range items {
    processItem(item)
    reporter.Add(1)
}

// Error handling
if err := operation(); err != nil {
    reporter.FinishWithError(fmt.Sprintf("Failed: %v", err))
    return err
}

// Rate limiting
if rateLimited() {
    reporter.SwitchToSpinner()
    reporter.SetDescription("Waiting...")
    time.Sleep(waitTime)
    reporter.ResumeProgress()
}

// Multi-step
multi := progress.NewMulti()
bar1 := multi.AddBar(n1, "Step 1")
// ... use bar1 ...
bar1.Finish("✓ Done")
bar2 := multi.AddBar(n2, "Step 2")
// ... use bar2 ...
multi.Wait()

// Testing
mock := progress.NewMockProgressBar()
operation(mock)
assert.True(t, mock.StartCalled)
assert.Equal(t, expectedAdds, mock.Adds)
```

## Support

- **Feature Spec**: [spec.md](spec.md)
- **Implementation Plan**: [plan.md](plan.md)
- **API Contracts**: [contracts/](contracts/)
- **GitHub Issues**: File bugs/feature requests

---

**Ready to implement?** Start with TDD: write tests first, then implement the `internal/progress` package!
