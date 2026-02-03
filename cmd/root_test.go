package cmd

import (
	"testing"

	"github.com/spf13/viper"
)

func TestFlagsContain(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		contains []string
		expected bool
	}{
		{
			name:     "contains one flag",
			flags:    []string{"--verbose", "--host=example.com"},
			contains: []string{"--verbose"},
			expected: true,
		},
		{
			name:     "contains multiple flags",
			flags:    []string{"--verbose", "--host=example.com"},
			contains: []string{"--verbose", "--host"},
			expected: true,
		},
		{
			name:     "does not contain flag",
			flags:    []string{"--verbose", "--host=example.com"},
			contains: []string{"--missing"},
			expected: false,
		},
		{
			name:     "empty flags",
			flags:    []string{},
			contains: []string{"--verbose"},
			expected: false,
		},
		{
			name:     "empty contains",
			flags:    []string{"--verbose"},
			contains: []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flagsContain(tt.flags, tt.contains...)
			if result != tt.expected {
				t.Errorf("flagsContain(%v, %v) = %v, want %v",
					tt.flags, tt.contains, result, tt.expected)
			}
		})
	}
}

func TestViperDefaults(t *testing.T) {
	// Viper defaults are set in init(), which runs automatically
	// We can verify they exist after package initialization
	tests := []struct {
		key      string
		expected interface{}
	}{
		{"pipeline", "master"},
		{"products.rs.search_name", "ingredi"},
		{"products.rs.display_name", "RS"},
		{"products.pra.search_name", "bpam"},
		{"products.pra.display_name", "PRA"},
		{"deployment.domain", "dev.bomgar.com"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value := viper.Get(tt.key)
			if value != tt.expected {
				t.Errorf("viper.Get(%s) = %v, want %v", tt.key, value, tt.expected)
			}
		})
	}
}

func TestVerboseLogging(t *testing.T) {
	// Save original verbose level
	oldVerbose := Verbose
	defer func() { Verbose = oldVerbose }()

	t.Run("verbose level 0", func(t *testing.T) {
		Verbose = 0
		// verbose() should not output anything, but we can't easily test that
		// Just verify it doesn't panic
		verbose("test message")
	})

	t.Run("verbose level 1", func(t *testing.T) {
		Verbose = 1
		// verbose() should output
		verbose("test message")
	})

	t.Run("vVerbose only at level 2", func(t *testing.T) {
		Verbose = 1
		// vVerbose() should not output at level 1
		vVerbose("test message")

		Verbose = 2
		// vVerbose() should output at level 2
		vVerbose("test message")
	})
}
