package merge_test

import (
	"testing"

	"github.com/lastfm-reader/lastfm-sync/internal/merge"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
)

// Helper function to create MBID pointer
func strPtr(s string) *string {
	return &s
}

func TestDeduplicationMap_Add_Default(t *testing.T) {
	tests := []struct {
		name      string
		scrobbles []*models.Scrobble
		want      int
	}{
		{
			name: "no duplicates",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
				{Artist: "Pink Floyd", Album: "Dark Side", Track: "Time", UTS: 1735689700},
			},
			want: 2,
		},
		{
			name: "exact duplicate",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
			},
			want: 1,
		},
		{
			name: "case insensitive duplicate",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
				{Artist: "the beatles", Album: "abbey road", Track: "come together", UTS: 1735689600},
			},
			want: 1,
		},
		{
			name: "different timestamp - not duplicate",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689700},
			},
			want: 2,
		},
		{
			name: "different album - not duplicate",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
				{Artist: "The Beatles", Album: "Let It Be", Track: "Come Together", UTS: 1735689600},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := merge.NewDeduplicationMap(merge.StrategyDefault, merge.ResolutionCompleteness)
			for _, s := range tt.scrobbles {
				dm.Add(s)
			}
			if got := dm.UniqueCount(); got != tt.want {
				t.Errorf("got %d unique scrobbles, want %d", got, tt.want)
			}
		})
	}
}

func TestDeduplicationMap_Add_Strict(t *testing.T) {
	tests := []struct {
		name      string
		scrobbles []*models.Scrobble
		want      int
	}{
		{
			name: "same metadata - duplicate (strict mode)",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := merge.NewDeduplicationMap(merge.StrategyStrict, merge.ResolutionCompleteness)
			for _, s := range tt.scrobbles {
				dm.Add(s)
			}
			if got := dm.UniqueCount(); got != tt.want {
				t.Errorf("got %d unique scrobbles, want %d", got, tt.want)
			}
		})
	}
}

func TestDeduplicationMap_Add_Relaxed(t *testing.T) {
	tests := []struct {
		name      string
		scrobbles []*models.Scrobble
		want      int
	}{
		{
			name: "same artist+track+time, different album - duplicate (relaxed)",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
				{Artist: "The Beatles", Album: "Let It Be", Track: "Come Together", UTS: 1735689600},
			},
			want: 1,
		},
		{
			name: "same artist+track+time, one missing album - duplicate (relaxed)",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
				{Artist: "The Beatles", Album: "", Track: "Come Together", UTS: 1735689600},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := merge.NewDeduplicationMap(merge.StrategyRelaxed, merge.ResolutionCompleteness)
			for _, s := range tt.scrobbles {
				dm.Add(s)
			}
			if got := dm.UniqueCount(); got != tt.want {
				t.Errorf("got %d unique scrobbles, want %d", got, tt.want)
			}
		})
	}
}

func TestDeduplicationMap_Add_MBID(t *testing.T) {
	tests := []struct {
		name      string
		scrobbles []*models.Scrobble
		want      int
	}{
		{
			name: "same MBID+timestamp - duplicate (mbid)",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Track: "Come Together", UTS: 1735689600, MBID: strPtr("12345")},
				{Artist: "Beatles", Track: "Come Together", UTS: 1735689600, MBID: strPtr("12345")},
			},
			want: 1,
		},
		{
			name: "same MBID, different timestamp - not duplicate (mbid)",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Track: "Come Together", UTS: 1735689600, MBID: strPtr("12345")},
				{Artist: "The Beatles", Track: "Come Together", UTS: 1735689700, MBID: strPtr("12345")},
			},
			want: 2,
		},
		{
			name: "no MBID - fallback to default strategy",
			scrobbles: []*models.Scrobble{
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
				{Artist: "The Beatles", Album: "Abbey Road", Track: "Come Together", UTS: 1735689600},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := merge.NewDeduplicationMap(merge.StrategyMBID, merge.ResolutionCompleteness)
			for _, s := range tt.scrobbles {
				dm.Add(s)
			}
			if got := dm.UniqueCount(); got != tt.want {
				t.Errorf("got %d unique scrobbles, want %d", got, tt.want)
			}
		})
	}
}

func TestDeduplicationMap_GetAll(t *testing.T) {
	dm := merge.NewDeduplicationMap(merge.StrategyDefault, merge.ResolutionCompleteness)
	scrobbles := []*models.Scrobble{
		{Artist: "Artist1", Album: "Album1", Track: "Track1", UTS: 1000},
		{Artist: "Artist2", Album: "Album2", Track: "Track2", UTS: 2000},
		{Artist: "Artist1", Album: "Album1", Track: "Track1", UTS: 1000}, // Duplicate
	}

	for _, s := range scrobbles {
		dm.Add(s)
	}

	all := dm.GetAll()
	if len(all) != 2 {
		t.Errorf("got %d scrobbles, want 2", len(all))
	}
}

func TestDeduplicationMap_Stats(t *testing.T) {
	dm := merge.NewDeduplicationMap(merge.StrategyDefault, merge.ResolutionCompleteness)
	scrobbles := []*models.Scrobble{
		{Artist: "Artist1", Album: "Album1", Track: "Track1", UTS: 1000},
		{Artist: "Artist2", Album: "Album2", Track: "Track2", UTS: 2000},
		{Artist: "Artist1", Album: "Album1", Track: "Track1", UTS: 1000}, // Duplicate
	}

	for _, s := range scrobbles {
		dm.Add(s)
	}

	if dm.TotalAdded() != 3 {
		t.Errorf("got %d total added, want 3", dm.TotalAdded())
	}

	if dm.UniqueCount() != 2 {
		t.Errorf("got %d unique, want 2", dm.UniqueCount())
	}

	if dm.DuplicateCount() != 1 {
		t.Errorf("got %d duplicates, want 1", dm.DuplicateCount())
	}
}
