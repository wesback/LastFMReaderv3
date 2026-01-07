package progress

import (
	"io"
	"os"
	"time"
)

// Options configures progress bar display and behavior.
type Options struct {
	// Width is the terminal width for the bar (0 = auto-detect)
	Width int

	// ShowSpeed enables display of items/second
	ShowSpeed bool

	// ShowETA enables display of estimated time remaining
	ShowETA bool

	// ShowElapsed enables display of elapsed time
	ShowElapsed bool

	// ShowPercent enables display of percentage complete
	ShowPercent bool

	// ShowCount enables display of current/total count
	ShowCount bool

	// RefreshRate is the update frequency (default 100ms)
	RefreshRate time.Duration

	// Style is the visual style to use
	Style Style

	// EnableColors enables ANSI color codes
	EnableColors bool

	// AutoClear clears the bar on completion
	AutoClear bool

	// Writer is the output destination (default os.Stdout)
	Writer io.Writer
}

// DefaultOptions returns the default configuration for progress bars.
func DefaultOptions() Options {
	return Options{
		Width:        0, // Auto-detect
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
}

// Option is a functional option for configuring Options.
type Option func(*Options)

// WithWidth sets the bar width (0 = auto-detect).
func WithWidth(w int) Option {
	return func(o *Options) {
		o.Width = w
	}
}

// WithStyle sets the visual style.
func WithStyle(s Style) Option {
	return func(o *Options) {
		o.Style = s
	}
}

// WithColors enables or disables ANSI colors.
func WithColors(enabled bool) Option {
	return func(o *Options) {
		o.EnableColors = enabled
	}
}

// WithSpeed enables or disables speed display.
func WithSpeed(show bool) Option {
	return func(o *Options) {
		o.ShowSpeed = show
	}
}

// WithETA enables or disables ETA display.
func WithETA(show bool) Option {
	return func(o *Options) {
		o.ShowETA = show
	}
}

// WithElapsed enables or disables elapsed time display.
func WithElapsed(show bool) Option {
	return func(o *Options) {
		o.ShowElapsed = show
	}
}

// WithPercentage enables or disables percentage display.
func WithPercentage(show bool) Option {
	return func(o *Options) {
		o.ShowPercent = show
	}
}

// WithCount enables or disables current/total count display.
func WithCount(show bool) Option {
	return func(o *Options) {
		o.ShowCount = show
	}
}

// WithRefreshRate sets the update frequency.
func WithRefreshRate(rate time.Duration) Option {
	return func(o *Options) {
		o.RefreshRate = rate
	}
}

// WithAutoClear enables or disables auto-clearing on completion.
func WithAutoClear(enabled bool) Option {
	return func(o *Options) {
		o.AutoClear = enabled
	}
}

// WithWriter sets the output writer.
func WithWriter(w io.Writer) Option {
	return func(o *Options) {
		o.Writer = w
	}
}
