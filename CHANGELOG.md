# Changelog

All notable changes to lastfm-sync will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added - Feature 005: Console Progress Bar

- **Progress Bar Implementation** (`internal/progress/`)
  - Real-time progress tracking with visual feedback using `github.com/schollz/progressbar/v3`
  - ETA calculation based on actual throughput
  - Speed metrics (scrobbles per second)
  - Current/total scrobble counts with percentage
  - Automatic terminal width detection
  - Thread-safe progress updates for concurrent operations
  - Configurable via `--no-progress` flag or `LASTFM_NO_PROGRESS` environment variable
  - Graceful degradation for non-interactive terminals (CI/CD pipelines)

- **Progress Reporting**
  - Factory pattern for creating progress bars vs no-op implementations
  - `ProgressReporter` interface for flexible progress tracking
  - Progress state tracking (completed, in-progress, pending)
  - Terminal capability detection using `golang.org/x/term`
  - Comprehensive test coverage including terminal emulation tests

### Added - Feature 004: Normalized Title Field

- **Title Normalization** (`internal/normalize/`)
  - Automatic removal of track title annotations for better data quality
  - Pattern-based removal of:
    - Remaster/Remastered annotations (e.g., "2009 Remaster", "Remastered 2015")
    - Live performance markers (e.g., "Live at Wembley", "Live 1969")
    - Version/edit labels (e.g., "Radio Edit", "Extended Version")
    - Date/year markers in parentheses or brackets
    - Remix labels (e.g., "Dave's Remix")
    - Featuring/collaboration markers (e.g., "feat. Artist", "with Orchestra")
  - Thread-safe global feature flag for enabling/disabling normalization
  - Preserves original title in `track` field while adding clean `normalized_title` field
  - Configurable minimum length protection to avoid over-aggressive cleaning
  - DEBUG-level logging for title changes (when logger provided)
  - YAML-based configuration for customizable patterns (`internal/normalize/patterns.go`)

- **Scrobble Model Enhancement** (`internal/models/scrobble.go`)
  - Added `normalized_title` field to all NDJSON output
  - Automatic normalization in `NewScrobble` constructor
  - Original title preserved for audit trail and debugging

### Added - Feature 002: Containerization & Documentation

- **Configuration Documentation** (`docs/configuration.md`)
  - Complete environment variable and CLI flag reference with descriptions and defaults
  - Configuration precedence explanation (flags > env > file > defaults)
  - Validation rules and error messages
  - Example configurations for local, Docker, and Azure deployments

- **Environment Template** (`.env.example`)
  - Template with all configuration variables and inline comments
  - Security warnings for sensitive values
  - Docker Compose and Azure-specific sections

- **Docker Enhancement** (`docs/docker.md`)
  - Multi-stage build architecture documentation
  - Container security features (non-root user, distroless base)
  - Docker Compose workflow guide
  - Troubleshooting section

- **Development Scripts**
  - `scripts/dev-up.sh` - Automated environment startup with validation
  - `scripts/dev-down.sh` - Environment cleanup with optional volume/image removal

- **Azure Deployment** (`docs/azure-deployment.md`)
  - Step-by-step Azure Container Instances deployment guide
  - Azure Key Vault integration with managed identity
  - Log Analytics monitoring with structured JSON logging format
  - Observability: custom metrics, container exit codes, KQL queries
  - `azure/deploy-aci.sh` - Automated deployment script
  - `azure/aci-params.json.example` - Deployment parameters template

- **Security Documentation** (`docs/security.md`)
  - Secret management best practices
  - Azure Key Vault setup and integration
  - Managed identity and RBAC configuration
  - Container security (distroless, non-root, read-only filesystem)
  - Network security (private endpoints, NSG rules)
  - Secret rotation procedures
  - Audit and compliance guidelines
  - Security checklists

- **Troubleshooting Guide** (`docs/troubleshooting.md`)
  - Configuration issues (missing API key, invalid QPS, timeout parsing)
  - Docker issues (permissions, build failures, distroless container behavior)
  - Docker Compose issues (.env, permissions, V1/V2 compatibility)
  - Azure deployment issues (authentication, Key Vault, quotas, network)
  - API and network issues (rate limiting, timeouts, certificate errors)
  - Debugging commands and validation checklists
  - FAQ section

### Changed
- Enhanced Dockerfile with comprehensive inline comments
- Updated README.md with links to all new documentation
- Verified .gitignore excludes .env and .env.* files

## [1.0.0] - 2026-01-06

### Added

- **Core Features**
  - Last.fm API client with pagination support
  - Local NDJSON output with atomic writes
  - Azure Blob Storage output with time-partitioned paths (dt=YYYY-MM-DD/)
  - Incremental sync using watermarks (tracks last successfully synced timestamp)
  - Automatic watermark persistence (local files or Azure blobs)
  - Dry-run mode for testing configurations without side effects

