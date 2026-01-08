package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/lastfm-reader/lastfm-sync/internal/config"
	"github.com/lastfm-reader/lastfm-sync/internal/logging"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
	"github.com/lastfm-reader/lastfm-sync/internal/normalize"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// Exit codes for normalize command
const (
	NormalizeExitSuccess    = 0 // Successful completion
	NormalizeExitGeneral    = 1 // General error
	NormalizeExitValidation = 2 // Validation error (missing args, invalid config)
)

// ProcessingError represents an error encountered while processing a file
type ProcessingError struct {
	FilePath  string
	ErrorType string // "parse_error", "missing_track_field", "permission_denied", "read_error", "write_error"
	Details   error
}

// ProcessingSummary contains the results of the normalize operation
type ProcessingSummary struct {
	TotalFiles     int
	UpdatedFiles   int
	UnchangedFiles int
	ErrorCount     int
	Errors         []ProcessingError
	DryRun         bool
	Duration       time.Duration
}

// NormalizeCommand returns the "normalize" cobra command
func NormalizeCommand() *cobra.Command {
	var (
		username          string
		dryRun            bool
		logLevel          string
		azureContainer    string
		azureAccount      string
		azureAuth         string
		azurePrefix       string
		azureContainerURL string
		azureAccountKey   string
		azureSASToken     string
	)

	cmd := &cobra.Command{
		Use:   "normalize",
		Short: "Re-normalize existing scrobble files",
		Long: `Process existing NDJSON scrobble files and update the normalized_title field
by reapplying the current normalization logic.

This command is useful when normalization rules have been updated and you want to
retroactively apply them to historical data without re-fetching from Last.fm.

STORAGE MODES:
  - Local: Processes files in local filesystem (default)
  - Azure: Processes files in Azure Blob Storage

FILE DISCOVERY:
  Matches files with pattern: {username}_*.ndjson
  - Local: Searches in current directory or configured base path
  - Azure: Searches with prefix filter in specified container

AZURE AUTHENTICATION:
  - default: DefaultAzureCredential (recommended for Azure VMs/AKS)
  - mi: Managed Identity
  - connstr: Connection string from AZURE_STORAGE_CONNECTION_STRING env var
  - key: Storage account key (use --azure-account-key)
  - sas: SAS token (use --azure-sas-token with query string format: sv=...&sig=...)

PROCESSING:
  - Reads each file line-by-line (NDJSON format)
  - Applies current normalization logic to track field
  - Updates only files where normalized_title has changed
  - Continues processing remaining files if individual files fail

DRY-RUN MODE:
  Use --dry-run to preview changes without modifying files.
  Shows which files would be updated and displays old vs new normalized_title values.

Examples:
  # Basic usage - normalize all files for user (local storage)
  lastfm-sync normalize --user john_doe

  # Dry-run preview (local storage)
  lastfm-sync normalize --user john_doe --dry-run

  # Azure Blob Storage with DefaultAzureCredential
  lastfm-sync normalize --user john_doe \
    --azure-container scrobbles --azure-account myaccount --azure-auth default

  # Azure with connection string
  export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=..."
  lastfm-sync normalize --user john_doe \
    --azure-container scrobbles --azure-auth connstr

  # Azure with custom prefix
  lastfm-sync normalize --user john_doe \
    --azure-container scrobbles --azure-account myaccount \
    --azure-prefix "archives/2026/"

  # Debug logging
  lastfm-sync normalize --user john_doe --log-level debug

Notes:
  - Does not prevent concurrent execution. Administrators should coordinate multiple
    normalize operations on the same user's files.
  - Processing is idempotent - running multiple times produces the same result.`,

		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Initialize logger
			logger, err := logging.New(logLevel)
			if err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}
			defer logger.Sync()

			// Validate required arguments
			if username == "" {
				return fmt.Errorf("--user is required")
			}

			// Determine storage mode
			storageMode := "local"
			if azureContainer != "" {
				storageMode = "azure"
			}

			logger.Info("Starting normalize operation",
				zap.String("username", username),
				zap.String("storage", storageMode),
				zap.Bool("dry_run", dryRun),
			)

			var files []string
			if storageMode == "azure" {
				// Azure storage - discover files via ListBlobs
				logger.Info("Discovering Azure blobs",
					zap.String("container", azureContainer),
					zap.String("prefix", azurePrefix),
					zap.String("pattern", username+"_*.ndjson"))

				// Create Azure client
				azureClient, err := createAzureClient(azureContainer, azureAccount, azureAuth,
					azureContainerURL, azureAccountKey, azureSASToken)
				if err != nil {
					return fmt.Errorf("failed to create Azure client: %w", err)
				}

				files, err = discoverAzureFiles(ctx, azureClient, azureContainer, azurePrefix, username, logger)
				if err != nil {
					return fmt.Errorf("failed to discover Azure files: %w", err)
				}
			} else {
				// Local storage - discover files via glob
				logger.Info("Discovering files", zap.String("pattern", username+"_*.ndjson or "+username+"-*.ndjson"))
				var err error
				files, err = DiscoverLocalFiles(username, logger)
				if err != nil {
					return fmt.Errorf("failed to discover files: %w", err)
				}
			}

			if len(files) == 0 {
				logger.Warn("No files found for user", zap.String("username", username))
				fmt.Printf("No files found for user: %s\n", username)
				return nil
			}

			logger.Info("Found files", zap.Int("count", len(files)))
			fmt.Printf("\nProcessing files for user: %s\n", username)
			fmt.Printf("Storage: %s\n\n", storageMode)

			// Process files
			startTime := time.Now()
			summary := ProcessingSummary{
				TotalFiles: len(files),
				DryRun:     dryRun,
			}

			for i, filePath := range files {
				fmt.Printf("Processing: %s [%d/%d]\n", filepath.Base(filePath), i+1, len(files))

				var updated bool
				var err error

				if storageMode == "azure" {
					azureClient, _ := createAzureClient(azureContainer, azureAccount, azureAuth,
						azureContainerURL, azureAccountKey, azureSASToken)
					updated, err = processAzureFile(ctx, azureClient, azureContainer, filePath, dryRun, logger)
				} else {
					updated, err = ProcessFile(ctx, filePath, dryRun, logger)
				}

				if err != nil {
					logger.Error("Failed to process file",
						zap.String("file", filePath),
						zap.Error(err),
					)
					summary.ErrorCount++
					summary.Errors = append(summary.Errors, ProcessingError{
						FilePath:  filePath,
						ErrorType: CategorizeError(err),
						Details:   err,
					})
					fmt.Printf("  Status: Error: %s\n", CategorizeError(err))
				} else if updated {
					summary.UpdatedFiles++
					if dryRun {
						fmt.Printf("  Status: Would update\n")
					} else {
						fmt.Printf("  Status: Updated\n")
					}
				} else {
					summary.UnchangedFiles++
					fmt.Printf("  Status: No change needed\n")
				}
			}

			summary.Duration = time.Since(startTime)

			// Display summary
			displaySummary(summary, username)

			return nil
		},
	}

	// Required flags
	cmd.Flags().StringVarP(&username, "user", "u", "", "Username whose files to process (required)")
	cmd.MarkFlagRequired("user")

	// Behavior flags
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn, error")

	// Azure Blob Storage options
	cmd.Flags().StringVar(&azureContainer, "azure-container", "", "Azure container name (enables Azure mode)")
	cmd.Flags().StringVar(&azureAccount, "azure-account", "", "Azure storage account name")
	cmd.Flags().StringVar(&azureAuth, "azure-auth", "default", "Azure auth method: default, mi, connstr, key, sas")
	cmd.Flags().StringVar(&azurePrefix, "azure-prefix", "", "Azure blob prefix")
	cmd.Flags().StringVar(&azureContainerURL, "azure-container-url", "", "Azure container URL")
	cmd.Flags().StringVar(&azureAccountKey, "azure-account-key", "", "Azure storage account key")
	cmd.Flags().StringVar(&azureSASToken, "azure-sas-token", "", "Azure SAS token")

	return cmd
}

