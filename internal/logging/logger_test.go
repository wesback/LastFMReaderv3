package logging

import (
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestLoggerCreation(t *testing.T) {
	tests := []struct {
		level string
	}{
		{"debug"},
		{"info"},
		{"warn"},
		{"error"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			logger, err := New(tt.level)
			if err != nil {
				t.Fatalf("New(%s) failed: %v", tt.level, err)
			}
			if logger == nil {
				t.Fatal("logger should not be nil")
			}
		})
	}
}

func TestSecretRedaction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "****",
		},
		{
			name:     "short string",
			input:    "123",
			expected: "****",
		},
		{
			name:     "long string",
			input:    "abcdefghij0123456789",
			expected: "****6789",
		},
		{
			name:     "exactly 4 chars",
			input:    "abcd",
			expected: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactSecret(tt.input)
			if result != tt.expected {
				t.Errorf("redactSecret(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWithRedacted(t *testing.T) {
	logger, err := New("info")
	if err != nil {
		t.Fatalf("New(info) failed: %v", err)
	}

	field := logger.WithRedacted("api_key", "secretkey123456")
	if field.String != "****3456" {
		t.Errorf("WithRedacted field value = %q, want ****3456", field.String)
	}
}

func TestEventLogging(t *testing.T) {
	logger, err := New("debug")
	if err != nil {
		t.Fatalf("New(debug) failed: %v", err)
	}

	// These should not panic
	logger.InfoEvent("test.info", zap.String("key", "value"))
	logger.DebugEvent("test.debug", zap.String("key", "value"))
	logger.WarnEvent("test.warn", zap.String("key", "value"))
	logger.ErrorEvent("test.error", zap.String("key", "value"))
}

func TestRedactSecretsPatterns(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(string) bool // Check that result is properly redacted
	}{
		{
			name:  "api key",
			input: "api_key=abc123def456ghi789",
			check: func(s string) bool {
				return strings.Contains(s, "api_key=****") && !strings.Contains(s, "abc123")
			},
		},
		{
			name:  "connection string",
			input: "AccountKey=mysupersecretekey;",
			check: func(s string) bool {
				return strings.Contains(s, "AccountKey=****") && !strings.Contains(s, "mysupersecretekey")
			},
		},
		{
			name:  "plain text unchanged",
			input: "plain text without secrets",
			check: func(s string) bool {
				return s == "plain text without secrets"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSecrets(tt.input)
			if !tt.check(result) {
				t.Errorf("RedactSecrets(%q) = %q, redaction failed", tt.input, result)
			}
		})
	}
}

// TestDebugLogLevel tests that debug level shows debug and info messages
func TestDebugLogLevel(t *testing.T) {
	logger, err := New("debug")
	if err != nil {
		t.Fatalf("New(debug) failed: %v", err)
	}
	if logger == nil {
		t.Fatal("logger should not be nil")
	}

	// Verify logger is at debug level
	if !logger.Core().Enabled(zap.DebugLevel) {
		t.Error("Expected debug level to be enabled")
	}

	// Verify info level is also enabled
	if !logger.Core().Enabled(zap.InfoLevel) {
		t.Error("Expected info level to be enabled in debug mode")
	}
}

// TestInfoLogLevel tests that info level does not show debug messages
func TestInfoLogLevel(t *testing.T) {
	logger, err := New("info")
	if err != nil {
		t.Fatalf("New(info) failed: %v", err)
	}
	if logger == nil {
		t.Fatal("logger should not be nil")
	}

	// Verify debug level is NOT enabled
	if logger.Core().Enabled(zap.DebugLevel) {
		t.Error("Expected debug level to be disabled in info mode")
	}

	// Verify info level is enabled
	if !logger.Core().Enabled(zap.InfoLevel) {
		t.Error("Expected info level to be enabled")
	}
}
