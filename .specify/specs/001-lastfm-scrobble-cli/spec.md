# Feature Specification: Last.fm Scrobble CLI with Incremental Sync

**Feature Branch**: `001-lastfm-scrobble-cli`  
**Created**: 2025-10-30  
**Status**: Draft  
**Input**: User description: "Last.fm scrobble CLI with incremental sync, JSON output, and optional Azure Blob upload"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Initial Full Scrobble History Fetch (Priority: P1)

Music listener Alice wants to export her complete Last.fm scrobble history to a local JSON file for data portability and archival purposes. She runs the CLI once pointing to her username and receives a complete dump of all available scrobbles in NDJSON format, stored locally.

**Why this priority**: This is the core MVP feature. Without the ability to fetch and store scrobbles, all other features are blocked. P1 enables basic data portability and establishes the data pipeline.

**Independent Test**: Can be fully tested by: (1) mocking Last.fm API to return paginated scrobbles, (2) running `lastfm-sync fetch --user alice --output local --out-path /tmp/alice.ndjson`, (3) verifying the output file contains correctly formatted NDJSON records with username, artist, track, album, uts, mbid, source, and ingested_at fields, and (4) confirming no records are duplicated.

**Acceptance Scenarios**:

1. **Given** a Last.fm user "alice" with 5,000 scrobbles, **When** running `lastfm-sync fetch --user alice --output local`, **Then** a file at `~/.lastfm/alice.ndjson` is created containing all 5,000 records in NDJSON format with no duplicates.

2. **Given** the CLI is run with `--page-size 200`, **When** fetching alice's scrobbles, **Then** the tool makes multiple paginated API calls (5,000 / 200 = 25 pages) and accumulates all results.

3. **Given** the output file does not exist, **When** running the fetch, **Then** a new file is created at the specified path with proper JSON formatting.

4. **Given** scrobbles lack an `mbid` field in the Last.fm API response, **When** output is written, **Then** the `mbid` field is either omitted or set to `null` consistently.

---

### User Story 2 - Incremental Sync with Watermarking (Priority: P1)

Alice runs the sync CLI daily. On the second run, the tool should skip already-fetched scrobbles and only sync new ones since the last successful run. The tool persists a watermark (max timestamp) locally to track progress and avoid re-processing.

**Why this priority**: Incremental sync is critical for production use. Without it, every run re-fetches all history (wasteful, violates rate limits). This is the second pillar of the MVP.

**Independent Test**: Can be fully tested by: (1) first run fetches scrobbles, watermark stored at `~/.lastfm/alice.watermark`, (2) mock API to return only new scrobbles, (3) second run with same user and `--since` not specified uses stored watermark, (4) verify only new scrobbles are appended to output file, (5) watermark updated to new max uts.

**Acceptance Scenarios**:

1. **Given** alice's first sync captured scrobbles up to uts=1000, **When** running sync again and API returns new scrobbles uts=1001..1005, **Then** watermark is read from storage, lower bound set to max(--since, watermark), only new scrobbles fetched and appended, watermark updated to 1005.

2. **Given** `--since 500` is provided and stored watermark is 1000, **When** fetching, **Then** effective lower bound is max(500, 1000) = 1000; earlier scrobbles not re-fetched.

3. **Given** a page returns zero records with uts > current watermark, **When** processing that page, **Then** pagination stops immediately (short-circuit); no further pages requested.

4. **Given** watermark stored locally, **When** sync completes after page N, **Then** watermark file updated atomically (avoid data loss on crash mid-run).

---

### User Story 3 - Azure Blob Storage Output (Priority: P2)

Alice has a data pipeline in Azure. She wants scrobbles written to Azure Blob Storage instead of a local file, with time-based partitioning (YYYY-MM-DD) for easy data governance and archival. She provides Azure credentials and specifies container name and prefix.

**Why this priority**: Cloud integration is valuable for production deployments but not required for MVP. Enables integration with data lakes and cloud-based workflows. P2 can be implemented after P1 is stable.

**Independent Test**: Can be fully tested by: (1) mock Azure SDK or use test container, (2) run `lastfm-sync fetch --user alice --output azure --azure-container lastfm --azure-prefix users/ --azure-auth default`, (3) verify blobs are written to `lastfm/users/dt=2025-10-30/alice-20251030-HHMMSS.ndjson`, (4) verify watermark also stored in Azure, (5) verify secrets not logged.

