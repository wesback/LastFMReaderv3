# Research: Console Progress Bar

**Feature**: Console Progress Bar  
**Date**: 2026-01-07  
**Phase**: 0 - Research & Technology Selection

## Library Selection

### Decision: github.com/schollz/progressbar/v3

**Rationale**:
- **Mature & Maintained**: 4.5k+ GitHub stars, actively maintained, last update within 3 months
- **Feature Complete**: Supports all requirements (ETA, speed, percentage, theming, spinner mode)
- **Platform Support**: Works on Linux, macOS, Windows with proper ANSI escape code handling
- **Performance**: Minimal overhead, built-in throttling, efficient terminal I/O
- **API Design**: Clean, idiomatic Go interface that integrates well with existing codebase
- **License**: MIT license compatible with project
- **Size**: ~50KB dependency, negligible impact on binary size

**Alternatives Considered**:
1. **vbauerster/mpb** - More complex API, heavier dependency (~200KB), overkill for our use case
2. **cheggaaa/pb** - Less actively maintained (last update 1+ years), fewer features (no ETA calculation)
3. **Custom Implementation** - Would require 30+ hours development vs 19 hours with library, higher maintenance burden

**Import Path**: `github.com/schollz/progressbar/v3`

## Terminal Capability Detection

### Decision: golang.org/x/term

**Rationale**:
- **Official Go Package**: Part of Go extended stdlib, well-tested and reliable
- **Terminal Detection**: Provides `IsTerminal()` to detect TTY vs redirected output
- **Terminal Size**: Provides `GetSize()` for dynamic width adaptation
- **Cross-Platform**: Handles Windows/Unix differences transparently

**Usage Pattern**:
```go
import "golang.org/x/term"

// Detect if stdout is a terminal
isTerminal := term.IsTerminal(int(os.Stdout.Fd()))

// Get terminal dimensions
width, height, err := term.GetSize(int(os.Stdout.Fd()))
```

**Alternatives Considered**:
1. **mattn/go-isatty** - Only provides TTY detection, no size detection
2. **Custom syscalls** - Platform-specific code, maintenance burden

## Best Practices for Progress Bars in Go

### 1. Concurrent Safety

**Pattern**: Mutex-protected updates
```go
type SafeProgressBar struct {
    bar   *progressbar.ProgressBar
    mu    sync.Mutex
}

func (s *SafeProgressBar) Add(n int) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.bar.Add(n)
}
```

**Rationale**: Go operations (Last.fm sync, normalization) use goroutines for concurrency. Progress bars must be thread-safe to avoid race conditions.

### 2. Update Throttling

**Pattern**: Time-based throttling (100ms default)
```go
bar := progressbar.NewOptions(100,
    progressbar.OptionThrottle(100 * time.Millisecond),
)
```

**Rationale**: Prevents terminal flickering and reduces CPU overhead for high-frequency updates (> 1000 items/sec).

### 3. Cleanup Patterns

**Pattern**: Defer-based cleanup
```go
bar := progressbar.New(total)
defer bar.Finish() // Ensures final state displayed and cleanup occurs
```

**Rationale**: Guarantees progress bar completion even if operation returns early or panics.

### 4. Context Integration

**Pattern**: Context-aware progress
```go
func processWithProgress(ctx context.Context, items []Item) error {
    bar := progressbar.NewOptions(len(items))
    defer bar.Finish()
    
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err() // Respects cancellation
        default:
            process(item)
            bar.Add(1)
        }
    }
    return nil
}
```

**Rationale**: Allows Ctrl+C (SIGINT) to cleanly cancel operations and clean up progress display.

## Integration Strategy

### 1. Progress Reporter Interface

**Decision**: Define interface for progress reporting

```go
// ProgressReporter provides progress update capability
type ProgressReporter interface {
    Start(total int64, description string)
    Add(n int)
    SetDescription(desc string)
    Finish(message string)
    FinishWithError(message string)
}
```

**Rationale**: 
- Decouples business logic from progress display implementation
- Enables testing with mock reporters
- Allows disabling progress (NoOp implementation) without code changes

### 2. Configuration Integration

**Decision**: Extend existing config package

```go
// In internal/config/types.go
type ProgressConfig struct {
    Enabled      bool          `yaml:"enabled" default:"true"`
    Style        string        `yaml:"style" default:"blocks"`
    ShowSpeed    bool          `yaml:"show_speed" default:"true"`
    ShowETA      bool          `yaml:"show_eta" default:"true"`
    ShowCount    bool          `yaml:"show_count" default:"true"`
    Width        int           `yaml:"width" default:"0"` // 0 = auto
    RefreshRate  time.Duration `yaml:"refresh_rate" default:"100ms"`
    Colors       bool          `yaml:"colors" default:"true"`
    AutoClear    bool          `yaml:"auto_clear" default:"true"`
}

type Config struct {
    // ... existing fields
    Progress ProgressConfig `yaml:"progress"`
}
```

**Rationale**: Consistent with existing configuration patterns in codebase.

### 3. Environment Variable Support

**Decision**: Check environment variables before config file

```go
// Priority: ENV > Config File > Defaults
func IsProgressEnabled(cfg *Config) bool {
    if val := os.Getenv("SPECKIT_NO_PROGRESS"); val == "true" {
        return false
    }
    return cfg.Progress.Enabled
}
```

