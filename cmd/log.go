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

	orange = lipgloss.Color("#FF5500")
	gray   = lipgloss.Color("#222222")
	white  = lipgloss.Color("#FFFFFF")
	red    = lipgloss.Color("#FF0000")
	green  = lipgloss.Color("#00FF00")
)

var vPrefix = fmt.Sprintf(" →(%d) ", os.Getpid())

func verbose(format string, a ...any) {
	if Verbose == 1 {
		fmt.Println(verboseStyle.Render(vPrefix + fmt.Sprintf(format, a...)))
	}
}
func vVerbose(format string, a ...any) {
	if Verbose > 1 {
		fmt.Println(verboseStyle.Render(vPrefix + fmt.Sprintf(format, a...)))
	}
}
