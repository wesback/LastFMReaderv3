package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Loader loads configuration from multiple sources with proper precedence:
// 1. Command-line flags (highest priority)
// 2. Environment variables
// 3. Config file (~/.lastfm/config.yaml)
// 4. Defaults (lowest priority)
type Loader struct {
	v *viper.Viper
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	v := viper.New()
	v.SetEnvPrefix("LASTFM")
	v.AutomaticEnv()

	return &Loader{v: v}
}

// LoadConfig loads configuration from all sources and returns a validated Config
func (l *Loader) LoadConfig() (*Config, error) {
	// Start with defaults
	defaults := Defaults()

	// Set default values in Viper
	l.v.SetDefault("qps", defaults.QPS)
	l.v.SetDefault("timeout", defaults.Timeout)
	l.v.SetDefault("page_size", defaults.PageSize)
	l.v.SetDefault("max_pages", defaults.MaxPages)
	l.v.SetDefault("output", defaults.Output)
	l.v.SetDefault("azure_prefix", defaults.AzurePrefix)
	l.v.SetDefault("azure_auth", defaults.AzureAuth)
	l.v.SetDefault("watermark_store", defaults.WatermarkStore)
	l.v.SetDefault("state_path", defaults.StatePath)
	l.v.SetDefault("log_level", defaults.LogLevel)

	// Try to load config file from standard locations
	l.loadConfigFile()

	// Read environment variables
	l.v.BindEnv("api_key", "LASTFM_API_KEY")
	l.v.BindEnv("qps", "LASTFM_QPS")
	l.v.BindEnv("timeout", "LASTFM_TIMEOUT")
	l.v.BindEnv("log_level", "LASTFM_LOG")
	l.v.BindEnv("state_path", "LASTFM_STATE")
	l.v.BindEnv("azure_account_key", "LASTFM_AZURE_ACCOUNT_KEY")
	l.v.BindEnv("azure_sas_token", "LASTFM_AZURE_SAS_TOKEN")
	l.v.BindEnv("azure_connection_string", "AZURE_STORAGE_CONNECTION_STRING")

	// Build config from Viper
	cfg := &Config{
		APIKey:                l.v.GetString("api_key"),
		QPS:                   l.v.GetInt("qps"),
		Timeout:               l.v.GetDuration("timeout"),
		User:                  l.v.GetString("user"),
		Since:                 l.v.GetInt64("since"),
		Until:                 l.v.GetInt64("until"),
		PageSize:              l.v.GetInt("page_size"),
		MaxPages:              l.v.GetInt("max_pages"),
		Output:                l.v.GetString("output"),
		OutPath:               l.v.GetString("out_path"),
		AzureContainer:        l.v.GetString("azure_container"),
		AzurePrefix:           l.v.GetString("azure_prefix"),
		AzureAuth:             l.v.GetString("azure_auth"),
		AzureAccount:          l.v.GetString("azure_account"),
		AzureContainerURL:     l.v.GetString("azure_container_url"),
		AzureAccountKey:       l.v.GetString("azure_account_key"),
		AzureSASToken:         l.v.GetString("azure_sas_token"),
		AzureConnectionString: l.v.GetString("azure_connection_string"),
		WatermarkStore:        l.v.GetString("watermark_store"),
		StatePath:             l.v.GetString("state_path"),
		DryRun:                l.v.GetBool("dry_run"),
		LogLevel:              l.v.GetString("log_level"),
		Progress:              l.loadProgressConfig(),
	}

	// Expand paths
	var err error
	cfg.StatePath, err = ExpandPath(cfg.StatePath)
	if err != nil {
		return nil, fmt.Errorf("expand state_path: %w", err)
	}

	if cfg.OutPath != "" {
		cfg.OutPath, err = ExpandPath(cfg.OutPath)
		if err != nil {
			return nil, fmt.Errorf("expand out_path: %w", err)
		}
	} else {
		// Set default out_path based on state_path and user
		if cfg.User != "" {
			cfg.OutPath = filepath.Join(cfg.StatePath, cfg.User+".ndjson")
		}
	}

	// Auto-select watermark store based on output if not explicitly set
	if cfg.WatermarkStore == "" || cfg.WatermarkStore == "file" {
		if cfg.Output == "azure" && !l.v.IsSet("watermark_store") {
			cfg.WatermarkStore = "azure"
		} else if cfg.Output == "local" && !l.v.IsSet("watermark_store") {
			cfg.WatermarkStore = "file"
		}
	}

	return cfg, nil
}

