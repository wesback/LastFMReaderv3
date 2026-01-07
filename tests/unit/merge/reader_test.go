package merge_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/lastfm-reader/lastfm-sync/internal/merge"
)

func TestReadNDJSON_ValidInput(t *testing.T) {
	input := `{"username":"user1","artist":"Artist1","track":"Track1","album":"Album1","uts":1000,"local_time":"2000-01-01","source":"lastfm","ingested_at":"2026-01-07"}
{"username":"user1","artist":"Artist2","track":"Track2","album":"Album2","uts":2000,"local_time":"2000-01-02","source":"lastfm","ingested_at":"2026-01-07"}
{"username":"user1","artist":"Artist3","track":"Track3","album":"","uts":3000,"local_time":"2000-01-03","source":"lastfm","ingested_at":"2026-01-07"}
`

	reader := bytes.NewBufferString(input)
	scrobbles, errors := merge.ReadNDJSON(reader)

	if len(scrobbles) != 3 {
		t.Errorf("got %d scrobbles, want 3", len(scrobbles))
	}

	if len(errors) != 0 {
		t.Errorf("got %d errors, want 0: %v", len(errors), errors)
	}

	// Verify first scrobble
	if scrobbles[0].Artist != "Artist1" {
		t.Errorf("first scrobble artist = %s, want Artist1", scrobbles[0].Artist)
	}
	if scrobbles[0].Track != "Track1" {
		t.Errorf("first scrobble track = %s, want Track1", scrobbles[0].Track)
	}
	if scrobbles[0].UTS != 1000 {
		t.Errorf("first scrobble UTS = %d, want 1000", scrobbles[0].UTS)
	}
}

func TestReadNDJSON_InvalidJSON(t *testing.T) {
	input := `{"username":"user1","artist":"Artist1","track":"Track1","uts":1000}
{invalid json line
{"username":"user1","artist":"Artist2","track":"Track2","uts":2000}
`

	reader := bytes.NewBufferString(input)
	scrobbles, errors := merge.ReadNDJSON(reader)

	if len(scrobbles) != 2 {
		t.Errorf("got %d scrobbles, want 2 (invalid line skipped)", len(scrobbles))
	}

	if len(errors) != 1 {
		t.Errorf("got %d errors, want 1", len(errors))
	}
}

func TestReadNDJSON_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantValid  int
		wantErrors int
	}{
		{
			name:       "missing artist",
			input:      `{"username":"user1","track":"Track1","uts":1000}` + "\n",
			wantValid:  0,
			wantErrors: 1,
		},
		{
			name:       "missing track",
			input:      `{"username":"user1","artist":"Artist1","uts":1000}` + "\n",
			wantValid:  0,
			wantErrors: 1,
		},
		{
			name:       "missing uts",
			input:      `{"username":"user1","artist":"Artist1","track":"Track1"}` + "\n",
			wantValid:  0,
			wantErrors: 1,
		},
		{
			name:       "zero uts (invalid)",
			input:      `{"username":"user1","artist":"Artist1","track":"Track1","uts":0}` + "\n",
			wantValid:  0,
			wantErrors: 1,
		},
		{
			name:       "negative uts (invalid)",
			input:      `{"username":"user1","artist":"Artist1","track":"Track1","uts":-1000}` + "\n",
			wantValid:  0,
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			scrobbles, errors := merge.ReadNDJSON(reader)

			if len(scrobbles) != tt.wantValid {
				t.Errorf("got %d valid scrobbles, want %d", len(scrobbles), tt.wantValid)
			}

			if len(errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d", len(errors), tt.wantErrors)
			}
		})
	}
}

func TestReadNDJSON_EmptyLines(t *testing.T) {
	input := `{"username":"user1","artist":"Artist1","track":"Track1","uts":1000}

{"username":"user1","artist":"Artist2","track":"Track2","uts":2000}

`

	reader := bytes.NewBufferString(input)
	scrobbles, errors := merge.ReadNDJSON(reader)

	if len(scrobbles) != 2 {
		t.Errorf("got %d scrobbles, want 2 (empty lines ignored)", len(scrobbles))
	}

	if len(errors) != 0 {
		t.Errorf("got %d errors, want 0 (empty lines should be silently skipped)", len(errors))
	}
}

func TestReadNDJSON_WithMBID(t *testing.T) {
	input := `{"username":"user1","artist":"Artist1","track":"Track1","uts":1000,"mbid":"12345"}
{"username":"user1","artist":"Artist2","track":"Track2","uts":2000,"mbid":null}
{"username":"user1","artist":"Artist3","track":"Track3","uts":3000}
`

	reader := bytes.NewBufferString(input)
	scrobbles, errors := merge.ReadNDJSON(reader)

	if len(scrobbles) != 3 {
		t.Errorf("got %d scrobbles, want 3", len(scrobbles))
	}

	if len(errors) != 0 {
		t.Errorf("got %d errors, want 0: %v", len(errors), errors)
	}

	// First has MBID
	if scrobbles[0].MBID == nil || *scrobbles[0].MBID != "12345" {
		t.Error("first scrobble should have MBID = 12345")
	}

	// Second has null MBID
	if scrobbles[1].MBID != nil {
		t.Error("second scrobble MBID should be nil")
	}

	// Third has no MBID field
	if scrobbles[2].MBID != nil {
		t.Error("third scrobble MBID should be nil")
	}
}

func TestReadNDJSON_LargeFile(t *testing.T) {
	// Generate 1000 scrobbles
	var sb strings.Builder
	for i := 1; i <= 1000; i++ {
		sb.WriteString(fmt.Sprintf(`{"username":"user1","artist":"Artist","track":"Track","uts":%d}`, i+1000))
		sb.WriteString("\n")
	}

	reader := strings.NewReader(sb.String())
	scrobbles, errors := merge.ReadNDJSON(reader)

	if len(scrobbles) != 1000 {
		t.Errorf("got %d scrobbles, want 1000", len(scrobbles))
	}

	if len(errors) != 0 {
		t.Errorf("got %d errors, want 0", len(errors))
	}
}

func TestReadNDJSON_EmptyInput(t *testing.T) {
	reader := strings.NewReader("")
	scrobbles, errors := merge.ReadNDJSON(reader)

	if len(scrobbles) != 0 {
		t.Errorf("got %d scrobbles, want 0", len(scrobbles))
	}

	if len(errors) != 0 {
		t.Errorf("got %d errors, want 0", len(errors))
	}
}
