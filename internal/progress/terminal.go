package progress

import (
	"os"

	"golang.org/x/term"
)

// TerminalInfo holds detected terminal capabilities.
type TerminalInfo struct {
	// Width is the terminal width in characters
	Width int

	// Height is the terminal height in lines
	Height int

	// SupportsUTF8 indicates if the terminal can display Unicode
	SupportsUTF8 bool

	// SupportsColor indicates if the terminal can display ANSI colors
	SupportsColor bool

	// IsInteractive indicates if this is a TTY (not redirected)
	IsInteractive bool
}

// DetectTerminal queries terminal capabilities using golang.org/x/term.
func DetectTerminal() (*TerminalInfo, error) {
	info := &TerminalInfo{
		Width:         80,   // Default fallback
		Height:        24,   // Default fallback
		SupportsUTF8:  true, // Assume UTF-8 by default (modern systems)
		SupportsColor: true, // Assume color support by default
		IsInteractive: false,
	}

	// Check if stdout is a terminal
	fd := int(os.Stdout.Fd())
	if term.IsTerminal(fd) {
		info.IsInteractive = true

		// Get terminal size
		width, height, err := term.GetSize(fd)
		if err == nil {
			info.Width = width
			info.Height = height
		}

		// Check for color support via environment variables
		colorTerm := os.Getenv("COLORTERM")
		term := os.Getenv("TERM")

		// NO_COLOR environment variable disables colors (standard)
		if os.Getenv("NO_COLOR") != "" {
			info.SupportsColor = false
		}

		// Detect UTF-8 support from locale
		lang := os.Getenv("LANG")
		lc_all := os.Getenv("LC_ALL")
		if lang != "" || lc_all != "" {
			// Simple check: if locale contains "UTF-8" or "utf8", assume UTF-8
			if contains(lang, "UTF-8") || contains(lang, "utf8") ||
				contains(lc_all, "UTF-8") || contains(lc_all, "utf8") {
				info.SupportsUTF8 = true
			} else {
				info.SupportsUTF8 = false
			}
		}

		// Check terminal type for color support
		if colorTerm != "" || contains(term, "256color") || contains(term, "color") {
			info.SupportsColor = true
		}

		// Windows cmd.exe has limited support
		if term == "dumb" {
			info.SupportsColor = false
			info.SupportsUTF8 = false
		}
	}

	return info, nil
}

// BestStyle returns the recommended style for this terminal.
func (t *TerminalInfo) BestStyle() Style {
	if !t.SupportsUTF8 {
		return StyleASCII
	}
	return StyleBlocks // Default to blocks for UTF-8 terminals
}

// ShouldDisplay returns whether progress bars should be shown.
// Returns false for non-interactive terminals (pipes, redirects).
func (t *TerminalInfo) ShouldDisplay() bool {
	return t.IsInteractive
}

// contains is a simple substring check helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
