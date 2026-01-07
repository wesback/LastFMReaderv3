package normalize

import (
	"regexp"
	"strings"
	"sync/atomic"
)

// Global feature flag (thread-safe)
var enabled atomic.Bool

// Precompiled regex patterns for cleanup (avoid recompilation)
var (
	multiSpacePattern      = regexp.MustCompile(`\s+`)
	trailingDashPattern    = regexp.MustCompile(`\s*[-–—]+\s*$`)
	trailingBracketPattern = regexp.MustCompile(`\s*[([\s]*$`)
)

func init() {
	enabled.Store(true) // Default: enabled
}

// SetEnabled enables or disables normalization globally.
// When disabled, NormalizeTitle returns input unchanged.
// Thread-safe for concurrent use.
func SetEnabled(state bool) {
	enabled.Store(state)
}

// IsEnabled returns whether normalization is currently enabled.
// Thread-safe for concurrent use.
func IsEnabled() bool {
	return enabled.Load()
}

// NormalizeTitle removes common annotations from track titles.
// Returns the original title if:
//   - Normalization is disabled (feature flag)
//   - Input is empty
//   - Normalized result would be too short (< minLength)
//
// The function applies patterns sequentially by priority:
//  1. Remaster annotations (highest priority)
//  2. Live performance markers
//  3. Version/edit labels
//  4. Date/year markers
//  5. Remix labels
//  6. Featuring/collaboration markers (lowest priority)
//
// Thread-safe and performant for concurrent use.
//
// Examples:
//
//	"Bohemian Rhapsody - Remastered 2011" → "Bohemian Rhapsody"
//	"Song - Live at Venue" → "Song"
//	"Track (feat. Artist)" → "Track"
//	"Live" → "Live" (preserved - too short)
//	"" → "" (preserved)
func NormalizeTitle(title string) string {
	// Check feature flag first
	if !IsEnabled() {
		return title
	}

	// Handle empty input
	if title == "" {
		return title
	}

	// Initial cleanup: trim whitespace
	normalized := strings.TrimSpace(title)
	if normalized == "" {
		return title
	}

	// Apply patterns sequentially by priority
	patterns := []struct {
		name  string
		regex *regexp.Regexp
	}{
		{"remaster", RemasterPattern},
		{"live", LivePattern},
		{"version", VersionPattern},
		{"date", DatePattern},
		{"remix", RemixPattern},
		{"featuring", FeaturingPattern},
	}

	for _, p := range patterns {
		normalized = p.regex.ReplaceAllString(normalized, "")
		// Clean up multiple spaces after each pattern
		normalized = strings.TrimSpace(normalized)
		normalized = multiSpacePattern.ReplaceAllString(normalized, " ")
		// Clean up trailing delimiters
		normalized = trailingDashPattern.ReplaceAllString(normalized, "")
		normalized = strings.TrimSpace(normalized)
	}

	// Final cleanup
	normalized = strings.TrimSpace(normalized)
	// Remove trailing delimiters and unmatched brackets
	normalized = trailingDashPattern.ReplaceAllString(normalized, "")
	// Remove unmatched opening brackets at the end
	normalized = trailingBracketPattern.ReplaceAllString(normalized, "")
	normalized = strings.TrimSpace(normalized)

	// Minimum length check: if result is too short, return original
	const minLength = 2
	if len(normalized) < minLength {
		return title
	}

	return normalized
}

// NormalizeTitleWithConfig applies normalization with custom configuration.
// Useful for advanced use cases with custom patterns or selective pattern disabling.
func NormalizeTitleWithConfig(title string, cfg *Config) string {
	// Check global feature flag first
	if !IsEnabled() {
		return title
	}

	// Check config-level enable
	if cfg != nil && !cfg.Enabled {
		return title
	}

	// Handle empty input
	if title == "" {
		return title
	}

	// Initial cleanup
	normalized := strings.TrimSpace(title)
	if normalized == "" {
		return title
	}

	// Use default config if none provided
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Apply built-in patterns (respecting disabled_patterns)
	builtInPatterns := BuiltInPatterns()
	for _, p := range builtInPatterns {
		if cfg.IsPatternEnabled(p.Name) {
			normalized = p.Regex.ReplaceAllString(normalized, "")
			normalized = strings.TrimSpace(normalized)
			normalized = multiSpacePattern.ReplaceAllString(normalized, " ")
		}
	}

	// Apply custom patterns (sorted by priority)
	for _, customPattern := range cfg.CustomPatterns {
		re, err := regexp.Compile(customPattern.Pattern)
		if err != nil {
			// Skip invalid patterns (should be caught by Validate)
			continue
		}
		normalized = re.ReplaceAllString(normalized, "")
		normalized = strings.TrimSpace(normalized)
		normalized = multiSpacePattern.ReplaceAllString(normalized, " ")
	}

	// Final cleanup
	normalized = strings.TrimSpace(normalized)
	normalized = trailingDashPattern.ReplaceAllString(normalized, "")
	normalized = trailingBracketPattern.ReplaceAllString(normalized, "")
	normalized = strings.TrimSpace(normalized)

	// Minimum length check
	minLength := cfg.MinLength
	if len(normalized) < minLength {
		return title
	}

	return normalized
}
