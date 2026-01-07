# Data Model: Console Progress Bar

**Feature**: Console Progress Bar  
**Date**: 2026-01-07  
**Phase**: 1 - Design & Contracts

## Core Entities

### ProgressBar

Represents a single progress bar instance tracking operation progress.

**Fields**:
```go
type ProgressBar struct {
    current      int64              // Current progress value
    total        int64              // Total items to process
    description  string             // Operation description
    startTime    time.Time          // When progress started
    style        Style              // Visual style configuration
    options      Options            // Display options
    bar          *progressbar.ProgressBar  // Underlying library instance
    mu           sync.Mutex         // Thread safety
    finished     bool               // Whether bar is complete
    state        ProgressState      // Current state (active/success/error/warning)
}
```

**States**:
- `StateActive`: Progress is actively updating
- `StateSuccess`: Operation completed successfully
- `StateError`: Operation failed
- `StateWarning`: Operation completed with warnings
- `StateSpinner`: Indeterminate progress (unknown total)

**Methods**:
- `New(total int64, desc string, opts ...Option) *ProgressBar`: Create new progress bar
- `Add(n int) error`: Increment progress by n
- `SetCurrent(n int64) error`: Set progress to specific value
- `SetDescription(desc string)`: Update operation description
- `Finish(message string)`: Mark as successfully complete
- `FinishWithError(message string)`: Mark as failed
- `FinishWithWarning(message string)`: Mark as complete with warning
- `SwitchToSpinner()`: Convert to indeterminate spinner mode
- `ResumeProgress()`: Resume from spinner mode to progress bar
- `IsFinished() bool`: Check if bar is complete

**Validation Rules**:
- `total` must be > 0 for progress mode, 0 for spinner mode
- `current` must be >= 0 and <= total
- `description` must be non-empty
- Cannot modify finished bar (operations return error)

**Lifecycle**:
```
Created → Active → (Spinner?) → Finished
```

### Options

Configuration settings for progress bar display.

**Fields**:
```go
type Options struct {
    Width         int           // Terminal width for bar (0 = auto-detect)
    ShowSpeed     bool          // Display items/second
    ShowETA       bool          // Display estimated time remaining
    ShowElapsed   bool          // Display elapsed time
    ShowPercent   bool          // Display percentage
    ShowCount     bool          // Display current/total
    RefreshRate   time.Duration // Update frequency (default 100ms)
    Style         Style         // Visual style
    EnableColors  bool          // Use ANSI colors
    AutoClear     bool          // Clear on completion
    Writer        io.Writer     // Output destination (default os.Stdout)
}
```

**Defaults**:
```go
var DefaultOptions = Options{
    Width:        0,     // Auto-detect
    ShowSpeed:    true,
    ShowETA:      true,
    ShowElapsed:  false,
    ShowPercent:  true,
    ShowCount:    true,
    RefreshRate:  100 * time.Millisecond,
    Style:        StyleBlocks,
    EnableColors: true,
    AutoClear:    true,
    Writer:       os.Stdout,
}
```

**Builder Pattern**:
```go
// Functional options pattern
func WithWidth(w int) Option { return func(o *Options) { o.Width = w } }
func WithStyle(s Style) Option { return func(o *Options) { o.Style = s } }
// ... etc
```

### Style

Visual style definition for progress bar appearance.

**Fields**:
```go
type Style struct {
    BarStart    string   // Start character (e.g., "[")
    BarEnd      string   // End character (e.g., "]")
    Complete    string   // Completed segment (e.g., "█")
    InProgress  string   // In-progress segment (e.g., "▓")
    Incomplete  string   // Incomplete segment (e.g., "░")
    SpinnerFrames []string // Animation frames for spinner
}
```

**Predefined Styles**:
```go
var StyleBlocks = Style{
    BarStart:    "[",
    BarEnd:      "]",
    Complete:    "█",
    InProgress:  "▓",
    Incomplete:  "░",
    SpinnerFrames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
}

var StyleArrows = Style{
    BarStart:    "[",
    BarEnd:      "]",
    Complete:    "=",
    InProgress:  ">",
    Incomplete:  " ",
    SpinnerFrames: []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
}

var StyleDots = Style{
    BarStart:    "[",
    BarEnd:      "]",
    Complete:    "●",
    InProgress:  "◐",
    Incomplete:  "○",
    SpinnerFrames: []string{"◐", "◓", "◑", "◒"},
}

var StyleASCII = Style{
    BarStart:    "[",
    BarEnd:      "]",
    Complete:    "#",
    InProgress:  "-",
    Incomplete:  " ",
    SpinnerFrames: []string{"|", "/", "-", "\\"},
}
```

