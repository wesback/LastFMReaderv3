package config

import (
	"os"
	"path/filepath"
	"time"
)

// Defaults returns a Config with sensible default values
func Defaults() *Config {
	return &Config{
		QPS:            3,
		Timeout:        15 * time.Second,
		PageSize:       200,
		MaxPages:       0,
		Output:         "local",
		AzurePrefix:    "lastfm/",
		AzureAuth:      "default",
		WatermarkStore: "file",
		StatePath:      stateDir(),
		LogLevel:       "info",
		OutPath:        filepath.Join(stateDir(), "scrobbles.ndjson"),
		Progress:       DefaultProgressConfig(),
	}
}

// DefaultProgressConfig returns default progress bar configuration.
func DefaultProgressConfig() ProgressConfig {
	return ProgressConfig{
		Enabled:        true,
		Style:          "blocks",
		ShowSpeed:      true,
		ShowETA:        true,
		ShowCount:      true,
		ShowPercentage: true,
		ShowElapsed:    false,
		Width:          0, // Auto-detect
		RefreshRate:    100 * time.Millisecond,
		Colors:         true,
		AutoClear:      true,
	}
}

// stateDir returns the default state directory (~/.lastfm)
func stateDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".lastfm"
	}
	return filepath.Join(homeDir, ".lastfm")
}

// ExpandPath expands ~ and environment variables in a path
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Replace ~ with home directory
	if path[0:1] == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	return path, nil
}
