package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestParseRetryAfterSeconds tests parsing Retry-After header with seconds value
func TestParseRetryAfterSeconds(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "valid seconds",
			header:  "120",
			want:    120 * time.Second,
			wantErr: false,
		},
		{
			name:    "zero seconds",
			header:  "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "large value",
			header:  "3600",
			want:    3600 * time.Second,
			wantErr: false,
		},
		{
			name:    "invalid format",
			header:  "abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "negative value",
			header:  "-10",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty header",
			header:  "",
			want:    0,
			wantErr: false, // Empty header is treated like missing header
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{
					"Retry-After": []string{tt.header},
				},
			}

			got, err := ParseRetryAfter(resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRetryAfter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseRetryAfterHTTPDate tests parsing Retry-After header with HTTP date
func TestParseRetryAfterHTTPDate(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(60 * time.Second)
	past := now.Add(-10 * time.Second)

	tests := []struct {
		name    string
		date    time.Time
		wantMin time.Duration
		wantMax time.Duration
		wantErr bool
	}{
		{
			name:    "future date",
			date:    future,
			wantMin: 55 * time.Second, // Allow 5s variance
			wantMax: 65 * time.Second,
			wantErr: false,
		},
		{
			name:    "past date",
			date:    past,
			wantMin: 0,
			wantMax: 1 * time.Second,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{
					"Retry-After": []string{tt.date.Format(http.TimeFormat)},
				},
			}

			got, err := ParseRetryAfter(resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRetryAfter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("ParseRetryAfter() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestParseRetryAfterNoHeader tests behavior when Retry-After header is missing
func TestParseRetryAfterNoHeader(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}

	got, err := ParseRetryAfter(resp)
	if err != nil {
		t.Errorf("ParseRetryAfter() with no header should not error, got %v", err)
	}
	if got != 0 {
		t.Errorf("ParseRetryAfter() with no header = %v, want 0", got)
	}
}

// TestParseRetryAfterInvalidDate tests parsing invalid HTTP date
func TestParseRetryAfterInvalidDate(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"Retry-After": []string{"Not a date or number"},
		},
	}

	_, err := ParseRetryAfter(resp)
	if err == nil {
		t.Error("ParseRetryAfter() with invalid date should return error")
	}
}

// TestBackoffWithRetryAfter tests that backoff respects Retry-After when present
func TestBackoffWithRetryAfter(t *testing.T) {
	limiter := NewLimiter(100, 3) // High QPS so rate limiting doesn't interfere

	// Create a response with Retry-After header
	resp := &http.Response{
		Header: http.Header{
			"Retry-After": []string{"2"}, // 2 seconds
		},
	}

	attempt := 0
	startTime := time.Now()

	err := limiter.DoWithRetry(context.Background(), func() error {
		attempt++
		if attempt == 1 {
			// First attempt fails with Retry-After
			return &HTTPError{
				Err:      fmt.Errorf("rate limited (429)"),
				Response: resp,
			}
		}
		// Second attempt succeeds
		return nil
	})

	elapsed := time.Since(startTime)

	if err != nil {
		t.Fatalf("DoWithRetry() unexpected error = %v", err)
	}

	if attempt != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempt)
	}

	// Should wait approximately 2 seconds (allow 500ms tolerance)
	if elapsed < 2*time.Second || elapsed > 2500*time.Millisecond {
		t.Errorf("Expected ~2s wait for Retry-After, got %v", elapsed)
	}
}
