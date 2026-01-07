package merge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"go.uber.org/zap"

	"github.com/lastfm-reader/lastfm-sync/internal/models"
	"github.com/lastfm-reader/lastfm-sync/internal/progress"
)

// Merger orchestrates the merge operation
type Merger struct {
	config      MergeConfig
	progress    progress.ProgressReporter
	logger      *zap.Logger
	azureClient *azblob.Client
}

// NewMerger creates a new Merger with the given configuration
func NewMerger(cfg MergeConfig) *Merger {
	// Set defaults
	if cfg.CheckpointInterval <= 0 {
		cfg.CheckpointInterval = 10000
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 128 * 1024 // 128KB default
	}
	if cfg.StorageBackend == "" {
		cfg.StorageBackend = "local"
	}

	return &Merger{
		config: cfg,
	}
}

// SetProgress sets the progress reporter for the merger
func (m *Merger) SetProgress(reporter progress.ProgressReporter) {
	m.progress = reporter
}

// SetLogger sets the logger for the merger
func (m *Merger) SetLogger(logger *zap.Logger) {
	m.logger = logger
}

// Merge performs the merge operation on input files and writes to output
// Returns MergeResult with statistics or error
func (m *Merger) Merge(inputFiles []string, outputPath string) (*MergeResult, error) {
	// Validate inputs
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("no input files specified")
	}
	if outputPath == "" {
		return nil, fmt.Errorf("output path is required")
	}

	// Initialize result
	result := &MergeResult{
		OutputPath: outputPath,
		Stats:      *NewStats(),
	}

	// T084: Resume from checkpoint if configured
	var resumeFrom string
	if m.config.Resume && m.config.CheckpointPath != "" {
		checkpoint, err := LoadCheckpoint(m.config.CheckpointPath)
		if err == nil {
			// Validate checkpoint config matches
			if err := checkpoint.ValidateConfig(m.config); err != nil {
				if m.logger != nil {
					m.logger.Warn("Checkpoint config mismatch, starting fresh",
						zap.Error(err))
				}
			} else {
				// Resume from checkpoint
				result.Stats.TotalScrobbles = checkpoint.TotalScrobbles
				result.Stats.SkippedLines = checkpoint.SkippedLines
				resumeFrom = checkpoint.CurrentFile

				if m.logger != nil {
					m.logger.Info("Resuming from checkpoint",
						zap.String("file", resumeFrom),
						zap.Int("scrobbles", checkpoint.TotalScrobbles))
				}
			}
		} else if m.logger != nil {
			m.logger.Debug("No valid checkpoint found, starting fresh",
				zap.Error(err))
		}
	}

	// Start progress tracking
	if m.progress != nil {
		totalFiles := int64(len(inputFiles))
		m.progress.Start(totalFiles, "Merging scrobble files")
	}

	// Log start
	if m.logger != nil {
		m.logger.Info("Starting merge operation",
			zap.Int("input_files", len(inputFiles)),
			zap.String("output", outputPath),
			zap.String("strategy", string(m.config.Strategy)))
	}

	// Create deduplication map
	dedupMap := NewDeduplicationMap(m.config.Strategy, m.config.ConflictResolution)

	// Process each input file
	for idx, filePath := range inputFiles {
		// T084: Skip files if resuming from checkpoint
		if resumeFrom != "" && filePath != resumeFrom {
			if m.logger != nil {
				m.logger.Debug("Skipping file (before resume point)",
					zap.String("file", filePath))
			}
			if m.progress != nil {
				m.progress.Add(1)
			}
			continue
		}
		// Once we hit the resume file, process it and all subsequent files
		if resumeFrom == filePath {
			resumeFrom = "" // Clear so we process remaining files
		}

		if m.progress != nil {
			m.progress.SetDescription(fmt.Sprintf("Processing %s (%d/%d)", filepath.Base(filePath), idx+1, len(inputFiles)))
		}

		if err := m.processFile(filePath, dedupMap, &result.Stats); err != nil {
			if m.logger != nil {
				m.logger.Error("Failed to process file", zap.String("file", filePath), zap.Error(err))
			}
			return nil, fmt.Errorf("failed to process file %s: %w", filePath, err)
		}
		result.Stats.ProcessedFiles++

		if m.progress != nil {
			m.progress.Add(1)
		}

		if m.logger != nil {
			m.logger.Debug("Processed file",
				zap.String("file", filePath),
				zap.Int("scrobbles_so_far", result.Stats.TotalScrobbles))
		}
	}

	// Get all unique scrobbles
	scrobbles := dedupMap.GetAll()

	// Update statistics
	result.Stats.UniqueScrobbles = len(scrobbles)
	result.Stats.Duplicates = result.Stats.TotalScrobbles - result.Stats.UniqueScrobbles
	result.Stats.EndTime = result.Stats.StartTime.Add(result.Stats.StartTime.Sub(result.Stats.StartTime))

	// T058, T059: Calculate date range and unique counts for all scrobbles
	m.calculateAdditionalStats(scrobbles, &result.Stats)

	if m.logger != nil {
		m.logger.Info("Deduplication complete",
			zap.Int("total", result.Stats.TotalScrobbles),
			zap.Int("unique", result.Stats.UniqueScrobbles),
			zap.Int("duplicates", result.Stats.Duplicates))

		// Log summary of skipped lines if any
		if result.Stats.SkippedLines > 0 {
			skipPercentage := float64(result.Stats.SkippedLines) / float64(result.Stats.TotalScrobbles+result.Stats.SkippedLines) * 100
			m.logger.Warn("Merge completed with skipped lines",
				zap.Int("total_skipped", result.Stats.SkippedLines),
				zap.Int("total_processed", result.Stats.TotalScrobbles),
				zap.Float64("skip_percentage", skipPercentage))
		}
	}

	// Update progress for sorting phase
	if m.progress != nil {
		m.progress.SetDescription("Sorting by timestamp...")
	}

	// Sort by timestamp
	m.sortScrobbles(scrobbles)

	// Write output
	if !m.config.DryRun {
		if m.progress != nil {
			m.progress.SetDescription("Writing output...")
		}

		if err := m.writeOutput(scrobbles, outputPath); err != nil {
			if m.logger != nil {
				m.logger.Error("Failed to write output", zap.Error(err))
			}
			return nil, fmt.Errorf("failed to write output: %w", err)
		}

		// T057: Get actual output size after writing
		if fileInfo, err := os.Stat(outputPath); err == nil {
			result.OutputSize = fileInfo.Size()
		}

		if m.logger != nil {
			m.logger.Info("Output written successfully", zap.String("path", outputPath))
		}
	} else {
		// T057: In dry-run mode, estimate output size
		result.OutputSize = m.estimateOutputSize(scrobbles)

		if m.logger != nil {
			m.logger.Info("Dry-run mode: no output written",
				zap.Int64("estimated_size_bytes", result.OutputSize))
		}
	}

	// Finish progress
	if m.progress != nil {
		m.progress.Finish("Merge complete")
	}

	// T085: Delete checkpoint on successful completion
	if m.config.CheckpointPath != "" {
		if err := DeleteCheckpoint(m.config.CheckpointPath); err != nil {
			if m.logger != nil {
				m.logger.Warn("Failed to delete checkpoint after success",
					zap.String("checkpoint", m.config.CheckpointPath),
					zap.Error(err))
			}
		} else if m.logger != nil {
			m.logger.Debug("Checkpoint deleted after successful merge",
				zap.String("checkpoint", m.config.CheckpointPath))
		}
	}

	return result, nil
}

