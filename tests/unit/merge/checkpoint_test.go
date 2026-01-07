package merge_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lastfm-reader/lastfm-sync/internal/merge"
)

// T074 [P] [US6] Unit test for checkpoint save/load round-trip
func TestCheckpointSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointPath := filepath.Join(tmpDir, "checkpoint.json")

	// Create a checkpoint
	original := &merge.MergeCheckpoint{
		Version:            1,
		Strategy:           merge.StrategyDefault,
		ConflictResolution: merge.ResolutionCompleteness,
		InputFiles:         []string{"file1.json", "file2.json", "file3.json"},
		ProcessedFiles:     []string{"file1.json", "file2.json"},
		CurrentFile:        "file3.json",
		CurrentLine:        12345,
		TotalScrobbles:     50000,
		UniqueScrobbles:    48000,
		Duplicates:         2000,
		SkippedLines:       100,
	}

	// Save checkpoint
	if err := original.Save(checkpointPath); err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		t.Fatal("Checkpoint file was not created")
	}

	// Load checkpoint
	loaded, err := merge.LoadCheckpoint(checkpointPath)
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	// Verify all fields match
	if loaded.Version != original.Version {
		t.Errorf("Version mismatch: expected %d, got %d", original.Version, loaded.Version)
	}
	if loaded.Strategy != original.Strategy {
		t.Errorf("Strategy mismatch: expected %s, got %s", original.Strategy, loaded.Strategy)
	}
	if loaded.ConflictResolution != original.ConflictResolution {
		t.Errorf("ConflictResolution mismatch: expected %s, got %s", original.ConflictResolution, loaded.ConflictResolution)
	}
	if len(loaded.InputFiles) != len(original.InputFiles) {
		t.Errorf("InputFiles length mismatch: expected %d, got %d", len(original.InputFiles), len(loaded.InputFiles))
	}
	if loaded.CurrentFile != original.CurrentFile {
		t.Errorf("CurrentFile mismatch: expected %s, got %s", original.CurrentFile, loaded.CurrentFile)
	}
	if loaded.CurrentLine != original.CurrentLine {
		t.Errorf("CurrentLine mismatch: expected %d, got %d", original.CurrentLine, loaded.CurrentLine)
	}
	if loaded.TotalScrobbles != original.TotalScrobbles {
		t.Errorf("TotalScrobbles mismatch: expected %d, got %d", original.TotalScrobbles, loaded.TotalScrobbles)
	}
	if loaded.UniqueScrobbles != original.UniqueScrobbles {
		t.Errorf("UniqueScrobbles mismatch: expected %d, got %d", original.UniqueScrobbles, loaded.UniqueScrobbles)
	}
}

// T075 [P] [US6] Unit test for checkpoint version validation
func TestCheckpointVersionValidation(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointPath := filepath.Join(tmpDir, "checkpoint.json")

	tests := []struct {
		name        string
		version     int
		expectError bool
	}{
		{
			name:        "valid version 1",
			version:     1,
			expectError: false,
		},
		{
			name:        "invalid version 0",
			version:     0,
			expectError: true,
		},
		{
			name:        "invalid version 2",
			version:     2,
			expectError: true,
		},
		{
			name:        "invalid version -1",
			version:     -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create checkpoint with specific version
			checkpoint := &merge.MergeCheckpoint{
				Version:            tt.version,
				Strategy:           merge.StrategyDefault,
				ConflictResolution: merge.ResolutionCompleteness,
				InputFiles:         []string{"file1.json"},
				ProcessedFiles:     []string{},
				CurrentFile:        "file1.json",
				CurrentLine:        0,
			}

			// Write directly to bypass Save() validation
			data, _ := json.MarshalIndent(checkpoint, "", "  ")
			if err := os.WriteFile(checkpointPath, data, 0644); err != nil {
				t.Fatalf("Failed to write checkpoint file: %v", err)
			}

			// Try to load
			_, err := merge.LoadCheckpoint(checkpointPath)

			if tt.expectError && err == nil {
				t.Error("Expected error for invalid version, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// Test atomic write (temp + rename)
func TestCheckpointAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointPath := filepath.Join(tmpDir, "checkpoint.json")

	checkpoint := &merge.MergeCheckpoint{
		Version:            1,
		Strategy:           merge.StrategyDefault,
		ConflictResolution: merge.ResolutionCompleteness,
		InputFiles:         []string{"file1.json"},
		ProcessedFiles:     []string{},
		CurrentFile:        "file1.json",
		CurrentLine:        0,
	}

	// Save checkpoint
	if err := checkpoint.Save(checkpointPath); err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Verify no temp file remains
	tmpFiles, _ := filepath.Glob(filepath.Join(tmpDir, "*.tmp"))
	if len(tmpFiles) > 0 {
		t.Errorf("Expected no temp files, found: %v", tmpFiles)
	}

	// Verify final file exists
	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		t.Error("Checkpoint file does not exist after save")
	}
}

// Test checkpoint validation
func TestCheckpointValidation(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointPath := filepath.Join(tmpDir, "checkpoint.json")

	config := merge.MergeConfig{
		Strategy:           merge.StrategyStrict,
		ConflictResolution: merge.ResolutionFirst,
		CheckpointInterval: 10000,
	}

	tests := []struct {
		name        string
		checkpoint  *merge.MergeCheckpoint
		expectError bool
	}{
		{
			name: "matching config",
			checkpoint: &merge.MergeCheckpoint{
				Version:            1,
				Strategy:           merge.StrategyStrict,
				ConflictResolution: merge.ResolutionFirst,
				InputFiles:         []string{"file1.json"},
				ProcessedFiles:     []string{},
				CurrentFile:        "file1.json",
				CurrentLine:        0,
			},
			expectError: false,
		},
		{
			name: "mismatched strategy",
			checkpoint: &merge.MergeCheckpoint{
				Version:            1,
				Strategy:           merge.StrategyDefault, // Different!
				ConflictResolution: merge.ResolutionFirst,
				InputFiles:         []string{"file1.json"},
				ProcessedFiles:     []string{},
				CurrentFile:        "file1.json",
				CurrentLine:        0,
			},
			expectError: true,
		},
		{
			name: "mismatched conflict resolution",
			checkpoint: &merge.MergeCheckpoint{
				Version:            1,
				Strategy:           merge.StrategyStrict,
				ConflictResolution: merge.ResolutionLast, // Different!
				InputFiles:         []string{"file1.json"},
				ProcessedFiles:     []string{},
				CurrentFile:        "file1.json",
				CurrentLine:        0,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save checkpoint
			if err := tt.checkpoint.Save(checkpointPath); err != nil {
				t.Fatalf("Failed to save checkpoint: %v", err)
			}

			// Validate against config
			err := tt.checkpoint.ValidateConfig(config)

			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}
