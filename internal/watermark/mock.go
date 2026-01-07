package watermark

import (
	"context"
)

// MockStore is a test double for WatermarkStore.
type MockStore struct {
	Data map[string]int64
	Err  error
}

func NewMockStore() *MockStore {
	return &MockStore{
		Data: make(map[string]int64),
	}
}

func (m *MockStore) Get(ctx context.Context, username string) (int64, bool, error) {
	if m.Err != nil {
		return 0, false, m.Err
	}
	uts, exists := m.Data[username]
	return uts, exists, nil
}

func (m *MockStore) Put(ctx context.Context, username string, uts int64) error {
	if m.Err != nil {
		return m.Err
	}
	m.Data[username] = uts
	return nil
}
