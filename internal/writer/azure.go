package writer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
	"io"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
)

// AzureWriter implements Writer interface for Azure Blob Storage
type AzureWriter struct {
	client    *azblob.Client
	container string
	prefix    string
	username  string
	tempFile  *os.File
	tempPath  string
	closed    bool
}

// NewAzureWriter creates a new Azure Blob Storage writer
func NewAzureWriter(client *azblob.Client, container, prefix, username string) (*AzureWriter, error) {
	// Create temp file for buffering
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, fmt.Sprintf("lastfm-azure-%s-*.ndjson", username))
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	return &AzureWriter{
		client:    client,
		container: container,
		prefix:    prefix,
		username:  username,
		tempFile:  tempFile,
		tempPath:  tempFile.Name(),
		closed:    false,
	}, nil
}

// SetUsername sets the username for this writer
func (w *AzureWriter) SetUsername(username string) {
	w.username = username
}

// WriteBatch writes a batch of scrobbles to the temp file
func (w *AzureWriter) WriteBatch(ctx context.Context, scrobbles []models.Scrobble) error {
	if w.closed {
		return fmt.Errorf("writer is closed")
	}

	for _, scrobble := range scrobbles {
		data, err := json.Marshal(scrobble)
		if err != nil {
			return fmt.Errorf("marshal scrobble: %w", err)
		}

		if _, err := w.tempFile.Write(data); err != nil {
			return fmt.Errorf("write to temp file: %w", err)
		}
		if _, err := w.tempFile.Write([]byte("\n")); err != nil {
			return fmt.Errorf("write newline: %w", err)
		}
	}

	return nil
}

// Flush uploads the temp file to Azure Blob Storage
func (w *AzureWriter) Flush(ctx context.Context) error {
	if w.closed {
		return fmt.Errorf("writer is closed")
	}

	// Sync temp file to disk
	if err := w.tempFile.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}

	// Get file info for upload
	info, err := w.tempFile.Stat()
	if err != nil {
		return fmt.Errorf("stat temp file: %w", err)
	}

	// Skip upload if file is empty
	if info.Size() == 0 {
		return nil
	}

	// Seek to beginning for reading
	if _, err := w.tempFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek temp file: %w", err)
	}

	// Generate blob path with current timestamp
	blobPath := formatAzureBlobPath(w.prefix, w.username, time.Now().UTC())

	// Upload to Azure Blob Storage
	_, err = w.client.UploadFile(ctx, w.container, blobPath, w.tempFile, &azblob.UploadFileOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: ptrString("application/x-ndjson"),
		},
	})
	if err != nil {
		return fmt.Errorf("upload to azure blob %s: %w", blobPath, err)
	}

	return nil
}

// Close closes the writer and cleans up the temp file
func (w *AzureWriter) Close(ctx context.Context) error {
	if w.closed {
		return nil
	}

	w.closed = true

	// Close temp file
	if err := w.tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Remove temp file
	if err := os.Remove(w.tempPath); err != nil {
		// Log but don't fail on temp file cleanup error
		return fmt.Errorf("remove temp file: %w", err)
	}

	return nil
}

// formatAzureBlobPath generates a time-partitioned blob path
// Format: {prefix}dt=YYYY-MM-DD/{username}-YYYYMMDD-HHMMSS.ndjson
func formatAzureBlobPath(prefix, username string, ts time.Time) string {
	datePartition := ts.Format("2006-01-02")
	timestamp := ts.Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.ndjson", username, timestamp)
	return fmt.Sprintf("%sdt=%s/%s", prefix, datePartition, filename)
}

// formatWatermarkBlobPath generates the watermark blob path
// Format: {prefix}{username}.watermark
func formatWatermarkBlobPath(prefix, username string) string {
	return prefix + username + ".watermark"
}

// ptrString returns a pointer to the given string
func ptrString(s string) *string {
	return &s
}
