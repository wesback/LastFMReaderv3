package progress

import "github.com/lastfm-reader/lastfm-sync/internal/config"

// NewProgressReporter creates an appropriate ProgressReporter based on configuration
// and terminal capabilities. Returns NoOpProgressBar if progress is disabled or
// terminal is non-interactive, otherwise returns a RealProgressBar (not yet started).
func NewProgressReporter(cfg *config.Config) ProgressReporter {
	// Check if progress is disabled in config
	if !cfg.Progress.Enabled {
		return NewNoOpProgressBar()
	}

	// Detect terminal capabilities
	termInfo, err := DetectTerminal()
	if err != nil || !termInfo.ShouldDisplay() {
		return NewNoOpProgressBar()
	}

	// Build options from config
	opts := []Option{
		WithWidth(cfg.Progress.Width),
		WithColors(cfg.Progress.Colors),
		WithSpeed(cfg.Progress.ShowSpeed),
		WithETA(cfg.Progress.ShowETA),
		WithPercentage(cfg.Progress.ShowPercentage),
		WithCount(cfg.Progress.ShowCount),
		WithElapsed(cfg.Progress.ShowElapsed),
		WithRefreshRate(cfg.Progress.RefreshRate),
		WithAutoClear(cfg.Progress.AutoClear),
	}

	// Select style based on config and terminal capabilities
	var style Style
	switch cfg.Progress.Style {
	case "arrows":
		style = StyleArrows
	case "dots":
		style = StyleDots
	case "ascii":
		style = StyleASCII
	case "blocks":
		fallthrough
	default:
		style = termInfo.BestStyle()
	}
	opts = append(opts, WithStyle(style))

	// Return unstarted RealProgressBar - caller will call Start()
	return NewRealProgressBar(0, "", opts...)
}
