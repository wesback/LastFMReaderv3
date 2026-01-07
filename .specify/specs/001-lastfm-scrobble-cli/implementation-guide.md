# Implementation Guide: Last.fm Scrobble CLI

**Feature**: `001-lastfm-scrobble-cli` | **Status**: Implementation Ready  
**Created**: 2025-10-30 | **Target Completion**: v1.0.0  
**Execution Model**: Phase-based with TDD (test-first, red-green-refactor)

---

## Quick Start for Implementers

### Prerequisites

Before starting implementation, ensure you have:

1. **Go 1.22+** installed
   ```bash
   go version
   ```

2. **GitHub CLI** for releases (optional)
   ```bash
   gh --version
   ```

3. **Docker** for container testing (optional but recommended)
   ```bash
   docker --version
   ```

4. **Project repository cloned**
   ```bash
   cd /home/wesleyb/git/LastFMReaderv3/LastFmReaderv3
   ```

### Reference Documents

- **Specification**: `specs/001-lastfm-scrobble-cli/spec.md` — User stories, requirements, success criteria
- **Implementation Plan**: `specs/001-lastfm-scrobble-cli/plan.md` — Architecture, tech stack, component design
- **Tasks**: `specs/001-lastfm-scrobble-cli/tasks.md` — Detailed task breakdown (110 tasks in 11 phases)
- **Constitution**: `.specify/memory/constitution.md` — Code quality, testing, performance standards

---

## Execution Model: Phase-Based Implementation

### Phase Overview

All 110 tasks are organized in **11 phases** with clear dependencies:

```
Phase 1: Setup (11 tasks)
  ↓
Phase 2: Foundational (22 tasks) ← BLOCKING: All later phases depend on this
  ↓
Phase 3-7: User Stories (38 tasks) ← Can execute in parallel after Phase 2
  ├─ Phase 3: US1 (12 tasks) — Fetch history MVP
  ├─ Phase 4: US2 (7 tasks) — Incremental sync MVP
  ├─ Phase 5: US3 (9 tasks) — Azure Blob output
  ├─ Phase 6: US4 (6 tasks) — Rate limiting
  └─ Phase 7: US5 (4 tasks) — Dry-run mode
  ↓
Phase 8: Testing (8 tasks) ← Can run in parallel with Phase 3-7
  ↓
Phase 9: Documentation (5 tasks) ← After Phase 8 or in parallel
  ↓
Phase 10: Docker & CI/CD (4 tasks) ← Parallel with Phase 9
  ↓
Phase 11: Polish & Release (10 tasks) ← Final phase
```

### Task Execution Rules

1. **Sequential Phases**: Each phase must complete before the next begins
2. **Parallel Tasks**: Tasks marked [P] can execute in parallel within a phase
3. **TDD Discipline**: Execute ALL test tasks before implementation tasks in each phase
4. **Dependency Tracking**: If task A depends on task B, execute B first
5. **Atomic Completion**: Mark tasks as complete only when all acceptance criteria met

---

## Phase 1: Setup & Project Initialization (11 Tasks)

### Purpose
Initialize Go project structure, dependencies, and build infrastructure

### Task Execution

#### T001-T003: Module & Structure Setup [P]
```bash
# T001: Create Go module
go mod init github.com/user/lastfm-reader
# Creates go.mod with module declaration

# T002: Add dependencies (parallel to T001)
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get go.uber.org/zap@latest
go get github.com/cenkalti/backoff/v4@latest
go get golang.org/x/time@latest
go get github.com/Azure/azure-sdk-for-go/sdk/storage/azblob@latest
# Verify go.mod + go.sum created/updated

# T003: Create directory structure (parallel)
mkdir -p cmd/lastfm-sync/commands
mkdir -p internal/{config,lastfm,watermark,writer,ratelimit,logging,version}
mkdir -p pkg/testutil
mkdir -p .github/workflows
```

**Acceptance Criteria**:
- [ ] `go.mod` exists with correct module path
- [ ] `go.sum` contains all dependencies
- [ ] Directory structure matches plan.md exactly
- [ ] `go mod tidy` succeeds with no errors

#### T004-T008: Build & CI/CD Tooling [P]

**T004: Makefile**
```bash
# Create file: Makefile
# Include targets: deps, lint, test, test-coverage, build, build-all, docker, release
# Each target must have clear documentation and error handling
```

**T005: Linting Config**
```bash
# Create file: .golangci.yml
# Enable strict rules:
# - gocyclo: max 10
# - errcheck: all errors must be handled
# - ineffassign: no unused assignments
# - misspell: catch typos
# Reference: https://golangci-lint.run/usage/configuration/
```

**T006: Dockerfile**
```bash
# Create file: Dockerfile
# Multi-stage: build stage (golang:1.22-alpine) → runtime (distroless)
# Reference the plan.md Dockerfile template
```

