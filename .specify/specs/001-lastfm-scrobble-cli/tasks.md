---
description: "Task list for Last.fm Scrobble CLI implementation"
---

# Tasks: Last.fm Scrobble CLI with Incremental Sync

**Input**: Design documents from `.specify/specs/001-lastfm-scrobble-cli/`  
**Prerequisites**: `plan.md` (tech stack, structure), `spec.md` (user stories with priorities)  
**Tests**: Test tasks included per TDD approach  
**Organization**: Tasks grouped by user story for independent implementation and testing

## Format: `- [ ] [ID] [P?] [Story?] Description`

- **Checkbox**: ALWAYS starts with `- [ ]`
- **[ID]**: Unique task identifier (T001, T002, etc.)
- **[P]**: Task can run in parallel with others (different files, no dependencies)
- **[Story]**: User story identifier (US1, US2, US3, US4, US5)
- **File paths**: Relative to repository root; Go package structure per `plan.md`

---

## Phase 1: Setup & Project Initialization

**Purpose**: Initialize Go project, dependencies, and build infrastructure

- [X] T001 Create Go module structure with go.mod and go.sum
- [X] T002 [P] Create directory structure: cmd/lastfm-sync/, internal/{config,lastfm,watermark,writer,ratelimit,logging,version}/
- [X] T003 [P] Create Makefile with targets: deps, lint, test, test-coverage, build, build-all, docker, release
- [X] T004 [P] Create .golangci.yml with linting rules (gocyclo < 10, errcheck, ineffassign)
- [X] T005 [P] Create Dockerfile (multi-stage: golang:1.24-alpine build + distroless runtime)
- [X] T006 [P] Create .github/workflows/test.yml (lint, test, coverage)
- [X] T007 [P] Create .github/workflows/release.yml (build, docker, GitHub release)
- [X] T008 [P] Create README.md template with installation and usage sections
- [X] T009 [P] Create LICENSE (Apache-2.0) and NOTICE files

---

## Phase 2: Foundational Infrastructure (Blocking Prerequisites)

**Purpose**: Core packages that all user stories depend on; MUST be complete before story implementation

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Configuration & Logging

- [X] T010 [P] Create internal/logging/logger.go with zap setup and secret redaction
  - Implement NewLogger(level string) returns *zap.Logger
  - Implement redactSecret(s string) returns ****{last4chars}
  - Unit tests for secret masking and JSON output

- [X] T011 [P] Create internal/config/types.go with Config struct hierarchy
  - Define Config, Auth, Client, Output, Watermark, Logging structs
  - Add mapstructure, yaml, json tags for Viper
  - Document default values inline

- [X] T012 [P] Create internal/config/defaults.go with default configuration values
  - Define constants for default paths, timeouts, QPS limits
  - Implement GetDefaults() function

- [X] T013 Create internal/config/loader.go with Viper-based configuration loading
  - Implement LoadConfig(flags, env, file) with priority: flags > env > file > defaults
  - Support path expansion for ~/.lastfm/ and $XDG_CONFIG_HOME
  - Unit tests for config merging and precedence

- [X] T014 [P] Create internal/config/validation.go with configuration validation
  - Implement ValidateConfig(*Config) with clear error messages
  - Validate required fields: API key, output target, Azure credentials if needed
  - Unit tests for valid/invalid configs

### Models & Data Structures

- [X] T015 [P] Create internal/models/scrobble.go with Scrobble struct
  - Define Scrobble with fields: Username, Artist, Track, Album, UTS, MBID, Source, IngestedAt, Raw
  - Add JSON tags for NDJSON serialization
  - Implement uniqueness key method: username + uts + artist + track
  - Unit tests for serialization and uniqueness

### Last.fm API Client (Core)

- [X] T016 [P] Create internal/lastfm/client.go with HTTP client
  - Define Client struct with BaseURL, APIKey, HTTPClient, Limiter
  - Implement NewClient(config) returns *Client
  - Implement buildURL() for API endpoint construction
  - Unit tests for client initialization

