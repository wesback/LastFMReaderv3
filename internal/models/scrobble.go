package models

import (
	"encoding/json"
	"time"

	"github.com/lastfm-reader/lastfm-sync/internal/logging"
	"github.com/lastfm-reader/lastfm-sync/internal/normalize"
	"go.uber.org/zap"
)

// Scrobble represents a single track listen event from Last.fm
// matching the NDJSON contract in spec.md
type Scrobble struct {
	Username        string          `json:"username"`
	Artist          string          `json:"artist"`
	Track           string          `json:"track"`
	NormalizedTitle string          `json:"normalized_title"` // Track title with annotations removed
	Album           string          `json:"album"`
	UTS             int64           `json:"uts"`
	LocalTime       string          `json:"local_time"`
	MBID            *string         `json:"mbid,omitempty"`
	Source          string          `json:"source"`
	IngestedAt      string          `json:"ingested_at"`
	Raw             json.RawMessage `json:"raw"`
}

// formatTimestamp converts Unix timestamp to RFC3339 UTC format.
// Returns empty string for invalid timestamps (uts <= 0).
func formatTimestamp(uts int64) string {
	if uts <= 0 {
		return ""
	}
	t := time.Unix(uts, 0).UTC()
	return t.Format(time.RFC3339)
}

// NewScrobble creates a Scrobble from API fields with current ingested_at timestamp.
// raw should be the original Last.fm API response object.
// If a logger is available globally, DEBUG logs normalization changes.
func NewScrobble(username, artist, track, album string, uts int64, mbid *string, raw json.RawMessage) *Scrobble {
	normalizedTitle := normalize.NormalizeTitle(track)

	// Log when title is changed by normalization
	if normalizedTitle != track {
		// Try to get logger, but don't fail if not available
		logger, err := logging.New("debug")
		if err == nil {
			logger.Debug("title normalized",
				zap.String("original", track),
				zap.String("normalized", normalizedTitle),
				zap.String("artist", artist),
				zap.String("username", username),
			)
		}
	}

	return &Scrobble{
		Username:        username,
		Artist:          artist,
		Track:           track,
		NormalizedTitle: normalizedTitle,
		Album:           album,
		UTS:             uts,
		LocalTime:       formatTimestamp(uts),
		MBID:            mbid,
		Source:          "lastfm",
		IngestedAt:      time.Now().UTC().Format(time.RFC3339),
		Raw:             raw,
	}
}

// MarshalJSON implements json.Marshaler to produce compact NDJSON format.
// Omits null MBID to reduce output size.
func (s *Scrobble) MarshalJSON() ([]byte, error) {
	type Alias Scrobble
	return json.Marshal((*Alias)(s))
}

// Watermark tracks the maximum processed unix timestamp per user for incremental syncs.
type Watermark struct {
	Username  string `json:"username"`
	MaxUTS    int64  `json:"max_uts"`
	UpdatedAt string `json:"updated_at"`
	SyncID    string `json:"sync_id"`
}

// NewWatermark creates a Watermark for tracking incremental progress.
func NewWatermark(username string, maxUTS int64, syncID string) *Watermark {
	return &Watermark{
		Username:  username,
		MaxUTS:    maxUTS,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		SyncID:    syncID,
	}
}
