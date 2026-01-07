package writer

import (
	"context"

	"github.com/lastfm-reader/lastfm-sync/internal/models"
)

// MockWriter is a test double for Writer.
type MockWriter struct {
	WriteBatchCalls [][]models.Scrobble
	FlushCalls      int
	CloseCalls      int
	Err             error
}

func (m *MockWriter) WriteBatch(ctx context.Context, records []models.Scrobble) error {
	m.WriteBatchCalls = append(m.WriteBatchCalls, records)
	return m.Err
}

func (m *MockWriter) Flush(ctx context.Context) error {
	m.FlushCalls++
	return m.Err
}

func (m *MockWriter) Close(ctx context.Context) error {
	m.CloseCalls++
	return m.Err
}