### TerminalInfo

Terminal capability information.

**Fields**:
```go
type TerminalInfo struct {
    Width         int  // Terminal width in characters
    Height        int  // Terminal height in lines
    SupportsUTF8  bool // Can display Unicode
    SupportsColor bool // Can display ANSI colors
    IsInteractive bool // Is a TTY (not redirected)
}
```

**Detection Methods**:
```go
func DetectTerminal() (*TerminalInfo, error)
func (t *TerminalInfo) BestStyle() Style  // Returns appropriate style
func (t *TerminalInfo) ShouldDisplay() bool // Whether to show progress
```

**Detection Logic**:
- `IsInteractive`: Use `term.IsTerminal(os.Stdout.Fd())`
- `Width/Height`: Use `term.GetSize(os.Stdout.Fd())`
- `SupportsUTF8`: Check `LANG` or `LC_ALL` environment variables
- `SupportsColor`: Check `TERM` environment variable and `NO_COLOR`

### MultiProgress

Container for managing multiple concurrent progress bars.

**Fields**:
```go
type MultiProgress struct {
    bars     []*ProgressBar  // All progress bars
    active   []*ProgressBar  // Currently updating bars
    complete []*ProgressBar  // Finished bars (shown with checkmarks)
    mu       sync.RWMutex    // Thread safety
    writer   io.Writer       // Output destination
    options  Options         // Shared options
}
```

**Methods**:
- `NewMulti(opts ...Option) *MultiProgress`: Create multi-bar container
- `AddBar(total int64, desc string) *ProgressBar`: Add new progress bar
- `RemoveBar(bar *ProgressBar)`: Remove completed bar from display
- `Wait()`: Wait for all bars to complete
- `Render()`: Manually trigger re-render (called automatically)
- `Clear()`: Clear all progress displays

**Rendering Strategy**:
```
[✓] Fetch Last.fm data     1000/1000  (45/s)  [Complete]
[✓] Normalize titles        800/800   (120/s) [Complete]
[████████████░░░░░] 80% (80/100) 45/s ETA 0:00:05  ← Active bar
```

**State Tracking**:
- Completed bars move to `complete` slice with checkmark
- Active bars remain in `active` slice with progress
- Failed bars shown in `complete` with ✗ marker

### ProgressReporter (Interface)

Abstract interface for progress reporting.

**Interface**:
```go
type ProgressReporter interface {
    Start(total int64, description string) error
    Add(n int) error
    SetCurrent(n int64) error
    SetDescription(desc string) error
    Finish(message string) error
    FinishWithError(message string) error
    FinishWithWarning(message string) error
    SwitchToSpinner() error
    ResumeProgress() error
    IsFinished() bool
}
```

**Implementations**:
- `RealProgressBar`: Full progress bar implementation
- `NoOpProgressBar`: Silent implementation (progress disabled)
- `MockProgressBar`: Test double for unit testing

**Factory**:
```go
func NewProgressReporter(cfg *config.Config, total int64, desc string) ProgressReporter {
    if !isProgressEnabled(cfg) {
        return NewNoOpProgressBar()
    }
    
    info, _ := DetectTerminal()
    if !info.ShouldDisplay() {
        return NewNoOpProgressBar()
    }
    
    return NewRealProgressBar(total, desc, buildOptions(cfg, info)...)
}
```

## Relationships

```
MultiProgress [1] ──────> [*] ProgressBar
ProgressBar [1] ──────> [1] Options
ProgressBar [1] ──────> [1] Style
Options [1] ──────> [1] Style
TerminalInfo [1] ──────> [1] Style (via BestStyle())
```

## Configuration Integration

### Config File Structure

```yaml
progress:
  enabled: true
  style: blocks              # blocks, arrows, dots, ascii
  show_speed: true
  show_eta: true
  show_count: true
  show_percentage: true
  show_elapsed: false
  width: 0                   # 0 = auto-detect
  refresh_rate: 100ms
  colors: true
  auto_clear: true
```

### Environment Variables

| Variable | Type | Purpose | Example |
|----------|------|---------|---------|
| `SPECKIT_NO_PROGRESS` | bool | Disable all progress bars | `true` |
| `SPECKIT_PROGRESS_ASCII` | bool | Force ASCII mode | `true` |
| `SPECKIT_NO_COLOR` | bool | Disable colors | `true` |
| `SPECKIT_PROGRESS_REFRESH` | int | Refresh rate (ms) | `100` |
| `SPECKIT_PROGRESS_WIDTH` | int | Bar width | `50` |

### Precedence Order

1. Environment variables (highest priority)
2. Config file settings
3. Default values (lowest priority)