**T007-T008: GitHub Actions**
```bash
# Create file: .github/workflows/test.yml
# - Lint with golangci-lint
# - Test with race detector (go test -race ./...)
# - Azurite service for integration tests
# - Coverage upload to codecov

# Create file: .github/workflows/release.yml
# - Tag push (v*) triggers build
# - Cross-compile: linux/amd64, arm64, darwin/amd64, darwin/arm64, windows/amd64
# - Docker build + push to GHCR
# - GitHub Release creation
```

#### T009-T011: Documentation & Licensing [P]

**T009: Build Info Injection**
```go
// cmd/lastfm-sync/main.go
var (
    Version   = "dev"
    BuildTime = ""
    GitCommit = ""
)

func main() {
    // Use these for --version output
}
```

**T010: README.md Template**
```bash
# Create file: README.md
# Sections: Installation, Quick Start, CLI Usage, Examples, Architecture, Testing
# Will be filled in Polish phase (T091)
```

**T011: Licensing**
```bash
# Create files: LICENSE (Apache-2.0 text), NOTICE
```

**Checkpoint**: Phase 1 Complete
- [ ] `go mod tidy` succeeds
- [ ] All configuration files created
- [ ] `make help` shows all targets
- [ ] Linting rules in place
- [ ] GitHub Actions workflows structured

---

## Phase 2: Foundational Infrastructure (22 Tasks)

### Purpose
Build core packages that all user stories depend on. **CRITICAL**: Phase 2 must be 100% complete before Phase 3-7 tasks begin.

### Configuration & Logging (4 Tasks)

#### T012: Logging Setup
```go
// File: internal/logging/logger.go

package logging

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// NewLogger creates a JSON-encoded logger with optional redaction
func NewLogger(level string) (*zap.Logger, error) {
    // Parse level: debug, info, warn, error
    // Create JSON encoder with custom SugaredLogger
    // Return logger or error
}

// RedactSecret redacts sensitive data: returns ****{last4chars}
func RedactSecret(s string) string {
    if len(s) < 4 {
        return "****"
    }
    return "****" + s[len(s)-4:]
}
```

**Acceptance Criteria**:
- [ ] Logger outputs valid JSON
- [ ] `--log-level debug|info` controls verbosity
- [ ] Secrets redacted as `****last4`
- [ ] Unit tests: secret masking, JSON structure validation

#### T013: Config Models
```go
// File: internal/config/models.go

package config

// Config represents the complete application configuration
type Config struct {
    Auth      AuthConfig      `mapstructure:"auth"`
    Client    ClientConfig    `mapstructure:"client"`
    Output    OutputConfig    `mapstructure:"output"`
    Watermark WatermarkConfig `mapstructure:"watermark"`
    Logging   LoggingConfig   `mapstructure:"logging"`
}

// Nested structs for Auth, Client, Output, Watermark, Logging
// Include field tags: mapstructure, yaml, json
// Document defaults inline
```

**Acceptance Criteria**:
- [ ] Struct definitions complete with all fields from plan.md
- [ ] Tags correctly applied for viper (mapstructure)
- [ ] Defaults documented as comments
- [ ] Can be marshaled/unmarshaled from YAML/TOML/JSON

#### T014: Config Loading
```go
// File: internal/config/config.go

package config

import "github.com/spf13/viper"

// LoadConfig merges: flags > env vars > config file > defaults
func LoadConfig(configPath string, flags map[string]interface{}) (*Config, error) {
    v := viper.New()
    
    // Set defaults for all fields
    // Read config file (YAML/TOML/JSON)
    // Bind env vars
    // Merge in flags (override)
    // Unmarshal into Config struct
    
    return &cfg, nil
}
```

**Acceptance Criteria**:
- [ ] Precedence: flags > env > file > defaults
- [ ] Supports YAML/TOML/JSON
- [ ] Path expansion: `~/.lastfm/` → home directory
- [ ] Returns error if required fields missing
- [ ] Unit tests: priority precedence, path expansion

#### T015: Config Validation
```go
// File: internal/config/validation.go

// ValidateConfig checks required fields and returns helpful errors
func ValidateConfig(cfg *Config) error {
    if cfg.Auth.APIKey == "" {
        return errors.New("LASTFM_API_KEY required (set via flag, env, or config file)")
    }
    // ... check other required fields
    return nil
}
```

**Acceptance Criteria**:
- [ ] Validates required fields (API key, output target, Azure if needed)
- [ ] Error messages include remediation steps
- [ ] Unit tests: valid configs pass, invalid fail with good messages

### Last.fm API Client (3 Tasks)

#### T016: Data Models
```go
// File: internal/lastfm/models.go

package lastfm

// ScrobbleResponse from Last.fm API
type ScrobbleResponse struct {
    Artist struct {
        MBID  string `json:"mbid"`
        Text  string `json:"#text"`
    } `json:"artist"`
    // ... other fields matching Last.fm API

    Track struct {
        MBID string `json:"mbid"`
        Text string `json:"#text"`
    } `json:"track"`
    
    Album struct {
        MBID string `json:"mbid"`
        Text string `json:"#text"`
    } `json:"album"`
    
    Date struct {
        UTS  string `json:"uts"`
        Text string `json:"#text"`
    } `json:"date"`
}

// Page represents a paginated result
type Page struct {
    Scrobbles []ScrobbleResponse
    Page      int
    Total     int
    TotalPages int
}
```

