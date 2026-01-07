package merge

import (
	"errors"
	"time"
)

// MergeConfig contains all configuration for a merge operation
type MergeConfig struct {
	// User configuration
	Username string `json:"username"` // Last.fm username (used for output filename)

	// Input/Output configuration (shared storage backend)
	InputPatterns  []string     `json:"input_patterns"`         // Glob patterns for input files (local) or blob patterns (Azure)
	InputFiles     []string     `json:"input_files"`            // Resolved input file paths
	Recursive      bool         `json:"recursive"`              // Recursively search subdirectories (local only)
	OutputPath     string       `json:"output_path"`            // Output file path (local or Azure blob name)
	StorageBackend string       `json:"storage_backend"`        // "local" or "azure" (applies to both input and output)
	AzureConfig    *AzureConfig `json:"azure_config,omitempty"` // Azure config (shared for input and output)

	// Deduplication configuration
	Strategy           DeduplicationStrategy `json:"strategy"`            // Deduplication strategy
	ConflictResolution ConflictResolution    `json:"conflict_resolution"` // Conflict resolution mode

	// Performance configuration
	CheckpointInterval int    `json:"checkpoint_interval"` // Save checkpoint every N scrobbles
	CheckpointPath     string `json:"checkpoint_path"`     // Checkpoint file path
	ProgressEnabled    bool   `json:"progress_enabled"`    // Show progress bar
	BufferSize         int    `json:"buffer_size"`         // Scanner buffer size (bytes)

	// Resume configuration
	Resume bool `json:"resume"` // Resume from checkpoint

	// Display options
	DryRun      bool   `json:"dry_run"`      // Preview mode (no output written)
	Verbose     bool   `json:"verbose"`      // Enable DEBUG logging
	LogLevel    string `json:"log_level"`    // "debug", "info", "warn", "error"
	SaveSkipped string `json:"save_skipped"` // Optional path to save skipped lines for analysis
}

// DeduplicationStrategy defines how duplicates are detected
type DeduplicationStrategy string

const (
	StrategyDefault DeduplicationStrategy = "default" // Artist+Album+Title+Timestamp
	StrategyStrict  DeduplicationStrategy = "strict"  // Default + Duration
	StrategyRelaxed DeduplicationStrategy = "relaxed" // Artist+Title+Timestamp (no Album)
	StrategyMBID    DeduplicationStrategy = "mbid"    // MusicBrainz Track ID + Timestamp
)

// ConflictResolution defines how duplicate scrobbles are resolved
type ConflictResolution string

const (
	ResolutionCompleteness ConflictResolution = "completeness" // Select most complete metadata
	ResolutionFirst        ConflictResolution = "first"        // Keep first occurrence
	ResolutionLast         ConflictResolution = "last"         // Keep last occurrence
)

// AzureConfig contains Azure Blob Storage configuration
type AzureConfig struct {
	AccountName   string `json:"account_name"`   // Storage account name
	ContainerName string `json:"container_name"` // Container name
	AuthMethod    string `json:"auth_method"`    // Auth method: default, mi, connstr, key, sas
	Prefix        string `json:"prefix"`         // Blob prefix path
	ContainerURL  string `json:"container_url"`  // Full container URL (optional)
	AccountKey    string `json:"account_key"`    // Account key (for key auth)
	SASToken      string `json:"sas_token"`      // SAS token (for sas auth)
}

// Validate checks if config is valid
func (c *MergeConfig) Validate() error {
	if len(c.InputPatterns) == 0 && len(c.InputFiles) == 0 {
		return errors.New("no input patterns or files specified")
	}
	if c.OutputPath == "" {
		return errors.New("output path is required")
	}
	if c.StorageBackend != "local" && c.StorageBackend != "azure" {
		return errors.New("storage backend must be 'local' or 'azure'")
	}
	if c.StorageBackend == "azure" && c.AzureConfig == nil {
		return errors.New("azure_config required when storage_backend is 'azure'")
	}
	if c.CheckpointInterval <= 0 {
		return errors.New("checkpoint_interval must be positive")
	}

	// Validate strategy
	switch c.Strategy {
	case StrategyDefault, StrategyStrict, StrategyRelaxed, StrategyMBID:
		// Valid
	default:
		return errors.New("invalid strategy: must be default, strict, relaxed, or mbid")
	}

	// Validate conflict resolution
	switch c.ConflictResolution {
	case ResolutionCompleteness, ResolutionFirst, ResolutionLast:
		// Valid
	default:
		return errors.New("invalid conflict resolution: must be completeness, first, or last")
	}

	return nil
}

// DefaultConfig returns a MergeConfig with default values
func DefaultConfig() *MergeConfig {
	return &MergeConfig{
		StorageBackend:     "local",
		Strategy:           StrategyDefault,
		ConflictResolution: ResolutionCompleteness,
		CheckpointInterval: 10000, // Every 10K scrobbles
		ProgressEnabled:    true,
		BufferSize:         128 * 1024, // 128KB
		LogLevel:           "info",
		CheckpointPath:     ".merge-checkpoint.json",
	}
}

// MergeStats tracks statistics for a merge operation
type MergeStats struct {
	// File counts
	TotalFiles     int `json:"total_files"`     // Total input files discovered
	ProcessedFiles int `json:"processed_files"` // Files fully processed

	// Scrobble counts
	TotalScrobbles  int `json:"total_scrobbles"`  // Total scrobbles read
	UniqueScrobbles int `json:"unique_scrobbles"` // Unique scrobbles after deduplication
	Duplicates      int `json:"duplicates"`       // Duplicate scrobbles removed

	// Error counts
	SkippedLines     int `json:"skipped_lines"`     // Lines with JSON parse errors
	SkippedScrobbles int `json:"skipped_scrobbles"` // Scrobbles failing validation

	// Conflict tracking
	Conflicts           int            `json:"conflicts"`                       // Duplicate keys resolved
	ConflictsByStrategy map[string]int `json:"conflicts_by_strategy,omitempty"` // Conflicts per strategy

	// Performance metrics
	StartTime time.Time `json:"start_time"`       // Merge start time
	EndTime   time.Time `json:"end_time"`         // Merge end time
	Duration  float64   `json:"duration_seconds"` // Total duration in seconds
	Rate      float64   `json:"rate_per_second"`  // Scrobbles processed per second

	// Date range tracking (for dry-run preview)
	EarliestTimestamp int64 `json:"earliest_timestamp,omitempty"` // Earliest scrobble timestamp
	LatestTimestamp   int64 `json:"latest_timestamp,omitempty"`   // Latest scrobble timestamp

	// Unique counts (for dry-run preview)
	UniqueArtists int `json:"unique_artists,omitempty"` // Unique artist count
	UniqueTracks  int `json:"unique_tracks,omitempty"`  // Unique track count (artist+title)
}

// MergeResult represents the outcome of a merge operation
type MergeResult struct {
	Success    bool       `json:"success"`               // Whether merge completed successfully
	OutputPath string     `json:"output_path"`           // Path to output file
	Stats      MergeStats `json:"stats"`                 // Merge statistics
	Warnings   []string   `json:"warnings"`              // Non-fatal warnings
	Error      error      `json:"error,omitempty"`       // Fatal error if failed
	OutputSize int64      `json:"output_size,omitempty"` // Output file size in bytes
}
