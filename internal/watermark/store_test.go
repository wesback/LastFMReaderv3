package watermark

import (
	"context"
	"testing"
)

// TestWatermarkStoreInterface verifies the interface is properly defined.
func TestWatermarkStoreInterface(t *testing.T) {
	var s WatermarkStore = NewMockStore()
	if s == nil {
		t.Fatal("WatermarkStore interface not implemented")
	}
}

// TestMockStoreGetNotExists tests that Get returns false for non-existent watermark.
func TestMockStoreGetNotExists(t *testing.T) {
	store := NewMockStore()

	uts, exists, err := store.Get(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if exists {
		t.Error("Expected exists=false")
	}
	if uts != 0 {
		t.Errorf("Expected uts=0, got %d", uts)
	}
}

// TestMockStorePutGet tests Put and Get round-trip.
func TestMockStorePutGet(t *testing.T) {
	store := NewMockStore()

	if err := store.Put(context.Background(), "testuser", 1725000000); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	uts, exists, err := store.Get(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !exists {
		t.Error("Expected exists=true")
	}
	if uts != 1725000000 {
		t.Errorf("Expected uts=1725000000, got %d", uts)
	}
}

// TestMockStoreMultipleUsers tests isolation between users.
func TestMockStoreMultipleUsers(t *testing.T) {
	store := NewMockStore()

	store.Put(context.Background(), "alice", 1000)
	store.Put(context.Background(), "bob", 2000)

	alice, _, _ := store.Get(context.Background(), "alice")
	bob, _, _ := store.Get(context.Background(), "bob")

	if alice != 1000 {
		t.Errorf("Alice watermark: expected 1000, got %d", alice)
	}
	if bob != 2000 {
		t.Errorf("Bob watermark: expected 2000, got %d", bob)
	}
}
