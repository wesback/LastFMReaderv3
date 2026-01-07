package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lastfm-reader/lastfm-sync/internal/normalize"
)

func TestScrobbleNDJSON(t *testing.T) {
	mbid := "12345678-1234-1234-1234-123456789012"
	raw := json.RawMessage(`{"artist":"Radiohead","track":"Idioteque"}`)

	scrobble := NewScrobble("alice", "Radiohead", "Idioteque", "Kid A", 971136000, &mbid, raw, nil)

	data, err := json.Marshal(scrobble)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify it's valid JSON
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Check required fields
	if unmarshaled["username"] != "alice" {
		t.Error("username mismatch")
	}
	if unmarshaled["artist"] != "Radiohead" {
		t.Error("artist mismatch")
	}
	if unmarshaled["track"] != "Idioteque" {
		t.Error("track mismatch")
	}
	// Check normalized_title field exists
	if normalized, exists := unmarshaled["normalized_title"]; !exists {
		t.Error("normalized_title field should exist")
	} else if normalized != "Idioteque" {
		t.Errorf("normalized_title should be 'Idioteque', got %v", normalized)
	}
	if unmarshaled["album"] != "Kid A" {
		t.Error("album mismatch")
	}
	if unmarshaled["uts"] != float64(971136000) {
		t.Error("uts mismatch")
	}
	if unmarshaled["mbid"] != mbid {
		t.Error("mbid mismatch")
	}
	if unmarshaled["source"] != "lastfm" {
		t.Error("source should be 'lastfm'")
	}
	if unmarshaled["ingested_at"] == nil {
		t.Error("ingested_at should be set")
	}
}

func TestScrobbleOmitNullMBID(t *testing.T) {
	raw := json.RawMessage(`{}`)
	scrobble := NewScrobble("alice", "Artist", "Track", "Album", 123456789, nil, raw, nil)

	data, err := json.Marshal(scrobble)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// mbid should be omitted when nil
	if _, exists := unmarshaled["mbid"]; exists && unmarshaled["mbid"] != nil {
		t.Error("mbid should be omitted when nil")
	}
}

func TestWatermark(t *testing.T) {
	wm := NewWatermark("alice", 1719792720, "sync-20251030-abc123")

	if wm.Username != "alice" {
		t.Error("username mismatch")
	}
	if wm.MaxUTS != 1719792720 {
		t.Error("max_uts mismatch")
	}
	if wm.SyncID != "sync-20251030-abc123" {
		t.Error("sync_id mismatch")
	}

	// updated_at should be recent
	updatedTime, err := time.Parse(time.RFC3339, wm.UpdatedAt)
	if err != nil {
		t.Fatalf("UpdatedAt parse failed: %v", err)
	}
	now := time.Now().UTC()
	if updatedTime.After(now.Add(1*time.Second)) || updatedTime.Before(now.Add(-1*time.Second)) {
		t.Error("updated_at should be recent")
	}
}

// T004: TestFormatTimestamp_Valid tests conversion of valid Unix timestamp
func TestFormatTimestamp_Valid(t *testing.T) {
	uts := int64(1704556800) // 2024-01-06T16:00:00Z
	expected := "2024-01-06T16:00:00Z"

	result := formatTimestamp(uts)

	if result != expected {
		t.Errorf("formatTimestamp(%d) = %q, want %q", uts, result, expected)
	}
}

// T005: TestFormatTimestamp_Zero tests handling of zero timestamp
func TestFormatTimestamp_Zero(t *testing.T) {
	uts := int64(0)
	expected := ""

	result := formatTimestamp(uts)

	if result != expected {
		t.Errorf("formatTimestamp(%d) = %q, want %q", uts, result, expected)
	}
}

// T006: TestFormatTimestamp_Negative tests handling of negative timestamp
func TestFormatTimestamp_Negative(t *testing.T) {
	uts := int64(-1000)
	expected := ""

	result := formatTimestamp(uts)

	if result != expected {
		t.Errorf("formatTimestamp(%d) = %q, want %q", uts, result, expected)
	}
}

// T007: TestFormatTimestamp_Large tests handling of large timestamp (2038+)
func TestFormatTimestamp_Large(t *testing.T) {
	uts := int64(2147483648) // 2038-01-19T03:14:08Z (beyond int32 max)
	expected := "2038-01-19T03:14:08Z"

	result := formatTimestamp(uts)

	if result != expected {
		t.Errorf("formatTimestamp(%d) = %q, want %q", uts, result, expected)
	}
}