- [X] T017 [P] Create internal/lastfm/pagination.go with Page struct and helpers
  - Define Page struct with Scrobbles, PageNum, TotalPages, Total
  - Implement FilterNowPlaying(scrobbles) to exclude entries without uts
  - Implement ParseResponse() for Last.fm API JSON parsing
  - Unit tests for filtering and parsing

### Rate Limiting & Retry

- [X] T018 [P] Create internal/ratelimit/limiter.go with rate limiter
  - Define Limiter struct wrapping golang.org/x/time/rate.Limiter
  - Implement NewLimiter(qps int) returns *Limiter
  - Implement Wait(ctx) for token bucket enforcement
  - Unit tests for QPS enforcement timing

- [X] T019 [P] Create internal/ratelimit/retry.go with exponential backoff
  - Implement DoWithRetry(ctx, fn, maxRetries) with exponential backoff (1s, 2s, 4s, 8s, max 32s)
  - Implement ParseRetryAfter(headers) to extract Retry-After header
  - Unit tests for backoff timing and retry limits

### CLI Foundation

- [X] T020 Create cmd/lastfm-sync/main.go with cobra root command setup
  - Initialize root command with version and global flags
  - Setup logging and config loading
  - Add version command with build info

- [X] T021 [P] Create internal/version/version.go with build information
  - Define Version, BuildTime, GitCommit variables (set via ldflags)
  - Implement GetVersionString() formatted output
  - Unit tests for version string formatting

**Checkpoint**: Foundation complete - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Initial Full Scrobble History Fetch (Priority: P1) 🎯 MVP

**Goal**: User can export complete Last.fm scrobble history to local NDJSON file for data portability

**Independent Test**: Mock Last.fm API with paginated scrobbles → run `lastfm-sync fetch --user alice --output local` → verify output file contains correctly formatted NDJSON with all records and no duplicates

### Tests for User Story 1 (TDD: Write First)

- [X] T022 [P] [US1] Unit test: LastfmClient.FetchPage in internal/lastfm/client_test.go
  - Test with mock HTTP server returning paginated responses
  - Test timeout handling
  - Test malformed JSON response handling
  - Test "now playing" filtering

- [X] T023 [P] [US1] Unit test: LocalWriter NDJSON output in internal/writer/local_test.go
  - Verify NDJSON format (one JSON per line)
  - Verify file creation and append mode
  - Verify atomic flush with fsync
  - Verify file permissions

- [X] T024 [P] [US1] Unit test: FileStore watermark in internal/watermark/file_test.go
  - Verify watermark file creation
  - Verify Get() returns correct value and exists=true/false
  - Verify Put() with atomic write
  - Test concurrent access scenarios

- [X] T025 [US1] Integration test: Full fetch → write → watermark flow in cmd/lastfm-sync/commands/fetch_test.go
  - Mock Last.fm API with 3 pages (200 records each)
  - Run fetch command
  - Verify NDJSON output file with 600 records
  - Verify watermark file created with max uts
  - Verify no duplicates (uniqueness check)

### Implementation for User Story 1

- [X] T026 [US1] Implement LastfmClient.FetchPage in internal/lastfm/client.go
  - Add FetchPage(ctx, username, from, until, pageNum, pageSize) method
  - Integrate HTTP request with rate limiting
  - Parse API response and filter now playing
  - Return Page with scrobbles and metadata

- [X] T027 [US1] Implement retry middleware in internal/lastfm/client.go
  - Integrate ratelimit.DoWithRetry for resilience
  - Handle 429 (rate limit) with exponential backoff
  - Handle 5xx (server error) with retry
  - Test with mock 429 and 5xx responses

- [X] T028 [P] [US1] Create internal/writer/writer.go with Writer interface
  - Define Writer interface: WriteBatch(ctx, []Scrobble), Flush(ctx), Close(ctx)
  - Add SetUsername(username string) for tracking user
  - Document writer contract and error handling

- [X] T029 [US1] Implement LocalWriter in internal/writer/local.go
  - Implement Writer interface for local NDJSON file output
  - Buffer records and write on Flush()
  - Use fsync for durability
  - Handle file creation, append mode, permissions
  - Unit tests for write correctness

