package writer

import (
	"encoding/json"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
	"testing"
)

// TestWriterInterface verifies the Writer interface is properly defined.
func TestWriterInterface(t *testing.T) {
	var w Writer = &MockWriter{}
	if w == nil {
		t.Fatal("Writer interface not implemented")
	}
}

// TestScrobbleJSON verifies models.Scrobble marshals to JSON correctly.
func TestScrobbleJSON(t *testing.T) {
	mbid := "test-mbid"
	s := models.Scrobble{
		Username:   "testuser",
		Artist:     "Artist Name",
		Track:      "Track Name",
		Album:      "Album Name",
		UTS:        1725000000,
		MBID:       &mbid,
		Source:     "last.fm",
		IngestedAt: "2025-10-30T12:00:00Z",
		Raw:        []byte(`{"nowplaying":false}`),
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled models.Scrobble
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.Username != s.Username {
		t.Errorf("Username mismatch: %q != %q", unmarshaled.Username, s.Username)
	}
	if unmarshaled.UTS != s.UTS {
		t.Errorf("UTS mismatch: %d != %d", unmarshaled.UTS, s.UTS)
	}
}

// TestScrobbleJSONOmitsNilMBID verifies MBID is omitted when nil.
func TestScrobbleJSONOmitsNilMBID(t *testing.T) {
	s := models.Scrobble{
		Username:   "testuser",
		Artist:     "Artist",
		Track:      "Track",
		UTS:        1725000000,
		MBID:       nil,
		IngestedAt: "2025-10-30T12:00:00Z",
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if string(data) != "" && len(data) > 0 {
		// Verify "mbid" key is not present
		if contains(string(data), `"mbid"`) {
			t.Error("MBID should be omitted when nil")
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if len(s[i:]) < len(substr) {
			return false
		}
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
