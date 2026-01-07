package watermark

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFileStoreCreate tests that FileStore creates state directory.
func TestFileStoreCreate(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	store, err := NewFileStore(stateDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}
	if store == nil {
		t.Fatal("Expected store, got nil")
	}

	if _, err := os.Stat(stateDir); err != nil {
		t.Fatalf("State directory not created: %v", err)
	}
}

// TestFileStoreGetNotExists tests Get on non-existent watermark.
func TestFileStoreGetNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

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

// TestFileStorePutGet tests Put and Get round-trip.
func TestFileStorePutGet(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

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

// TestFileStoreAtomicWrite tests atomic write-temp-rename.
func TestFileStoreAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	if err := store.Put(context.Background(), "user1", 1000); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify file on disk
	filePath := filepath.Join(tmpDir, "user1.watermark")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var wm watermarkData
	if err := json.Unmarshal(data, &wm); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if wm.Username != "user1" || wm.UTS != 1000 {
		t.Errorf("Watermark mismatch: %+v", wm)
	}
}

// TestFileStoreMultipleUsers tests isolation between users.
func TestFileStoreMultipleUsers(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

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

// TestFileStoreOverwrite tests that Put overwrites existing watermark.
func TestFileStoreOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	store.Put(context.Background(), "user", 1000)
	store.Put(context.Background(), "user", 2000)

	uts, exists, _ := store.Get(context.Background(), "user")
	if !exists || uts != 2000 {
		t.Errorf("Expected uts=2000, got %d", uts)
	}
}

// TestFileStorePersistence tests that watermarks persist across store instances.
func TestFileStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// First instance
	store1, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore 1 failed: %v", err)
	}

	if err := store1.Put(context.Background(), "user", 1725000000); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Second instance (simulates new process)
	store2, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore 2 failed: %v", err)
	}

	uts, exists, err := store2.Get(context.Background(), "user")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !exists || uts != 1725000000 {
		t.Errorf("Expected uts=1725000000, got %d", uts)
	}
}

// TestFileStoreContextCancellation tests context cancellation.
func TestFileStoreContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err = store.Get(ctx, "user")
	if err == nil {
		t.Error("Expected context error, got nil")
	}

	err = store.Put(ctx, "user", 1000)
	if err == nil {
		t.Error("Expected context error, got nil")
	}
}

// TestFileStoreTimeout tests timeout on operations.
func TestFileStoreTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// These should complete well within timeout
	_, _, err = store.Get(ctx, "user")
	if err != nil && err == context.DeadlineExceeded {
		t.Fatalf("Get timed out: %v", err)
	}

	err = store.Put(ctx, "user", 1000)
	if err != nil && err == context.DeadlineExceeded {
		t.Fatalf("Put timed out: %v", err)
	}
}

// TestFileStoreSpecialCharactersInUsername tests usernames with special chars.
func TestFileStoreSpecialCharactersInUsername(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	usernames := []string{
		"user_with_underscore",
		"user-with-dash",
		"user.with.dot",
	}

	for _, username := range usernames {
		if err := store.Put(context.Background(), username, 1000); err != nil {
			t.Fatalf("Put %q failed: %v", username, err)
		}

		uts, exists, err := store.Get(context.Background(), username)
		if err != nil {
			t.Fatalf("Get %q failed: %v", username, err)
		}
		if !exists || uts != 1000 {
			t.Errorf("Watermark for %q not persisted", username)
		}
	}
}