- [X] T030 [P] [US1] Create internal/watermark/store.go with WatermarkStore interface
  - Define interface: Get(ctx, username) returns (uts, exists, error)
  - Define interface: Put(ctx, username, uts) returns error
  - Document watermark contract

- [X] T031 [US1] Implement FileStore in internal/watermark/file.go
  - Implement WatermarkStore for local JSON file
  - Store at ~/.lastfm/{username}.watermark
  - Use atomic write: temp file + rename
  - Handle file not found (first run)
  - Unit tests for atomic write and concurrency

- [X] T032 [US1] Create cmd/lastfm-sync/commands/fetch.go with fetch command
  - Define fetch command with cobra
  - Add flags: --user, --since, --until, --page-size, --max-pages, --output, --out-path
  - Parse and validate flags
  - Wire up config, client, writer, watermark store
  - Implement orchestration logic (placeholder for now)

- [X] T033 [US1] Implement fetch command logic in cmd/lastfm-sync/commands/fetch.go
  - Load config and validate
  - Create Last.fm client, writer, watermark store
  - Implement pagination loop: FetchPage → WriteBatch → Flush
  - Add structured logging: fetch.start, fetch.page, fetch.write, fetch.complete
  - Track metrics: pages_fetched, records_written, bytes_written, total_duration
  - Handle errors with clear messages

- [X] T034 [US1] Add error handling and user-friendly messages in fetch command
  - Handle API key missing with actionable message
  - Handle invalid username with Last.fm API error
  - Handle file permission denied with clear path
  - Exit codes: 0 on success, 1 on failure

