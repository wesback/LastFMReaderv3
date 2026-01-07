package merge

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/lastfm-reader/lastfm-sync/internal/models"
)

// ReadError represents an error that occurred while reading NDJSON
type ReadError struct {
	Line    int
	Message string
	Err     error
	Sample  string // First 200 chars of the invalid line for debugging
}

func (e ReadError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("line %d: %s: %v", e.Line, e.Message, e.Err)
	}
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

// ReadNDJSON reads NDJSON format and returns scrobbles and any errors encountered
// Invalid lines are skipped with errors recorded, allowing processing to continue
func ReadNDJSON(reader io.Reader) ([]*models.Scrobble, []ReadError) {
	var scrobbles []*models.Scrobble
	var errors []ReadError

	scanner := bufio.NewScanner(reader)

	// Configure buffer for large lines (up to 1MB)
	buf := make([]byte, 0, 128*1024) // 128KB initial
	scanner.Buffer(buf, 1024*1024)   // 1MB max

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		// Skip empty lines silently
		if len(line) == 0 || len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		var scrobble models.Scrobble
		if err := json.Unmarshal(line, &scrobble); err != nil {
			// Capture sample of invalid line (first 200 chars)
			sample := string(line)
			if len(sample) > 200 {
				sample = sample[:200] + "..."
			}
			errors = append(errors, ReadError{
				Line:    lineNum,
				Message: "invalid JSON",
				Err:     err,
				Sample:  sample,
			})
			continue
		}

		// Validate required fields
		if err := validateScrobble(&scrobble); err != nil {
			// Capture sample for validation failures too
			sample := string(line)
			if len(sample) > 200 {
				sample = sample[:200] + "..."
			}
			errors = append(errors, ReadError{
				Line:    lineNum,
				Message: "validation failed",
				Err:     err,
				Sample:  sample,
			})
			continue
		}

		scrobbles = append(scrobbles, &scrobble)
	}

	if err := scanner.Err(); err != nil {
		errors = append(errors, ReadError{
			Line:    lineNum,
			Message: "scanner error",
			Err:     err,
		})
	}

	return scrobbles, errors
}

// validateScrobble checks if a scrobble has required fields
func validateScrobble(s *models.Scrobble) error {
	if s.Artist == "" {
		return fmt.Errorf("missing required field: artist")
	}
	if s.Track == "" {
		return fmt.Errorf("missing required field: track")
	}
	if s.UTS <= 0 {
		return fmt.Errorf("invalid timestamp: must be positive (got %d)", s.UTS)
	}
	return nil
}
