package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lastfm-reader/lastfm-sync/internal/config"
	"github.com/lastfm-reader/lastfm-sync/internal/lastfm"
	"github.com/lastfm-reader/lastfm-sync/internal/logging"
	"github.com/lastfm-reader/lastfm-sync/internal/progress"
	"github.com/lastfm-reader/lastfm-sync/internal/ratelimit"
	"github.com/lastfm-reader/lastfm-sync/internal/service"
	"github.com/lastfm-reader/lastfm-sync/internal/watermark"
	"github.com/lastfm-reader/lastfm-sync/internal/writer"
)

// TestProgressBarWithLastFmSync tests progress bar integration with Last.fm sync
func TestProgressBarWithLastFmSync(t *testing.T) {
	// Create mock Last.fm API server
	pageSize := 50
	totalPages := 3
	totalTracks := pageSize * totalPages

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}

		var pageNum int
		if _, err := fmt.Sscanf(page, "%d", &pageNum); err != nil {
			pageNum = 1
		}

		// Generate mock tracks for this page
		tracks := make([]map[string]interface{}, 0)
		if pageNum <= totalPages {
			for i := 0; i < pageSize; i++ {
				trackNum := (pageNum-1)*pageSize + i + 1
				tracks = append(tracks, map[string]interface{}{
					"artist": map[string]interface{}{
						"#text": "Test Artist",
						"mbid":  "",
					},
					"name": map[string]interface{}{
						"#text": fmt.Sprintf("Track %d", trackNum),
					},
					"album": map[string]interface{}{
						"#text": "Test Album",
						"mbid":  "",
					},
					"date": map[string]interface{}{
						"uts":   fmt.Sprintf("%d", 1600000000+trackNum),
						"#text": "01 Jan 2021",
					},
					"mbid": "",
				})
			}
		}

		response := map[string]interface{}{
			"recenttracks": map[string]interface{}{
				"track": tracks,
				"@attr": map[string]interface{}{
					"user":       "testuser",
					"page":       fmt.Sprintf("%d", pageNum),
					"perPage":    fmt.Sprintf("%d", pageSize),
					"totalPages": fmt.Sprintf("%d", totalPages),
					"total":      fmt.Sprintf("%d", totalTracks),
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create temp directory for output
	tmpDir, err := os.MkdirTemp("", "lastfm-sync-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outPath := filepath.Join(tmpDir, "testuser.ndjson")
	statePath := filepath.Join(tmpDir, "state")

	// Create configuration
	cfg := &config.Config{
		User:           "testuser",
		APIKey:         "test-api-key",
		Since:          0,
		Until:          time.Now().Unix(),
		PageSize:       pageSize,
		MaxPages:       0, // No limit
		Output:         "local",
		OutPath:        outPath,
		WatermarkStore: "file",
		StatePath:      statePath,
		QPS:            100, // High QPS for testing
		Timeout:        15 * time.Second,
		DryRun:         false,
		Progress: config.ProgressConfig{
			Enabled:        true,
			Style:          "blocks",
			ShowSpeed:      true,
			ShowETA:        true,
			ShowCount:      true,
			ShowPercentage: true,
			ShowElapsed:    false,
			Width:          80,
			RefreshRate:    100 * time.Millisecond,
			Colors:         false, // Disable colors for easier testing
			AutoClear:      false, // Don't clear so we can verify output
		},
	}

	// Create logger
	logger, err := logging.New("info")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create rate limiter
	limiter := ratelimit.NewLimiter(float64(cfg.QPS), 3)

	// Create Last.fm client with mock server
	lfmClient := lastfm.NewClient(cfg.APIKey, limiter)
	lfmClient.BaseURL = server.URL + "/" // Override BaseURL to use mock server

	// Create writer
	w, err := writer.NewLocalWriter(outPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer w.Close(context.Background())

	// Create watermark store
	wmStore, err := watermark.NewFileStore(statePath)
	if err != nil {
		t.Fatalf("Failed to create watermark store: %v", err)
	}

	// Create progress reporter with custom writer to capture output
	progressOutput := &bytes.Buffer{}
	progressReporter := progress.NewRealProgressBar(0, "", progress.WithWriter(progressOutput), progress.WithColors(false))

	// Create sync service
	svc := service.NewSyncService(
		cfg.User,
		cfg.Since,
		cfg.Until,
		false, // useSince
		cfg.PageSize,
		cfg.MaxPages,
		cfg.DryRun,
		lfmClient,
		w,
		wmStore,
		logger.Logger,
		progressReporter,
	)

	// Run sync
	ctx := context.Background()
	records, err := svc.Sync(ctx)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify results
	if records != totalTracks {
		t.Errorf("Expected %d records, got %d", totalTracks, records)
	}

	// Verify progress bar was used
	progressStr := progressOutput.String()
	if len(progressStr) == 0 {
		t.Error("Expected progress bar output, got empty string")
	}

	// Verify progress bar is finished
	if !progressReporter.IsFinished() {
		t.Error("Progress bar should be finished after sync")
	}

	// Verify output file exists and has correct number of records
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) != totalTracks {
		t.Errorf("Expected %d lines in output, got %d", totalTracks, len(lines))
	}

	// Verify watermark was set
	watermark, exists, err := wmStore.Get(ctx, cfg.User)
	if err != nil {
		t.Fatalf("Failed to get watermark: %v", err)
	}
	if !exists {
		t.Error("Expected watermark to exist")
	}
	if watermark == 0 {
		t.Error("Expected watermark to be non-zero")
	}
}

// TestProgressBarDisabled tests that progress bar can be disabled
func TestProgressBarDisabled(t *testing.T) {
	// Create configuration with progress disabled
	cfg := &config.Config{
		Progress: config.ProgressConfig{
			Enabled: false,
		},
	}

	// Create progress reporter
	progressReporter := progress.NewProgressReporter(cfg)

	// Should return NoOpProgressBar
	progressReporter.Start(100, "Test")
	progressReporter.Add(50)
	progressReporter.Finish("Done")

	// Should not error and be idempotent
	if progressReporter.IsFinished() {
		t.Error("NoOpProgressBar should always report not finished")
	}
}

// TestProgressBarErrorHandling tests progress bar with sync errors
func TestProgressBarErrorHandling(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lastfm-sync-error-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outPath := filepath.Join(tmpDir, "testuser.ndjson")
	statePath := filepath.Join(tmpDir, "state")

	// Create logger
	logger, _ := logging.New("info")

	// Create rate limiter
	limiter := ratelimit.NewLimiter(3.0, 3)

	// Create Last.fm client
	lfmClient := lastfm.NewClient("test-key", limiter)
	lfmClient.BaseURL = server.URL + "/"

	// Create writer
	w, _ := writer.NewLocalWriter(outPath)
	defer w.Close(context.Background())

	// Create watermark store
	wmStore, _ := watermark.NewFileStore(statePath)

	// Create progress reporter
	progressOutput := &bytes.Buffer{}
	progressReporter := progress.NewRealProgressBar(0, "", progress.WithWriter(progressOutput))

	// Create sync service
	svc := service.NewSyncService(
		"testuser",
		0,
		time.Now().Unix(),
		false, // useSince
		50,
		0,
		false,
		lfmClient,
		w,
		wmStore,
		logger.Logger,
		progressReporter,
	)

	// Run sync - should fail
	ctx := context.Background()
	_, err = svc.Sync(ctx)
	if err == nil {
		t.Error("Expected sync to fail with server error")
	}

	// Verify progress bar is finished with error
	if !progressReporter.IsFinished() {
		t.Error("Progress bar should be finished after error")
	}

	// Progress output should contain error indication
	progressStr := progressOutput.String()
	if len(progressStr) == 0 {
		t.Error("Expected progress bar output even on error")
	}
}

// TestProgressBarCancellation tests progress bar with context cancellation
func TestProgressBarCancellation(t *testing.T) {
	// Create mock server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay to allow cancellation

		response := map[string]interface{}{
			"recenttracks": map[string]interface{}{
				"track": []map[string]interface{}{},
				"@attr": map[string]interface{}{
					"page":       "1",
					"perPage":    "50",
					"totalPages": "10",
					"total":      "500",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lastfm-sync-cancel-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outPath := filepath.Join(tmpDir, "testuser.ndjson")
	statePath := filepath.Join(tmpDir, "state")

	// Create logger
	logger, _ := logging.New("info")

	// Create rate limiter
	limiter := ratelimit.NewLimiter(3.0, 3)

	// Create Last.fm client
	lfmClient := lastfm.NewClient("test-key", limiter)
	lfmClient.BaseURL = server.URL + "/"

	// Create writer
	w, _ := writer.NewLocalWriter(outPath)
	defer w.Close(context.Background())

	// Create watermark store
	wmStore, _ := watermark.NewFileStore(statePath)

	// Create progress reporter
	progressReporter := progress.NewRealProgressBar(0, "")

	// Create sync service
	svc := service.NewSyncService(
		"testuser",
		0,
		time.Now().Unix(),
		false, // useSince
		50,
		0,
		false,
		lfmClient,
		w,
		wmStore,
		logger.Logger,
		progressReporter,
	)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Run sync - should be cancelled
	_, err = svc.Sync(ctx)
	if err == nil {
		t.Error("Expected sync to be cancelled")
	}
	if ctx.Err() != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", ctx.Err())
	}
}
