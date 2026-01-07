package progress

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

var (
	// ErrBarFinished is returned when updating a finished bar
	ErrBarFinished = errors.New("progress bar already finished")

	// ErrNegativeValue is returned when progress value is negative
	ErrNegativeValue = errors.New("progress value cannot be negative")

	// ErrExceedsTotal is returned when progress exceeds total
	ErrExceedsTotal = errors.New("progress exceeds total")

	// ErrInvalidState is returned when operation is invalid for current state
	ErrInvalidState = errors.New("invalid operation for current state")
)

// ProgressBar is a real progress bar implementation using schollz/progressbar.
type ProgressBar struct {
	current     int64
	total       int64
	description string
	startTime   time.Time
	options     Options
	bar         *progressbar.ProgressBar
	mu          sync.Mutex
	finished    bool
	state       ProgressState
}

// NewRealProgressBar creates a new progress bar with the given configuration.
func NewRealProgressBar(total int64, desc string, opts ...Option) *ProgressBar {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	pb := &ProgressBar{
		total:       total,
		description: desc,
		startTime:   time.Now(),
		options:     options,
		state:       StateActive,
	}

	// Configure progressbar library options
	barOpts := []progressbar.Option{
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetWriter(options.Writer),
		progressbar.OptionThrottle(options.RefreshRate),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(options.ShowETA),
		progressbar.OptionSetElapsedTime(options.ShowElapsed),
	}

	// Only set width if explicitly specified (0 means auto-detect, don't pass to library)
	if options.Width > 0 {
		barOpts = append(barOpts, progressbar.OptionSetWidth(options.Width))
	}

	if options.ShowSpeed {
		barOpts = append(barOpts, progressbar.OptionShowIts())
	}

	if options.EnableColors {
		barOpts = append(barOpts, progressbar.OptionEnableColorCodes(true))
		barOpts = append(barOpts,
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        options.Style.Complete,
				SaucerHead:    options.Style.InProgress,
				SaucerPadding: options.Style.Incomplete,
				BarStart:      options.Style.BarStart,
				BarEnd:        options.Style.BarEnd,
			}),
		)
	} else {
		barOpts = append(barOpts, progressbar.OptionEnableColorCodes(false))
		barOpts = append(barOpts,
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        options.Style.Complete,
				SaucerHead:    options.Style.InProgress,
				SaucerPadding: options.Style.Incomplete,
				BarStart:      options.Style.BarStart,
				BarEnd:        options.Style.BarEnd,
			}),
		)
	}

	if options.AutoClear {
		barOpts = append(barOpts, progressbar.OptionClearOnFinish())
	}

	pb.bar = progressbar.NewOptions64(total, barOpts...)

	return pb
}

// Start initializes progress tracking.
func (p *ProgressBar) Start(total int64, description string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.finished {
		return ErrBarFinished
	}

	p.total = total
	p.description = description
	p.startTime = time.Now()
	p.current = 0
	p.state = StateActive

	// Reset the bar with new total and description
	p.bar.ChangeMax64(total)
	p.bar.Describe(description)

	return nil
}

// Add increments the progress by n.
func (p *ProgressBar) Add(n int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.finished {
		return ErrBarFinished
	}

	if n < 0 {
		return ErrNegativeValue
	}

	newCurrent := p.current + int64(n)
	if newCurrent > p.total {
		return ErrExceedsTotal
	}

	p.current = newCurrent
	return p.bar.Add(n)
}

// SetCurrent sets the progress to a specific value.
func (p *ProgressBar) SetCurrent(n int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.finished {
		return ErrBarFinished
	}

	if n < 0 {
		return ErrNegativeValue
	}

	if n > p.total {
		return ErrExceedsTotal
	}

	p.current = n
	return p.bar.Set64(n)
}

// SetDescription updates the operation description.
func (p *ProgressBar) SetDescription(desc string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.description = desc
	p.bar.Describe(desc)
	return nil
}

// Finish marks the progress as successfully complete.
func (p *ProgressBar) Finish(message string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.finished {
		return nil // Already finished, no-op
	}

	p.finished = true
	p.state = StateSuccess

	// Set to 100%
	if err := p.bar.Finish(); err != nil {
		return fmt.Errorf("failed to finish progress bar: %w", err)
	}

	if message != "" {
		fmt.Fprintf(p.options.Writer, "\n%s\n", message)
	}

	return nil
}

// FinishWithError marks the progress as failed.
func (p *ProgressBar) FinishWithError(message string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.finished {
		return nil
	}

	p.finished = true
	p.state = StateError

	p.bar.Clear()

	if message != "" {
		// Red color for errors if colors enabled
		if p.options.EnableColors {
			fmt.Fprintf(p.options.Writer, "\033[31m✗ %s\033[0m\n", message)
		} else {
			fmt.Fprintf(p.options.Writer, "✗ %s\n", message)
		}
	}

	return nil
}

// FinishWithWarning marks the progress as complete with warnings.
func (p *ProgressBar) FinishWithWarning(message string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.finished {
		return nil
	}

	p.finished = true
	p.state = StateWarning

	if err := p.bar.Finish(); err != nil {
		return fmt.Errorf("failed to finish progress bar: %w", err)
	}

	if message != "" {
		// Yellow color for warnings if colors enabled
		if p.options.EnableColors {
			fmt.Fprintf(p.options.Writer, "\033[33m⚠ %s\033[0m\n", message)
		} else {
			fmt.Fprintf(p.options.Writer, "⚠ %s\n", message)
		}
	}

	return nil
}

// SwitchToSpinner converts to indeterminate spinner mode.
func (p *ProgressBar) SwitchToSpinner() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.finished {
		return ErrBarFinished
	}

	p.state = StateSpinner
	// Note: schollz/progressbar doesn't have native spinner support
	// For now, we'll just update the description to indicate waiting
	return nil
}

// ResumeProgress resumes normal progress bar from spinner mode.
func (p *ProgressBar) ResumeProgress() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.finished {
		return ErrBarFinished
	}

	if p.state != StateSpinner {
		return ErrInvalidState
	}

	p.state = StateActive
	return nil
}

// IsFinished returns whether the progress bar is finished.
func (p *ProgressBar) IsFinished() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.finished
}

// Clear removes the progress bar from the terminal.
func (p *ProgressBar) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.bar.Clear()
}
