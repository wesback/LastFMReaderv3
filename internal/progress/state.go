package progress

// ProgressState represents the current state of a progress bar.
type ProgressState int

const (
	// StateActive indicates progress is actively updating
	StateActive ProgressState = iota

	// StateSuccess indicates operation completed successfully
	StateSuccess

	// StateError indicates operation failed
	StateError

	// StateWarning indicates operation completed with warnings
	StateWarning

	// StateSpinner indicates indeterminate progress (unknown total)
	StateSpinner
)

// String returns the string representation of the state.
func (s ProgressState) String() string {
	switch s {
	case StateActive:
		return "active"
	case StateSuccess:
		return "success"
	case StateError:
		return "error"
	case StateWarning:
		return "warning"
	case StateSpinner:
		return "spinner"
	default:
		return "unknown"
	}
}