// createAzureClient creates an Azure Blob Storage client based on auth method
func createAzureClient(container, account, authMethod, containerURL, accountKey, sasToken string) (*azblob.Client, error) {
	cfg := &config.Config{
		AzureContainer:  container,
		AzureAccount:    account,
		AzureAuth:       authMethod,
		AzureAccountURL: "",
		AzureAccountKey: accountKey,
		AzureSASToken:   sasToken,
	}

	// Build account URL if not using connection string
	if authMethod != "connstr" && account != "" {
		cfg.AzureAccountURL = fmt.Sprintf("https://%s.blob.core.windows.net/", account)
	}

	// For container URL-based auth
	if containerURL != "" {
		cfg.AzureAccountURL = containerURL
	}

	return config.CreateAzureBlobClient(cfg)
}

// discoverAzureFiles lists blobs matching the username pattern in Azure Blob Storage
// Supports both underscore (username_*.ndjson) and dash (username-*.ndjson) patterns
func discoverAzureFiles(ctx context.Context, client *azblob.Client, container, prefix, username string, logger *logging.Logger) ([]string, error) {
	var files []string
	patternUnderscore := username + "_"
	patternDash := username + "-"

	// Build prefix filter
	searchPrefix := prefix
	if searchPrefix != "" && !strings.HasSuffix(searchPrefix, "/") {
		searchPrefix += "/"
	}

	// List blobs with prefix
	pager := client.NewListBlobsFlatPager(container, &azblob.ListBlobsFlatOptions{
		Prefix: &searchPrefix,
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list blobs: %w", err)
		}

		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			blobName := *blob.Name
			baseName := filepath.Base(blobName)

			// Check if it matches either pattern: {username}_*.ndjson or {username}-*.ndjson
			if (strings.HasPrefix(baseName, patternUnderscore) || strings.HasPrefix(baseName, patternDash)) && strings.HasSuffix(baseName, ".ndjson") {
				files = append(files, blobName)
			}
		}
	}

	return files, nil
}

