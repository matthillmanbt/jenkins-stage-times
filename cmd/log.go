package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	lightGray = lipgloss.Color("#7C878E")
	orange    = lipgloss.Color("#FF5500")
	darkNavy  = lipgloss.Color("#081826")
	cyan      = lipgloss.Color("#4EC3E0")
	gray      = lipgloss.Color("#222222")
	white     = lipgloss.Color("#FFFFFF")
	red       = lipgloss.Color("#FF0000")
	green     = lipgloss.Color("#00FF00")
	// navy      = lipgloss.Color("#253746")

	textColor = lipgloss.AdaptiveColor{Light: string(darkNavy), Dark: string(white)}

	verboseRenderer = lipgloss.NewRenderer(os.Stderr)
	verboseStyle    = verboseRenderer.NewStyle().Foreground(lightGray)

	stdRe = lipgloss.NewRenderer(os.Stdout)
	errRe = lipgloss.NewRenderer(os.Stderr)

	errStyle     = errRe.NewStyle().Bold(true).Align(lipgloss.Center).Foreground(white).Background(red).Padding(1, 6).Width(102)
	noStyle      = stdRe.NewStyle().Foreground(lipgloss.NoColor{})
	textStyle    = stdRe.NewStyle().Foreground(textColor)
	orangeStyle  = stdRe.NewStyle().Foreground(orange)
	grayStyle    = stdRe.NewStyle().Foreground(lightGray)
	infoBoxStyle = stdRe.NewStyle().Bold(true).Foreground(white).Background(orange).Padding(1, 6)
)

var vPrefix = fmt.Sprintf(" →(%d) ", os.Getpid())

func verbose(format string, a ...any) {
	if Verbose > 0 {
		fmt.Fprintln(os.Stderr, verboseStyle.Render(vPrefix+fmt.Sprintf(format, a...)))
	}
}
func vVerbose(format string, a ...any) {
	if Verbose > 1 {
		fmt.Fprintln(os.Stderr, verboseStyle.Render(vPrefix+fmt.Sprintf(format, a...)))
	}
}
