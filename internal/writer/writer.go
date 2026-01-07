package writer

import (
	"context"

	"github.com/lastfm-reader/lastfm-sync/internal/models"
)

// Writer defines the interface for writing scrobbles to persistent storage.
type Writer interface {
	// WriteBatch writes a batch of scrobbles. May be buffered.
	WriteBatch(ctx context.Context, records []models.Scrobble) error

	// Flush ensures all buffered records are persisted.
	Flush(ctx context.Context) error

	// Close closes the writer and releases resources.
	Close(ctx context.Context) error
}
