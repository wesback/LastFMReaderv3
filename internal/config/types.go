package config

import (
	"fmt"
	"time"
)

// Config holds all CLI flags and configuration for lastfm-sync
type Config struct {
	// Last.fm API
	APIKey  string        `mapstructure:"api_key"`
	QPS     int           `mapstructure:"qps"`
	Timeout time.Duration `mapstructure:"timeout"`

	// Fetch options
	User     string `mapstructure:"user"`
	Since    int64  `mapstructure:"since"`
	Until    int64  `mapstructure:"until"`
	PageSize int    `mapstructure:"page_size"`
	MaxPages int    `mapstructure:"max_pages"`

	// Output destination
	Output  string `mapstructure:"output"`
	OutPath string `mapstructure:"out_path"`

	// Azure Blob Storage
	AzureContainer        string `mapstructure:"azure_container"`
	AzurePrefix           string `mapstructure:"azure_prefix"`
	AzureAuth             string `mapstructure:"azure_auth"`
	AzureAccount          string `mapstructure:"azure_account"`
	AzureAccountURL       string `mapstructure:"azure_account_url"`
	AzureContainerURL     string `mapstructure:"azure_container_url"`
	AzureConnectionString string `mapstructure:"azure_connection_string"`
	AzureAccountKey       string `mapstructure:"azure_account_key"`
	AzureSASToken         string `mapstructure:"azure_sas_token"`

	// Watermark storage
	WatermarkStore string `mapstructure:"watermark_store"`
	StatePath      string `mapstructure:"state_path"`

	// Flags
	DryRun   bool   `mapstructure:"dry_run"`
	LogLevel string `mapstructure:"log_level"`

	// Progress bars
	Progress ProgressConfig `mapstructure:"progress"`
}

// ProgressConfig holds configuration for progress bar display.
type ProgressConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	Style          string        `mapstructure:"style"`
	ShowSpeed      bool          `mapstructure:"show_speed"`
	ShowETA        bool          `mapstructure:"show_eta"`
	ShowCount      bool          `mapstructure:"show_count"`
	ShowPercentage bool          `mapstructure:"show_percentage"`
	ShowElapsed    bool          `mapstructure:"show_elapsed"`
	Width          int           `mapstructure:"width"`
	RefreshRate    time.Duration `mapstructure:"refresh_rate"`
	Colors         bool          `mapstructure:"colors"`
	AutoClear      bool          `mapstructure:"auto_clear"`
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("LASTFM_API_KEY is required")
	}

	if c.QPS <= 0 {
		return fmt.Errorf("qps must be > 0, got %d", c.QPS)
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0, got %v", c.Timeout)
	}

	if c.Output != "local" && c.Output != "azure" {
		return fmt.Errorf("output must be 'local' or 'azure', got %q", c.Output)
	}

	if c.Output == "azure" && c.AzureContainer == "" {
		return fmt.Errorf("--azure-container is required when --output is azure")
	}

	if c.WatermarkStore != "file" && c.WatermarkStore != "azure" {
		return fmt.Errorf("watermark_store must be 'file' or 'azure', got %q", c.WatermarkStore)
	}

	if c.LogLevel != "info" && c.LogLevel != "debug" {
		return fmt.Errorf("log_level must be 'info' or 'debug', got %q", c.LogLevel)
	}

	return nil
}

// LastfmAPIResponse represents the shape of Last.fm API responses
type LastfmAPIResponse struct {
	RecentTracks struct {
		Track []LastfmTrack `json:"track"`
		Attr  struct {
			Page       string `json:"page"`
			PerPage    string `json:"perPage"`
			Total      string `json:"total"`
			TotalPages string `json:"totalPages"`
		} `json:"@attr"`
	} `json:"recenttracks"`
	Error int `json:"error,omitempty"`

	Message string `json:"message,omitempty"`
}

// LastfmTrack represents a single track in Last.fm API response
type LastfmTrack struct {
	Artist struct {
		MBID string `json:"mbid"`
		Text string `json:"#text"`
	} `json:"artist"`
	Album struct {
		MBID string `json:"mbid"`
		Text string `json:"#text"`
	} `json:"album"`
	Track struct {
		MBID string `json:"mbid"`
		Text string `json:"#text"`
	} `json:"name"`
	Date struct {
		UTS  string `json:"uts"`
		Text string `json:"#text"`
	} `json:"date"`
	NowPlaying string `json:"nowplaying,omitempty"` // Present if currently playing
	MBID       string `json:"mbid"`
	URL        string `json:"url"`
	Image      []struct {
		Size string `json:"size"`
		Text string `json:"#text"`
	} `json:"image"`
}