- **Rate Limiting & Reliability**
  - 3 QPS rate limiting (configurable) compliant with Last.fm API limits
  - Exponential backoff retry logic for transient errors (429, 5xx)
  - Retry-After header support for intelligent backoff delays
  - HTTP timeout handling with configurable timeouts (default: 15s)
  - Maximum retry limit (5 attempts) with detailed error messages

- **Azure Integration**
  - Multiple authentication methods:
    - DefaultAzureCredential (recommended for Azure VMs/AKS)
    - Managed Identity (workload identity)
    - Connection string
    - Account key (SharedKeyCredential)
    - SAS token
  - Time-partitioned blob paths for efficient data organization
  - Separate watermark blob storage for state management
  - Auto-detection of watermark storage based on output mode

- **Logging & Observability**
  - Structured logging using zap
  - Automatic secret redaction for API keys, connection strings, SAS tokens
  - Debug mode with detailed operation logs
  - Log levels: info (default) and debug
  - Comprehensive event logging:
    - sync.start, fetch.page.start, fetch.page.success
    - fetch.write.start, watermark.updated
    - dry_run.preview, dry_run.output, dry_run.watermark

- **CLI Features**
  - Comprehensive command-line interface built with Cobra
  - Environment variable support for all major configurations
  - Flexible flag-based configuration
  - Detailed help text with examples
  - Support for unix timestamp-based time ranges (--since, --until)
  - Page size configuration (1-200 records per page)
  - Max pages limit for testing and quota management

- **Testing & Quality**
  - 105 passing tests across all packages
  - Unit tests for all core functionality
  - Integration tests for API client, writers, watermark stores
  - Mock implementations for testing
  - Coverage reports available via go test

- **Development**
  - Multi-stage Docker build with distroless runtime
  - Makefile for common build tasks
  - Go modules for dependency management
  - CI/CD pipeline configuration (GitHub Actions)
  - Comprehensive documentation (README, specification, architecture plan)

### Implementation Details

- **Phases Completed:**
  - Phase 1: Project Setup & Foundation
  - Phase 2: Foundational Components (models, logging, config)
  - Phase 3: User Story 1 - Initial Fetch (local output)
  - Phase 4: User Story 2 - Incremental Sync
  - Phase 5: User Story 3 - Azure Blob Storage
  - Phase 6: User Story 4 - Rate Limit Compliance
  - Phase 7: User Story 5 - Dry Run & Debugging
  - Phase 8: Polish & Documentation

- **Architecture:**
  - Clean architecture with clear separation of concerns
  - Internal packages: config, lastfm, logging, models, ratelimit, service, watermark, writer
  - Interface-based design for testability and extensibility
  - Dependency injection for mocking and testing

- **Code Quality:**
  - No TODO comments left in production code
  - All code formatted with gofmt
  - Passes go vet static analysis
  - Consistent error handling patterns
  - Clear function responsibilities

### Security

- Secret redaction in logs (API keys, connection strings, SAS tokens)
- Docker container runs as non-root user (nonroot:nonroot)
- No secrets stored in environment by default
- Credential handling via Azure SDK best practices
- HTTPS-only communication with Last.fm API

### Performance

- Efficient pagination handling (200 records per page default)
- Rate limiting prevents API bans
- Streaming NDJSON write for memory efficiency
- Atomic flush operations for crash safety
- Expected throughput: ~600 scrobbles/minute at 3 QPS

### Dependencies

- Go 1.24+
- github.com/spf13/cobra v1.8.1
- github.com/spf13/viper v1.19.0
- go.uber.org/zap v1.27.0
- github.com/cenkalti/backoff/v4 v4.3.0
- golang.org/x/time/rate v0.8.0
- github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.3
- github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1

## Release Process

1. **Version Bump**: Update version in internal/version/version.go
2. **Changelog**: Update this file with release notes
3. **Git Tag**: Create annotated tag `git tag -a v1.0.0 -m "Release v1.0.0"`
4. **Push Tag**: `git push origin v1.0.0`
5. **Build Binaries**: `make build` for linux/darwin/windows
6. **GitHub Release**: Create release with binaries and changelog
7. **Docker Image**: `docker build -t lastfm-sync:v1.0.0 .`

## Support

- **Issues**: https://github.com/lastfm-reader/lastfm-sync/issues
- **Documentation**: See README.md
- **Specification**: .specify/specs/001-lastfm-scrobble-cli/

---

[Unreleased]: https://github.com/lastfm-reader/lastfm-sync/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/lastfm-reader/lastfm-sync/releases/tag/v1.0.0
