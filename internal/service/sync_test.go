package service

import (
	"context"
	"testing"

	"github.com/lastfm-reader/lastfm-sync/internal/lastfm"
	"github.com/lastfm-reader/lastfm-sync/internal/progress"
	"github.com/lastfm-reader/lastfm-sync/internal/watermark"
	"github.com/lastfm-reader/lastfm-sync/internal/writer"
	"go.uber.org/zap"
)

// TestParseUTS tests UTS parsing.
func TestParseUTS(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1000", 1000},
		{"0", 0},
		{"1725000000", 1725000000},
		{"invalid", 0},
		{"", 0},
	}

	for _, test := range tests {
		result := parseUTS(test.input)
		if result != test.expected {
			t.Errorf("parseUTS(%q) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

// TestSyncServiceCreation tests that SyncService can be created.
func TestSyncServiceCreation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	mockClient := &lastfm.Client{
		BaseURL: "http://example.com",
		APIKey:  "test",
	}

	mockWriter := &writer.MockWriter{}
	mockWM := watermark.NewMockStore()
	mockProgress := progress.NewNoOpProgressBar()

	svc := NewSyncService(
		"testuser",
		100, 2000, // from, to
		200, 0, // pageSize, maxPages
		false, // dryRun
		mockClient,
		mockWriter,
		mockWM,
		logger,
		mockProgress,
	)

	if svc == nil {
		t.Fatal("Expected SyncService, got nil")
	}

	if svc.username != "testuser" {
		t.Errorf("Username mismatch: %q", svc.username)
	}

	if svc.from != 100 || svc.to != 2000 {
		t.Errorf("Bounds mismatch: from=%d, to=%d", svc.from, svc.to)
	}
}

// TestSyncServiceContextCancellation tests context cancellation.
func TestSyncServiceContextCancellation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	mockClient := &lastfm.Client{
		BaseURL: "http://example.com",
		APIKey:  "test",
	}
	mockWriter := &writer.MockWriter{}
	mockWM := watermark.NewMockStore()
	mockProgress := progress.NewNoOpProgressBar()

	svc := NewSyncService("testuser", 100, 2000, 200, 0, false, mockClient, mockWriter, mockWM, logger, mockProgress)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.Sync(ctx)
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
}