- [X] T035 [US1] Implement --dry-run mode in fetch command
  - Skip API calls and writes when --dry-run flag set
  - Log preview: "Would fetch X pages, Y records to {output_path}"
  - Verify watermark loading (but don't update)

**Checkpoint**: User Story 1 complete. User can fetch full history to local file. P1 MVP foundation ready.

---

## Phase 4: User Story 2 - Incremental Sync with Watermarking (Priority: P1) 🎯 MVP

**Goal**: Subsequent runs fetch only new scrobbles since last sync; watermark-based idempotent resumption

**Independent Test**: First run fetches all → second run (with new API data) fetches only delta → verify no duplicates and watermark updated

### Tests for User Story 2 (TDD: Write First)

- [X] T036 [P] [US2] Unit test: Watermark logic in internal/lastfm/client_test.go
  - Test effective lower bound: max(--since, watermark)
  - Test watermark override with --since flag
  - Verify from parameter passed to API correctly

- [X] T037 [P] [US2] Unit test: Short-circuit logic in internal/lastfm/pagination_test.go
  - Test IsShortCircuit(page, watermark) when no records > watermark
  - Verify pagination stops early
  - Test with various watermark values

- [X] T038 [P] [US2] Unit test: Idempotency in internal/writer/local_test.go
  - Write same batch twice to verify no duplicates in output
  - Verify uniqueness by (username, uts, artist, track)
  - Test append vs overwrite behavior

- [X] T039 [US2] Integration test: Incremental sync flow in cmd/lastfm-sync/commands/fetch_test.go
  - Run 1: Mock API returns pages 1-3 (uts 100-300), verify watermark = 300
  - Run 2: Mock API returns pages 1-2 (uts 100-200) + pages 3-5 (new, uts 301-500)
  - Verify only pages 3-5 fetched (effective_from = 300)
  - Verify output file has 600 total records (no duplicates)
  - Verify watermark updated to 500

- [X] T040 [US2] Integration test: Crash recovery in cmd/lastfm-sync/commands/fetch_test.go
  - Simulate process kill after page 1 written but before watermark update
  - Rerun and verify page 1 records NOT duplicated
  - Verify correct recovery from watermark = 0

### Implementation for User Story 2

- [X] T041 [US2] Enhance LastfmClient.FetchPage to accept from parameter in internal/lastfm/client.go
  - Add from int64 parameter to FetchPage method
  - Pass from as query parameter to Last.fm API
  - Unit tests: verify from parameter in request URL

- [X] T042 [US2] Implement short-circuit detection in internal/lastfm/pagination.go
  - Implement IsShortCircuit(page *Page, watermark int64) bool
  - Check if all scrobbles have uts <= watermark
  - Return true if no new data to process

- [X] T043 [US2] Enhance fetch command to load and use watermark in cmd/lastfm-sync/commands/fetch.go
  - Load watermark before pagination loop
  - Calculate effective_from = max(--since flag, watermark.max_uts)
  - Pass effective_from to FetchPage
  - Log effective lower bound

- [X] T044 [US2] Implement watermark updates after each page in cmd/lastfm-sync/commands/fetch.go
  - After each successful Flush(), update watermark with max uts from page
  - Use atomic Put() to prevent corruption on crash
  - Log watermark.update events
  - Handle watermark write failures

- [X] T045 [US2] Implement short-circuit in pagination loop in cmd/lastfm-sync/commands/fetch.go
  - After each page, check IsShortCircuit(page, current_watermark)
  - If true, exit pagination loop early
  - Log early termination: "No new scrobbles found"
  - Unit + integration tests

- [X] T046 [US2] Add --since flag validation and watermark interaction in cmd/lastfm-sync/commands/fetch.go
  - Validate --since is valid unix timestamp if provided
  - Implement max(--since, watermark) logic
  - Log which value is used as effective lower bound
  - Test all combinations: no flag, flag < watermark, flag > watermark

- [X] T047 [US2] Implement deduplication in LocalWriter in internal/writer/local.go
  - Track written scrobbles by uniqueness key: username + uts + artist + track
  - Skip duplicates in WriteBatch()
  - Log skipped duplicate count
  - Unit tests for duplicate detection

**Checkpoint**: User Story 2 complete. Incremental sync working with crash-safe watermarks. Full P1 MVP ready.

---

## Phase 3 Status: NOT YET IMPLEMENTED

**Implementation Note**: As of 2025-10-30, Phases 1-2.4 are complete (T001-T047). Phase 3 (Azure integration) has NOT been started. The tasks below (T048+) are planned but pending implementation.

**Completed to Date**:
- ✅ Phase 1: Setup & Project Initialization (T001-T009)
- ✅ Phase 2: Foundational Infrastructure (T010-T021)
- ✅ Phase 3 (tasks.md): User Story 1 - Initial Fetch (T022-T035)
- ✅ Phase 4 (tasks.md): User Story 2 - Incremental Sync (T036-T047)
- ❌ Phase 5 (tasks.md): User Story 3 - Azure Integration (T048-T057) - **NOT STARTED**
- ❌ Phase 6 (tasks.md): User Story 4 - Enhanced Rate Limiting (T054-T057) - **NOT STARTED**
- ❌ Phase 7 (tasks.md): User Story 5 - Dry-run mode (T058+) - **NOT STARTED**

**Current State**: 84 tests passing, 82% coverage. Binary compiles successfully. MVP (P1) complete. Ready to begin Phase 5 (Azure) implementation.

**Reference**: See PHASE2_4_SUMMARY.md for complete implementation details.

---

## Phase 5: User Story 3 - Azure Blob Storage Output (Priority: P2) ⚠️ NOT IMPLEMENTED

**Goal**: User can write scrobbles to Azure Blob Storage with time-partitioned paths and managed identity auth

**Independent Test**: Mock Azure (Azurite) → run fetch with --output azure → verify blob structure and watermark in Azure

**Status**: Configuration validation for Azure exists, but AzureWriter and AzureStore NOT implemented.

### Tests for User Story 3 (TDD: Write First)

- [X] T048 [P] [US3] Unit test: Azure blob path formatting in internal/writer/azure_test.go
  - Test path construction: {prefix}dt=YYYY-MM-DD/{username}-YYYYMMDD-HHMMSS.ndjson
  - Verify date partitioning logic
  - Verify filename uniqueness (timestamp component)
  - Test various prefix values

- [X] T049 [P] [US3] Unit test: AzureStore watermark in internal/watermark/azure_test.go
  - Test watermark blob path: {prefix}{username}.watermark
  - Mock Azure SDK for Get/Put operations
  - Test ETag concurrency scenarios
  - Test blob not found (first run)

- [X] T050 [US3] Integration test: Azure end-to-end with Azurite in cmd/lastfm-sync/commands/fetch_test.go
  - Start Azurite container or use test doubles
  - Mock Last.fm API with 2 pages
  - Run fetch --output azure --azure-container test --azure-prefix lastfm/ --azure-auth default
  - Verify blob created: lastfm/dt=YYYY-MM-DD/alice-YYYYMMDD-HHMMSS.ndjson
  - Verify watermark blob: lastfm/alice.watermark
  - Rerun and verify new blob created (not overwritten)

- [X] T051 [US3] Integration test: Azure auth methods in cmd/lastfm-sync/commands/fetch_test.go
  - Test DefaultAzureCredential with mock
  - Test connection string auth with env var
  - Test SAS token auth with container URL + token
  - Verify correct credential provider used

### Implementation for User Story 3

- [X] T052 [P] [US3] Create internal/config/azure.go with Azure auth resolution
  - Parse --azure-auth flag (default, mi, connstr, sas)
  - Create appropriate credential provider based on auth method
  - Implement GetAzureCredential(config) function
  - Unit tests for credential selection logic

- [X] T053 [US3] Implement AzureWriter in internal/writer/azure.go
  - Implement Writer interface for Azure Blob Storage
  - Use local temp file for buffering records
  - On Flush(), upload temp file to time-partitioned blob path
  - Generate blob name: {prefix}dt=YYYY-MM-DD/{username}-YYYYMMDD-HHMMSS.ndjson
  - Set Content-Type: application/x-ndjson
  - On Close(), delete temp file on success
  - Handle Azure SDK errors with retries
  - Unit + integration tests with Azurite

- [X] T054 [US3] Implement AzureStore in internal/watermark/azure.go
  - Implement WatermarkStore for Azure Blob
  - Blob path: {prefix}{username}.watermark
  - Get() reads watermark blob, handles not found
  - Put() writes watermark blob with ETag check for concurrency
  - Handle Azure SDK errors
  - Unit + integration tests with Azurite

- [X] T055 [US3] Enhance fetch command for Azure output in cmd/lastfm-sync/commands/fetch.go
  - Add Azure flags: --azure-container, --azure-prefix, --azure-auth, --azure-account, --azure-container-url
  - Validate required Azure fields when --output azure
  - Create AzureWriter and AzureStore based on config
  - Wire up Azure auth resolution
  - Add structured logging for Azure operations

- [X] T056 [US3] Implement auto-watermark-store selection in cmd/lastfm-sync/commands/fetch.go
  - If --output azure and --watermark-store not specified, use azure
  - If --output local and --watermark-store not specified, use file
  - Allow explicit override via --watermark-store flag
  - Log selected watermark store
  - Unit tests for selection logic

- [X] T057 [US3] Add Azure secret redaction in internal/logging/logger.go
  - Extend redactSecret() for connection strings and SAS tokens
  - Redact as ****last4 in logs
  - Test connection string patterns
  - Test SAS token patterns
  - Verify no secrets in log output

**Checkpoint**: User Story 3 complete. Azure Blob output with incremental sync. Cloud integration ready.

---

## Phase 6: User Story 4 - Rate Limit Compliance and Backoff (Priority: P2)

**Goal**: Respect Last.fm 3 QPS limit; automatic exponential backoff on 429/5xx; no API bans

**Independent Test**: Throttle to 3 QPS → verify requests take ≥ correct time for N calls; mock 429 → verify exponential backoff applied

### Tests for User Story 4 (TDD: Write First)

- [X] T058 [P] [US4] Unit test: QPS throttling in internal/ratelimit/limiter_test.go
  - Make 10 requests at 3 QPS
  - Verify total time ≈ 10/3 = 3.33 seconds
  - Test burst allowance
  - Test context cancellation

- [X] T059 [P] [US4] Unit test: Exponential backoff in internal/ratelimit/retry_test.go
  - Mock function that fails N times then succeeds
  - Verify backoff delays: 1s, 2s, 4s, 8s
  - Verify max retries honored
  - Test immediate success (no backoff)

- [X] T060 [P] [US4] Unit test: Retry-After header parsing in internal/ratelimit/retry_test.go
  - Test Retry-After with seconds value
  - Test Retry-After with HTTP date
  - Verify backoff uses Retry-After when present
  - Test invalid Retry-After values

- [X] T061 [US4] Integration test: Rate limit handling in cmd/lastfm-sync/commands/fetch_test.go
  - Mock Last.fm API returning 429 for first 2 requests, then 200
  - Run fetch command with --qps 3
  - Verify exponential backoff applied
  - Verify eventual success after retries
  - Check logs for retry events

- [X] T062 [US4] Integration test: 5xx error handling in cmd/lastfm-sync/commands/fetch_test.go
  - Mock API returning 503 (service unavailable) for N requests
  - Verify exponential backoff
  - Verify max retries honored
  - Verify clear error message after max retries exhausted

### Implementation for User Story 4

- [X] T063 [US4] Enhance rate limiter integration in internal/lastfm/client.go
  - Wrap all HTTP requests with limiter.Wait(ctx)
  - Ensure 3 QPS default (configurable via config)
  - Add logging for rate limit waits
  - Unit tests with time measurement

- [X] T064 [US4] Implement Retry-After header support in internal/ratelimit/retry.go
  - Parse Retry-After from HTTP response headers
  - Use Retry-After value for backoff delay if present
  - Fall back to exponential backoff if not present
  - Unit tests for header parsing

- [X] T065 [US4] Add retry logging in internal/lastfm/client.go
  - Log retry attempts with level=warn
  - Include: attempt number, status code, backoff duration
  - Log when max retries exhausted with level=error
  - Verify structured logging format

- [X] T066 [US4] Add --qps and --timeout flags to fetch command in cmd/lastfm-sync/commands/fetch.go
  - Add --qps flag with default 3
  - Add --timeout flag with default 15s
  - Pass to client configuration
  - Validate values are positive
  - Unit tests for flag parsing

- [X] T067 [US4] Implement request timeout enforcement in internal/lastfm/client.go
  - Set HTTP client timeout from config
  - Create per-request context with timeout
  - Handle context deadline exceeded errors
  - Treat timeouts as retryable errors
  - Unit tests with mock slow server

**Checkpoint**: User Story 4 complete. Rate limiting and backoff working. Production-ready resilience.

---

## Phase 7: User Story 5 - Dry Run and Debugging (Priority: P3)

**Goal**: User can test CLI configuration without writing data or consuming API quota

**Independent Test**: Run with --dry-run → verify no API calls, no data written, no watermark updated, preview shown in logs

### Tests for User Story 5 (TDD: Write First)

- [X] T068 [P] [US5] Unit test: Dry-run flag handling in cmd/lastfm-sync/commands/fetch_test.go
  - Set --dry-run flag
  - Verify config.DryRun = true
  - Verify client detects dry-run mode
  - Verify writer detects dry-run mode

- [X] T069 [US5] Integration test: Dry-run end-to-end in cmd/lastfm-sync/commands/fetch_test.go
  - Run fetch with --dry-run
  - Verify no HTTP requests made (or mock call counter = 0)
  - Verify no output file created
  - Verify watermark not updated
  - Verify logs show preview information

- [X] T070 [P] [US5] Unit test: Debug logging in internal/logging/logger_test.go
  - Set --log-level debug
  - Verify debug-level events logged
  - Verify info-level events still logged
  - Test log level filtering

### Implementation for User Story 5

- [X] T071 [US5] Add --dry-run flag to fetch command in cmd/lastfm-sync/commands/fetch.go
  - Add --dry-run boolean flag
  - Pass to config
  - Modify execution flow: skip API calls, writes, watermark updates
  - Log preview information instead

- [X] T072 [US5] Implement dry-run preview in cmd/lastfm-sync/commands/fetch.go
  - Load watermark (if exists)
  - Calculate effective_from
  - Log planned behavior: "Would fetch from uts=X to uts=Y"
  - Log "Would write to {output_path}"
  - Log "Would update watermark to {new_uts}"
  - Return success without side effects

- [X] T073 [US5] Add --log-level flag to root command in cmd/lastfm-sync/main.go
  - Add --log-level flag with options: info, debug
  - Default to info
  - Configure logger based on flag
  - Verify debug logs only show when --log-level debug

- [X] T074 [US5] Enhance debug logging throughout codebase
  - Add fetch.page events at debug level in cmd/lastfm-sync/commands/fetch.go
  - Add fetch.write events at debug level
  - Add watermark.update events at debug level
  - Add rate.limit events at debug level
  - Verify debug logs provide useful troubleshooting information

**Checkpoint**: User Story 5 complete. Dry-run and debugging capabilities ready.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements affecting multiple user stories

- [X] T075 [P] Complete README.md with full documentation
  - Installation instructions (binary, Docker, go install)
  - Configuration examples (local and Azure)
  - Usage examples for all user stories
  - Troubleshooting section
  - FAQ section

- [X] T076 [P] Add comprehensive help text to all commands in cmd/lastfm-sync/
  - Ensure --help shows clear descriptions for all flags
  - Add examples in long help text
  - Document flag defaults
  - Document environment variables

- [X] T077 [P] Verify NDJSON output format compliance
  - Test output with jq: `jq -R 'fromjson' output.ndjson`
  - Verify no parse errors
  - Verify all fields present and correctly typed
  - Test with various NDJSON parsers

- [X] T078 Code cleanup and refactoring
  - Remove any TODO comments
  - Ensure consistent error handling patterns
  - Verify all functions have clear responsibilities
  - Run golangci-lint and fix all issues

- [X] T079 [P] Performance optimization
  - Profile fetch command with pprof
  - Optimize hot paths (pagination loop, NDJSON writing)
  - Verify 200+ scrobbles/sec throughput
  - Verify 10K scrobbles in < 10 minutes

- [X] T080 [P] Security review
  - Audit all credential handling
  - Verify no secrets in logs (run tests and grep for patterns)
  - Verify Docker image runs as non-root
  - Review dependency versions for CVEs

- [X] T081 Integration test coverage improvements
  - Ensure all user stories have integration tests
  - Add edge case tests (empty results, malformed data, network errors)
  - Verify test isolation (each test can run independently)
  - Add integration test documentation

- [X] T082 CI/CD pipeline validation
  - Verify GitHub Actions workflows run on PRs
  - Verify linting passes
  - Verify tests pass with Azurite
  - Verify coverage threshold met (80%+)
  - Verify Docker build succeeds

- [X] T083 [P] Create release documentation
  - Write CHANGELOG.md
  - Document release process
  - Tag v1.0.0
  - Create GitHub release with binaries

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phases 3-7)**: All depend on Foundational phase completion
  - User Story 1 (P1): Can start immediately after Phase 2
  - User Story 2 (P1): Depends on User Story 1 completion (uses watermark and writer from US1)
  - User Story 3 (P2): Can start after Phase 2, independent of US1/US2 (adds Azure support)
  - User Story 4 (P2): Can start after Phase 2, enhances client from US1
  - User Story 5 (P3): Depends on US1 completion (adds dry-run to existing fetch)
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: No dependencies on other stories - implements core fetch to local file
- **User Story 2 (P1)**: Depends on User Story 1 - extends with watermarking and incremental sync
- **User Story 3 (P2)**: Independent of US1/US2 - adds parallel Azure output capability
- **User Story 4 (P2)**: Independent - enhances rate limiting and retry in existing client
- **User Story 5 (P3)**: Depends on User Story 1 - adds dry-run mode to existing fetch

