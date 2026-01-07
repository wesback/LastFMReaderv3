package watermark

import (
	"context"
	"errors"
	"testing"
)

// TestAzureStoreCreation tests creating an AzureStore instance
func TestAzureStoreCreation(t *testing.T) {
	// This test will verify that we can create an AzureStore
	// Implementation will be added when we implement azure.go
	t.Skip("Skipping until AzureStore implementation is added")
}

// TestAzureStoreGetNotExists tests Get() when watermark blob doesn't exist
func TestAzureStoreGetNotExists(t *testing.T) {
	t.Skip("Skipping until AzureStore implementation is added")
}

// TestAzureStorePutGet tests Put() and Get() operations
func TestAzureStorePutGet(t *testing.T) {
	t.Skip("Skipping until AzureStore implementation is added")
}

// TestAzureStoreWatermarkBlobPath tests the watermark blob path format
func TestAzureStoreWatermarkBlobPath(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		username string
		want     string
	}{
		{
			name:     "with prefix",
			prefix:   "lastfm/",
			username: "alice",
			want:     "lastfm/alice.watermark",
		},
		{
			name:     "no prefix",
			prefix:   "",
			username: "bob",
			want:     "bob.watermark",
		},
		{
			name:     "complex prefix",
			prefix:   "scrobbles/prod/",
			username: "charlie",
			want:     "scrobbles/prod/charlie.watermark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test uses the actual function from azure.go
			got := formatAzureWatermarkPath(tt.prefix, tt.username)
			if got != tt.want {
				t.Errorf("formatAzureWatermarkPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// MockAzureStore for testing - implements WatermarkStore interface
type MockAzureStore struct {
	watermarks map[string]int64
	getErr     error
	putErr     error
}

func NewMockAzureStore() *MockAzureStore {
	return &MockAzureStore{
		watermarks: make(map[string]int64),
	}
}

func (m *MockAzureStore) Get(ctx context.Context, username string) (int64, bool, error) {
	if m.getErr != nil {
		return 0, false, m.getErr
	}

	uts, exists := m.watermarks[username]
	return uts, exists, nil
}

func (m *MockAzureStore) Put(ctx context.Context, username string, uts int64) error {
	if m.putErr != nil {
		return m.putErr
	}

	m.watermarks[username] = uts
	return nil
}

func (m *MockAzureStore) SetGetError(err error) {
	m.getErr = err
}

func (m *MockAzureStore) SetPutError(err error) {
	m.putErr = err
}

// TestMockAzureStore tests the mock implementation
func TestMockAzureStore(t *testing.T) {
	ctx := context.Background()
	store := NewMockAzureStore()

	// Test Get when not exists
	_, exists, err := store.Get(ctx, "alice")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if exists {
		t.Error("Get() exists = true, want false")
	}

	// Test Put and Get
	if err := store.Put(ctx, "alice", 12345); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	uts, exists, err := store.Get(ctx, "alice")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !exists {
		t.Error("Get() exists = false, want true")
	}
	if uts != 12345 {
		t.Errorf("Get() uts = %v, want 12345", uts)
	}

	// Test error injection
	testErr := errors.New("test error")
	store.SetGetError(testErr)
	_, _, err = store.Get(ctx, "alice")
	if err != testErr {
		t.Errorf("Get() error = %v, want %v", err, testErr)
	}

	store.SetGetError(nil)
	store.SetPutError(testErr)
	err = store.Put(ctx, "bob", 67890)
	if err != testErr {
		t.Errorf("Put() error = %v, want %v", err, testErr)
	}
}