**Acceptance Scenarios**:

1. **Given** `--output azure --azure-container lastfm --azure-prefix users/`, **When** fetching on 2025-10-30, **Then** scrobbles written to blob path `lastfm/users/dt=2025-10-30/alice-20251030-HHMMSS.ndjson` (time-partitioned).

2. **Given** `--azure-auth default`, **When** connecting to Azure, **Then** use default credential chain (managed identity, environment, etc.); no credentials logged.

3. **Given** multiple runs on the same day, **When** scrobbles fetched, **Then** each run creates a new blob with unique timestamp suffix; earlier blobs not overwritten.

4. **Given** `--watermark-store azure`, **When** sync completes, **Then** watermark persisted to blob (e.g., `lastfm/users/alice.watermark`) for multi-machine deployments.

---

### User Story 4 - Rate Limit Compliance and Backoff (Priority: P2)

The Last.fm API enforces rate limits (~3 requests per second). Alice's CLI should respect these limits, automatically back off on 429 (Too Many Requests) responses, and avoid blocking the user with slow retries.

**Why this priority**: Rate limit compliance is essential for long-running syncs and multi-user scenarios but can be implemented after basic sync works. Prevents API bans and service disruptions. P2.

**Independent Test**: Can be fully tested by: (1) mock API returning 429 for N requests, then 200, (2) run fetch with `--qps 3 --timeout 15s`, (3) verify exponential backoff applied (retry after delay), (4) verify max-retries honored, (5) verify total run time accounts for backoff delays.

**Acceptance Scenarios**:

1. **Given** API returns 429 (rate limited), **When** fetching, **Then** tool backs off exponentially (e.g., 1s, 2s, 4s, 8s) and retries.

2. **Given** `--qps 3`, **When** making requests, **Then** rate is throttled to max 3 requests per second; requests delayed as needed.

3. **Given** API returns 5xx error, **When** fetching, **Then** tool backs off and retries with same exponential strategy; retry limit respected.

4. **Given** `--timeout 15s`, **When** any single request exceeds 15 seconds, **Then** request cancelled and treated as failure; backoff applied.

---

### User Story 5 - Dry Run and Debugging (Priority: P3)

Alice wants to test her CLI configuration without writing data or consuming API quota. `--dry-run` flag allows her to preview what would be fetched and validate configuration before committing changes.

**Why this priority**: Dry-run is valuable for verification and debugging but not critical for MVP. Reduces risk of accidental data issues. P3.

**Independent Test**: Can be fully tested by: (1) run with `--dry-run`, (2) verify no API calls made (or mocked calls count only), (3) verify no data written to file or blob, (4) verify logging shows what would be done (e.g., "Would fetch X pages, Y records"), (5) watermark not updated.

**Acceptance Scenarios**:

1. **Given** `--dry-run` flag provided, **When** running fetch, **Then** no HTTP requests made to Last.fm API; no files written.

2. **Given** `--dry-run` with `--log-level debug`, **When** running, **Then** output shows detailed plan (e.g., "Would fetch from uts=1000 to uts=2000; page size 200").

3. **Given** watermark already stored, **When** running with `--dry-run`, **Then** watermark read and shown but not updated.

---

### Edge Cases

- What happens when Last.fm API returns an empty page (no scrobbles found)?
  - **Expected**: Pagination stops; watermark updated; exit cleanly with message "No new scrobbles."

- What happens when ScrobbleData is null?
  - **Expected**: This is the currently playing song, ignore this record and the end of the stream is reached.

- What happens if output file is not writable (permission denied)?
  - **Expected**: Clear error message; exit with non-zero code; no partial file left.

- What happens if Azure credentials are invalid or container doesn't exist?
  - **Expected**: Error logged; no retry loop on auth failures; exit with guidance on fixing credentials.

- What happens if a scrobble record is malformed (missing required field)?
  - **Expected**: Log warning with record details; skip malformed record; continue processing.

- What happens if a network error occurs mid-page?
  - **Expected**: Log error; backoff and retry up to max retries; if max exceeded, checkpoint at previous successful page.

- What happens if `--until` is before `--since`?
  - **Expected**: Validation error; message "until < since"; exit with non-zero code.

