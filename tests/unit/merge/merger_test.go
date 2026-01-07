package merge_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lastfm-reader/lastfm-sync/internal/merge"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
)

// T052 [P] [US4] Unit test for dry-run mode - verify no output written
func TestMergerDryRun(t *testing.T) {
	tests := []struct {
		name         string
		scrobbles    []*models.Scrobble
		dryRun       bool
		expectOutput bool
	}{
		{
			name: "dry-run prevents file write",
			scrobbles: []*models.Scrobble{
				{Artist: "Artist1", Track: "Track1", UTS: 1000},
				{Artist: "Artist2", Track: "Track2", UTS: 2000},
			},
			dryRun:       true,
			expectOutput: false,
		},
		{
			name: "normal mode writes file",
			scrobbles: []*models.Scrobble{
				{Artist: "Artist1", Track: "Track1", UTS: 1000},
				{Artist: "Artist2", Track: "Track2", UTS: 2000},
			},
			dryRun:       false,
			expectOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "output.json")

			// Create temp input file
			inputPath := filepath.Join(tmpDir, "input.json")
			f, err := os.Create(inputPath)
			if err != nil {
				t.Fatalf("Failed to create input file: %v", err)
			}
			defer f.Close()

			// Write scrobbles to input file
			for _, s := range tt.scrobbles {
				jsonBytes, _ := json.Marshal(s)
				if _, err := f.Write(jsonBytes); err != nil {
					t.Fatalf("Failed to write scrobble: %v", err)
				}
				f.WriteString("\n")
			}
			f.Close()

			// Configure merger
			config := merge.MergeConfig{
				Strategy:           merge.StrategyDefault,
				ConflictResolution: merge.ResolutionCompleteness,
				CheckpointInterval: 10000,
				DryRun:             tt.dryRun,
			}

			merger := merge.NewMerger(config)

			// Run merge
			result, err := merger.Merge([]string{inputPath}, outputPath)
			if err != nil {
				t.Fatalf("Merge failed: %v", err)
			}

			// Verify result stats are still accurate
			if result.Stats.TotalScrobbles != len(tt.scrobbles) {
				t.Errorf("Expected %d total scrobbles, got %d",
					len(tt.scrobbles), result.Stats.TotalScrobbles)
			}

			// Check if output file exists
			_, statErr := os.Stat(outputPath)
			fileExists := statErr == nil

			if tt.expectOutput && !fileExists {
				t.Error("Expected output file to exist, but it doesn't")
			}

			if !tt.expectOutput && fileExists {
				t.Error("Expected no output file in dry-run mode, but file exists")
			}
		})
	}
}

// Test that dry-run mode still calculates accurate statistics
func TestMergerDryRunStatistics(t *testing.T) {
	tmpDir := t.TempDir()

	// Create input file with duplicates
	inputPath := filepath.Join(tmpDir, "input.json")
	f, err := os.Create(inputPath)
	if err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}
	defer f.Close()

	scrobbles := []*models.Scrobble{
		{Artist: "Artist1", Track: "Track1", UTS: 1000},
		{Artist: "Artist1", Track: "Track1", UTS: 1000}, // duplicate
		{Artist: "Artist2", Track: "Track2", UTS: 2000},
	}

	for _, s := range scrobbles {
		jsonBytes, _ := json.Marshal(s)
		if _, err := f.Write(jsonBytes); err != nil {
			t.Fatalf("Failed to write scrobble: %v", err)
		}
		f.WriteString("\n")
	}
	f.Close()

	config := merge.MergeConfig{
		Strategy:           merge.StrategyDefault,
		ConflictResolution: merge.ResolutionCompleteness,
		CheckpointInterval: 10000,
		DryRun:             true,
	}

	merger := merge.NewMerger(config)
	result, err := merger.Merge([]string{inputPath}, filepath.Join(tmpDir, "output.json"))
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Verify statistics are calculated correctly
	if result.Stats.TotalScrobbles != 3 {
		t.Errorf("Expected 3 total scrobbles, got %d", result.Stats.TotalScrobbles)
	}

	if result.Stats.UniqueScrobbles != 2 {
		t.Errorf("Expected 2 unique scrobbles, got %d", result.Stats.UniqueScrobbles)
	}

	if result.Stats.Duplicates != 1 {
		t.Errorf("Expected 1 duplicate, got %d", result.Stats.Duplicates)
	}
}