**Rationale**: Environment variables provide quick override for CI/CD and scripting without config file changes.

## Signal Handling for Clean Termination

### Decision: Implement signal handler

```go
func SetupSignalHandler(bar ProgressReporter) {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        bar.Finish("Operation cancelled")
        os.Exit(130) // Standard exit code for SIGINT
    }()
}
```

**Rationale**: Ensures progress bar clears properly on Ctrl+C, providing clean user experience.

## Multi-Bar Implementation Strategy

### Decision: Stack-based multi-bar renderer

**Pattern**:
```go
type MultiProgress struct {
    bars    []*ProgressBar
    mu      sync.Mutex
    writer  io.Writer
}

func (m *MultiProgress) AddBar(total int, desc string) *ProgressBar {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    bar := newProgressBar(total, desc)
    m.bars = append(m.bars, bar)
    return bar
}

func (m *MultiProgress) Render() {
    // Clear previous lines
    // Render each completed bar with checkmark
    // Render active bars with progress
}
```

**Rationale**: Matches clarified requirement for stacking completed operations with checkmarks above active progress.

## Performance Optimization Strategies

### 1. Buffered Terminal I/O

**Decision**: Use buffered writer
```go
writer := bufio.NewWriter(os.Stdout)
bar := progressbar.NewOptions(total,
    progressbar.OptionSetWriter(writer),
)
defer writer.Flush()
```

**Rationale**: Reduces syscall overhead for frequent updates.

### 2. Conditional Rendering

**Decision**: Skip rendering if terminal too narrow
```go
func NewProgressBar(total int) *ProgressBar {
    width, _, _ := term.GetSize(int(os.Stdout.Fd()))
    if width < 20 {
        return NewNoOpProgressBar() // Silent implementation
    }
    return NewRealProgressBar(total)
}
```

**Rationale**: Avoids performance degradation and display issues on very narrow terminals.

## Error Handling Patterns

### Decision: Non-blocking error handling

**Pattern**:
```go
func (b *ProgressBar) Add(n int) error {
    if err := b.bar.Add(n); err != nil {
        // Log error but don't fail operation
        log.Debug("Progress bar update failed: %v", err)
        return nil // Return nil to not block operation
    }
    return nil
}
```

**Rationale**: Progress bar failures (e.g., terminal issues) should never block the underlying operation. Log for debugging but continue.

## Testing Strategy

### Unit Tests

1. **State Management Tests**: Verify percentage calculations, ETA accuracy, speed calculations
2. **Terminal Detection Tests**: Mock terminal capabilities, verify fallback behavior
3. **Multi-Bar Tests**: Verify stack management, completion marking
4. **Configuration Tests**: Verify environment variable precedence, config loading

### Integration Tests

1. **Real Terminal Tests**: Verify display in actual terminal environments
2. **Sync Operation Tests**: Verify progress during Last.fm sync with mock API
3. **Multi-Operation Tests**: Verify sequential operation display
4. **Signal Handling Tests**: Verify clean Ctrl+C handling

### Performance Benchmarks

1. **Update Throughput**: Measure ops/sec for progress updates
2. **Memory Usage**: Verify < 1MB memory consumption
3. **CPU Overhead**: Verify < 1% CPU impact during 10k item processing

## Risk Mitigation

### Risk 1: Terminal Compatibility Issues

**Mitigation**: 
- ASCII fallback mode for limited terminals
- Comprehensive testing matrix (Linux/macOS/Windows × multiple terminal emulators)
- Auto-detection with safe defaults

### Risk 2: Performance Impact

**Mitigation**:
- Built-in throttling (100ms default)
- Benchmarks in CI/CD to catch regressions
- Ability to disable via config/environment

### Risk 3: Multi-Threading Race Conditions

**Mitigation**:
- Mutex-protected progress bar wrapper
- Race detector in tests (`go test -race`)
- Documented concurrency patterns

## Dependencies Summary

| Package | Version | Purpose | License |
|---------|---------|---------|---------|
| github.com/schollz/progressbar/v3 | v3.14.1+ | Core progress bar | MIT |
| golang.org/x/term | latest | Terminal detection | BSD-3-Clause |

**Total Additional Dependencies**: 2 (minimal impact)  
**Estimated Binary Size Impact**: ~50-75KB

## Timeline Estimates (Validated)

Based on library selection research:

- **Library Integration**: 3 hours (reduced from initial estimate due to good API design)
- **Configuration & Environment Handling**: 2 hours
- **Multi-Bar Implementation**: 3 hours (increased slightly for stack rendering)
- **Terminal Detection & Fallbacks**: 2 hours
- **Unit Tests**: 3 hours
- **Integration Tests**: 3 hours
- **Performance Benchmarks**: 2 hours
- **Documentation**: 2 hours
- **Polish & Edge Cases**: 2 hours

**Total**: 22 hours (~3 days) - slightly higher than initial 19-hour estimate due to multi-bar complexity

## Next Steps

1. ✅ Research complete - all technology choices finalized
2. → Proceed to Phase 1: Data model and contracts definition
3. → Define ProgressReporter interface and implementation contracts
4. → Create integration quickstart guide