- What happens if user runs two instances simultaneously (same user, output file)?
  - **Expected**: File lock or atomic append enforced; later write waits or fails gracefully with clear error.

- What happens on API deprecation or schema change?
  - **Expected**: Non-goals for v1, but code structured to log warnings on unexpected fields.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST fetch scrobbles from Last.fm API using the `user.getRecentTracks` endpoint with pagination.

- **FR-002**: System MUST support `--since` and `--until` unix timestamp filters to limit fetch date range.

- **FR-003**: System MUST output scrobbles in NDJSON format (one JSON object per line) with fields: `username`, `artist`, `track`, `album`, `uts`, `mbid`, `source`, `ingested_at`, `raw`.

- **FR-004**: System MUST store and update a watermark (max uts) after each successfully flushed page to enable incremental re-runs.

- **FR-005**: System MUST use effective lower bound = max(`--since`, stored_watermark) to determine fetch start point.

- **FR-006**: System MUST support `--output local` (default) and `--output azure` destinations.

- **FR-007**: System MUST append new NDJSON records to existing local file (idempotent write).

- **FR-008**: System MUST write Azure blobs to time-partitioned paths: `{prefix}dt=YYYY-MM-DD/{username}-YYYYMMDD-HHMMSS.ndjson`.

- **FR-009**: System MUST persist watermark locally (file) or in Azure (blob) depending on `--watermark-store` option.

- **FR-010**: System MUST respect Last.fm API rate limits: enforce `--qps` (queries per second) and back off on 429/5xx responses.

- **FR-011**: System MUST skip already-processed scrobbles (idempotent) based on username + uts + artist + track uniqueness.

- **FR-012**: System MUST support authentication methods: `--api-key` flag or `LASTFM_API_KEY` environment variable.

- **FR-013**: System MUST support Azure authentication: `--azure-auth default|mi|connstr|sas` with appropriate credential providers.

- **FR-014**: System MUST provide `--dry-run` flag to preview behavior without writing data or consuming quota.

- **FR-015**: System MUST provide `--log-level debug|info` for debugging and observability.

- **FR-016**: System MUST enforce `--timeout` per request; cancel and retry on timeout.

- **FR-017**: System MUST provide clear error messages on validation failures, API errors, and file I/O errors.

### Key Entities

- **Scrobble Record**: Represents a single track listen event.
  - Attributes: `username` (string), `artist` (string), `track` (string), `album` (string), `uts` (unix timestamp), `mbid` (optional string), `source` (fixed: "lastfm"), `ingested_at` (ISO 8601 timestamp), `raw` (object with original Last.fm response).
  - Uniqueness: Combination of `username`, `uts`, `artist`, `track`.

- **Watermark**: Tracks the maximum processed unix timestamp per user for incremental syncs.
  - Attributes: `username` (string), `max_uts` (unix timestamp), `updated_at` (ISO 8601 timestamp), `sync_id` (unique ID for the run that produced this watermark).
  - Storage: Local file `~/.lastfm/{username}.watermark` or Azure blob.

- **CLI Command**: `lastfm-sync fetch` with options as specified in goals.
  - Responsible for: parsing arguments, validating options, orchestrating fetch → transform → write pipeline.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: User can fetch 10,000 scrobbles and write to local file in < 10 minutes (includes API latency and I/O). *Measurement: Manual benchmark test or deferred to post-MVP observability phase.*

- **SC-002**: Incremental re-run on same user fetches only new scrobbles in < 30 seconds (watermark lookup + delta fetch). *Measurement: Manual benchmark test or deferred to post-MVP observability phase.*

- **SC-003**: Rate limit compliance: CLI respects `--qps 3` and does not trigger Last.fm API abuse alerts; zero unhandled 429s.

- **SC-004**: Zero data loss on crash: watermark updated atomically; if process killed mid-page, rerun resumes from last-committed watermark.

- **SC-005**: Azure blob writes succeed with 99% success rate; transient errors retried; secrets never logged.

- **SC-006**: NDJSON output is valid and parseable: `jq -R 'fromjson' output.ndjson` returns all records without errors.

- **SC-007**: Dry-run mode: 100% accuracy; no actual API calls or writes; previewed actions match what would happen in real run.

