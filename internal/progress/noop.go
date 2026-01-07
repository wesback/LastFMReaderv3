package progress

// NoOpProgressBar is a silent implementation of ProgressReporter that does nothing.
// Used when progress bars are disabled or terminal is non-interactive.
type NoOpProgressBar struct{}

// NewNoOpProgressBar creates a new no-op progress reporter.
func NewNoOpProgressBar() ProgressReporter {
	return &NoOpProgressBar{}
}

// Start does nothing and always returns nil.
func (n *NoOpProgressBar) Start(total int64, description string) error {
	return nil
}

// Add does nothing and always returns nil.
func (n *NoOpProgressBar) Add(count int) error {
	return nil
}

// SetCurrent does nothing and always returns nil.
func (n *NoOpProgressBar) SetCurrent(n64 int64) error {
	return nil
}

// SetDescription does nothing and always returns nil.
func (n *NoOpProgressBar) SetDescription(desc string) error {
	return nil
}

// Finish does nothing and always returns nil.
func (n *NoOpProgressBar) Finish(message string) error {
	return nil
}

// FinishWithError does nothing and always returns nil.
func (n *NoOpProgressBar) FinishWithError(message string) error {
	return nil
}

// FinishWithWarning does nothing and always returns nil.
func (n *NoOpProgressBar) FinishWithWarning(message string) error {
	return nil
}

// SwitchToSpinner does nothing and always returns nil.
func (n *NoOpProgressBar) SwitchToSpinner() error {
	return nil
}

// ResumeProgress does nothing and always returns nil.
func (n *NoOpProgressBar) ResumeProgress() error {
	return nil
}

// IsFinished always returns false for NoOp implementation.
func (n *NoOpProgressBar) IsFinished() bool {
	return false
}