**Acceptance Criteria**:
- [ ] JSON struct tags correctly map Last.fm API response
- [ ] Can unmarshal real Last.fm API responses
- [ ] Handles missing optional fields (mbid, etc.)
- [ ] Unit tests: JSON unmarshaling, edge cases

#### T017: HTTP Client
```go
// File: internal/lastfm/client.go

package lastfm

import "net/http"

type Client struct {
    BaseURL    string
    APIKey     string
    Timeout    time.Duration
    HTTPClient *http.Client
    // Rate limiter and retry config (populated later in Phase 6)
}

func NewClient(cfg *config.Config) (*Client, error) {
    c := &Client{
        BaseURL: "https://ws.audioscrobbler.com/2.0/",
        APIKey:  cfg.Auth.APIKey,
        Timeout: cfg.Client.Timeout,
    }
    c.HTTPClient = &http.Client{Timeout: c.Timeout}
    return c, nil
}

func (c *Client) FetchPage(ctx context.Context, username string, from, until int64, pageNum, pageSize int) (*Page, error) {
    // Build query: method=user.getRecentTracks, user, from, to, limit, page
    // Make HTTP request
    // Parse response
    // Return Page or error
}
```

**Acceptance Criteria**:
- [ ] Client construction works
- [ ] FetchPage builds correct URL
- [ ] Timeout enforcement works
- [ ] Handles HTTP errors gracefully
- [ ] Unit tests: basic client, timeout behavior

#### T018: Pagination Helpers
```go
// File: internal/lastfm/pagination.go

func FilterNowPlaying(scrobbles []ScrobbleResponse) []ScrobbleResponse {
    // Remove entries without date.uts (now playing)
    // Return filtered list
}

func IsShortCircuit(page *Page, watermarkUts int64) bool {
    // Return true if page has 0 records with uts > watermarkUts
    // This signals we can stop paginating (no new records)
}
```

**Acceptance Criteria**:
- [ ] FilterNowPlaying correctly excludes entries without date
- [ ] IsShortCircuit detects stop condition
- [ ] Unit tests: filtering, short-circuit detection

### Rate Limiting & Retry (2 Tasks)

#### T019: Rate Limiter
```go
// File: internal/ratelimit/limiter.go

package ratelimit

import "golang.org/x/time/rate"

type Limiter struct {
    limiter *rate.Limiter
    // backoff config (added later)
}

func NewLimiter(qps int, maxRetries int) *Limiter {
    // qps: queries per second (e.g., 3 for Last.fm)
    // Create token bucket: rate.Limit(qps)
    return &Limiter{
        limiter: rate.NewLimiter(rate.Limit(qps), 1),
    }
}

func (l *Limiter) Wait(ctx context.Context) error {
    return l.limiter.Wait(ctx)
}
```

**Acceptance Criteria**:
- [ ] Token bucket rate limiting works
- [ ] `Wait(ctx)` blocks if over rate
- [ ] Unit tests: QPS enforcement (measure time)

#### T020: Backoff & Retry
```go
// File: internal/ratelimit/backoff.go

package ratelimit

import "github.com/cenkalti/backoff/v4"

func DoWithRetry(ctx context.Context, fn func() error, maxRetries int) error {
    // Execute fn() with exponential backoff
    // Sequence: 1s, 2s, 4s, 8s, max 32s
    // Retry up to maxRetries times
    // Return error after max retries exceeded
}

func IsRetryable(err error) bool {
    // 429 (Too Many Requests) → true
    // 5xx errors → true
    // Context deadline/timeout → true
    // 4xx errors (except 429) → false
}

func ParseRetryAfter(headers http.Header) *time.Duration {
    // Extract Retry-After header if present
    // Return duration or nil
}
```

**Acceptance Criteria**:
- [ ] Exponential backoff sequence correct
- [ ] Max retries enforced
- [ ] Retry-After header parsed if present
- [ ] Error classification correct
- [ ] Unit tests: backoff sequence, max retries, error types

### Writer Interface & Local Implementation (2 Tasks)

#### T021: Writer Interface & Scrobble Type
```go
// File: internal/writer/writer.go

package writer

import "encoding/json"

type Scrobble struct {
    Username   string          `json:"username"`
    Artist     string          `json:"artist"`
    Track      string          `json:"track"`
    Album      string          `json:"album"`
    UTS        int64           `json:"uts"`
    MBID       *string         `json:"mbid,omitempty"`
    Source     string          `json:"source"`
    IngestedAt string          `json:"ingested_at"`
    Raw        json.RawMessage `json:"raw"`
}

type Writer interface {
    WriteBatch(ctx context.Context, records []Scrobble) error
    Flush(ctx context.Context) error
    Close(ctx context.Context) error
}
```

