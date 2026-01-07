# LastFMReaderv3 Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-06

## Active Technologies
- Go 1.24.0+ + `github.com/schollz/progressbar/v3`, `golang.org/x/term` (005-console-progress-bar)
- N/A (UI component only) (005-console-progress-bar)

- Go 1.24.0+ (alpine-based Docker build) (002-containerization-documentation)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.24.0+ (alpine-based Docker build)

## Code Style

Go 1.24.0+ (alpine-based Docker build): Follow standard conventions

## Recent Changes
- 005-console-progress-bar: Added Go 1.24.0+ + `github.com/schollz/progressbar/v3`, `golang.org/x/term`
- 004-normalized-title-field: Adding `normalized_title` field to remove annotations (Live, Remastered, featuring, etc.) from track titles for better matching and grouping. Uses internal/normalize package with gopkg.in/yaml.v3 for configuration. DEBUG logging when titles modified.
- 002-containerization-documentation: Added [if applicable, e.g., PostgreSQL, CoreData, files or N/A]

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