// BindFlag binds a command-line flag to the Viper config
// This ensures flags take precedence over env vars and config files
func (l *Loader) BindFlag(flagName, configKey string, value interface{}) error {
	switch v := value.(type) {
	case string:
		l.v.Set(configKey, v)
	case int:
		l.v.Set(configKey, v)
	case int64:
		l.v.Set(configKey, v)
	case time.Duration:
		l.v.Set(configKey, v)
	case bool:
		l.v.Set(configKey, v)
	}
	return nil
}

// loadConfigFile attempts to load config from standard locations
func (l *Loader) loadConfigFile() {
	// Search paths: current dir, ~/.lastfm/, /etc/lastfm/
	searchPaths := []string{
		".",
		filepath.Join(os.ExpandEnv("$HOME"), ".lastfm"),
		"/etc/lastfm",
	}

	// Try LASTFM_CONFIG env var
	if configPath := os.Getenv("LASTFM_CONFIG"); configPath != "" {
		searchPaths = append([]string{configPath}, searchPaths...)
	}

	l.v.SetConfigName("config")
	l.v.SetConfigType("yaml")

	for _, path := range searchPaths {
		l.v.AddConfigPath(path)
	}

	// Try to read the config file (don't fail if not found)
	_ = l.v.ReadInConfig()
}

// SetFlag sets a flag value, overriding all other sources
func (l *Loader) SetFlag(key string, value interface{}) {
	l.v.Set(key, value)
}

// loadProgressConfig loads progress bar configuration with environment variable overrides.
func (l *Loader) loadProgressConfig() ProgressConfig {
	cfg := DefaultProgressConfig()

	// Override from config file if present
	if l.v.IsSet("progress.enabled") {
		cfg.Enabled = l.v.GetBool("progress.enabled")
	}
	if l.v.IsSet("progress.style") {
		cfg.Style = l.v.GetString("progress.style")
	}
	if l.v.IsSet("progress.show_speed") {
		cfg.ShowSpeed = l.v.GetBool("progress.show_speed")
	}
	if l.v.IsSet("progress.show_eta") {
		cfg.ShowETA = l.v.GetBool("progress.show_eta")
	}
	if l.v.IsSet("progress.show_count") {
		cfg.ShowCount = l.v.GetBool("progress.show_count")
	}
	if l.v.IsSet("progress.show_percentage") {
		cfg.ShowPercentage = l.v.GetBool("progress.show_percentage")
	}
	if l.v.IsSet("progress.show_elapsed") {
		cfg.ShowElapsed = l.v.GetBool("progress.show_elapsed")
	}
	if l.v.IsSet("progress.width") {
		cfg.Width = l.v.GetInt("progress.width")
	}
	if l.v.IsSet("progress.refresh_rate") {
		cfg.RefreshRate = l.v.GetDuration("progress.refresh_rate")
	}
	if l.v.IsSet("progress.colors") {
		cfg.Colors = l.v.GetBool("progress.colors")
	}
	if l.v.IsSet("progress.auto_clear") {
		cfg.AutoClear = l.v.GetBool("progress.auto_clear")
	}

	// Environment variable overrides (highest priority)
	if os.Getenv("SPECKIT_NO_PROGRESS") != "" {
		val := os.Getenv("SPECKIT_NO_PROGRESS")
		cfg.Enabled = val != "true" && val != "1"
	}

	if os.Getenv("SPECKIT_PROGRESS_ASCII") == "true" || os.Getenv("SPECKIT_PROGRESS_ASCII") == "1" {
		cfg.Style = "ascii"
	}

	if os.Getenv("SPECKIT_NO_COLOR") == "true" || os.Getenv("SPECKIT_NO_COLOR") == "1" || os.Getenv("NO_COLOR") != "" {
		cfg.Colors = false
	}

	if val := os.Getenv("SPECKIT_PROGRESS_REFRESH"); val != "" {
		if ms, err := time.ParseDuration(val + "ms"); err == nil {
			cfg.RefreshRate = ms
		}
	}

	if val := os.Getenv("SPECKIT_PROGRESS_WIDTH"); val != "" {
		var width int
		fmt.Sscanf(val, "%d", &width)
		if width > 0 {
			cfg.Width = width
		}
	}

	return cfg
}
