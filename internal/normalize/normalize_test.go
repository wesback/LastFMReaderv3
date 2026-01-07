package normalize

import (
	"os"
	"testing"
)

// TestNormalizeTitle tests standard normalization cases.
func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Remaster patterns
		{"remaster simple", "Bohemian Rhapsody - Remastered 2011", "Bohemian Rhapsody"},
		{"remaster parens", "Stairway to Heaven (Remaster)", "Stairway to Heaven"},
		{"remaster year", "Hotel California - 2013 Remaster", "Hotel California"},
		{"reissue", "Yesterday - Reissue", "Yesterday"},
		{"remasterise french", "La Vie en Rose - Remasterisé", "La Vie en Rose"},

		// Live patterns
		{"live simple", "Hotel California - Live", "Hotel California"},
		{"live at venue", "Comfortably Numb - Live at Pompeii", "Comfortably Numb"},
		{"live year", "Dream On - Live 2023", "Dream On"},
		{"en vivo spanish", "Bésame Mucho - En Vivo", "Bésame Mucho"},
		{"ao vivo portuguese", "Garota de Ipanema - Ao Vivo", "Garota de Ipanema"},
		{"en direct french", "Non, Je Ne Regrette Rien - En Direct", "Non, Je Ne Regrette Rien"},

		// Version patterns
		{"album version", "Imagine - Album Version", "Imagine"},
		{"radio edit", "Smells Like Teen Spirit - Radio Edit", "Smells Like Teen Spirit"},
		{"extended mix", "Blue Monday - Extended Mix", "Blue Monday"},
		{"explicit version", "Lose Yourself - Explicit Version", "Lose Yourself"},

		// Date patterns
		{"year simple", "Wonderwall - 2011", "Wonderwall"},
		{"year parens", "Hallelujah (2008)", "Hallelujah"},
		{"year remaster", "Let It Be - 2009 Remaster", "Let It Be"},

		// Remix patterns
		{"remix artist", "Vogue - Shep Pettibone Remix", "Vogue"},
		{"club mix", "Rhythm Is a Dancer - Club Mix", "Rhythm Is a Dancer"},
		{"acoustic", "Layla - Acoustic", "Layla"},
		{"unplugged", "About a Girl - Unplugged", "About a Girl"},
		{"instrumental", "Europa - Instrumental", "Europa"},

		// Featuring patterns
		{"feat dot", "Love the Way You Lie (feat. Rihanna)", "Love the Way You Lie"},
		{"ft abbreviation", "Empire State of Mind - ft. Alicia Keys", "Empire State of Mind"},
		{"featuring full", "Walk This Way - Featuring Run-D.M.C.", "Walk This Way"},
		{"with", "Under Pressure - With David Bowie", "Under Pressure"},

		// No changes needed
		{"already clean", "Bohemian Rhapsody", "Bohemian Rhapsody"},
		{"no annotations", "Stairway to Heaven", "Stairway to Heaven"},
		{"cut in title", "God's Gonna Cut You Down", "God's Gonna Cut You Down"},
		{"break in title", "Break On Through", "Break On Through"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeTitle(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNormalizeTitle_EdgeCases tests edge cases and boundary conditions.
func TestNormalizeTitle_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Empty and whitespace
		{"empty string", "", ""},
		{"only spaces", "   ", "   "},
		{"whitespace around", "  Title  ", "Title"},

		// Too short after normalization (preserved)
		{"band named Live", "Live", "Live"},
		{"single char", "X", "X"},
		{"two chars", "OK", "OK"},
		{"short live", "Go - Live", "Go"}, // "Go" is still >= minLength (2)

		// Multiple annotations
		{"multiple remove", "Song - Live (2011 Remaster)", "Song"},
		{"all patterns", "Track - Live at Venue (2023 Remaster) [feat. Artist]", "Track"},
		{"complex", "Title - Remastered 2011 - Live - Radio Edit", "Title"},

		// Unicode and international
		{"emoji", "Happy 😊 - Remastered", "Happy 😊"},
		{"cyrillic", "Калинка - Ремастер", "Калинка - Ремастер"}, // Not matched (pattern is Latin-based)
		{"accents", "Ñoño - En Vivo", "Ñoño"},
		{"chinese", "春节 - Live", "春节"},

		// Long titles
		{"very long", "Some Very Long Title About Something Interesting - Remastered 2023", "Some Very Long Title About Something Interesting"},
		{"500 chars", string(make([]byte, 500)) + " - Live", string(make([]byte, 500))},

		// Nested delimiters
		{"nested parens", "Song (Part 1) - (Remaster)", "Song (Part 1)"},
		{"mixed delimiters", "Track [Extended] - (Live)", "Track [Extended]"},

		// Only annotations (should preserve original)
		{"only remaster", "Remastered 2011", "Remastered 2011"},
		{"only live", "Live at Venue", "Live at Venue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeTitle(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNormalizeTitle_International tests international pattern variants.
func TestNormalizeTitle_International(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Spanish
		{"spanish live", "Canción - En Vivo", "Canción"},
		{"spanish remaster", "Título - Remasterizada", "Título"},

		// French
		{"french live", "Chanson - En Direct", "Chanson"},
		{"french remaster", "Titre - Remasterisé", "Titre"},

		// Portuguese
		{"portuguese live", "Música - Ao Vivo", "Música"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeTitle(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFeatureFlag tests enable/disable functionality.
func TestFeatureFlag(t *testing.T) {
	// Save original state
	originalState := IsEnabled()
	defer SetEnabled(originalState)

	input := "Song - Remastered 2011"
	expected := "Song"

	// Test enabled (default)
	SetEnabled(true)
	if !IsEnabled() {
		t.Error("Expected normalization to be enabled")
	}
	result := NormalizeTitle(input)
	if result != expected {
		t.Errorf("With enabled=true, got %q, want %q", result, expected)
	}

	// Test disabled
	SetEnabled(false)
	if IsEnabled() {
		t.Error("Expected normalization to be disabled")
	}
	result = NormalizeTitle(input)
	if result != input {
		t.Errorf("With enabled=false, got %q, want %q (original)", result, input)
	}

	// Re-enable
	SetEnabled(true)
	if !IsEnabled() {
		t.Error("Expected normalization to be re-enabled")
	}
	result = NormalizeTitle(input)
	if result != expected {
		t.Errorf("After re-enabling, got %q, want %q", result, expected)
	}
}

// TestConfig_Load tests configuration loading.
func TestConfig_Load(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cfg := DefaultConfig()
		if !cfg.Enabled {
			t.Error("Expected default config to be enabled")
		}
		if cfg.MinLength != 2 {
			t.Errorf("Expected default MinLength=2, got %d", cfg.MinLength)
		}
		if !cfg.International {
			t.Error("Expected default International=true")
		}
	})

	t.Run("env var override", func(t *testing.T) {
		// Set env vars
		os.Setenv("NORMALIZE_ENABLED", "false")
		os.Setenv("NORMALIZE_MIN_LENGTH", "5")
		os.Setenv("NORMALIZE_INTERNATIONAL", "false")
		defer func() {
			os.Unsetenv("NORMALIZE_ENABLED")
			os.Unsetenv("NORMALIZE_MIN_LENGTH")
			os.Unsetenv("NORMALIZE_INTERNATIONAL")
		}()

		cfg, err := LoadConfig("")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}
		if cfg.Enabled {
			t.Error("Expected Enabled=false from env var")
		}
		if cfg.MinLength != 5 {
			t.Errorf("Expected MinLength=5 from env var, got %d", cfg.MinLength)
		}
		if cfg.International {
			t.Error("Expected International=false from env var")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		cfg, err := LoadConfig("/nonexistent/path.yaml")
		if err != nil {
			t.Fatalf("Expected no error for nonexistent file, got: %v", err)
		}
		// Should return defaults
		if !cfg.Enabled {
			t.Error("Expected defaults when file doesn't exist")
		}
	})
}

// TestConfig_Validate tests configuration validation.
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name:      "valid default",
			config:    *DefaultConfig(),
			wantError: false,
		},
		{
			name: "negative min length",
			config: Config{
				Enabled:   true,
				MinLength: -1,
			},
			wantError: true,
		},
		{
			name: "excessive min length",
			config: Config{
				Enabled:   true,
				MinLength: 200,
			},
			wantError: true,
		},
		{
			name: "invalid custom pattern regex",
			config: Config{
				Enabled:   true,
				MinLength: 2,
				CustomPatterns: []PatternConfig{
					{Name: "test", Pattern: "[invalid(", Priority: 10},
				},
			},
			wantError: true,
		},
		{
			name: "invalid disabled pattern",
			config: Config{
				Enabled:          true,
				MinLength:        2,
				DisabledPatterns: []string{"nonexistent"},
			},
			wantError: true,
		},
		{
			name: "valid custom pattern",
			config: Config{
				Enabled:   true,
				MinLength: 2,
				CustomPatterns: []PatternConfig{
					{Name: "test", Pattern: `\s*-\s*Test$`, Priority: 10, Description: "Test pattern"},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestNormalizeTitleWithConfig tests custom configuration.
func TestNormalizeTitleWithConfig(t *testing.T) {
	t.Run("disabled pattern", func(t *testing.T) {
		cfg := &Config{
			Enabled:          true,
			MinLength:        2,
			DisabledPatterns: []string{"remaster"},
		}
		input := "Song - Remastered 2011"
		result := NormalizeTitleWithConfig(input, cfg)
		// Remaster pattern disabled, but date pattern (2011) will still be applied
		expected := "Song - Remastered"
		if result != expected {
			t.Errorf("Expected %q (remaster pattern disabled), got %q", expected, result)
		}
	})

	t.Run("custom min length", func(t *testing.T) {
		cfg := &Config{
			Enabled:   true,
			MinLength: 10,
		}
		input := "Short - Live"
		result := NormalizeTitleWithConfig(input, cfg)
		// "Short" is < 10 chars, should preserve original
		if result != input {
			t.Errorf("Expected %q (too short), got %q", input, result)
		}
	})

	t.Run("config disabled", func(t *testing.T) {
		cfg := &Config{
			Enabled:   false,
			MinLength: 2,
		}
		input := "Song - Remastered 2011"
		result := NormalizeTitleWithConfig(input, cfg)
		if result != input {
			t.Errorf("Expected %q (config disabled), got %q", input, result)
		}
	})
}

// Benchmark tests (Task 1.7)

// BenchmarkNormalizeTitle_NoAnnotations tests fast path (no changes needed).
func BenchmarkNormalizeTitle_NoAnnotations(b *testing.B) {
	input := "Bohemian Rhapsody"
	for i := 0; i < b.N; i++ {
		_ = NormalizeTitle(input)
	}
}

// BenchmarkNormalizeTitle_SingleAnnotation tests single pattern match.
func BenchmarkNormalizeTitle_SingleAnnotation(b *testing.B) {
	input := "Bohemian Rhapsody - Remastered 2011"
	for i := 0; i < b.N; i++ {
		_ = NormalizeTitle(input)
	}
}

// BenchmarkNormalizeTitle_MultipleAnnotations tests multiple pattern matches.
func BenchmarkNormalizeTitle_MultipleAnnotations(b *testing.B) {
	input := "Song - Live at Venue (2023 Remaster) [feat. Artist]"
	for i := 0; i < b.N; i++ {
		_ = NormalizeTitle(input)
	}
}

// BenchmarkNormalizeTitle_LongTitle tests performance with long titles.
func BenchmarkNormalizeTitle_LongTitle(b *testing.B) {
	input := "This Is a Very Long Title With Many Words and Possibly Some Annotations That Need To Be Removed - Remastered 2023 - Live at Some Venue"
	for i := 0; i < b.N; i++ {
		_ = NormalizeTitle(input)
	}
}

// BenchmarkNormalizeTitle_Disabled tests performance when feature is disabled.
func BenchmarkNormalizeTitle_Disabled(b *testing.B) {
	SetEnabled(false)
	defer SetEnabled(true)
	input := "Song - Remastered 2011 - Live - feat. Artist"
	for i := 0; i < b.N; i++ {
		_ = NormalizeTitle(input)
	}
}