// processAzureFile processes a single Azure blob file
func processAzureFile(ctx context.Context, client *azblob.Client, container, blobPath string, dryRun bool, logger *logging.Logger) (bool, error) {
	// Download blob
	response, err := client.DownloadStream(ctx, container, blobPath, nil)
	if err != nil {
		return false, fmt.Errorf("read_error: download blob: %w", err)
	}
	defer response.Body.Close()

	// Parse NDJSON line by line
	scanner := bufio.NewScanner(response.Body)
	var scrobbles []models.Scrobble
	var updated bool
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var scrobble models.Scrobble
		if err := json.Unmarshal([]byte(line), &scrobble); err != nil {
			return false, fmt.Errorf("parse_error at line %d: %w", lineNum, err)
		}

		// Check if track field exists
		if scrobble.Track == "" {
			return false, fmt.Errorf("missing_track_field at line %d", lineNum)
		}

		// Apply normalization
		newNormalized := normalize.NormalizeTitle(scrobble.Track)

		// Check if normalized_title changed
		if scrobble.NormalizedTitle != newNormalized {
			scrobble.NormalizedTitle = newNormalized
			updated = true
		}

		scrobbles = append(scrobbles, scrobble)
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("read_error: %w", err)
	}

	// If no changes, skip writing
	if !updated {
		return false, nil
	}

	// In dry-run mode, don't write changes
	if dryRun {
		logger.Info("Dry-run: Would update blob",
			zap.String("blob", blobPath),
			zap.Int("scrobbles", len(scrobbles)),
		)
		return true, nil
	}

	// Upload updated blob
	// Create in-memory buffer with NDJSON content
	var buf strings.Builder
	for _, scrobble := range scrobbles {
		data, err := json.Marshal(scrobble)
		if err != nil {
			return false, fmt.Errorf("write_error: marshal: %w", err)
		}
		buf.Write(data)
		buf.WriteString("\n")
	}

	// Upload to Azure
	content := strings.NewReader(buf.String())
	_, err = client.UploadStream(ctx, container, blobPath, content, nil)
	if err != nil {
		return false, fmt.Errorf("write_error: upload blob: %w", err)
	}

	return true, nil
}

// DiscoverLocalFiles finds all NDJSON files matching the username pattern in local storage
// Supports both underscore (username_*.ndjson) and dash (username-*.ndjson) patterns
func DiscoverLocalFiles(username string, logger *logging.Logger) ([]string, error) {
	baseDir := "." // TODO: Get from config

	// Check for underscore pattern (username_*.ndjson)
	pattern1 := filepath.Join(baseDir, username+"_*.ndjson")
	matches1, err := filepath.Glob(pattern1)
	if err != nil {
		return nil, fmt.Errorf("glob pattern failed: %w", err)
	}

	// Check for dash pattern (username-*.ndjson)
	pattern2 := filepath.Join(baseDir, username+"-*.ndjson")
	matches2, err := filepath.Glob(pattern2)
	if err != nil {
		return nil, fmt.Errorf("glob pattern failed: %w", err)
	}

	// Combine results
	allMatches := append(matches1, matches2...)
	return allMatches, nil
}