// openReader opens a reader for a file path, supporting both local files and Azure blobs
// Returns the reader, a closer function, and any error
func (m *Merger) openReader(filePath string) (io.Reader, func(), error) {
	if m.config.StorageBackend == "azure" {
		// Download blob from Azure
		if m.azureClient == nil {
			client, err := m.createAzureClient()
			if err != nil {
				return nil, nil, fmt.Errorf("create azure client: %w", err)
			}
			m.azureClient = client
		}

		ctx := context.Background()
		containerName := m.config.AzureConfig.ContainerName

		// Download blob to a buffer
		response, err := m.azureClient.DownloadStream(ctx, containerName, filePath, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("download blob %s: %w", filePath, err)
		}

		// Return the response body and a closer function
		return response.Body, func() { response.Body.Close() }, nil
	}

	// Local file
	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}

// processFile reads and processes a single NDJSON file
func (m *Merger) processFile(filePath string, dedupMap *DeduplicationMap, stats *MergeStats) error {
	// Open file or download blob based on storage backend
	reader, closer, err := m.openReader(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer closer()

	// Read scrobbles using NDJSON reader
	scrobbles, readErrors := ReadNDJSON(reader)

	// T083: Process scrobbles with checkpoint saving
	for i, s := range scrobbles {
		stats.TotalScrobbles++
		dedupMap.Add(s)

		// T083: Save checkpoint every N scrobbles if configured
		if m.config.CheckpointPath != "" && stats.TotalScrobbles%m.config.CheckpointInterval == 0 {
			if err := m.saveCheckpoint(filePath, i+1, stats, dedupMap); err != nil {
				if m.logger != nil {
					m.logger.Warn("Failed to save checkpoint",
						zap.Int("scrobbles", stats.TotalScrobbles),
						zap.Error(err))
				}
			} else if m.logger != nil {
				m.logger.Debug("Checkpoint saved",
					zap.Int("scrobbles", stats.TotalScrobbles),
					zap.String("checkpoint", m.config.CheckpointPath))
			}
		}
	}

	// Track and log read errors
	if len(readErrors) > 0 {
		stats.SkippedLines += len(readErrors)

		if m.logger != nil {
			// Categorize errors
			errorCounts := make(map[string]int)
			for _, err := range readErrors {
				errorCounts[err.Message]++
			}

			// Log summary of errors for this file
			m.logger.Warn("Skipped lines in file",
				zap.String("file", filepath.Base(filePath)),
				zap.Int("total_skipped", len(readErrors)),
				zap.Any("error_breakdown", errorCounts))

			// Log first few errors at debug level for investigation
			for i, err := range readErrors {
				if i >= 3 { // Only log first 3 errors per file
					break
				}
				m.logger.Debug("Skipped line details",
					zap.String("file", filepath.Base(filePath)),
					zap.Int("line", err.Line),
					zap.String("reason", err.Message),
					zap.String("content", err.Sample),
					zap.Error(err.Err))
			}
		}
	}

	return nil
}

// T083: saveCheckpoint creates and saves a checkpoint of current merge state
func (m *Merger) saveCheckpoint(currentFile string, currentLine int, stats *MergeStats, dedupMap *DeduplicationMap) error {
	checkpoint := &MergeCheckpoint{
		Version:            CheckpointVersion,
		Strategy:           m.config.Strategy,
		ConflictResolution: m.config.ConflictResolution,
		InputFiles:         m.config.InputFiles,
		ProcessedFiles:     []string{}, // Could track fully processed files
		CurrentFile:        currentFile,
		CurrentLine:        currentLine,
		TotalScrobbles:     stats.TotalScrobbles,
		UniqueScrobbles:    dedupMap.UniqueCount(),
		Duplicates:         dedupMap.DuplicateCount(),
		SkippedLines:       stats.SkippedLines,
	}

	return checkpoint.Save(m.config.CheckpointPath)
}

// sortScrobbles sorts scrobbles by timestamp (UTS) in ascending order
func (m *Merger) sortScrobbles(scrobbles []*models.Scrobble) {
	sort.Slice(scrobbles, func(i, j int) bool {
		return scrobbles[i].UTS < scrobbles[j].UTS
	})
}

// writeOutput writes scrobbles to output file as JSON array
func (m *Merger) writeOutput(scrobbles []*models.Scrobble, outputPath string) error {
	// Determine if Azure or local storage based on config
	if m.config.StorageBackend == "azure" {
		return m.writeAzureOutput(scrobbles, outputPath)
	}

	return m.writeLocalOutput(scrobbles, outputPath)
}

// writeLocalOutput writes to local filesystem using atomic write pattern
func (m *Merger) writeLocalOutput(scrobbles []*models.Scrobble, outputPath string) error {
	// Create parent directory if needed
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Use temp file + rename for atomic write
	tempPath := outputPath + ".tmp"
	f, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write JSON array
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ") // Pretty-print for readability

	if err := encoder.Encode(scrobbles); err != nil {
		f.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, outputPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// writeAzureOutput writes to Azure Blob Storage
func (m *Merger) writeAzureOutput(scrobbles []*models.Scrobble, outputPath string) error {
	// Create Azure client if not already initialized
	if m.azureClient == nil {
		client, err := m.createAzureClient()
		if err != nil {
			return fmt.Errorf("create azure client: %w", err)
		}
		m.azureClient = client
	}

	// Marshal scrobbles to JSON
	data, err := json.MarshalIndent(scrobbles, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Upload to Azure Blob Storage
	ctx := context.Background()
	containerName := m.config.AzureConfig.ContainerName

	_, err = m.azureClient.UploadBuffer(ctx, containerName, outputPath, data, nil)
	if err != nil {
		return fmt.Errorf("failed to upload blob %s: %w", outputPath, err)
	}

	if m.logger != nil {
		m.logger.Info("Output uploaded to Azure",
			zap.String("container", containerName),
			zap.String("blob", outputPath),
			zap.Int("size_bytes", len(data)))
	}

	return nil
}

// DiscoverFiles discovers input files matching patterns
// Supports local glob patterns and Azure blob listing
func (m *Merger) DiscoverFiles(patterns []string) ([]string, error) {
	// Check if using Azure storage
	if m.config.StorageBackend == "azure" {
		return m.discoverAzureBlobs(patterns)
	}

	// Local file discovery
	return m.discoverLocalFiles(patterns)
}

// discoverAzureBlobs lists Azure blobs matching patterns
func (m *Merger) discoverAzureBlobs(patterns []string) ([]string, error) {
	// Create Azure client if not already initialized
	if m.azureClient == nil {
		client, err := m.createAzureClient()
		if err != nil {
			return nil, fmt.Errorf("create azure client: %w", err)
		}
		m.azureClient = client
	}

	containerName := m.config.AzureConfig.ContainerName
	var allBlobs []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		// Convert glob pattern to prefix and suffix for filtering
		// For example: "lastfm/dt=*/dis4ea-*.ndjson" -> prefix="lastfm/dt="
		prefix, hasWildcard := extractPrefix(pattern)

		if m.logger != nil {
			m.logger.Debug("Listing Azure blobs",
				zap.String("pattern", pattern),
				zap.String("prefix", prefix),
				zap.Bool("has_wildcard", hasWildcard))
		}

		// List blobs with the prefix
		ctx := context.Background()
		pager := m.azureClient.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
			Prefix: &prefix,
		})

		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("list blobs (pattern=%s): %w", pattern, err)
			}

			for _, blob := range page.Segment.BlobItems {
				if blob.Name == nil {
					continue
				}
				blobName := *blob.Name

				// If there's a wildcard, apply glob-style matching
				if hasWildcard {
					matched, err := filepath.Match(pattern, blobName)
					if err != nil {
						// Invalid pattern, skip
						if m.logger != nil {
							m.logger.Warn("Invalid glob pattern",
								zap.String("pattern", pattern),
								zap.Error(err))
						}
						continue
					}
					if !matched {
						continue
					}
				}

				// Add if not already seen
				if !seen[blobName] {
					allBlobs = append(allBlobs, blobName)
					seen[blobName] = true

					if m.logger != nil {
						m.logger.Debug("Found matching blob",
							zap.String("blob", blobName))
					}
				}
			}
		}
	}

	// Sort for consistent ordering
	sort.Strings(allBlobs)

	if m.logger != nil {
		m.logger.Info("Azure blob discovery complete",
			zap.Int("total_blobs", len(allBlobs)))
	}

	return allBlobs, nil
}