**Acceptance Criteria**:
- [ ] Scrobble struct matches spec exactly
- [ ] JSON tags correct for NDJSON format
- [ ] Can marshal Scrobble to JSON
- [ ] Unit tests: marshaling, field validation

#### T022: LocalWriter Implementation
```go
// File: internal/writer/local.go

type LocalWriter struct {
    filePath string
    file     *os.File
    writer   *bufio.Writer
}

func NewLocalWriter(filepath string) (*LocalWriter, error) {
    // Create or open existing file (append mode)
    // Initialize buffered writer
}

func (w *LocalWriter) WriteBatch(ctx context.Context, records []Scrobble) error {
    // Write each record as NDJSON (one JSON per line)
    // Buffer writes for efficiency
}

func (w *LocalWriter) Flush(ctx context.Context) error {
    // Flush buffer
    // fsync to disk
}

func (w *LocalWriter) Close(ctx context.Context) error {
    // Close file handle
    // Cleanup
}
```

**Acceptance Criteria**:
- [ ] Appends to existing file (idempotent)
- [ ] NDJSON format: one JSON per line
- [ ] Flush calls fsync
- [ ] Error handling: permission denied, disk full
- [ ] Unit tests: append, fsync, NDJSON format, error handling

### Watermark Store Interface & File Implementation (2 Tasks)

#### T023: WatermarkStore Interface & Watermark Type
```go
// File: internal/watermark/store.go

package watermark

type Watermark struct {
    Username  string    `json:"username"`
    MaxUts    int64     `json:"max_uts"`
    UpdatedAt time.Time `json:"updated_at"`
    SyncID    string    `json:"sync_id"`
}

type WatermarkStore interface {
    Get(ctx context.Context, username string) (uts int64, exists bool, err error)
    Put(ctx context.Context, username string, uts int64) error
}
```

**Acceptance Criteria**:
- [ ] Watermark struct matches spec
- [ ] Can marshal/unmarshal to JSON
- [ ] Interface is clean and minimal

#### T024: FileStore Implementation
```go
// File: internal/watermark/file.go

type FileStore struct {
    basePath string
}

func NewFileStore(basePath string) *FileStore {
    return &FileStore{basePath: basePath}
}

func (s *FileStore) Get(ctx context.Context, username string) (int64, bool, error) {
    // Read watermark file: {basePath}/{username}.watermark
    // Parse JSON: extract max_uts
    // Return (uts, exists=true, nil) or (0, exists=false, nil) or error
}

func (s *FileStore) Put(ctx context.Context, username string, uts int64) error {
    // Create Watermark struct with current timestamp, sync ID
    // Write to temp file: {basePath}/.{username}.watermark.tmp
    // Atomic rename: tmp → final
}
```

**Acceptance Criteria**:
- [ ] Read/write watermark JSON correctly
- [ ] Atomic writes (temp file + rename)
- [ ] Handles missing files gracefully
- [ ] Unit tests: read, write, atomic updates, concurrent access

### Azure Writer & Watermark Store Stubs (2 Tasks)

#### T025: AzureWriter Stub
```go
// File: internal/writer/azure.go

type AzureWriter struct {
    // Container client, prefix, temp dir (to be filled)
}

func NewAzureWriter(container, prefix, username, tempDir string) (*AzureWriter, error) {
    // Stub: return error or incomplete implementation
    // Full implementation in Phase 5 (US3)
}

func (w *AzureWriter) WriteBatch(ctx context.Context, records []Scrobble) error {
    // Stub
}

func (w *AzureWriter) Flush(ctx context.Context) error {
    // Stub
}

func (w *AzureWriter) Close(ctx context.Context) error {
    // Stub
}
```

#### T026: AzureStore Stub
```go
// File: internal/watermark/azure.go

type AzureStore struct {
    // Container client, prefix (to be filled)
}

func (s *AzureStore) Get(ctx context.Context, username string) (int64, bool, error) {
    // Stub
}

func (s *AzureStore) Put(ctx context.Context, username string, uts int64) error {
    // Stub
}
```

### CLI Framework & Command Skeleton (5 Tasks)

#### T027: Main Entry Point
```go
// File: cmd/lastfm-sync/main.go

package main

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "lastfm-sync",
    Short: "Sync Last.fm scrobbles to local or Azure storage",
    Long:  "...",
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

#### T028: Fetch Command Skeleton
```go
// File: cmd/lastfm-sync/commands/fetch.go

func NewFetchCmd() *cobra.Command {
    var (
        user      string
        since     int64
        until     int64
        output    string
        // ... all other flags
    )
    
    cmd := &cobra.Command{
        Use:   "fetch",
        Short: "Fetch scrobbles from Last.fm",
        Long:  "...",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Stub: will be filled in Phase 3
            return nil
        },
    }
    
    cmd.Flags().StringVar(&user, "user", "", "Last.fm username (required)")
    // ... bind all other flags
    
    return cmd
}

