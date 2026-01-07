# API Contract: Progress Bar Package

**Package**: `internal/progress`  
**Date**: 2026-01-07  
**Language**: Go

## Public API

### ProgressReporter Interface

The primary interface for all progress reporting operations.

```go
package progress

// ProgressReporter provides progress update capability for long-running operations
type ProgressReporter interface {
    // Start initializes progress tracking with total items and description
    // Returns error if already started
    Start(total int64, description string) error
    
    // Add increments progress by n items
    // Returns error if bar is finished or value invalid
    Add(n int) error
    
    // SetCurrent sets progress to specific value
    // Returns error if value > total or bar is finished
    SetCurrent(n int64) error
    
    // SetDescription updates the operation description
    SetDescription(desc string) error
    
    // Finish marks operation as successfully complete with optional message
    Finish(message string) error
    
    // FinishWithError marks operation as failed with error message
    FinishWithError(message string) error
    
    // FinishWithWarning marks operation as complete with warnings
    FinishWithWarning(message string) error
    
    // SwitchToSpinner converts to indeterminate spinner mode (e.g., for rate limiting)
    SwitchToSpinner() error
    
    // ResumeProgress resumes normal progress bar from spinner mode
    ResumeProgress() error
    
    // IsFinished returns whether progress tracking is complete
    IsFinished() bool
}
```

### Factory Functions

```go
// NewProgressReporter creates appropriate reporter based on configuration
// Returns NoOpProgressBar if progress disabled or terminal non-interactive
func NewProgressReporter(cfg *config.Config, total int64, desc string) ProgressReporter

// NewRealProgressBar creates a full-featured progress bar
// Options configured via functional options pattern
func NewRealProgressBar(total int64, desc string, opts ...Option) *ProgressBar

// NewNoOpProgressBar creates a silent no-op implementation
func NewNoOpProgressBar() ProgressReporter

// NewMockProgressBar creates a test double for unit testing
func NewMockProgressBar() *MockProgressBar
```

### Options (Functional Options Pattern)

```go
// Option configures ProgressBar behavior
type Option func(*Options)

// Display options
func WithWidth(w int) Option
func WithStyle(s Style) Option
func WithColors(enabled bool) Option
func WithWriter(w io.Writer) Option

// Information display toggles
func WithSpeed(show bool) Option
func WithETA(show bool) Option
func WithElapsed(show bool) Option
func WithPercentage(show bool) Option
func WithCount(show bool) Option

// Behavior options
func WithRefreshRate(rate time.Duration) Option
func WithAutoClear(enabled bool) Option
```

### Predefined Styles

```go
// Predefined visual styles
var (
    StyleBlocks Style // Unicode blocks (default): [████░░] 
    StyleArrows Style // Arrow progression: [====>  ]
    StyleDots   Style // Circle progression: [●●●○○]
    StyleASCII  Style // ASCII fallback: [####   ]
)
```

### MultiProgress API

```go
// MultiProgress manages multiple concurrent progress bars
type MultiProgress struct {
    // private fields
}

// NewMulti creates a multi-bar container
func NewMulti(opts ...Option) *MultiProgress

// AddBar adds a new progress bar to the display
func (m *MultiProgress) AddBar(total int64, desc string) *ProgressBar

// RemoveBar removes a completed bar from display
func (m *MultiProgress) RemoveBar(bar *ProgressBar)

// Wait blocks until all bars complete
func (m *MultiProgress) Wait()

// Clear removes all progress displays from terminal
func (m *MultiProgress) Clear()
```

### Terminal Detection

```go
// DetectTerminal queries terminal capabilities
func DetectTerminal() (*TerminalInfo, error)

// TerminalInfo holds detected terminal capabilities
type TerminalInfo struct {
    Width         int  // Terminal width in characters
    Height        int  // Terminal height in lines
    SupportsUTF8  bool // Can display Unicode
    SupportsColor bool // Can display ANSI colors
    IsInteractive bool // Is a TTY (not redirected)
}

// BestStyle returns recommended style for this terminal
func (t *TerminalInfo) BestStyle() Style

// ShouldDisplay returns whether progress bars should be shown
func (t *TerminalInfo) ShouldDisplay() bool
```

## Usage Patterns

### Basic Progress Bar

```go
import "internal/progress"

func ProcessItems(cfg *config.Config, items []Item) error {
    // Create progress reporter
    reporter := progress.NewProgressReporter(cfg, int64(len(items)), "Processing items")
    defer reporter.Finish("Processing complete")
    
    // Process with progress updates
    for _, item := range items {
        if err := process(item); err != nil {
            reporter.FinishWithError(fmt.Sprintf("Failed: %v", err))
            return err
        }
        reporter.Add(1)
    }
    
    return nil
}
```

### Progress with Rate Limiting

```go
func FetchWithRateLimit(reporter progress.ProgressReporter) error {
    for page := 1; page <= totalPages; page++ {
        // Check for rate limit
        if isRateLimited() {
            reporter.SwitchToSpinner()
            reporter.SetDescription("Rate limited - waiting...")
            time.Sleep(waitDuration)
            reporter.ResumeProgress()
            reporter.SetDescription("Fetching data")
        }
        
        // Fetch data
        data := fetchPage(page)
        reporter.Add(len(data))
    }
    return nil
}
```

### Multi-Operation Workflow

