package watermark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
)

// AzureStore implements WatermarkStore interface using Azure Blob Storage
type AzureStore struct {
	client    *azblob.Client
	container string
	prefix    string
}

// azureWatermarkData represents the JSON structure stored in the watermark blob
type azureWatermarkData struct {
	Username string `json:"username"`
	MaxUTS   int64  `json:"max_uts"`
}

// NewAzureStore creates a new Azure Blob Storage watermark store
func NewAzureStore(client *azblob.Client, container, prefix string) *AzureStore {
	return &AzureStore{
		client:    client,
		container: container,
		prefix:    prefix,
	}
}

// Get retrieves the watermark for a user from Azure Blob Storage
func (s *AzureStore) Get(ctx context.Context, username string) (int64, bool, error) {
	blobPath := formatAzureWatermarkPath(s.prefix, username)

	// Download blob
	resp, err := s.client.DownloadStream(ctx, s.container, blobPath, nil)
	if err != nil {
		// Check if blob doesn't exist (first run)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && bloberror.HasCode(err, bloberror.BlobNotFound) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("download watermark blob: %w", err)
	}
	defer resp.Body.Close()

	// Read blob content
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false, fmt.Errorf("read watermark blob: %w", err)
	}

	// Parse JSON
	var wm azureWatermarkData
	if err := json.Unmarshal(data, &wm); err != nil {
		return 0, false, fmt.Errorf("unmarshal watermark: %w", err)
	}

	return wm.MaxUTS, true, nil
}

// Put stores the watermark for a user in Azure Blob Storage
func (s *AzureStore) Put(ctx context.Context, username string, uts int64) error {
	blobPath := formatAzureWatermarkPath(s.prefix, username)

	// Create watermark data
	wm := azureWatermarkData{
		Username: username,
		MaxUTS:   uts,
	}

	// Marshal to JSON
	data, err := json.Marshal(wm)
	if err != nil {
		return fmt.Errorf("marshal watermark: %w", err)
	}

	// Upload blob with ETag concurrency control
	// Note: For now we use simple upload. In production, you might want to:
	// 1. Get the current ETag
	// 2. Upload with If-Match condition to prevent concurrent overwrites
	_, err = s.client.UploadBuffer(ctx, s.container, blobPath, data, &azblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: ptrString("application/json"),
		},
	})
	if err != nil {
		return fmt.Errorf("upload watermark blob: %w", err)
	}

	return nil
}

// formatAzureWatermarkPath generates the watermark blob path
// Format: {prefix}{username}.watermark
func formatAzureWatermarkPath(prefix, username string) string {
	return prefix + username + ".watermark"
}

// ptrString returns a pointer to the given string
func ptrString(s string) *string {
	return &s
}