// ProcessFile processes a single NDJSON file and returns true if it was updated
func ProcessFile(ctx context.Context, filePath string, dryRun bool, logger *logging.Logger) (bool, error) {
	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("read_error: %w", err)
	}
	defer file.Close()

	// Parse NDJSON line by line
	scanner := bufio.NewScanner(file)
	var scrobbles []models.Scrobble
	var updated bool
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var scrobble models.Scrobble
		if err := json.Unmarshal([]byte(line), &scrobble); err != nil {
			return false, fmt.Errorf("parse_error at line %d: %w", lineNum, err)
		}

		// Check if track field exists
		if scrobble.Track == "" {
			return false, fmt.Errorf("missing_track_field at line %d", lineNum)
		}

		// Apply normalization
		newNormalized := normalize.NormalizeTitle(scrobble.Track)

		// Check if normalized_title changed
		if scrobble.NormalizedTitle != newNormalized {
			scrobble.NormalizedTitle = newNormalized
			updated = true
		}

		scrobbles = append(scrobbles, scrobble)
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("read_error: %w", err)
	}

	// If no changes and not dry-run, skip writing
	if !updated {
		return false, nil
	}

	// In dry-run mode, don't write changes
	if dryRun {
		logger.Info("Dry-run: Would update file",
			zap.String("file", filePath),
			zap.Int("scrobbles", len(scrobbles)),
		)
		return true, nil
	}

	// Write updated file
	// Create temporary file for atomic write
	tempPath := filePath + ".tmp"
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return false, fmt.Errorf("write_error: %w", err)
	}

	writer := bufio.NewWriter(tempFile)
	for _, scrobble := range scrobbles {
		data, err := json.Marshal(scrobble)
		if err != nil {
			tempFile.Close()
			os.Remove(tempPath)
			return false, fmt.Errorf("write_error: %w", err)
		}
		if _, err := writer.Write(data); err != nil {
			tempFile.Close()
			os.Remove(tempPath)
			return false, fmt.Errorf("write_error: %w", err)
		}
		if _, err := writer.WriteString("\n"); err != nil {
			tempFile.Close()
			os.Remove(tempPath)
			return false, fmt.Errorf("write_error: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return false, fmt.Errorf("write_error: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		return false, fmt.Errorf("write_error: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return false, fmt.Errorf("write_error: %w", err)
	}

	return true, nil
}

// CategorizeError determines the error type from an error message
func CategorizeError(err error) string {
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "parse_error"):
		return "parse_error"
	case strings.Contains(errStr, "missing_track_field"):
		return "missing_track_field"
	case strings.Contains(errStr, "permission denied"):
		return "permission_denied"
	case strings.Contains(errStr, "read_error"):
		return "read_error"
	case strings.Contains(errStr, "write_error"):
		return "write_error"
	default:
		return "unknown_error"
	}
}

// displaySummary prints the processing summary to stdout
func displaySummary(summary ProcessingSummary, username string) {
	fmt.Printf("\nProcessing complete for user: %s\n\n", username)

	fmt.Printf("Summary:\n")
	fmt.Printf("  Total files:    %d\n", summary.TotalFiles)
	fmt.Printf("  Updated:        %d\n", summary.UpdatedFiles)
	fmt.Printf("  Unchanged:      %d\n", summary.UnchangedFiles)
	fmt.Printf("  Errors:         %d\n", summary.ErrorCount)
	fmt.Printf("  Duration:       %s\n", summary.Duration.Round(time.Millisecond))

	if summary.DryRun {
		fmt.Printf("\nDry-run mode: No changes written to storage\n")
	}

	if summary.ErrorCount > 0 {
		fmt.Printf("\nErrors encountered:\n")
		for _, err := range summary.Errors {
			fmt.Printf("  - %s: %s\n", filepath.Base(err.FilePath), err.ErrorType)
		}
	}
}
