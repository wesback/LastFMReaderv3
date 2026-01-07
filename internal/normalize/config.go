package normalize

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds normalization configuration.
type Config struct {
	Enabled          bool            `yaml:"enabled"`           // Master enable/disable
	MinLength        int             `yaml:"min_length"`        // Minimum normalized title length
	International    bool            `yaml:"international"`     // Enable international patterns
	CustomPatterns   []PatternConfig `yaml:"custom_patterns"`   // User-defined patterns
	DisabledPatterns []string        `yaml:"disabled_patterns"` // Patterns to skip
}

// PatternConfig represents a custom user-defined pattern.
type PatternConfig struct {
	Name        string `yaml:"name"`        // Pattern name
	Pattern     string `yaml:"pattern"`     // Regex pattern string
	Priority    int    `yaml:"priority"`    // Priority order
	Description string `yaml:"description"` // Human-readable description
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Enabled:          true,
		MinLength:        2,
		International:    true,
		CustomPatterns:   []PatternConfig{},
		DisabledPatterns: []string{},
	}
}

// LoadConfig loads configuration from a YAML file.
// Falls back to defaults if file doesn't exist.
// Environment variables override file settings.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Load from YAML file if provided
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				// File doesn't exist, use defaults
			} else {
				return nil, fmt.Errorf("read config file: %w", err)
			}
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config YAML: %w", err)
			}
		}
	}

	// Environment variables override file settings
	if envEnabled := os.Getenv("NORMALIZE_ENABLED"); envEnabled != "" {
		enabled, err := strconv.ParseBool(envEnabled)
		if err != nil {
			return nil, fmt.Errorf("invalid NORMALIZE_ENABLED value: %w", err)
		}
		cfg.Enabled = enabled
	}

	if envMinLen := os.Getenv("NORMALIZE_MIN_LENGTH"); envMinLen != "" {
		minLen, err := strconv.Atoi(envMinLen)
		if err != nil {
			return nil, fmt.Errorf("invalid NORMALIZE_MIN_LENGTH value: %w", err)
		}
		cfg.MinLength = minLen
	}

	if envIntl := os.Getenv("NORMALIZE_INTERNATIONAL"); envIntl != "" {
		intl, err := strconv.ParseBool(envIntl)
		if err != nil {
			return nil, fmt.Errorf("invalid NORMALIZE_INTERNATIONAL value: %w", err)
		}
		cfg.International = intl
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	// Validate minimum length
	if c.MinLength < 0 {
		return fmt.Errorf("min_length must be non-negative, got %d", c.MinLength)
	}
	if c.MinLength > 100 {
		return fmt.Errorf("min_length must be <= 100, got %d", c.MinLength)
	}

	// Validate custom patterns
	for i, p := range c.CustomPatterns {
		if p.Name == "" {
			return fmt.Errorf("custom pattern %d: name is required", i)
		}
		if p.Pattern == "" {
			return fmt.Errorf("custom pattern %q: pattern is required", p.Name)
		}
		// Validate regex compiles
		if _, err := regexp.Compile(p.Pattern); err != nil {
			return fmt.Errorf("custom pattern %q: invalid regex: %w", p.Name, err)
		}
		if p.Priority < 0 {
			return fmt.Errorf("custom pattern %q: priority must be non-negative", p.Name)
		}
	}

	// Validate disabled patterns
	validPatterns := map[string]bool{
		"remaster": true, "live": true, "version": true,
		"date": true, "remix": true, "featuring": true,
	}
	for _, name := range c.DisabledPatterns {
		if !validPatterns[name] {
			return fmt.Errorf("invalid pattern name in disabled_patterns: %q", name)
		}
	}

	return nil
}

// IsPatternEnabled checks if a pattern is enabled.
func (c *Config) IsPatternEnabled(name string) bool {
	for _, disabled := range c.DisabledPatterns {
		if disabled == name {
			return false
		}
	}
	return true
}