// discoverLocalFiles discovers local files matching glob patterns
func (m *Merger) discoverLocalFiles(patterns []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
		}

		for _, match := range matches {
			// Check if it's a file
			info, err := os.Stat(match)
			if err != nil {
				continue // Skip inaccessible files
			}

			if info.IsDir() && m.config.Recursive {
				// Recursively find NDJSON files
				dirFiles, err := m.findNDJSONFiles(match)
				if err != nil {
					return nil, err
				}
				for _, f := range dirFiles {
					if !seen[f] {
						files = append(files, f)
						seen[f] = true
					}
				}
			} else if !info.IsDir() {
				// Add file directly
				if !seen[match] {
					files = append(files, match)
					seen[match] = true
				}
			}
		}
	}

	return files, nil
}

// createAzureClient creates an Azure Blob Storage client from config
func (m *Merger) createAzureClient() (*azblob.Client, error) {
	azCfg := m.config.AzureConfig
	if azCfg == nil {
		return nil, fmt.Errorf("azure config is required")
	}

	// Determine account URL
	accountURL := azCfg.ContainerURL
	if accountURL == "" && azCfg.AccountName != "" {
		accountURL = fmt.Sprintf("https://%s.blob.core.windows.net/", azCfg.AccountName)
	}

	// Get credential based on auth method
	switch azCfg.AuthMethod {
	case "default", "":
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("create default azure credential: %w", err)
		}
		if accountURL == "" {
			return nil, fmt.Errorf("azure account URL required for credential-based auth")
		}
		return azblob.NewClient(accountURL, cred, nil)

	case "mi":
		cred, err := azidentity.NewManagedIdentityCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("create managed identity credential: %w", err)
		}
		if accountURL == "" {
			return nil, fmt.Errorf("azure account URL required for managed identity auth")
		}
		return azblob.NewClient(accountURL, cred, nil)

	case "key":
		if azCfg.AccountKey == "" {
			return nil, fmt.Errorf("account key required for key auth method")
		}
		if azCfg.AccountName == "" {
			return nil, fmt.Errorf("account name required for key auth method")
		}
		if accountURL == "" {
			return nil, fmt.Errorf("azure account URL required for key auth")
		}
		sharedKeyCred, err := azblob.NewSharedKeyCredential(azCfg.AccountName, azCfg.AccountKey)
		if err != nil {
			return nil, fmt.Errorf("create shared key credential: %w", err)
		}
		return azblob.NewClientWithSharedKeyCredential(accountURL, sharedKeyCred, nil)

	case "sas":
		if azCfg.SASToken == "" {
			return nil, fmt.Errorf("SAS token required for sas auth method")
		}
		if accountURL == "" {
			return nil, fmt.Errorf("azure account URL required for SAS auth")
		}
		sasURL := accountURL
		if sasURL[len(sasURL)-1] == '/' {
			sasURL = sasURL[:len(sasURL)-1]
		}
		if len(azCfg.SASToken) > 0 && azCfg.SASToken[0] != '?' {
			sasURL += "?"
		}
		return azblob.NewClientWithNoCredential(sasURL+azCfg.SASToken, nil)

	default:
		return nil, fmt.Errorf("unsupported azure auth method: %s", azCfg.AuthMethod)
	}
}

