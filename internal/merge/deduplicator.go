package merge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lastfm-reader/lastfm-sync/internal/models"
)

// DeduplicationMap tracks unique scrobbles using hash-based deduplication
type DeduplicationMap struct {
	data               map[string]*models.Scrobble
	strategy           DeduplicationStrategy
	conflictResolution ConflictResolution
	totalAdded         int
	duplicates         int
}

// NewDeduplicationMap creates a new deduplication map with the specified strategy
func NewDeduplicationMap(strategy DeduplicationStrategy, conflictResolution ConflictResolution) *DeduplicationMap {
	return &DeduplicationMap{
		data:               make(map[string]*models.Scrobble),
		strategy:           strategy,
		conflictResolution: conflictResolution,
	}
}

// Add adds a scrobble to the deduplication map
// Returns true if the scrobble was added (new), false if it was a duplicate
func (dm *DeduplicationMap) Add(scrobble *models.Scrobble) bool {
	dm.totalAdded++
	key := dm.generateKey(scrobble)

	if existing, exists := dm.data[key]; exists {
		// Conflict: resolve using configured strategy
		dm.duplicates++
		resolved := ResolveConflict(existing, scrobble, dm.conflictResolution)
		dm.data[key] = resolved
		return false // Duplicate
	}

	dm.data[key] = scrobble
	return true // New
}

// GetAll returns all unique scrobbles
func (dm *DeduplicationMap) GetAll() []*models.Scrobble {
	result := make([]*models.Scrobble, 0, len(dm.data))
	for _, s := range dm.data {
		result = append(result, s)
	}
	return result
}

// TotalAdded returns the total number of scrobbles attempted to add
func (dm *DeduplicationMap) TotalAdded() int {
	return dm.totalAdded
}

// UniqueCount returns the number of unique scrobbles
func (dm *DeduplicationMap) UniqueCount() int {
	return len(dm.data)
}

// DuplicateCount returns the number of duplicates found
func (dm *DeduplicationMap) DuplicateCount() int {
	return dm.duplicates
}

// generateKey generates a unique hash key for a scrobble based on the strategy
func (dm *DeduplicationMap) generateKey(s *models.Scrobble) string {
	h := sha256.New()

	switch dm.strategy {
	case StrategyStrict:
		// Artist + Album + Track + UTS + Duration (not available in current model)
		h.Write([]byte(strings.ToLower(s.Artist)))
		h.Write([]byte(strings.ToLower(s.Album)))
		h.Write([]byte(strings.ToLower(s.Track)))
		h.Write([]byte(fmt.Sprintf("%d", s.UTS)))
		// Note: Duration field not in current model, using 0 as placeholder
		h.Write([]byte(fmt.Sprintf("%d", 0)))

	case StrategyRelaxed:
		// Artist + Track + UTS (no Album)
		h.Write([]byte(strings.ToLower(s.Artist)))
		h.Write([]byte(strings.ToLower(s.Track)))
		h.Write([]byte(fmt.Sprintf("%d", s.UTS)))

	case StrategyMBID:
		// MusicBrainz Track ID + UTS (fallback to default if no MBID)
		if s.MBID != nil && *s.MBID != "" {
			h.Write([]byte(*s.MBID))
			h.Write([]byte(fmt.Sprintf("%d", s.UTS)))
		} else {
			// Fallback to default strategy
			return dm.generateKeyDefault(s)
		}

	default: // StrategyDefault
		return dm.generateKeyDefault(s)
	}

	return hex.EncodeToString(h.Sum(nil))
}

// generateKeyDefault generates the default deduplication key
// Artist + Album + Track + UTS
func (dm *DeduplicationMap) generateKeyDefault(s *models.Scrobble) string {
	h := sha256.New()
	h.Write([]byte(strings.ToLower(s.Artist)))
	h.Write([]byte(strings.ToLower(s.Album)))
	h.Write([]byte(strings.ToLower(s.Track)))
	h.Write([]byte(fmt.Sprintf("%d", s.UTS)))
	return hex.EncodeToString(h.Sum(nil))
}
