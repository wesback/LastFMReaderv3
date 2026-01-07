package config

import (
	"testing"
	"time"
)

// TestConfigValidateAPIKey tests that APIKey is required.
func TestConfigValidateAPIKey(t *testing.T) {
	cfg := &Config{
		APIKey: "",
		QPS:    3,
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for missing APIKey")
	}
}

// TestConfigValidateQPS tests that QPS must be positive.
func TestConfigValidateQPS(t *testing.T) {
	cfg := &Config{
		APIKey: "test",
		QPS:    0,
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for QPS=0")
	}
}

// TestConfigValidateTimeout tests that Timeout must be positive.
func TestConfigValidateTimeout(t *testing.T) {
	cfg := &Config{
		APIKey:  "test",
		QPS:     3,
		Timeout: 0,
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for Timeout=0")
	}
}

// TestConfigValidateOutput tests that Output must be valid.
func TestConfigValidateOutput(t *testing.T) {
	cfg := &Config{
		APIKey:   "test",
		QPS:      3,
		Timeout:  15 * time.Second,
		Output:   "invalid",
		LogLevel: "info",
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid Output")
	}
}

// TestConfigValidateAzureContainer tests that AzureContainer is required for azure output.
func TestConfigValidateAzureContainer(t *testing.T) {
	cfg := &Config{
		APIKey:    "test",
		QPS:       3,
		Timeout:   15 * time.Second,
		Output:    "azure",
		LogLevel:  "info",
		AzureAuth: "default",
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for missing AzureContainer")
	}
}

// TestConfigValidateWatermarkStore tests that WatermarkStore must be valid.
func TestConfigValidateWatermarkStore(t *testing.T) {
	cfg := &Config{
		APIKey:         "test",
		QPS:            3,
		Timeout:        15 * time.Second,
		Output:         "local",
		LogLevel:       "info",
		WatermarkStore: "invalid",
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid WatermarkStore")
	}
}

// TestConfigValidateLogLevel tests that LogLevel must be valid.
func TestConfigValidateLogLevel(t *testing.T) {
	cfg := &Config{
		APIKey:   "test",
		QPS:      3,
		Timeout:  15 * time.Second,
		Output:   "local",
		LogLevel: "trace",
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid LogLevel")
	}
}

// TestConfigValidateValid tests a valid configuration.
func TestConfigValidateValid(t *testing.T) {
	cfg := &Config{
		APIKey:         "test",
		QPS:            3,
		Timeout:        15 * time.Second,
		Output:         "local",
		LogLevel:       "info",
		WatermarkStore: "file",
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// TestConfigValidateValidAzure tests a valid Azure configuration.
func TestConfigValidateValidAzure(t *testing.T) {
	cfg := &Config{
		APIKey:         "test",
		QPS:            3,
		Timeout:        15 * time.Second,
		Output:         "azure",
		AzureContainer: "scrobbles",
		LogLevel:       "info",
		WatermarkStore: "azure",
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// TestParseTimeout tests timeout parsing.
// Note: parseTimeout is now in commands/fetch.go, testing via Config instead.
func TestConfigTimeout(t *testing.T) {
	cfg := &Config{
		APIKey:         "test",
		QPS:            3,
		Timeout:        30 * time.Second,
		LogLevel:       "info",
		Output:         "local",
		WatermarkStore: "file",
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected timeout=30s, got %v", cfg.Timeout)
	}
}
