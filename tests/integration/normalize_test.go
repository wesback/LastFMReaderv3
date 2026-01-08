package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lastfm-reader/lastfm-sync/cmd/lastfm-sync/commands"
	"github.com/lastfm-reader/lastfm-sync/internal/logging"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNormalizeLocalStorage tests end-to-end normalization on local filesystem
func TestNormalizeLocalStorage(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create test files with scrobbles needing normalization
	testFiles := map[string][]models.Scrobble{
		"testuser_001.ndjson": {
			{Artist: "The Beatles", Track: "Hey Jude - Remastered 2009", NormalizedTitle: ""},
			{Artist: "Queen", Track: "Bohemian Rhapsody - Live at Wembley", NormalizedTitle: ""},
		},
		"testuser_002.ndjson": {
			{Artist: "Pink Floyd", Track: "Comfortably Numb (feat. David Gilmour)", NormalizedTitle: ""},
			{Artist: "Led Zeppelin", Track: "Stairway To Heaven - 2007 Remaster", NormalizedTitle: "old_value"},
		},
	}

	for filename, scrobbles := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		f, err := os.Create(filePath)
		require.NoError(t, err)

		for _, scrobble := range scrobbles {
			data, _ := json.Marshal(scrobble)
			f.Write(data)
			f.WriteString("\n")
		}
		f.Close()
	}

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run normalize command
	logger, err := logging.New("error")
	require.NoError(t, err)

	files, err := commands.DiscoverLocalFiles("testuser", logger)
	require.NoError(t, err)
	assert.Len(t, files, 2)

	ctx := context.Background()
	updatedCount := 0
	for _, file := range files {
		updated, err := commands.ProcessFile(ctx, file, false, logger)
		require.NoError(t, err)
		if updated {
			updatedCount++
		}
	}

	assert.Equal(t, 2, updatedCount, "both files should be updated")

	// Verify normalized_title fields were updated
	expectedNormalized := map[string]string{
		"Hey Jude - Remastered 2009":             "Hey Jude",
		"Bohemian Rhapsody - Live at Wembley":    "Bohemian Rhapsody",
		"Comfortably Numb (feat. David Gilmour)": "Comfortably Numb",
		"Stairway To Heaven - 2007 Remaster":     "Stairway To Heaven",
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err)

		// Parse each line
		lines := string(content)
		for i, line := range splitLines(lines) {
			if line == "" {
				continue
			}
			var scrobble models.Scrobble
			err := json.Unmarshal([]byte(line), &scrobble)
			require.NoError(t, err, "file %s line %d", file, i+1)

			expected, exists := expectedNormalized[scrobble.Track]
			if exists {
				assert.Equal(t, expected, scrobble.NormalizedTitle,
					"file %s line %d: track '%s'", file, i+1, scrobble.Track)
			}
		}
	}
}

// TestNormalizeDryRun tests that dry-run mode doesn't modify files
func TestNormalizeDryRun(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create test file
	filePath := filepath.Join(tmpDir, "testuser_001.ndjson")
	scrobbles := []models.Scrobble{
		{Artist: "The Beatles", Track: "Hey Jude - Remastered 2009", NormalizedTitle: ""},
	}

	f, err := os.Create(filePath)
	require.NoError(t, err)
	for _, scrobble := range scrobbles {
		data, _ := json.Marshal(scrobble)
		f.Write(data)
		f.WriteString("\n")
	}
	f.Close()

	// Read original content
	originalContent, err := os.ReadFile(filePath)
	require.NoError(t, err)

	// Run normalize in dry-run mode
	logger, _ := logging.New("error")
	ctx := context.Background()
	updated, err := commands.ProcessFile(ctx, filePath, true, logger)

	require.NoError(t, err)
	assert.True(t, updated, "should report as updated")

	// Verify file was NOT modified
	currentContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, currentContent, "file should not be modified in dry-run mode")
}

// TestNormalizeUnchangedFiles tests handling of files where normalized_title is already correct
func TestNormalizeUnchangedFiles(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create test file with already-normalized titles
	filePath := filepath.Join(tmpDir, "testuser_001.ndjson")
	scrobbles := []models.Scrobble{
		{Artist: "The Rolling Stones", Track: "Paint It Black", NormalizedTitle: "Paint It Black"},
		{Artist: "Nirvana", Track: "Smells Like Teen Spirit", NormalizedTitle: "Smells Like Teen Spirit"},
	}

	f, err := os.Create(filePath)
	require.NoError(t, err)
	for _, scrobble := range scrobbles {
		data, _ := json.Marshal(scrobble)
		f.Write(data)
		f.WriteString("\n")
	}
	f.Close()

	// Run normalize
	logger, _ := logging.New("error")
	ctx := context.Background()
	updated, err := commands.ProcessFile(ctx, filePath, false, logger)

	require.NoError(t, err)
	assert.False(t, updated, "should report as unchanged")

	// Run again to test idempotency
	updated, err = commands.ProcessFile(ctx, filePath, false, logger)
	require.NoError(t, err)
	assert.False(t, updated, "second run should also report unchanged")
}

// TestNormalizeErrorHandling tests processing continues after individual file errors
func TestNormalizeErrorHandling(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create valid file
	validFile := filepath.Join(tmpDir, "testuser_001.ndjson")
	f1, _ := os.Create(validFile)
	scrobble := models.Scrobble{Artist: "Test", Track: "Test Track", NormalizedTitle: ""}
	data, _ := json.Marshal(scrobble)
	f1.Write(data)
	f1.WriteString("\n")
	f1.Close()

	// Create file with malformed JSON
	malformedFile := filepath.Join(tmpDir, "testuser_002.ndjson")
	os.WriteFile(malformedFile, []byte("{this is not valid json\n"), 0644)

	// Create file with missing track field
	missingTrackFile := filepath.Join(tmpDir, "testuser_003.ndjson")
	os.WriteFile(missingTrackFile, []byte(`{"artist":"Test","timestamp":123}`+"\n"), 0644)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Discover and process all files
	logger, _ := logging.New("error")
	files, err := commands.DiscoverLocalFiles("testuser", logger)
	require.NoError(t, err)
	assert.Len(t, files, 3)

	ctx := context.Background()
	successCount := 0
	errorCount := 0
	var errors []error

	for _, file := range files {
		_, err := commands.ProcessFile(ctx, file, false, logger)
		if err != nil {
			errorCount++
			errors = append(errors, err)
		} else {
			successCount++
		}
	}

	// Verify processing continued despite errors
	assert.Equal(t, 1, successCount, "valid file should process successfully")
	assert.Equal(t, 2, errorCount, "two files should fail")

	// Verify error types
	assert.Contains(t, errors[0].Error(), "parse_error", "first error should be parse error")
	assert.Contains(t, errors[1].Error(), "missing_track_field", "second error should be missing field")
}

// TestNormalizeAzureStorage tests end-to-end normalization on Azure Blob Storage
func TestNormalizeAzureStorage(t *testing.T) {
	// TODO: Implement integration test for Azure storage normalization
	// Requires Azure emulator or test account
	t.Skip("Azure integration test not yet implemented - requires Azure test environment")
}

// TestNormalizeProgressDisplay tests progress bar display during processing
func TestNormalizeProgressDisplay(t *testing.T) {
	// This is more of a visual/manual test since progress display is output-based
	// The functionality is tested implicitly in other integration tests
	t.Skip("Progress display is output-based - tested manually")
}

// Helper function to split lines for parsing
func splitLines(content string) []string {
	var lines []string
	current := ""
	for _, ch := range content {
		if ch == '\n' {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
