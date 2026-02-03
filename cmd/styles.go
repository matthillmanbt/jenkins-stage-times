package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Color definitions
var (
	lightGray = lipgloss.Color("#7C878E")
	orange    = lipgloss.Color("#FF5500")
	darkNavy  = lipgloss.Color("#081826")
	cyan      = lipgloss.Color("#4EC3E0")
	gray      = lipgloss.Color("#222222")
	white     = lipgloss.Color("#FFFFFF")
	red       = lipgloss.Color("#FF0000")
	green     = lipgloss.Color("#00FF00")

	textColor = lipgloss.AdaptiveColor{Light: string(darkNavy), Dark: string(white)}
)

// Renderers for different output streams
var (
	verboseRenderer = lipgloss.NewRenderer(os.Stderr)
	stdRe           = lipgloss.NewRenderer(os.Stdout)
	errRe           = lipgloss.NewRenderer(os.Stderr)
)

// Base styles
var (
	noStyle      = stdRe.NewStyle().Foreground(lipgloss.NoColor{})
	textStyle    = stdRe.NewStyle().Foreground(textColor)
	orangeStyle  = stdRe.NewStyle().Foreground(orange)
	grayStyle    = stdRe.NewStyle().Foreground(lightGray)
	verboseStyle = verboseRenderer.NewStyle().Foreground(lightGray)
)

// Status and info styles
var (
	errStyle     = errRe.NewStyle().Bold(true).Align(lipgloss.Center).Foreground(white).Background(red).Padding(1, 6).Width(102)
	infoBoxStyle = stdRe.NewStyle().Bold(true).Foreground(white).Background(orange).Padding(1, 6)
	infoBoldStyle = stdRe.NewStyle().Bold(true).Foreground(orange)
	successStyle = stdRe.NewStyle().Bold(true).Foreground(white).Background(green)
	failureStyle = stdRe.NewStyle().Bold(true).Foreground(white).Background(red)
)

// vPrefix is used for verbose logging to show process ID
var vPrefix = fmt.Sprintf(" →(%d) ", os.Getpid())