func init() {
    rootCmd.AddCommand(NewFetchCmd())
}
```

#### T029-T031: Other Commands
```go
// Files: commands/show-watermark.go, set-watermark.go, version.go
// Stubs only (full implementation in Phase 11)
```

### Test Utilities & Fixtures (2 Tasks)

#### T032: Mock Data Generators
```go
// File: pkg/testutil/fixtures.go

func GenerateScrobbles(count int) []writer.Scrobble {
    // Generate count fake scrobbles with realistic data
}

func MockLastfmPage(pageNum, pageSize int, maxUts int64) *lastfm.Page {
    // Generate mock Last.fm API response
}

func MockHTTPResponse(pageNum, pageSize int) *http.Response {
    // Create mock HTTP response body
}
```

#### T033: Mock Implementations
```go
// File: pkg/testutil/mocks.go

type MockWriter struct {
    WriteBatches [][]writer.Scrobble
    FlushCount   int
}

func (m *MockWriter) WriteBatch(ctx context.Context, records []writer.Scrobble) error {
    m.WriteBatches = append(m.WriteBatches, records)
    return nil
}

// ... similar for WatermarkStore, LastfmClient
```

**Checkpoint: Phase 2 Complete**
- [ ] All packages compile: `go build ./...`
- [ ] All interfaces defined and testable
- [ ] Configuration system works
- [ ] Logging configured
- [ ] Build tooling functional

---

## Phase 3: User Story 1 - Initial Full Scrobble History Fetch (P1 MVP) (12 Tasks)

### Overview
Users can fetch their complete Last.fm scrobble history to a local NDJSON file.

### Test Tasks (TDD: Write First)

#### T034: Last.fm Client Pagination Tests
```go
// File: internal/lastfm/client_test.go

func TestFetchPage(t *testing.T) {
    // Create mock HTTP server
    // Mock Last.fm API responses for multiple pages
    
    // Test: Page 1 with 200 records
    // Test: Page 2 with 200 records
    // Test: Last page with < 200 records
    
    // Verify: from parameter passed correctly
    // Verify: all records parsed
    // Verify: pagination info (page, total, totalPages)
}

func TestFilterNowPlaying(t *testing.T) {
    // Test: "now playing" entries (no date.uts) filtered out
    // Test: normal scrobbles preserved
}
```

#### T035: LocalWriter Tests
```go
// File: internal/writer/local_test.go

func TestNDJSONFormat(t *testing.T) {
    writer := NewLocalWriter(tmpFile)
    defer writer.Close(context.Background())
    
    records := testutil.GenerateScrobbles(10)
    writer.WriteBatch(context.Background(), records)
    writer.Flush(context.Background())
    
    // Verify: each line is valid JSON
    // Verify: all fields present
}

func TestAppendBehavior(t *testing.T) {
    // Test: multiple WriteBatch calls append correctly
    // Test: file reopened, new data appended (not overwritten)
}
```

#### T036: FileStore Watermark Tests
```go
// File: internal/watermark/file_test.go

func TestAtomicWrite(t *testing.T) {
    store := NewFileStore(tmpDir)
    
    store.Put(context.Background(), "alice", 1000)
    uts, exists, err := store.Get(context.Background(), "alice")
    
    // Verify: uts = 1000
    // Verify: exists = true
    // Verify: file exists at expected path
}
```

#### T037: Integration Test - Full Fetch Flow
```go
// File: cmd/lastfm-sync/integration_test.go

