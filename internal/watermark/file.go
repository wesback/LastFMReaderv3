package watermark

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileStore persists watermarks to JSON files in a directory.
// File layout: {stateDir}/{username}.watermark
// Each file contains: {"username":"user","uts":1725000000}
type FileStore struct {
	stateDir string
	mu       sync.RWMutex
}

// watermarkData represents the JSON structure of a watermark file.
type watermarkData struct {
	Username string `json:"username"`
	UTS      int64  `json:"uts"`
}

// NewFileStore creates a FileStore that persists watermarks to stateDir.
// The directory will be created if it doesn't exist.
func NewFileStore(stateDir string) (*FileStore, error) {
	// Ensure directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}

	return &FileStore{
		stateDir: stateDir,
	}, nil
}

// Get retrieves the watermark for the given username.
// Returns (uts, exists, error).
func (f *FileStore) Get(ctx context.Context, username string) (int64, bool, error) {
	if ctx.Err() != nil {
		return 0, false, ctx.Err()
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	filePath := filepath.Join(f.stateDir, username+".watermark")

	// Check if file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("stat watermark file: %w", err)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, false, fmt.Errorf("read watermark file: %w", err)
	}

	var wm watermarkData
	if err := json.Unmarshal(data, &wm); err != nil {
		return 0, false, fmt.Errorf("unmarshal watermark: %w", err)
	}

	return wm.UTS, true, nil
}

// Put atomically updates the watermark for the given username.
// Uses write-to-temp-then-rename pattern for atomicity.
func (f *FileStore) Put(ctx context.Context, username string, uts int64) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	wm := watermarkData{
		Username: username,
		UTS:      uts,
	}

	data, err := json.Marshal(wm)
	if err != nil {
		return fmt.Errorf("marshal watermark: %w", err)
	}

	filePath := filepath.Join(f.stateDir, username+".watermark")
	tempPath := filePath + ".tmp"

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("write temp watermark: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		// Clean up temp file on error
		_ = os.Remove(tempPath)
		return fmt.Errorf("rename watermark file: %w", err)
	}

	return nil
}
