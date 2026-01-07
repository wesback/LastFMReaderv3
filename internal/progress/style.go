package progress

// Style defines the visual appearance of a progress bar.
type Style struct {
	// BarStart is the character(s) at the start of the bar (e.g., "[")
	BarStart string

	// BarEnd is the character(s) at the end of the bar (e.g., "]")
	BarEnd string

	// Complete is the character(s) for completed segments (e.g., "█")
	Complete string

	// InProgress is the character(s) for in-progress segments (e.g., "▓")
	InProgress string

	// Incomplete is the character(s) for incomplete segments (e.g., "░")
	Incomplete string

	// SpinnerFrames are the animation frames for spinner mode
	SpinnerFrames []string
}

// Predefined visual styles for progress bars

// StyleBlocks uses Unicode block characters (default style for modern terminals)
var StyleBlocks = Style{
	BarStart:      "[",
	BarEnd:        "]",
	Complete:      "█",
	InProgress:    "▓",
	Incomplete:    "░",
	SpinnerFrames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
}

// StyleArrows uses arrow-based progression (alternative Unicode style)
var StyleArrows = Style{
	BarStart:      "[",
	BarEnd:        "]",
	Complete:      "=",
	InProgress:    ">",
	Incomplete:    " ",
	SpinnerFrames: []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
}

// StyleDots uses circle-based progression (alternative Unicode style)
var StyleDots = Style{
	BarStart:      "[",
	BarEnd:        "]",
	Complete:      "●",
	InProgress:    "◐",
	Incomplete:    "○",
	SpinnerFrames: []string{"◴", "◷", "◶", "◵"},
}

// StyleASCII uses only ASCII characters (fallback for limited terminals)
var StyleASCII = Style{
	BarStart:      "[",
	BarEnd:        "]",
	Complete:      "#",
	InProgress:    "#",
	Incomplete:    " ",
	SpinnerFrames: []string{"|", "/", "-", "\\"},
}