// T008: TestNewScrobble_LocalTime tests that LocalTime field is populated
func TestNewScrobble_LocalTime(t *testing.T) {
	raw := json.RawMessage(`{}`)
	uts := int64(1704556800) // 2024-01-06T16:00:00Z

	scrobble := NewScrobble("alice", "Artist", "Track", "Album", uts, nil, raw, nil)

	expectedLocalTime := "2024-01-06T16:00:00Z"
	if scrobble.LocalTime != expectedLocalTime {
		t.Errorf("LocalTime = %q, want %q", scrobble.LocalTime, expectedLocalTime)
	}
}

// T009: TestScrobble_MarshalJSON_LocalTime tests that local_time is in JSON output
func TestScrobble_MarshalJSON_LocalTime(t *testing.T) {
	raw := json.RawMessage(`{}`)
	uts := int64(1704556800)

	scrobble := NewScrobble("alice", "Artist", "Track", "Album", uts, nil, raw, nil)

	data, err := json.Marshal(scrobble)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify local_time field exists
	localTime, exists := unmarshaled["local_time"]
	if !exists {
		t.Error("local_time field should exist in JSON output")
	}

	// Verify local_time has expected value
	expectedLocalTime := "2024-01-06T16:00:00Z"
	if localTime != expectedLocalTime {
		t.Errorf("local_time = %v, want %q", localTime, expectedLocalTime)
	}
}

// TestNewScrobble_WithNormalization tests that NewScrobble() correctly populates normalized_title
func TestNewScrobble_WithNormalization(t *testing.T) {
	tests := []struct {
		name               string
		track              string
		expectedNormalized string
	}{
		{"clean title", "Bohemian Rhapsody", "Bohemian Rhapsody"},
		{"remastered", "Bohemian Rhapsody - Remastered 2011", "Bohemian Rhapsody"},
		{"live", "Hotel California - Live", "Hotel California"},
		{"featuring", "Song (feat. Artist)", "Song"},
		{"multiple", "Track - Live (2023 Remaster)", "Track"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := json.RawMessage(`{}`)
			scrobble := NewScrobble("user", "Queen", tt.track, "Album", 123456789, nil, raw, nil)

			if scrobble.Track != tt.track {
				t.Errorf("Track = %q, want %q", scrobble.Track, tt.track)
			}
			if scrobble.NormalizedTitle != tt.expectedNormalized {
				t.Errorf("NormalizedTitle = %q, want %q", scrobble.NormalizedTitle, tt.expectedNormalized)
			}
		})
	}
}

// TestScrobble_JSONMarshalWithNormalization tests that normalized_title is included in JSON output
func TestScrobble_JSONMarshalWithNormalization(t *testing.T) {
	raw := json.RawMessage(`{"test":"data"}`)
	scrobble := NewScrobble("alice", "Queen", "Song - Remastered 2011", "Greatest Hits", 123456789, nil, raw, nil)

	data, err := json.Marshal(scrobble)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify both track and normalized_title exist
	if track, exists := unmarshaled["track"]; !exists {
		t.Error("track field should exist")
	} else if track != "Song - Remastered 2011" {
		t.Errorf("track = %v, want 'Song - Remastered 2011'", track)
	}

	if normalized, exists := unmarshaled["normalized_title"]; !exists {
		t.Error("normalized_title field should exist")
	} else if normalized != "Song" {
		t.Errorf("normalized_title = %v, want 'Song'", normalized)
	}
}

// TestNormalization_Disabled tests behavior when normalization is disabled
func TestNormalization_Disabled(t *testing.T) {
	// Save original state
	originalState := normalize.IsEnabled()
	defer normalize.SetEnabled(originalState)

	// Disable normalization
	normalize.SetEnabled(false)

	raw := json.RawMessage(`{}`)
	track := "Song - Remastered 2011"
	scrobble := NewScrobble("user", "Artist", track, "Album", 123456789, nil, raw, nil)

	// When disabled, normalized_title should equal original track
	if scrobble.NormalizedTitle != track {
		t.Errorf("With normalization disabled, NormalizedTitle should equal Track (%q), got %q",
			track, scrobble.NormalizedTitle)
	}
}