```go
func MultiStepWorkflow(cfg *config.Config) error {
    multi := progress.NewMulti()
    
    // Step 1: Fetch
    bar1 := multi.AddBar(1000, "Fetching data")
    if err := fetchData(bar1); err != nil {
        bar1.FinishWithError("Fetch failed")
        return err
    }
    bar1.Finish("Fetch complete")
    
    // Step 2: Normalize
    bar2 := multi.AddBar(800, "Normalizing titles")
    if err := normalize(bar2); err != nil {
        bar2.FinishWithError("Normalization failed")
        return err
    }
    bar2.Finish("Normalization complete")
    
    // Step 3: Export
    bar3 := multi.AddBar(800, "Exporting results")
    if err := export(bar3); err != nil {
        bar3.FinishWithError("Export failed")
        return err
    }
    bar3.Finish("Export complete")
    
    multi.Wait()
    return nil
}
```

### Custom Configuration

```go
func CustomProgressBar() progress.ProgressReporter {
    return progress.NewRealProgressBar(
        100,
        "Custom operation",
        progress.WithStyle(progress.StyleArrows),
        progress.WithColors(false),
        progress.WithWidth(80),
        progress.WithRefreshRate(50*time.Millisecond),
        progress.WithETA(true),
        progress.WithSpeed(true),
    )
}
```

### Testing with Mock

```go
func TestProcessing(t *testing.T) {
    mock := progress.NewMockProgressBar()
    
    // Run operation with mock
    err := ProcessItems(mockConfig, items, mock)
    
    // Verify progress updates
    assert.Equal(t, len(items), mock.Adds)
    assert.True(t, mock.FinishCalled)
}
```

## Configuration Contract

### Config File Schema

```yaml
progress:
  enabled: boolean        # Enable/disable progress bars (default: true)
  style: string          # Visual style: blocks|arrows|dots|ascii (default: blocks)
  show_speed: boolean    # Display items/second (default: true)
  show_eta: boolean      # Display estimated time (default: true)
  show_count: boolean    # Display current/total (default: true)
  show_percentage: boolean # Display percentage (default: true)
  show_elapsed: boolean  # Display elapsed time (default: false)
  width: integer         # Bar width in chars, 0=auto (default: 0)
  refresh_rate: duration # Update frequency (default: 100ms)
  colors: boolean        # Enable ANSI colors (default: true)
  auto_clear: boolean    # Clear on completion (default: true)
```

### Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `SPECKIT_NO_PROGRESS` | bool | false | Disable all progress bars |
| `SPECKIT_PROGRESS_ASCII` | bool | false | Force ASCII style |
| `SPECKIT_NO_COLOR` | bool | false | Disable colors |
| `SPECKIT_PROGRESS_REFRESH` | int | 100 | Refresh rate in milliseconds |
| `SPECKIT_PROGRESS_WIDTH` | int | 0 | Bar width (0=auto) |

## Error Handling Contract

### Error Types

```go
var (
    // ErrBarFinished returned when updating a finished bar
    ErrBarFinished = errors.New("progress bar already finished")
    
    // ErrNegativeValue returned when progress value is negative
    ErrNegativeValue = errors.New("progress value cannot be negative")
    
    // ErrExceedsTotal returned when progress exceeds total
    ErrExceedsTotal = errors.New("progress exceeds total")
    
    // ErrTerminalWrite returned when terminal write fails
    ErrTerminalWrite = errors.New("failed to write to terminal")
    
    // ErrInvalidState returned when operation invalid for current state
    ErrInvalidState = errors.New("invalid operation for current state")
)
```

### Error Behavior

- Progress bar errors are **non-blocking** - operations log but continue
- Errors are returned for API contract compliance but don't halt operations
- Failed terminal writes degrade to no-op silently
- Configuration errors default to safe fallback values

## Thread Safety Contract

All public methods on `ProgressReporter` implementations are **thread-safe**:

```go
// Safe concurrent usage
var reporter = progress.NewProgressReporter(cfg, 1000, "Processing")

// Multiple goroutines can safely update
for i := 0; i < 10; i++ {
    go func() {
        for j := 0; j < 100; j++ {
            reporter.Add(1) // Thread-safe
        }
    }()
}
```

## Performance Contract

Guaranteed performance characteristics:

| Metric | Target | Enforcement |
|--------|--------|-------------|
| CPU Overhead | < 1% | Benchmarks in CI/CD |
| Memory Usage | < 1MB | Unit tests verify |
| Update Rate | 10-60 FPS | Throttled automatically |
| Initial Display | < 100ms | Integration tests |
| Resize Response | < 200ms | Integration tests |

## Compatibility Contract

### Platform Support

- **Linux**: Full support (all features)
- **macOS**: Full support (all features)
- **Windows**: Full support (Unicode in Windows Terminal, ASCII fallback in cmd.exe)

### Terminal Support

- **Modern terminals** (UTF-8, ANSI): StyleBlocks with colors
- **Limited terminals** (ASCII only): StyleASCII without colors
- **Non-interactive** (pipes, redirects): NoOp (silent)

### Go Version

- **Minimum**: Go 1.24.0
- **Dependencies**: Compatible with Go modules

## Breaking Changes Policy

Public API changes follow semantic versioning:

- **Major**: Breaking interface changes
- **Minor**: New methods or options (backward compatible)
- **Patch**: Bug fixes, internal changes

Current API version: `v1.0.0`

## Migration Guide

N/A - This is a new feature. No migration required.

## Deprecation Policy

If methods are deprecated:
1. Mark with `// Deprecated:` comment
2. Provide alternative in deprecation notice
3. Maintain for at least 2 minor versions
4. Remove in next major version

## Next Steps

1. ✅ API contracts defined
2. → Create quickstart guide
3. → Update agent context
4. → Begin implementation with TDD
