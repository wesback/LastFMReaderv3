package normalize

import "regexp"

// Pattern represents a normalization pattern with metadata.
type Pattern struct {
	Name        string         // Descriptive name
	Regex       *regexp.Regexp // Compiled regex pattern
	Priority    int            // Lower number = higher priority
	Description string         // Human-readable description
}

// Built-in patterns compiled at package initialization.
// All patterns are anchored to end of string ($) and case-insensitive.
var (
	// RemasterPattern removes remaster/reissue annotations.
	// Priority: 10 (highest)
	// Matches: "Remastered", "Remaster", "Reissue", "2011 Remaster", etc.
	// International: "Remasterisé" (FR), "Remasterizada" (ES)
	RemasterPattern = regexp.MustCompile(`(?i)\s*[-–—([]?\s*((\d{4}\s+)?remaster(ed|is[ée]|izada)?(\s+\d{4})?|reissue).*$`)

	// LivePattern removes live performance annotations.
	// Priority: 20
	// Matches: "Live", "Live at Venue", "Live 2023", etc.
	// International: "En Vivo" (ES), "En Direct" (FR), "Ao Vivo" (PT)
	LivePattern = regexp.MustCompile(`(?i)\s*[-–—([]?\s*(live|en\s+(vivo|direct)|ao\s+vivo).*$`)

	// VersionPattern removes version/edit annotations.
	// Priority: 30
	// Matches: "Album Version", "Radio Edit", "7\" Edit", "12\" Mix", etc.
	// Note: Requires dash delimiter for standalone version/edit to avoid matching descriptive subtitles
	// Matches qualified versions (with adjectives/formats) with any delimiter, but standalone "version/edit" only after dashes
	VersionPattern = regexp.MustCompile(`(?i)\s*[-–—([]?\s*((album|radio|extended|original|single|deluxe|special|explicit|\d+["']?(\s*inch)?)\s+(version|edit|cut|mix)).*$|\s+[-–—]\s+(version|edit).*$`)

	// SourcePattern removes source annotations (film, soundtrack, etc.).
	// Priority: 35
	// Matches: "From the Film X", "From the Movie Y", "From \"Title\" Soundtrack", etc.
	// Allows optional text between "from" and the source type keyword
	SourcePattern = regexp.MustCompile(`(?i)\s*[-–—([]?\s*from\s+.*?(film|movie|soundtrack|album|musical|show|series).*$`)

	// DatePattern removes year/date annotations.
	// Priority: 40
	// Matches: "Song - 2011", "(2023 Remaster)", "(Eurovision 1988)", "[Live 2023]", etc.
	// Handles years in parentheses/brackets with or without preceding text
	DatePattern = regexp.MustCompile(`(?i)\s+[-–—]\s+\d{4}$|[([][^)\]]*\d{4}.*$|\s+\d{4}(\s+(version|remaster|recording|mix|edition))?$`)

	// RemixPattern removes remix annotations.
	// Priority: 50
	// Matches: "Artist Remix", "Club Mix", "Acoustic Mix", etc.
	// Note: "Mix" as standalone is included here, not in VersionPattern
	// The pattern requires either a delimiter before the match, or the match to start the string
	RemixPattern = regexp.MustCompile(`(?i)\s*[-–—([]?\s*([a-zA-Z0-9\s&]+\s+)?(remix|rework|mix|acoustic|unplugged|instrumental).*$`)

	// FeaturingPattern removes featuring/with annotations.
	// Priority: 60 (lowest)
	// Matches: "feat. Artist", "ft. Artist", "featuring Artist", "with Artist"
	// International: "con" (ES), "avec" (FR)
	// Note: "with" requires a delimiter before it to avoid matching mid-title occurrences
	FeaturingPattern = regexp.MustCompile(`(?i)\s*[-–—([]?\s*(feat\.?|ft\.?|featuring)\s+.*$|\s+[-–—]\s+(with|con|avec)\s+.*$`)
)

// BuiltInPatterns returns all built-in patterns sorted by priority.
func BuiltInPatterns() []Pattern {
	return []Pattern{
		{
			Name:        "remaster",
			Regex:       RemasterPattern,
			Priority:    10,
			Description: "Removes remaster/reissue annotations",
		},
		{
			Name:        "live",
			Regex:       LivePattern,
			Priority:    20,
			Description: "Removes live performance annotations",
		},
		{
			Name:        "version",
			Regex:       VersionPattern,
			Priority:    30,
			Description: "Removes version/edit annotations",
		},
		{
			Name:        "source",
			Regex:       SourcePattern,
			Priority:    35,
			Description: "Removes source annotations (film/soundtrack/etc.)",
		},
		{
			Name:        "date",
			Regex:       DatePattern,
			Priority:    40,
			Description: "Removes year/date annotations",
		},
		{
			Name:        "remix",
			Regex:       RemixPattern,
			Priority:    50,
			Description: "Removes remix/acoustic annotations",
		},
		{
			Name:        "featuring",
			Regex:       FeaturingPattern,
			Priority:    60,
			Description: "Removes featuring/collaboration annotations",
		},
	}
}