// extractPrefix extracts the prefix part of a glob pattern (before first wildcard)
// Returns the prefix and whether the pattern contains wildcards
func extractPrefix(pattern string) (prefix string, hasWildcard bool) {
	// Find first wildcard character
	wildcardIdx := strings.IndexAny(pattern, "*?[]")
	if wildcardIdx == -1 {
		// No wildcard, pattern is the prefix
		return pattern, false
	}

	// Extract everything before the wildcard
	beforeWildcard := pattern[:wildcardIdx]

	// If the character before wildcard is a slash, keep everything up to and including it
	// Otherwise, find the last slash and keep everything up to and including it
	if wildcardIdx > 0 && pattern[wildcardIdx-1] == '/' {
		// Wildcard is at the start of a path component, keep full path prefix
		prefix = beforeWildcard
	} else {
		// Wildcard is within a path component, trim to last complete component
		lastSlash := strings.LastIndex(beforeWildcard, "/")
		if lastSlash != -1 {
			prefix = beforeWildcard[:lastSlash+1]
		} else {
			prefix = ""
		}
	}

	return prefix, true
}

// findNDJSONFiles recursively finds all .ndjson files in a directory
func (m *Merger) findNDJSONFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".ndjson") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// T058, T059: calculateAdditionalStats calculates date range and unique counts
// Used for dry-run preview and enhanced statistics
func (m *Merger) calculateAdditionalStats(scrobbles []*models.Scrobble, stats *MergeStats) {
	if len(scrobbles) == 0 {
		return
	}

	// Track unique artists and tracks
	uniqueArtists := make(map[string]bool)
	uniqueTracks := make(map[string]bool) // Artist+Track combination

	// Initialize date range
	stats.EarliestTimestamp = scrobbles[0].UTS
	stats.LatestTimestamp = scrobbles[0].UTS

	// Iterate through scrobbles
	for _, s := range scrobbles {
		// Track date range
		if s.UTS < stats.EarliestTimestamp {
			stats.EarliestTimestamp = s.UTS
		}
		if s.UTS > stats.LatestTimestamp {
			stats.LatestTimestamp = s.UTS
		}

		// Track unique artists
		uniqueArtists[s.Artist] = true

		// Track unique tracks (Artist + Track combination)
		trackKey := s.Artist + "\x00" + s.Track
		uniqueTracks[trackKey] = true
	}

	stats.UniqueArtists = len(uniqueArtists)
	stats.UniqueTracks = len(uniqueTracks)
}

// T057: estimateOutputSize calculates estimated file size for output
// Used for dry-run preview
func (m *Merger) estimateOutputSize(scrobbles []*models.Scrobble) int64 {
	if len(scrobbles) == 0 {
		return 0
	}

	// Sample first 100 scrobbles (or all if less than 100)
	sampleSize := 100
	if len(scrobbles) < sampleSize {
		sampleSize = len(scrobbles)
	}

	// Calculate average scrobble size
	totalBytes := 0
	for i := 0; i < sampleSize; i++ {
		jsonBytes, err := json.Marshal(scrobbles[i])
		if err == nil {
			totalBytes += len(jsonBytes) + 1 // +1 for newline
		}
	}

	avgSize := float64(totalBytes) / float64(sampleSize)
	estimatedSize := int64(avgSize * float64(len(scrobbles)))

	return estimatedSize
}