- **SC-008**: Error messages are actionable: user can diagnose and fix issue. Actionable means: (1) what failed (component/operation), (2) why it failed (root cause), (3) specific remediation step (e.g., "Azure container 'foo' not found in subscription 'bar'; verify container exists or create with: az storage container create -n foo").

- **SC-009**: Code coverage: ≥ 80% for new code (unit + integration tests per constitution); critical paths (watermark, dedup, retry) at 90%+.

- **SC-010**: UX consistency: all error messages follow same format; loading/success/error states clear via `--log-level debug`; help text complete (`--help`).

---

## Data Contracts

### Scrobble Record (NDJSON Format)

Each line in the output file is a JSON object:

```json
{
  "username": "alice",
  "artist": "Radiohead",
  "track": "Idioteque",
  "album": "Kid A",
  "uts": 971136000,
  "mbid": "12345678-1234-1234-1234-123456789012",
  "source": "lastfm",
  "ingested_at": "2025-10-30T18:12:00Z",
  "raw": {
    "artist": { "#text": "Radiohead", "mbid": "12345678-1234-1234-1234-123456789012" },
    "track": { "#text": "Idioteque", "mbid": "..." },
    "album": { "#text": "Kid A", "mbid": "..." },
    "date": { "uts": "971136000", "#text": "Oct 30, 2025" },
    "loved": "0",
    "image": [...],
    "url": "..."
  }
}
```

**Notes on fields**:
- `username`: Last.fm username (from `--user` argument).
- `artist`, `track`, `album`: Extracted from Last.fm response; whitespace normalized.
- `uts`: Unix timestamp (epoch seconds); used for sorting and watermarking.
- `mbid`: MusicBrainz ID if available; omitted or `null` if not present.
- `source`: Fixed string "lastfm" (future-proofs for multi-source data).
- `ingested_at`: ISO 8601 timestamp of when record was processed (not Last.fm's date).
- `raw`: Original unmodified Last.fm API response object.

### Watermark Record

```json
{
  "username": "alice",
  "max_uts": 1719792720,
  "updated_at": "2025-10-30T18:12:00Z",
  "sync_id": "sync-20251030-181200-abc123"
}
```

**Notes**:
- `username`: User this watermark belongs to.
- `max_uts`: Highest `uts` value successfully written in this run.
- `updated_at`: ISO 8601 timestamp of watermark update (for auditing).
- `sync_id`: Unique identifier for the sync run (for tracing failures).

### Watermarking Rules

1. **Effective lower bound**: `max(--since, stored_watermark.max_uts)`.
2. **Watermark update**: After each successfully flushed page, update watermark atomically.
3. **Short-circuit**: If a page returns zero records with uts > current watermark, stop pagination.
4. **Crash safety**: Watermark file/blob written last; if write fails, do not mark records as processed.

## CLI Options Reference

```bash
lastfm-sync fetch --user <username> \
  [--since <unix_ts>] \
  [--until <unix_ts>] \
  [--page-size 200] \
  [--max-pages 0] \
  [--dry-run] \
  [--output local|azure] \
  [--out-path ~/.lastfm/{user}.ndjson] \
  [--azure-container <name>] \
  [--azure-prefix lastfm/{user}/] \
  [--azure-auth default|mi|connstr|sas] \
  [--watermark-store file|azure] \
  [--log-level info|debug] \
  [--qps 3] \
  [--timeout 15s] \
  [--api-key ... | env LASTFM_API_KEY]
```

**Option Defaults**:
- `--output`: `local`
- `--out-path`: `~/.lastfm/{user}.ndjson`
- `--page-size`: `200` (Last.fm max)
- `--max-pages`: `0` (unlimited)
- `--qps`: `3` (Last.fm recommended)
- `--timeout`: `15s`
- `--log-level`: `info`
- `--watermark-store`: `file` (or `azure` if `--output azure`)
- `--azure-auth`: `default`

## Non-Goals (v1)

- Schema evolution/versioning beyond the scrobble structure defined above.
- Deduplication across replays beyond username + uts + artist + track.
- Support for scrobble corrections/deletions (reverse sync).
- Data transformation, aggregation, or analytics.
- GUI or web UI; CLI-only for v1.
- Integration with other music platforms (Spotify, Apple Music, etc.) in v1.

