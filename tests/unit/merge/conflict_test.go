package merge_test

import (
	"testing"

	"github.com/lastfm-reader/lastfm-sync/internal/merge"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
)

func TestCompletenessScore(t *testing.T) {
	tests := []struct {
		name     string
		scrobble *models.Scrobble
		want     int
	}{
		{
			name: "minimal scrobble (required fields only)",
			scrobble: &models.Scrobble{
				Artist: "Artist",
				Track:  "Track",
				UTS:    1000,
			},
			want: 3, // Artist + Track + UTS
		},
		{
			name: "scrobble with album",
			scrobble: &models.Scrobble{
				Artist: "Artist",
				Album:  "Album",
				Track:  "Track",
				UTS:    1000,
			},
			want: 4, // Artist + Album + Track + UTS
		},
		{
			name: "scrobble with MBID (higher weight)",
			scrobble: &models.Scrobble{
				Artist: "Artist",
				Track:  "Track",
				UTS:    1000,
				MBID:   strPtr("12345"),
			},
			want: 5, // Artist + Track + UTS + MBID (weighted +2)
		},
		{
			name: "fully complete scrobble",
			scrobble: &models.Scrobble{
				Username: "user1",
				Artist:   "Artist",
				Album:    "Album",
				Track:    "Track",
				UTS:      1000,
				MBID:     strPtr("12345"),
				Source:   "lastfm",
			},
			want: 8, // Username + Artist + Album + Track + UTS + MBID(+2) + Source
		},
		{
			name: "empty optional fields don't count",
			scrobble: &models.Scrobble{
				Artist: "Artist",
				Album:  "", // Empty
				Track:  "Track",
				UTS:    1000,
				MBID:   nil, // Nil pointer
			},
			want: 3, // Only Artist + Track + UTS
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := merge.CompletenessScore(tt.scrobble)
			if got != tt.want {
				t.Errorf("CompletenessScore() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResolveConflict(t *testing.T) {
	tests := []struct {
		name     string
		existing *models.Scrobble
		new      *models.Scrobble
		wantNew  bool // true if new should win, false if existing should win
	}{
		{
			name: "new has higher completeness score",
			existing: &models.Scrobble{
				Artist: "Artist",
				Track:  "Track",
				UTS:    1000,
			},
			new: &models.Scrobble{
				Artist: "Artist",
				Album:  "Album",
				Track:  "Track",
				UTS:    1000,
			},
			wantNew: true,
		},
		{
			name: "existing has higher completeness score",
			existing: &models.Scrobble{
				Artist: "Artist",
				Album:  "Album",
				Track:  "Track",
				UTS:    1000,
				MBID:   strPtr("12345"),
			},
			new: &models.Scrobble{
				Artist: "Artist",
				Track:  "Track",
				UTS:    1000,
			},
			wantNew: false,
		},
		{
			name: "tie - same completeness, new has MBID (tie-breaker)",
			existing: &models.Scrobble{
				Artist: "Artist",
				Album:  "Album",
				Track:  "Track",
				UTS:    1000,
			},
			new: &models.Scrobble{
				Artist: "Artist",
				Track:  "Track",
				UTS:    1000,
				MBID:   strPtr("12345"),
			},
			wantNew: true, // MBID gives higher score
		},
		{
			name: "tie - same completeness, prefer later timestamp",
			existing: &models.Scrobble{
				Artist: "Artist",
				Album:  "Album",
				Track:  "Track",
				UTS:    1000,
			},
			new: &models.Scrobble{
				Artist: "Artist",
				Album:  "Album2",
				Track:  "Track",
				UTS:    2000,
			},
			wantNew: true, // Later timestamp
		},
		{
			name: "tie - same completeness and timestamp, prefer existing",
			existing: &models.Scrobble{
				Artist: "Artist",
				Album:  "Album",
				Track:  "Track",
				UTS:    1000,
			},
			new: &models.Scrobble{
				Artist: "Artist",
				Album:  "Album2",
				Track:  "Track",
				UTS:    1000,
			},
			wantNew: false, // Same completeness and timestamp, keep existing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := merge.ResolveConflict(tt.existing, tt.new, merge.ResolutionCompleteness)

			if tt.wantNew && result != tt.new {
				t.Errorf("ResolveConflict() should have returned new scrobble")
			}
			if !tt.wantNew && result != tt.existing {
				t.Errorf("ResolveConflict() should have returned existing scrobble")
			}
		})
	}
}

func TestResolveConflict_Modes(t *testing.T) {
	existing := &models.Scrobble{
		Artist: "Artist",
		Track:  "Track",
		UTS:    1000,
	}

	new := &models.Scrobble{
		Artist: "Artist",
		Album:  "Album",
		Track:  "Track",
		UTS:    1000,
	}

	t.Run("completeness mode - prefers more complete", func(t *testing.T) {
		result := merge.ResolveConflict(existing, new, merge.ResolutionCompleteness)
		if result != new {
			t.Error("completeness mode should prefer new (more complete)")
		}
	})

	t.Run("first mode - always keeps existing", func(t *testing.T) {
		result := merge.ResolveConflict(existing, new, merge.ResolutionFirst)
		if result != existing {
			t.Error("first mode should always keep existing")
		}
	})

	t.Run("last mode - always takes new", func(t *testing.T) {
		result := merge.ResolveConflict(existing, new, merge.ResolutionLast)
		if result != new {
			t.Error("last mode should always take new")
		}
	})
}
