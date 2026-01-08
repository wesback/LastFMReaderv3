package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lastfm-reader/lastfm-sync/cmd/lastfm-sync/commands"
	"github.com/lastfm-reader/lastfm-sync/internal/logging"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileDiscovery tests file discovery with various username patterns
func TestFileDiscovery(t *testing.T) {
	tests := []struct {
		name     string
		username string
		files    []string // Files to create in temp dir
		want     []string // Expected matches (basenames)
	}{
		{
			name:     "single file match",
			username: "user1",
			files:    []string{"user1_001.ndjson"},
			want:     []string{"user1_001.ndjson"},
		},
		{
			name:     "multiple files match",
			username: "user1",
			files:    []string{"user1_001.ndjson", "user1_002.ndjson", "user2_001.ndjson"},
			want:     []string{"user1_001.ndjson", "user1_002.ndjson"},
		},
		{
			name:     "no matches",
			username: "user1",
			files:    []string{"user2_001.ndjson", "user2_002.ndjson"},
			want:     []string{},
		},
		{
			name:     "files with different extensions ignored",
			username: "user1",
			files:    []string{"user1_001.ndjson", "user1_002.json", "user1_003.txt"},
			want:     []string{"user1_001.ndjson"},
		},
		{
			name:     "files without underscore separator ignored",
			username: "user1",
			files:    []string{"user1_001.ndjson", "user1-002.ndjson", "user1.ndjson"},
			want:     []string{"user1_001.ndjson"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Create test files
			for _, file := range tt.files {
				path := filepath.Join(tmpDir, file)
				err := os.WriteFile(path, []byte(""), 0644)
				require.NoError(t, err)
			}

			// Change to temp directory for file discovery
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(oldWd)

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Run file discovery
			logger, _ := logging.New("error")
			files, err := commands.DiscoverLocalFiles(tt.username, logger)
			require.NoError(t, err)

			// Extract basenames for comparison
			var gotBasenames []string
			for _, f := range files {
				gotBasenames = append(gotBasenames, filepath.Base(f))
			}

			assert.ElementsMatch(t, tt.want, gotBasenames)
		})
	}
}

// TestErrorCategorization tests error type categorization
func TestErrorCategorization(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantType string
	}{
		{
			name:     "parse error",
			err:      fmt.Errorf("parse_error at line 5: invalid JSON"),
			wantType: "parse_error",
		},
		{
			name:     "missing track field",
			err:      fmt.Errorf("missing_track_field at line 10"),
			wantType: "missing_track_field",
		},
		{
			name:     "permission denied",
			err:      fmt.Errorf("permission denied: cannot write file"),
			wantType: "permission_denied",
		},
		{
			name:     "read error",
			err:      fmt.Errorf("read_error: file not found"),
			wantType: "read_error",
		},
		{
			name:     "write error",
			err:      fmt.Errorf("write_error: disk full"),
			wantType: "write_error",
		},
		{
			name:     "unknown error",
			err:      fmt.Errorf("something unexpected happened"),
			wantType: "unknown_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commands.CategorizeError(tt.err)
			assert.Equal(t, tt.wantType, got)
		})
	}
}

// TestProcessFileNormalization tests the core normalization logic
func TestProcessFileNormalization(t *testing.T) {
	tests := []struct {
		name           string
		input          []models.Scrobble
		dryRun         bool
		wantUpdated    bool
		wantNormalized map[string]string // track -> expected normalized_title
	}{
		{
			name: "normalize removes remaster annotation",
			input: []models.Scrobble{
				{Artist: "The Beatles", Track: "Hey Jude - Remastered 2009", NormalizedTitle: ""},
			},
			dryRun:      false,
			wantUpdated: true,
			wantNormalized: map[string]string{
				"Hey Jude - Remastered 2009": "Hey Jude",
			},
		},
		{
			name: "normalize removes live annotation",
			input: []models.Scrobble{
				{Artist: "Queen", Track: "Bohemian Rhapsody - Live at Wembley", NormalizedTitle: ""},
			},
			dryRun:      false,
			wantUpdated: true,
			wantNormalized: map[string]string{
				"Bohemian Rhapsody - Live at Wembley": "Bohemian Rhapsody",
			},
		},
		{
			name: "normalize removes featuring annotation",
			input: []models.Scrobble{
				{Artist: "Pink Floyd", Track: "Comfortably Numb (feat. David Gilmour)", NormalizedTitle: ""},
			},
			dryRun:      false,
			wantUpdated: true,
			wantNormalized: map[string]string{
				"Comfortably Numb (feat. David Gilmour)": "Comfortably Numb",
			},
		},
		{
			name: "already normalized - no update",
			input: []models.Scrobble{
				{Artist: "The Rolling Stones", Track: "Paint It Black", NormalizedTitle: "Paint It Black"},
			},
			dryRun:      false,
			wantUpdated: false,
			wantNormalized: map[string]string{
				"Paint It Black": "Paint It Black",
			},
		},
		{
			name: "dry-run mode returns updated but doesn't write",
			input: []models.Scrobble{
				{Artist: "The Beatles", Track: "Hey Jude - Remastered 2009", NormalizedTitle: ""},
			},
			dryRun:      true,
			wantUpdated: true,
			wantNormalized: map[string]string{
				"Hey Jude - Remastered 2009": "Hey Jude",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with input data
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.ndjson")

			f, err := os.Create(tmpFile)
			require.NoError(t, err)
			for _, scrobble := range tt.input {
				data, _ := json.Marshal(scrobble)
				f.Write(data)
				f.WriteString("\n")
			}
			f.Close()

			// Process file
			logger, _ := logging.New("error")
			ctx := context.Background()
			updated, err := commands.ProcessFile(ctx, tmpFile, tt.dryRun, logger)

			require.NoError(t, err)
			assert.Equal(t, tt.wantUpdated, updated)

			// In dry-run mode, file shouldn't be modified
			if tt.dryRun {
				// Read file and verify it's unchanged
				content, err := os.ReadFile(tmpFile)
				require.NoError(t, err)

				// Parse and check normalized_title is still empty
				lines := string(content)
				var scrobble models.Scrobble
				json.Unmarshal([]byte(lines), &scrobble)
				assert.Empty(t, scrobble.NormalizedTitle, "dry-run should not modify file")
			} else if tt.wantUpdated {
				// Read file and verify normalized_title was updated
				content, err := os.ReadFile(tmpFile)
				require.NoError(t, err)

				// Parse each line and check normalized_title
				lines := string(content)
				var scrobble models.Scrobble
				json.Unmarshal([]byte(lines), &scrobble)

				expectedNormalized := tt.wantNormalized[scrobble.Track]
				assert.Equal(t, expectedNormalized, scrobble.NormalizedTitle)
			}
		})
	}
}

// TestProcessFileErrors tests error handling scenarios
func TestProcessFileErrors(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		wantErr     bool
		errContains string
	}{
		{
			name:        "malformed JSON",
			fileContent: "{this is not valid json\n",
			wantErr:     true,
			errContains: "parse_error",
		},
		{
			name:        "missing track field",
			fileContent: `{"artist":"Test Artist","timestamp":1609459200}` + "\n",
			wantErr:     true,
			errContains: "missing_track_field",
		},
		{
			name:        "valid JSON",
			fileContent: `{"artist":"Test","track":"Test Track","timestamp":1609459200,"normalized_title":""}` + "\n",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.ndjson")

			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			// Process file
			logger, _ := logging.New("error")
			ctx := context.Background()
			_, err = commands.ProcessFile(ctx, tmpFile, false, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
