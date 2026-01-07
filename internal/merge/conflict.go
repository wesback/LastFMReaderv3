package merge

import (
	"github.com/lastfm-reader/lastfm-sync/internal/models"
)

// CompletenessScore calculates a completeness score for a scrobble
// Higher scores indicate more complete metadata
func CompletenessScore(s *models.Scrobble) int {
	score := 0

	// Core fields (always present in valid scrobbles)
	if s.Artist != "" {
		score++
	}
	if s.Track != "" {
		score++
	}
	if s.UTS > 0 {
		score++
	}

	// Optional metadata fields
	if s.Album != "" {
		score++
	}
	if s.Username != "" {
		score++
	}
	if s.Source != "" {
		score++
	}

	// MusicBrainz ID gets extra weight (authoritative source)
	if s.MBID != nil && *s.MBID != "" {
		score += 2
	}

	return score
}

// ResolveConflict determines which scrobble to keep when duplicates are found
// Returns the scrobble that should be kept based on the resolution mode
func ResolveConflict(existing, new *models.Scrobble, mode ConflictResolution) *models.Scrobble {
	switch mode {
	case ResolutionFirst:
		// Always keep the first occurrence
		return existing

	case ResolutionLast:
		// Always take the latest occurrence
		return new

	case ResolutionCompleteness:
		// Select based on completeness score
		existingScore := CompletenessScore(existing)
		newScore := CompletenessScore(new)

		if newScore > existingScore {
			// New has more complete metadata
			return new
		} else if newScore < existingScore {
			// Existing has more complete metadata
			return existing
		}

		// Tie-breaker: prefer later timestamp (assumption: later exports may have corrections)
		if new.UTS > existing.UTS {
			return new
		} else if new.UTS < existing.UTS {
			return existing
		}

		// Final tie-breaker: keep existing (stable deduplication)
		return existing

	default:
		// Default to completeness mode
		return ResolveConflict(existing, new, ResolutionCompleteness)
	}
}