### Recommended Execution Order

**MVP (P1 Stories)**:
1. Phase 1: Setup (all tasks can run in parallel marked [P])
2. Phase 2: Foundational (critical blocking phase, some tasks parallelizable)
3. Phase 3: User Story 1 - Initial Fetch (sequential within story, tests first)
4. Phase 4: User Story 2 - Incremental Sync (sequential, builds on US1)

**Extended Features (P2 Stories)**:
5. Phase 5: User Story 3 - Azure Blob Output (can be done in parallel with US4)
6. Phase 6: User Story 4 - Rate Limiting (can be done in parallel with US3)

**Nice-to-Have (P3 Stories)**:
7. Phase 7: User Story 5 - Dry Run (quick addition after MVP)

**Finalization**:
8. Phase 8: Polish (final improvements and release prep)

### Parallel Opportunities

Within **Phase 1 (Setup)**: All tasks marked [P] can run in parallel (8/9 tasks)

Within **Phase 2 (Foundational)**: These can run in parallel:
- T010 (logging), T011 (config types), T012 (config defaults), T014 (validation)
- T015 (models)
- T016 (client), T017 (pagination)
- T018 (limiter), T019 (retry)
- T021 (version)

Within **Phase 3 (User Story 1)**: Tests T022-T024 can run in parallel, then implementations