## State Transitions

### Single Progress Bar

```
┌─────────┐
│ Created │
└────┬────┘
     │
     ▼
┌─────────┐    SwitchToSpinner()    ┌─────────┐
│ Active  │ ──────────────────────> │ Spinner │
└────┬────┘                         └────┬────┘
     │                                   │
     │         ResumeProgress()          │
     │ <─────────────────────────────────┘
     │
     ▼
┌──────────┐
│ Finished │
│ (Success/│
│ Error/   │
│ Warning) │
└──────────┘
```

### Multi-Bar Sequence

```
Operation 1: Active → Complete (✓) → Moved to completed list
Operation 2:         Active → Complete (✓) → Moved to completed list
Operation 3:                 Active → ...
```

## Memory Estimation

| Component | Size | Count | Total |
|-----------|------|-------|-------|
| ProgressBar struct | ~200 bytes | 1-5 typical | 1KB |
| Style strings | ~100 bytes | 4 styles | 400 bytes |
| Options | ~100 bytes | 1 | 100 bytes |
| Library overhead | ~500KB | 1 | 500KB |
| **Total** | | | **~501KB** |

Well under 1MB constraint. ✅

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Notes |
|-----------|-----------|--------|
| Add(n) | O(1) | Update counter |
| Render() | O(w) | w = terminal width |
| MultiRender() | O(n·w) | n = number of bars |

### Update Rate

- **Throttled**: 100ms default (10 FPS)
- **Maximum**: Configurable up to 60 FPS
- **CPU Impact**: < 0.1% per bar at 10 FPS

## Thread Safety

All public methods on `ProgressBar` and `MultiProgress` are thread-safe via mutex protection:

```go
func (b *ProgressBar) Add(n int) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    // ... update logic
}
```

## Error Handling

Progress bar operations return errors but never panic:

```go
// Error conditions
var (
    ErrBarFinished   = errors.New("progress bar already finished")
    ErrNegativeValue = errors.New("progress value cannot be negative")
    ErrExceedsTotal  = errors.New("progress exceeds total")
    ErrTerminalWrite = errors.New("failed to write to terminal")
)
```

Operations log errors but don't fail the underlying task:

```go
if err := bar.Add(1); err != nil {
    log.Debug("Progress update failed: %v", err)
    // Continue processing despite progress error
}
```

## Integration Points

### Last.fm Sync Service

```go
func (s *SyncService) FetchScrobbles(ctx context.Context, user string) error {
    total := s.estimateTotal(user) // Estimate from API
    reporter := progress.NewProgressReporter(s.cfg, total, "Fetching scrobbles")
    defer reporter.Finish("Fetch complete")
    
    for page := range s.fetchPages(ctx, user) {
        // Process scrobbles
        reporter.Add(len(page.Scrobbles))
    }
    return nil
}
```

### Title Normalization

```go
func (n *Normalizer) NormalizeBatch(tracks []Track) error {
    reporter := progress.NewProgressReporter(n.cfg, int64(len(tracks)), "Normalizing titles")
    defer reporter.Finish("Normalization complete")
    
    for _, track := range tracks {
        track.NormalizedTitle = n.Normalize(track.Title)
        reporter.Add(1)
    }
    return nil
}
```

### API Rate Limiting

```go
func (c *Client) waitForRateLimit(reporter progress.ProgressReporter, waitTime time.Duration) {
    reporter.SwitchToSpinner()
    reporter.SetDescription(fmt.Sprintf("Rate limited - waiting %v", waitTime))
    
    time.Sleep(waitTime)
    
    reporter.ResumeProgress()
    reporter.SetDescription("Resuming fetch")
}
```

## Testing Data

### Test Fixtures

```go
var testCases = []struct {
    name     string
    total    int64
    updates  []int
    expected float64 // Expected percentage
}{
    {"empty", 100, []int{}, 0.0},
    {"half", 100, []int{50}, 50.0},
    {"complete", 100, []int{100}, 100.0},
    {"overfull", 100, []int{150}, 100.0}, // Clamped
}
```

### Mock Terminal

```go
type MockTerminal struct {
    width         int
    supportsColor bool
    supportsUTF8  bool
    isInteractive bool
}

func (m *MockTerminal) GetInfo() *TerminalInfo {
    return &TerminalInfo{
        Width:         m.width,
        SupportsColor: m.supportsColor,
        SupportsUTF8:  m.supportsUTF8,
        IsInteractive: m.isInteractive,
    }
}
```

## Next Steps

1. ✅ Data model complete
2. → Proceed to contract definitions
3. → Create quickstart guide
4. → Update agent context
