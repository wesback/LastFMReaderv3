package watermark

import (
	"context"
)

// WatermarkStore defines the interface for persisting and retrieving watermarks.
// A watermark is the highest UTS (unix timestamp) successfully processed for a user.
// It enables incremental sync on subsequent runs.
type WatermarkStore interface {
	// Get retrieves the watermark (highest UTS) for the given username.
	// Returns (uts, exists, error).
	// If no watermark exists, exists=false and uts=0.
	Get(ctx context.Context, username string) (uts int64, exists bool, err error)

	// Put atomically updates the watermark for the given username.
	// uts is the highest UTS successfully processed.
	Put(ctx context.Context, username string, uts int64) error
}
