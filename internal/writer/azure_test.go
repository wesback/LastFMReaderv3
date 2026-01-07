package writer

import (
	"testing"
	"time"
)

// TestAzureBlobPathFormatting tests the path construction logic for Azure blob paths
// Expected format: {prefix}dt=YYYY-MM-DD/{username}-YYYYMMDD-HHMMSS.ndjson
func TestAzureBlobPathFormatting(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		username string
		ts       time.Time
		want     string
	}{
		{
			name:     "simple path with prefix",
			prefix:   "lastfm/",
			username: "alice",
			ts:       time.Date(2025, 10, 30, 14, 30, 45, 0, time.UTC),
			want:     "lastfm/dt=2025-10-30/alice-20251030-143045.ndjson",
		},
		{
			name:     "no prefix",
			prefix:   "",
			username: "bob",
			ts:       time.Date(2026, 1, 6, 9, 15, 30, 0, time.UTC),
			want:     "dt=2026-01-06/bob-20260106-091530.ndjson",
		},
		{
			name:     "prefix without trailing slash",
			prefix:   "data",
			username: "charlie",
			ts:       time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
			want:     "datadt=2025-12-31/charlie-20251231-235959.ndjson",
		},
		{
			name:     "complex prefix with subdirectories",
			prefix:   "scrobbles/prod/lastfm/",
			username: "david",
			ts:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want:     "scrobbles/prod/lastfm/dt=2025-01-01/david-20250101-000000.ndjson",
		},
		{
			name:     "username with special characters",
			prefix:   "lastfm/",
			username: "user.name_123",
			ts:       time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
			want:     "lastfm/dt=2025-06-15/user.name_123-20250615-120000.ndjson",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAzureBlobPath(tt.prefix, tt.username, tt.ts)
			if got != tt.want {
				t.Errorf("formatAzureBlobPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAzureBlobPathUniqueness verifies that paths generated at different times are unique
func TestAzureBlobPathUniqueness(t *testing.T) {
	prefix := "lastfm/"
	username := "testuser"

	ts1 := time.Date(2025, 10, 30, 14, 30, 45, 0, time.UTC)
	ts2 := time.Date(2025, 10, 30, 14, 30, 46, 0, time.UTC) // 1 second later
	ts3 := time.Date(2025, 10, 31, 14, 30, 45, 0, time.UTC) // Different day

	path1 := formatAzureBlobPath(prefix, username, ts1)
	path2 := formatAzureBlobPath(prefix, username, ts2)
	path3 := formatAzureBlobPath(prefix, username, ts3)

	if path1 == path2 {
		t.Errorf("Paths should be unique for different timestamps: %v == %v", path1, path2)
	}

	if path1 == path3 {
		t.Errorf("Paths should be unique for different days: %v == %v", path1, path3)
	}

	// Verify different day results in different partition
	if !containsSubstring(path1, "dt=2025-10-30") {
		t.Errorf("Path should contain date partition dt=2025-10-30, got %v", path1)
	}
	if !containsSubstring(path3, "dt=2025-10-31") {
		t.Errorf("Path should contain date partition dt=2025-10-31, got %v", path3)
	}
}

// TestAzureBlobPathDatePartitioning verifies the date partitioning logic
func TestAzureBlobPathDatePartitioning(t *testing.T) {
	prefix := "lastfm/"
	username := "alice"

	// Test various dates to ensure correct partitioning
	tests := []struct {
		date      time.Time
		partition string
	}{
		{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "dt=2025-01-01"},
		{time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC), "dt=2025-12-31"},
		{time.Date(2026, 6, 15, 12, 30, 45, 0, time.UTC), "dt=2026-06-15"},
	}

	for _, tt := range tests {
		path := formatAzureBlobPath(prefix, username, tt.date)
		if !containsSubstring(path, tt.partition) {
			t.Errorf("Path %v should contain partition %v", path, tt.partition)
		}
	}
}

// TestAzureWatermarkBlobPath tests the watermark blob path construction
// Expected format: {prefix}{username}.watermark
func TestAzureWatermarkBlobPath(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		username string
		want     string
	}{
		{
			name:     "with prefix",
			prefix:   "lastfm/",
			username: "alice",
			want:     "lastfm/alice.watermark",
		},
		{
			name:     "no prefix",
			prefix:   "",
			username: "bob",
			want:     "bob.watermark",
		},
		{
			name:     "complex prefix",
			prefix:   "scrobbles/prod/",
			username: "charlie",
			want:     "scrobbles/prod/charlie.watermark",
		},
		{
			name:     "username with special characters",
			prefix:   "data/",
			username: "user.name_123",
			want:     "data/user.name_123.watermark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatWatermarkBlobPath(tt.prefix, tt.username)
			if got != tt.want {
				t.Errorf("formatWatermarkBlobPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
