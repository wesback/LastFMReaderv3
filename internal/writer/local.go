package writer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
	"os"
	"sync"
)

// LocalWriter writes scrobbles to a local NDJSON file with buffering.
type LocalWriter struct {
	filePath string
	file     *os.File
	writer   *bufio.Writer
	mu       sync.Mutex
}

// NewLocalWriter creates a LocalWriter that appends NDJSON to filePath.
// If the file doesn't exist, it will be created.
// If it exists, records are appended.
func NewLocalWriter(filePath string) (*LocalWriter, error) {
	// Open file for append; create if not exists
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return &LocalWriter{
		filePath: filePath,
		file:     file,
		writer:   bufio.NewWriter(file),
	}, nil
}

// WriteBatch writes records to the buffer (not yet flushed to disk).
func (l *LocalWriter) WriteBatch(ctx context.Context, records []models.Scrobble) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return fmt.Errorf("writer closed")
	}

	for _, scrobble := range records {
		data, err := json.Marshal(scrobble)
		if err != nil {
			return fmt.Errorf("marshal scrobble: %w", err)
		}

		if _, err := l.writer.Write(data); err != nil {
			return fmt.Errorf("write to buffer: %w", err)
		}

		if err := l.writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("write newline: %w", err)
		}
	}

	return nil
}

// Flush writes all buffered records to disk and syncs the file.
func (l *LocalWriter) Flush(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return fmt.Errorf("writer closed")
	}

	// Flush buffer to file
	if err := l.writer.Flush(); err != nil {
		return fmt.Errorf("flush buffer: %w", err)
	}

	// Sync file to disk
	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}

	return nil
}

// Close flushes any remaining data and closes the file.
func (l *LocalWriter) Close(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return nil // Already closed
	}

	// Flush buffer first
	if err := l.writer.Flush(); err != nil {
		_ = l.file.Close()
		l.file = nil
		return fmt.Errorf("flush on close: %w", err)
	}

	// Sync file
	if err := l.file.Sync(); err != nil {
		_ = l.file.Close()
		l.file = nil
		return fmt.Errorf("sync on close: %w", err)
	}

	// Close file
	if err := l.file.Close(); err != nil {
		l.file = nil
		return fmt.Errorf("close file: %w", err)
	}

	l.file = nil
	return nil
}
