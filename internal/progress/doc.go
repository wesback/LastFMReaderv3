// Package progress provides visual progress indication for long-running
// console operations in LastFMReaderv3.
//
// The package offers thread-safe progress bar implementations that automatically
// adapt to terminal capabilities, with support for:
//   - Real-time progress tracking with percentage, ETA, and speed display
//   - Multiple visual styles (Unicode blocks, ASCII, arrows, dots)
//   - Terminal capability detection and graceful fallbacks
//   - Multi-operation workflows with stacked completion states
//   - Configurable display options and error state handling
//
// # Basic Usage
//
//	reporter := progress.NewProgressReporter(cfg, total, "Processing items")
//	defer reporter.Finish("Complete")
//
//	for _, item := range items {
//	    processItem(item)
//	    reporter.Add(1)
//	}
//
// # Configuration
//
// Progress bars can be configured via config file or environment variables:
//   - SPECKIT_NO_PROGRESS: Disable all progress bars
//   - SPECKIT_PROGRESS_ASCII: Force ASCII mode
//   - SPECKIT_NO_COLOR: Disable colors
//   - SPECKIT_PROGRESS_REFRESH: Set refresh rate in milliseconds
//   - SPECKIT_PROGRESS_WIDTH: Set bar width
//
// # Terminal Detection
//
// Progress bars automatically disable in non-interactive environments
// (pipes, redirects) and adapt visual style based on terminal capabilities.
package progress