Within each user story: All test tasks marked [P] can run in parallel

**Cross-Story Parallelization** (if team capacity allows):
- After Phase 2 completes, User Story 3 and User Story 4 can be developed in parallel
- Both are independent enhancements to the core system

### Critical Path (MVP Only)

```
Phase 1 Setup → Phase 2 Foundational → Phase 3 US1 → Phase 4 US2 → Phase 8 Polish
   (1 day)         (2-3 days)            (2-3 days)    (2 days)       (1-2 days)
                                                                     
Total MVP Timeline: ~8-11 days (sequential, single developer)
```

### Parallel Execution Example (2-3 developers)

```
Dev 1: Phase 1 → Phase 2 (config, logging) → Phase 3 (US1) → Phase 4 (US2) → Phase 8
Dev 2: Phase 1 → Phase 2 (client, ratelimit) → Phase 5 (US3 Azure) → Phase 8
Dev 3: Phase 1 → Phase 2 (models, version) → Phase 6 (US4 rate limit) → Phase 7 (US5) → Phase 8

Total Timeline: ~5-7 days (parallel, 3 developers)
```

---

## Implementation Strategy

### MVP First (Phases 1-4)

The MVP consists of:
- **User Story 1**: Initial fetch to local NDJSON file
- **User Story 2**: Incremental sync with watermarking

