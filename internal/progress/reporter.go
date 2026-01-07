package progress

// ProgressReporter provides progress update capability for long-running operations.
// All implementations must be thread-safe.
type ProgressReporter interface {
	// Start initializes progress tracking with total items and description.
	// Returns error if already started or invalid parameters.
	Start(total int64, description string) error

	// Add increments progress by n items.
	// Returns error if bar is finished or value invalid.
	Add(n int) error

	// SetCurrent sets progress to specific value.
	// Returns error if value > total or bar is finished.
	SetCurrent(n int64) error

	// SetDescription updates the operation description.
	SetDescription(desc string) error

	// Finish marks operation as successfully complete with optional message.
	Finish(message string) error

	// FinishWithError marks operation as failed with error message.
	FinishWithError(message string) error

	// FinishWithWarning marks operation as complete with warnings.
	FinishWithWarning(message string) error

	// SwitchToSpinner converts to indeterminate spinner mode (e.g., for rate limiting).
	SwitchToSpinner() error

	// ResumeProgress resumes normal progress bar from spinner mode.
	ResumeProgress() error

	// IsFinished returns whether progress tracking is complete.
	IsFinished() bool
}
