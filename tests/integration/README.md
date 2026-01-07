# Integration Tests

This directory contains integration tests that validate multiple components working together in realistic scenarios.

## What Belongs Here

**Integration tests** validate end-to-end workflows spanning multiple packages:
- Full sync pipeline (API → processing → storage)
- External service interactions (Last.fm API, Azure Blob Storage)
- Cross-package data flow and state management
- System-level error handling and recovery

## What Doesn't Belong Here

**Unit tests** belong alongside the code they test in their respective packages:
- `internal/lastfm/client_test.go` - API client behavior
- `internal/writer/local_test.go` - File writer logic
- `internal/watermark/file_test.go` - State persistence
- `cmd/lastfm-sync/commands/fetch_test.go` - CLI command parsing

## Test Patterns

### Mock External Services
Use `httptest.NewServer()` to simulate Last.fm API responses:
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Return mock Last.fm response
}))
defer server.Close()
```

### Temporary Resources
Always clean up test artifacts:
```go
tmpDir, err := os.MkdirTemp("", "test-*")
if err != nil {
    t.Fatalf("Failed to create temp dir: %v", err)
}
defer os.RemoveAll(tmpDir)
```

### Context Handling
Test cancellation and timeout scenarios:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

## Running Integration Tests

```bash
# Run all integration tests
go test ./tests/integration/... -v

# Run with timeout
go test ./tests/integration/... -v -timeout 30s

# Run specific test
go test ./tests/integration/... -run TestSyncWithLastFm -v
```

## Test Coverage

Current integration tests:
- ✅ Full sync pipeline with progress tracking
- ✅ Progress bar disabled/enabled modes
- ✅ Error handling during sync
- ✅ Context cancellation

Planned additions (from spec):
- [ ] Incremental sync with watermarks (T039)
- [ ] Crash recovery (T040)
- [ ] Azure Blob Storage end-to-end (T050)
- [ ] Azure authentication methods (T051)
- [ ] Rate limit handling (T061)
- [ ] 5xx error retry behavior (T062)