This provides immediate value:
- Users can export their full scrobble history
- Users can run incremental syncs without re-fetching everything
- Crash-safe operation with atomic watermarks
- Foundation for all other features

### Incremental Delivery

After MVP, deliver in priority order:
1. **User Story 3 (P2)**: Azure Blob output - enables cloud integration
2. **User Story 4 (P2)**: Enhanced rate limiting - production hardening
3. **User Story 5 (P3)**: Dry-run mode - UX improvement

Each delivery is independently testable and provides incremental value.

### Testing Philosophy

- **TDD**: Write tests first (T022-T025 before T026-T035 in US1)
- **Unit tests**: Fast, isolated, mock external dependencies
- **Integration tests**: End-to-end flows with real Last.fm API mocks and Azurite
- **Test independence**: Each test can run in isolation
- **Coverage goal**: 80%+ overall, 90%+ for critical paths

---

## Summary

- **Total tasks**: 83
- **Setup tasks**: 9
- **Foundational tasks**: 12 (blocking all user stories)
- **User Story 1 (P1)**: 14 tasks (4 tests + 10 implementation)
- **User Story 2 (P1)**: 12 tasks (5 tests + 7 implementation)
- **User Story 3 (P2)**: 10 tasks (4 tests + 6 implementation)
- **User Story 4 (P2)**: 10 tasks (5 tests + 5 implementation)
- **User Story 5 (P3)**: 7 tasks (3 tests + 4 implementation)
- **Polish**: 9 tasks

**Parallel opportunities**: ~35 tasks marked [P] can run in parallel within their phase

**MVP scope**: Phases 1-4 (47 tasks) deliver P1 user stories - complete, independently testable CLI

**Independent test criteria**:
- US1: Mock API → fetch → verify NDJSON file
- US2: Two runs → verify only delta fetched, no duplicates
- US3: Azurite → fetch → verify blob structure
- US4: Mock 429 → verify backoff and recovery
- US5: Dry-run → verify no side effects, preview shown