func TestFullFetch(t *testing.T) {
    // Setup: mock Last.fm API with 3 pages (200 records each)
    // Execute: lastfm-sync fetch --user alice --output local
    // Verify:
    //   - Output file created
    //   - Contains 600 NDJSON records
    //   - No duplicates
    //   - Watermark file created
    //   - Watermark = max uts
}
```

### Implementation Tasks

#### T038: Retry Middleware in Client
```go
// Integrate ratelimit.DoWithRetry into Client.FetchPage
// Wrap HTTP request with retry logic
// Test: 429 response triggers retry + backoff
// Test: 5xx response triggers retry
```

#### T039: Complete LocalWriter
```go
// Add SetUsername() method
// Ensure all NDJSON records include username, ingested_at
// Add comprehensive error handling
```

#### T040: Complete FileStore
```go
// Implement atomic writes with proper temp file handling
// Add file locking if concurrent access expected
```

#### T041: Fetch Command Main Logic
```go
// cmd/lastfm-sync/commands/fetch.go
// Main fetch loop:
// 1. Load configuration from flags/env/file
// 2. Validate required fields
// 3. Create clients: LastfmClient, LocalWriter, FileStore
// 4. Load watermark for user
// 5. Calculate effective_from = max(--since, watermark)
// 6. Pagination loop:
//    - FetchPage(username, from, to, page)
//    - Transform to Scrobble records
//    - WriteBatch(records)
//    - UpdateWatermark(max_uts)
//    - Check short-circuit condition
// 7. Exit cleanly
```

#### T042: Structured Logging
```go
// Add logging events:
// - fetch.start: user, output_mode, watermark value
// - fetch.page: page_num, total_pages, record_count, duration
// - fetch.write: batch_size, bytes, duration
// - watermark.update: new_uts, store_type
// - fetch.complete: total_records, total_duration, avg_rate
```

#### T043: Error Handling & User Messages
```go
// Handle errors:
// - Missing API key: "LASTFM_API_KEY required..."
// - Invalid user: "User 'xyz' not found or private"
// - File permission: "Cannot write to {path}: permission denied"
// - Azure auth: "Azure credentials invalid..."
// All errors with exit code 1
```

#### T044: Dry-Run Mode
```go
// Add --dry-run flag
// When set: skip API calls and writes
// Log: "Would fetch X pages, Y records to {path}"
// Return exit code 0
```

**Checkpoint: Phase 3 Complete**
- [ ] User can fetch full history: `lastfm-sync fetch --user alice`
- [ ] NDJSON file created with all records
- [ ] Watermark file created
- [ ] Tests pass: `go test ./internal/lastfm ./internal/writer ./cmd/lastfm-sync`
- [ ] Coverage > 80%

---

## Phase 4: User Story 2 - Incremental Sync with Watermarking (P1 MVP) (7 Tasks)

### Overview
Subsequent runs fetch only new scrobbles; watermark-based idempotent resumption; crash-safe.

### Test Tasks

#### T045: Watermark Logic Tests
```go
// Test: effective_from = max(--since, watermark)
// Test: when --since provided, use if > watermark
// Test: when --since not provided, use watermark
// Test: short-circuit: 0 new records → stop pagination
```

#### T046: Idempotency Tests
```go
// Test: write same batch twice → no duplicates
// Test: uniqueness check: (username, uts, artist, track)
```

#### T047: Incremental Sync Integration Test
```go
// Run 1: Mock API pages 1-3 (uts 100-300) → watermark = 300
// Run 2: Mock API pages 1-2 (old) + 3-5 (new, 301-500)
// Verify: only pages 3-5 fetched
// Verify: no duplicates
// Verify: watermark = 500
```

#### T048: Crash Recovery Test
```go
// Simulate: kill process after page write but before watermark update
// Rerun: verify page 1 NOT duplicated (watermark still 0)
// Verify: correct recovery from page 1
```

### Implementation Tasks

#### T049: FetchPage `from` Parameter
```go
// Update LastfmClient.FetchPage signature to accept `from` parameter
// Pass as Last.fm API `from` parameter
// Test: correct URL parameter passed
```

#### T050: Fetch Command Load Watermark
```go
// Before pagination loop:
// 1. watermark, exists, err := store.Get(ctx, username)
// 2. effective_from := max(flags.Since, watermark)
// 3. Pass effective_from to FetchPage
```

#### T051: Short-Circuit Logic
```go
// After each page:
// if IsShortCircuit(page, watermark) {
//     break // Stop pagination
// }
```

#### T052: Atomic Watermark Updates
```go
// Update watermark AFTER each successful page flush
// Use atomic file writes (temp + rename)
// Test: kill after write, before watermark → correct recovery
```

#### T053: --since Flag Validation
```go
// If both --since and watermark present: use max
// If --until < --since: error
// Test: all combinations
```

**Checkpoint: Phase 4 Complete**
- [ ] Incremental sync working: run twice, second fetches only delta
- [ ] Crash recovery tested and working
- [ ] Watermark file updated atomically
- [ ] All tests pass
- [ ] No duplicate records on re-run

---

## Phase 5: User Story 3 - Azure Blob Storage Output (P2) (9 Tasks)

### Overview
Write scrobbles to Azure Blob Storage with time-partitioned paths; managed identity/SAS auth.

### Implementation Steps

1. **Azure SDK Setup**
   - Import: `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`
   - Create container client based on auth method

2. **AzureWriter Complete Implementation**
   - Temp file buffering (reuse LocalWriter pattern)
   - Flush: upload to Azure with path: `{prefix}dt=YYYY-MM-DD/{username}-HHMMSS.ndjson`
   - Close: delete temp file on success

3. **AzureStore Complete Implementation**
   - Get: read watermark blob
   - Put: write with ETag concurrency check

4. **Fetch Command Azure Support**
   - Parse `--output azure` flag
   - Parse Azure-specific flags: `--azure-container`, `--azure-auth`, etc.
   - Create appropriate writer/store

5. **Auto Watermark Store Selection**
   - If `--output azure` and no `--watermark-store`: auto-use azure
   - Otherwise: use file (default)

**Checkpoint: Phase 5 Complete**
- [ ] Azure blob writer uploads correctly
- [ ] Time partitioning works: `dt=2025-10-30/alice-HHMMSS.ndjson`
- [ ] Watermark stored in Azure
- [ ] Incremental sync with Azure works
- [ ] Integration tests pass with Azurite

---

## Phase 6: User Story 4 - Rate Limit Compliance & Backoff (P2) (6 Tasks)

### Overview
Respect Last.fm 3 QPS limit; automatic exponential backoff on 429/5xx.

### Implementation Steps

1. **Integrate Rate Limiter**
   - Every HTTP request: `limiter.Wait(ctx)` before execution
   - Ensures max 3 requests per second

2. **Implement Retry-After Parsing**
   - Extract header if present
   - Use specified duration instead of backoff sequence

3. **Add Flags**
   - `--qps`: queries per second (default 3)
   - `--timeout`: per-request timeout (default 15s)

4. **Timeout Enforcement**
   - Set `http.Client.Timeout` from config
   - Verify timeout errors retryable

5. **Retry Metrics**
   - Log retry events with attempt number, reason, backoff duration
   - Track counters: `retries_total`, `rate_limit_hits`

**Checkpoint: Phase 6 Complete**
- [ ] 3 QPS throttle enforced: 100 requests take ≥ 33 seconds
- [ ] 429 response triggers backoff + retry
- [ ] 5xx response triggers backoff + retry
- [ ] Retry-After header honored if present
- [ ] Tests pass: rate limiting, backoff sequence

---

## Phase 7: User Story 5 - Dry Run & Debugging (P3) (4 Tasks)

### Overview
Preview fetch plan without API calls or writes; debug with `--log-level debug`.

### Implementation Steps

1. **Dry-Run Flag**
   - Add `--dry-run` boolean flag
   - When true: skip API calls and writes
   - Log detailed plan

2. **Mock Writer & Store for Testing**
   - Implement Writer interface for testing
   - Track calls without writing

3. **Debug Event Details**
   - Page-by-page details when `--log-level debug`
   - Show: page_num, total_pages, records, bytes, duration

4. **Help Text & Examples**
   - `fetch --help` shows all flags
   - Include example commands

**Checkpoint: Phase 7 Complete**
- [ ] Dry-run mode works: no writes, no API calls
- [ ] Debug logging shows detailed plan
- [ ] Help text comprehensive

---

## Phase 8: Shared Testing & Quality Gates (8 Tasks)

### Focus
Cross-cutting tests, performance benchmarks, code quality validation.

### Key Tests

**T083**: Config loading from file
**T084**: Help and version commands
**T085**: Performance benchmark (NDJSON write throughput > 200/sec)
**T086**: Rate limiter overhead (< 1ms per call)
**T087**: Linting + race detection + coverage ≥ 80%
**T088**: Contract tests (NDJSON format, watermark structure)
**T089**: Full multi-page fetch (10 pages = 5K scrobbles)
**T090**: Mixed watermark modes (local + Azure)

**Checkpoint: Phase 8 Complete**
- [ ] All tests pass: `go test -race ./...`
- [ ] Coverage ≥ 80% overall, ≥ 90% critical packages
- [ ] Benchmarks meet targets
- [ ] Linting clean
- [ ] No race conditions

---

## Phase 9: Documentation & Examples (5 Tasks)

### T091: README.md
Sections:
- Installation (build, download, docker)
- Quick Start
- CLI Usage (all flags, env vars, config file)
- Examples (local, Azure, Kubernetes)
- Architecture
- Testing guide

### T092: Inline Code Comments
- Retry logic (client.go)
- Watermark atomicity (file.go)
- Rate limiter (limiter.go)
- Azure blob upload (azure.go)

### T093: ARCHITECTURE.md
- Component diagram
- Data flow
- Error handling strategies

### T094: SECURITY.md
- Secret redaction
- Credential handling
- Container security
- Access controls

### T095: Example Scripts & Configs
- `examples/local-fetch.sh`
- `examples/azure-fetch.sh`
- `examples/config.yaml`
- `examples/k8s-cronjob.yaml`

**Checkpoint: Phase 9 Complete**
- [ ] README complete and accurate
- [ ] All examples tested
- [ ] Architecture documented
- [ ] Security guide written

---

## Phase 10: Docker & CI/CD (4 Tasks)

### T096: Dockerfile Validation
- Build image
- Verify distroless runtime
- Verify non-root user
- Test runs

### T097: GitHub Actions Test Workflow
- Linting
- Tests with race detector
- Azurite service for integration tests
- Coverage upload

### T098: GitHub Actions Release Workflow
- Cross-compile binaries
- Docker build + push
- GitHub Release creation

### T099: Branch Protection
- Require passing checks
- Require PR review

**Checkpoint: Phase 10 Complete**
- [ ] Docker image builds successfully
- [ ] GitHub Actions workflows functional
- [ ] Branch protection rules in place

---

## Phase 11: Polish & Release Preparation (10 Tasks)

### T100-T101: Watermark Commands
- `show-watermark` displays current value
- `set-watermark` allows manual override

### T102-T104: Final Validation
- Full test suite passes
- Code review checklist complete
- Manual testing scenarios verified

### T105-T110: Release
- CHANGELOG.md complete
- Version finalized (v1.0.0)
- Tag release: `git tag -a v1.0.0`
- Verify GitHub Release

**Checkpoint: Phase 11 Complete**
- [ ] v1.0.0 released
- [ ] All binaries published
- [ ] Docker images on GHCR
- [ ] GitHub Release with changelog

---

## Execution Tracking & Progress

### Task Status Template

For each task, track:
- **Status**: Not Started → In Progress → Testing → Complete
- **Assignee**: Who is working on it
- **Start Date**: When work began
- **Est. Completion**: When done
- **Issues**: Any blockers or problems
- **Code Review**: PR link (when applicable)

### Example Tracking Format

```markdown
## Phase 2 Progress

