package cmd

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStylesInitialized(t *testing.T) {
	// Verify that color constants are defined
	colors := []lipgloss.Color{lightGray, orange, darkNavy, cyan, gray, white, red, green}
	for i, color := range colors {
		if string(color) == "" {
			t.Errorf("color at index %d is empty", i)
		}
	}

	// Verify adaptive color
	if textColor.Light == "" || textColor.Dark == "" {
		t.Error("textColor adaptive color not properly initialized")
	}
}

func TestStylesNotNil(t *testing.T) {
	// Verify that style variables are not nil/empty
	styles := []struct {
		name  string
		style lipgloss.Style
	}{
		{"noStyle", noStyle},
		{"textStyle", textStyle},
		{"orangeStyle", orangeStyle},
		{"grayStyle", grayStyle},
		{"verboseStyle", verboseStyle},
		{"errStyle", errStyle},
		{"infoBoxStyle", infoBoxStyle},
		{"infoBoldStyle", infoBoldStyle},
		{"successStyle", successStyle},
		{"failureStyle", failureStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			// Styles should be initialized (we can't directly check if they're "nil"
			// but we can render something and verify it doesn't panic)
			result := s.style.Render("test")
			if result == "" {
				t.Errorf("%s.Render() returned empty string", s.name)
			}
		})
	}
}

func TestVPrefixInitialized(t *testing.T) {
	if vPrefix == "" {
		t.Error("vPrefix should be initialized with process ID")
	}
	// vPrefix should contain formatting for the process ID
	if len(vPrefix) < 5 {
		t.Errorf("vPrefix seems too short: %s", vPrefix)
	}
}
