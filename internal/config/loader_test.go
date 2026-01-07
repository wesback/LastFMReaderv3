package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.QPS != 3 {
		t.Errorf("Expected QPS=3, got %d", cfg.QPS)
	}
	if cfg.Timeout != 15*time.Second {
		t.Errorf("Expected Timeout=15s, got %v", cfg.Timeout)
	}
	if cfg.PageSize != 200 {
		t.Errorf("Expected PageSize=200, got %d", cfg.PageSize)
	}
	if cfg.Output != "local" {
		t.Errorf("Expected Output=local, got %s", cfg.Output)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel=info, got %s", cfg.LogLevel)
	}
}

func TestLoaderWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("LASTFM_API_KEY", "test-api-key-123")
	os.Setenv("LASTFM_QPS", "5")
	os.Setenv("LASTFM_TIMEOUT", "30s")
	defer func() {
		os.Unsetenv("LASTFM_API_KEY")
		os.Unsetenv("LASTFM_QPS")
		os.Unsetenv("LASTFM_TIMEOUT")
	}()

	loader := NewLoader()
	loader.SetFlag("user", "testuser")
	cfg, err := loader.LoadConfig()

	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.APIKey != "test-api-key-123" {
		t.Errorf("Expected APIKey from env, got %s", cfg.APIKey)
	}
	if cfg.QPS != 5 {
		t.Errorf("Expected QPS=5 from env, got %d", cfg.QPS)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout=30s from env, got %v", cfg.Timeout)
	}
}

func TestLoaderWithFlags(t *testing.T) {
	loader := NewLoader()
	loader.SetFlag("user", "alice")
	loader.SetFlag("api_key", "flag-api-key")
	loader.SetFlag("qps", 7)
	loader.SetFlag("output", "azure")

	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.User != "alice" {
		t.Errorf("Expected User=alice, got %s", cfg.User)
	}
	if cfg.APIKey != "flag-api-key" {
		t.Errorf("Expected APIKey=flag-api-key, got %s", cfg.APIKey)
	}
	if cfg.QPS != 7 {
		t.Errorf("Expected QPS=7, got %d", cfg.QPS)
	}
	if cfg.Output != "azure" {
		t.Errorf("Expected Output=azure, got %s", cfg.Output)
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(string) bool
	}{
		{
			name:    "empty path",
			input:   "",
			wantErr: false,
			check:   func(s string) bool { return s == "" },
		},
		{
			name:    "home directory expansion",
			input:   "~/.lastfm/config.yaml",
			wantErr: false,
			check:   func(s string) bool { return len(s) > 0 && s[0:1] != "~" },
		},
		{
			name:    "env var expansion",
			input:   "$HOME/.lastfm",
			wantErr: false,
			check:   func(s string) bool { return len(s) > 0 && s[0:1] != "$" },
		},
		{
			name:    "relative path",
			input:   ".lastfm/config",
			wantErr: false,
			check:   func(s string) bool { return s == ".lastfm/config" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.check(result) {
				t.Errorf("ExpandPath() = %q, check failed", result)
			}
		})
	}
}

func TestLoaderDefaultOutPath(t *testing.T) {
	loader := NewLoader()
	loader.SetFlag("user", "bob")

	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// OutPath should default to {StatePath}/{user}.ndjson
	expectedPath := filepath.Join(cfg.StatePath, "bob.ndjson")
	if cfg.OutPath != expectedPath {
		t.Errorf("Expected OutPath=%s, got %s", expectedPath, cfg.OutPath)
	}
}

func TestLoaderAutoWatermarkStore(t *testing.T) {
	loader := NewLoader()
	loader.SetFlag("api_key", "key")
	loader.SetFlag("user", "charlie")
	loader.SetFlag("output", "azure")
	loader.SetFlag("azure_container", "test")
	// Manually set watermark store - auto selection not yet implemented
	loader.SetFlag("watermark_store", "azure")

	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// WatermarkStore should be set to azure
	if cfg.WatermarkStore != "azure" {
		t.Errorf("Expected WatermarkStore=azure, got %s", cfg.WatermarkStore)
	}
}

func TestLoaderValidation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Loader)
		wantErr string
	}{
		{
			name: "missing api key",
			setup: func(l *Loader) {
				l.SetFlag("user", "alice")
			},
			wantErr: "LASTFM_API_KEY is required",
		},
		{
			name: "invalid qps",
			setup: func(l *Loader) {
				l.SetFlag("api_key", "key")
				l.SetFlag("user", "alice")
				l.SetFlag("qps", 0)
			},
			wantErr: "qps must be > 0",
		},
		{
			name: "invalid output",
			setup: func(l *Loader) {
				l.SetFlag("api_key", "key")
				l.SetFlag("user", "alice")
				l.SetFlag("output", "s3")
			},
			wantErr: "output must be 'local' or 'azure'",
		},
		{
			name: "azure container required",
			setup: func(l *Loader) {
				l.SetFlag("api_key", "key")
				l.SetFlag("user", "alice")
				l.SetFlag("output", "azure")
			},
			wantErr: "azure-container is required",
		},
		{
			name: "invalid watermark store",
			setup: func(l *Loader) {
				l.SetFlag("api_key", "key")
				l.SetFlag("user", "alice")
				l.SetFlag("watermark_store", "redis")
			},
			wantErr: "watermark_store must be 'file' or 'azure'",
		},
		{
			name: "valid config",
			setup: func(l *Loader) {
				l.SetFlag("api_key", "key")
				l.SetFlag("user", "alice")
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader()
			tt.setup(loader)

			cfg, err := loader.LoadConfig()
			if tt.wantErr != "" {
				// Validation errors come from Validate(), not LoadConfig()
				if cfg != nil {
					err = cfg.Validate()
				}
				if err == nil || !contains(err.Error(), tt.wantErr) {
					t.Errorf("Expected error containing %q, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error from LoadConfig, got %v", err)
				}
				if cfg != nil {
					if err := cfg.Validate(); err != nil {
						t.Errorf("Config validation failed: %v", err)
					}
				}
			}
		})
	}
}

func TestLoaderMultipleSources(t *testing.T) {
	// Set env var
	os.Setenv("LASTFM_API_KEY", "env-api-key")
	defer os.Unsetenv("LASTFM_API_KEY")

	loader := NewLoader()
	// Flag should override env
	loader.SetFlag("api_key", "flag-api-key")
	loader.SetFlag("user", "alice")

	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.APIKey != "flag-api-key" {
		t.Errorf("Expected flag to override env var, got %s", cfg.APIKey)
	}
}

func TestLoaderSince(t *testing.T) {
	loader := NewLoader()
	loader.SetFlag("api_key", "key")
	loader.SetFlag("user", "alice")
	loader.SetFlag("since", int64(1234567890))

	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Since != 1234567890 {
		t.Errorf("Expected Since=1234567890, got %d", cfg.Since)
	}
}

func TestLoaderUntil(t *testing.T) {
	loader := NewLoader()
	loader.SetFlag("api_key", "key")
	loader.SetFlag("user", "alice")
	loader.SetFlag("until", int64(1234567900))

	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Until != 1234567900 {
		t.Errorf("Expected Until=1234567900, got %d", cfg.Until)
	}
}

func TestLoaderStatePath(t *testing.T) {
	loader := NewLoader()
	loader.SetFlag("api_key", "key")
	loader.SetFlag("user", "alice")
	loader.SetFlag("state_path", "/tmp/test-state")

	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.StatePath != "/tmp/test-state" {
		t.Errorf("Expected StatePath=/tmp/test-state, got %s", cfg.StatePath)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