- [x] T012 Logging Setup (COMPLETE) - alice - PR#123
- [x] T013 Config Models (COMPLETE) - bob - PR#124
- [ ] T014 Config Loading (IN PROGRESS) - alice - ETA: 2025-11-02
- [ ] T015 Config Validation (NOT STARTED) - bob - ETA: 2025-11-03
```

### Checkpoint Validation

At each phase checkpoint, verify:

```bash
# Phase 2 validation example
cd /home/wesleyb/git/LastFMReaderv3/LastFmReaderv3

# 1. Code compiles
go build ./...

# 2. Tests pass
go test -race ./...

# 3. Coverage acceptable
go test -cover ./... | grep coverage

# 4. Linting passes
make lint

# 5. No uncommitted changes (ready for PR)
git status
```

---

## Common Implementation Patterns

### TDD Pattern: Red-Green-Refactor

1. **Red**: Write test that fails
   ```bash
   go test -run TestXxx -v ./... # Should fail
   ```

2. **Green**: Implement to make test pass
   ```bash
   go test -run TestXxx -v ./... # Should pass
   ```

3. **Refactor**: Clean up code
   ```bash
   go fmt ./...
   go vet ./...
   ```

### Error Handling Pattern

```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Log errors with structured logging
logger.Error("operation failed", zap.Error(err))
```

### Testing Pattern

```go
func TestSomething(t *testing.T) {
    // Arrange
    setup := setupTestEnvironment()
    
    // Act
    result, err := doSomething(setup)
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != expected {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

---

## Performance Targets (from Constitution & Plan)

| Metric | Target | Measurement |
|--------|--------|-------------|
| Initial fetch (10K scrobbles) | < 30 seconds | E2E with real API |
| Incremental re-run | < 5 seconds | Watermark lookup + delta |
| NDJSON write throughput | > 200 scrobbles/sec | Benchmark test |
| Rate limiter overhead | < 1ms per request | Benchmark test |
| Code coverage | ≥ 80% overall, ≥ 90% critical | go test -cover |

---

## Code Quality Checklist

For every PR, verify:

- [ ] Tests written first (TDD)
- [ ] Tests pass: `go test -race ./...`
- [ ] Linting clean: `make lint`
- [ ] Coverage adequate: ≥ 80% new code
- [ ] Error messages clear and actionable
- [ ] Logging structured (JSON)
- [ ] No secrets logged
- [ ] Code reviewed
- [ ] All comments explain WHY, not WHAT
- [ ] No dead code or unused imports

---

## Key Contacts & Resources

### Last.fm API
- **Docs**: https://www.last.fm/api/
- **Rate Limit**: 3 requests per second (+ burst)
- **Endpoint**: `user.getRecentTracks`
- **Auth**: API key via parameter

### Azure SDK for Go
- **Docs**: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob
- **DefaultAzureCredential**: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#NewDefaultAzureCredential
- **Azurite** (local testing): https://github.com/Azure/Azurite

### Go Best Practices
- **Error Handling**: https://pkg.go.dev/errors
- **Context**: https://pkg.go.dev/context
- **Testing**: https://pkg.go.dev/testing
- **Logging**: https://pkg.go.dev/go.uber.org/zap

---

## Final Checklist Before v1.0.0 Release

- [ ] All 110 tasks marked complete
- [ ] Tests pass: `go test -race ./...`
- [ ] Coverage ≥ 80%
- [ ] Linting clean: `make lint`
- [ ] Docker image builds: `make docker`
- [ ] README complete and tested
- [ ] CHANGELOG written
- [ ] Version tag created: `v1.0.0`
- [ ] GitHub Release published
- [ ] Binaries uploaded
- [ ] Docker images pushed to GHCR
- [ ] Security review completed
- [ ] Performance targets met

---

**Status**: Implementation guide complete. Ready to begin Phase 1.  
**Next Step**: Start with Phase 1 tasks (Setup & Project Initialization)  
**Questions**: Refer to spec.md (requirements), plan.md (design), or tasks.md (execution)

