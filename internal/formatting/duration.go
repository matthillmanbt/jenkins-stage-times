// Package formatting provides utilities for formatting data for display.
package formatting

import (
	"fmt"
	"time"
)

// Duration formats a duration as MM:SS.mmm
func Duration(d time.Duration) string {
	d = d.Round(time.Millisecond)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	d -= s * time.Second
	return fmt.Sprintf("%02d:%02d.%03d", m, s, d.Milliseconds())
}
