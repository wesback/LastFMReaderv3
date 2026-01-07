package writer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lastfm-reader/lastfm-sync/internal/models"
)

// TestLocalWriterCreate tests that LocalWriter creates a new file.
func TestLocalWriterCreate(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.ndjson")

	writer, err := NewLocalWriter(filePath)
	if err != nil {
		t.Fatalf("NewLocalWriter failed: %v", err)
	}
	defer writer.Close(context.Background())

	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("File not created: %v", err)
	}
}

// TestLocalWriterWriteBatch tests writing records.
func TestLocalWriterWriteBatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.ndjson")

	writer, err := NewLocalWriter(filePath)
	if err != nil {
		t.Fatalf("NewLocalWriter failed: %v", err)
	}
	defer writer.Close(context.Background())

	records := []models.Scrobble{
		{
			Username:   "testuser",
			Artist:     "Artist1",
			Track:      "Track1",
			Album:      "Album1",
			UTS:        1725000000,
			Source:     "lastfm",
			IngestedAt: "2025-10-30T12:00:00Z",
		},
		{
			Username:   "testuser",
			Artist:     "Artist2",
			Track:      "Track2",
			Album:      "Album2",
			UTS:        1725000100,
			Source:     "lastfm",
			IngestedAt: "2025-10-30T12:00:00Z",
		},
	}

	if err := writer.WriteBatch(context.Background(), records); err != nil {
		t.Fatalf("WriteBatch failed: %v", err)
	}

	if err := writer.Flush(context.Background()); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Verify file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	lines := parseNDJSON(string(content))
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	var s1, s2 models.Scrobble
	if err := json.Unmarshal([]byte(lines[0]), &s1); err != nil {
		t.Fatalf("Unmarshal line 0 failed: %v", err)
	}
	if s1.Artist != "Artist1" {
		t.Errorf("Expected Artist1, got %q", s1.Artist)
	}

	if err := json.Unmarshal([]byte(lines[1]), &s2); err != nil {
		t.Fatalf("Unmarshal line 1 failed: %v", err)
	}
	if s2.Artist != "Artist2" {
		t.Errorf("Expected Artist2, got %q", s2.Artist)
	}
}

// TestLocalWriterAppend tests appending to existing file.
func TestLocalWriterAppend(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.ndjson")

	// First batch
	writer1, err := NewLocalWriter(filePath)
	if err != nil {
		t.Fatalf("NewLocalWriter failed: %v", err)
	}

	records1 := []models.Scrobble{
		{Username: "user", Artist: "A1", Track: "T1", UTS: 1, IngestedAt: "2025-10-30T12:00:00Z"},
	}
	if err := writer1.WriteBatch(context.Background(), records1); err != nil {
		t.Fatalf("WriteBatch 1 failed: %v", err)
	}
	if err := writer1.Flush(context.Background()); err != nil {
		t.Fatalf("Flush 1 failed: %v", err)
	}
	writer1.Close(context.Background())

	// Second batch (append)
	writer2, err := NewLocalWriter(filePath)
	if err != nil {
		t.Fatalf("NewLocalWriter 2 failed: %v", err)
	}
	defer writer2.Close(context.Background())

	records2 := []models.Scrobble{
		{Username: "user", Artist: "A2", Track: "T2", UTS: 2, IngestedAt: "2025-10-30T12:00:00Z"},
	}
	if err := writer2.WriteBatch(context.Background(), records2); err != nil {
		t.Fatalf("WriteBatch 2 failed: %v", err)
	}
	if err := writer2.Flush(context.Background()); err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}

	// Verify file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	lines := parseNDJSON(string(content))
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

// TestLocalWriterContextCancellation tests that context cancellation is respected.
func TestLocalWriterContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.ndjson")

	writer, err := NewLocalWriter(filePath)
	if err != nil {
		t.Fatalf("NewLocalWriter failed: %v", err)
	}
	defer writer.Close(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	records := []models.Scrobble{
		{Username: "user", Artist: "A1", Track: "T1", UTS: 1, IngestedAt: "2025-10-30T12:00:00Z"},
	}

	err = writer.WriteBatch(ctx, records)
	if err == nil {
		t.Error("Expected context error, got nil")
	}
}

// TestLocalWriterCloseIdempotent tests that Close() can be called multiple times.
func TestLocalWriterCloseIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.ndjson")

	writer, err := NewLocalWriter(filePath)
	if err != nil {
		t.Fatalf("NewLocalWriter failed: %v", err)
	}

	if err := writer.Close(context.Background()); err != nil {
		t.Fatalf("First Close failed: %v", err)
	}

	// Second Close should not error
	if err := writer.Close(context.Background()); err != nil {
		t.Fatalf("Second Close failed: %v", err)
	}
}

// TestLocalWriterClosedError tests that operations on closed writer fail gracefully.
func TestLocalWriterClosedError(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.ndjson")

	writer, err := NewLocalWriter(filePath)
	if err != nil {
		t.Fatalf("NewLocalWriter failed: %v", err)
	}

	writer.Close(context.Background())

	records := []models.Scrobble{
		{Username: "user", Artist: "A1", Track: "T1", UTS: 1, IngestedAt: "2025-10-30T12:00:00Z"},
	}

	err = writer.WriteBatch(context.Background(), records)
	if err == nil {
		t.Error("Expected error on closed writer, got nil")
	}
}

// TestLocalWriterFlushSync tests that Flush performs fsync.
func TestLocalWriterFlushSync(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.ndjson")

	writer, err := NewLocalWriter(filePath)
	if err != nil {
		t.Fatalf("NewLocalWriter failed: %v", err)
	}
	defer writer.Close(context.Background())

	records := []models.Scrobble{
		{Username: "user", Artist: "A1", Track: "T1", UTS: 1, IngestedAt: "2025-10-30T12:00:00Z"},
	}

	if err := writer.WriteBatch(context.Background(), records); err != nil {
		t.Fatalf("WriteBatch failed: %v", err)
	}

	// Before flush, data might not be on disk
	// After flush, data should be synced
	if err := writer.Flush(context.Background()); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Verify data is on disk by reading it
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	lines := parseNDJSON(string(content))
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}
}

// TestLocalWriterTimeout tests that Flush respects timeout.
func TestLocalWriterTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.ndjson")

	writer, err := NewLocalWriter(filePath)
	if err != nil {
		t.Fatalf("NewLocalWriter failed: %v", err)
	}
	defer writer.Close(context.Background())

	records := []models.Scrobble{
		{Username: "user", Artist: "A1", Track: "T1", UTS: 1, IngestedAt: "2025-10-30T12:00:00Z"},
	}

	if err := writer.WriteBatch(context.Background(), records); err != nil {
		t.Fatalf("WriteBatch failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Flush should complete quickly
	if err := writer.Flush(ctx); err != nil {
		t.Fatalf("Flush with timeout failed: %v", err)
	}
}

// parseNDJSON splits NDJSON content into individual lines (non-empty).
func parseNDJSON(content string) []string {
	var lines []string
	var current string
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			if current != "" {
				lines = append(lines, current)
			}
			current = ""
		} else {
			current += string(content[i])
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
