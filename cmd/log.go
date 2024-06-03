package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	lightGray       = lipgloss.Color("#888888")
	verboseRenderer = lipgloss.NewRenderer(os.Stderr)
	verboseStyle    = verboseRenderer.NewStyle().Foreground(lightGray)

	stdRe    = lipgloss.NewRenderer(os.Stdout)
	errRe    = lipgloss.NewRenderer(os.Stderr)
	errStyle = errRe.NewStyle().
			Bold(true).
			Align(lipgloss.Center).
			Foreground(white).
			Background(red).
			Padding(1, 6).
			Width(102)
)

func verbose(format string, a ...any) {
	if Verbose == 1 {
		fmt.Println(verboseStyle.Render(" → " + fmt.Sprintf(format, a...)))
	}
}
func vVerbose(format string, a ...any) {
	if Verbose > 1 {
		fmt.Println(verboseStyle.Render(" → " + fmt.Sprintf(format, a...)))
	}
}
