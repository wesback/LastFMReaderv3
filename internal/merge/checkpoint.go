package merge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MergeCheckpoint represents the state of a merge operation for resumability
// Implements the checkpoint data model from spec.md
type MergeCheckpoint struct {
	// Metadata
	Version int `json:"version"` // Checkpoint format version (currently 1)

	// Configuration that must match on resume
	Strategy           DeduplicationStrategy `json:"strategy"`
	ConflictResolution ConflictResolution    `json:"conflict_resolution"`

	// Input files
	InputFiles     []string `json:"input_files"`     // All input files to process
	ProcessedFiles []string `json:"processed_files"` // Files fully processed
	CurrentFile    string   `json:"current_file"`    // File currently being processed
	CurrentLine    int      `json:"current_line"`    // Line number in current file

	// Progress tracking
	TotalScrobbles  int `json:"total_scrobbles"`  // Total scrobbles processed so far
	UniqueScrobbles int `json:"unique_scrobbles"` // Unique scrobbles so far
	Duplicates      int `json:"duplicates"`       // Duplicates found so far
	SkippedLines    int `json:"skipped_lines"`    // Invalid lines skipped
}

const CheckpointVersion = 1

// T078: Save writes checkpoint to disk using atomic write (temp + rename)
func (c *MergeCheckpoint) Save(path string) error {
	// Validate checkpoint before saving
	if c.Version != CheckpointVersion {
		return fmt.Errorf("invalid checkpoint version: %d (expected %d)", c.Version, CheckpointVersion)
	}

	// Marshal checkpoint to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	// Create temp file in same directory for atomic rename
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".checkpoint-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write checkpoint data: %w", err)
	}

	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync checkpoint: %w", err)
	}
	tmpFile.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename checkpoint: %w", err)
	}

	return nil
}

// T079: LoadCheckpoint reads checkpoint from disk with version validation
func LoadCheckpoint(path string) (*MergeCheckpoint, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("checkpoint file not found: %s", path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint: %w", err)
	}

	// Unmarshal JSON
	var checkpoint MergeCheckpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint: %w", err)
	}

	// T075: Validate version
	if checkpoint.Version != CheckpointVersion {
		return nil, fmt.Errorf("unsupported checkpoint version: %d (expected %d)", checkpoint.Version, CheckpointVersion)
	}

	return &checkpoint, nil
}

// T086: ValidateConfig ensures checkpoint matches current merge configuration
func (c *MergeCheckpoint) ValidateConfig(config MergeConfig) error {
	if c.Strategy != config.Strategy {
		return fmt.Errorf("checkpoint strategy mismatch: checkpoint uses %s, config uses %s", c.Strategy, config.Strategy)
	}

	if c.ConflictResolution != config.ConflictResolution {
		return fmt.Errorf("checkpoint conflict resolution mismatch: checkpoint uses %s, config uses %s", c.ConflictResolution, config.ConflictResolution)
	}

	return nil
}

// DeleteCheckpoint removes checkpoint file if it exists
func DeleteCheckpoint(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Already deleted or never existed
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	return nil
}
