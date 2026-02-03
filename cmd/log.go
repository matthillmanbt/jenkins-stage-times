package cmd

import (
	"fmt"
	"os"
)

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
