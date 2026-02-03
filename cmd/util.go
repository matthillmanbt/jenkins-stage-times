package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"golang.org/x/exp/constraints"
)

// Ptr returns a pointer to the given value. Useful for converting literals to pointers.
func Ptr[Value any](v Value) *Value {
	return &v
}

// Timestamp wraps time.Time to provide custom JSON unmarshaling for Unix millisecond timestamps
type Timestamp struct {
	time.Time
}

// UnmarshalJSON decodes an int64 timestamp into a time.Time object
func (p *Timestamp) UnmarshalJSON(bytes []byte) error {
	// 1. Decode the bytes into an int64
	var raw int64
	err := json.Unmarshal(bytes, &raw)

	if err != nil {
		fmt.Printf("error decoding timestamp: %s\n", err)
		return err
	}

	// 2. Parse the unix timestamp
	p.Time = time.Unix(raw/1000, 0)
	return nil
}

type Number interface {
	constraints.Float | constraints.Integer
}

// avg calculates the arithmetic mean of a slice of numbers.
// Returns 0 if the slice is empty.
func avg[T Number](data []T) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, v := range data {
		sum += float64(v)
	}
	return sum / float64(len(data))
}

// fmtDuration formats a duration as MM:SS.mmm
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	d -= s * time.Second
	return fmt.Sprintf("%02d:%02d.%03d", m, s, d.Milliseconds())
}

const (
	childEnv = "__IS_CHILD"
	// DefaultPollInterval is the default interval for URLPoller to check for responses
	DefaultPollInterval = 3 * time.Second
)

// SpawnBG spawns the current executable in the background with the given arguments
func SpawnBG(args ...string) (*exec.Cmd, error) {
	return Spawn(os.Args[0], args...)
}

// Spawn starts a command with the given arguments.
// Returns the started command or an error if the command fails to start.
func Spawn(command string, args ...string) (*exec.Cmd, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("%v=%v", childEnv, 1))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	verbose("spawning command [%v]", cmd.Args)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to spawn command %s: %w", command, err)
	}

	return cmd, nil
}

// URLPoller polls a URL at regular intervals until it receives a successful response
type URLPoller struct {
	ticker   *time.Ticker
	url      string
	Response <-chan *http.Response
	done     chan bool
}

// NewURLPoller creates a new URLPoller that polls the given URL.
// The response will be sent to the Response channel when available.
func NewURLPoller(url string) *URLPoller {
	c := make(chan *http.Response, 1)
	done := make(chan bool, 1)
	p := &URLPoller{
		ticker:   time.NewTicker(DefaultPollInterval),
		url:      url,
		Response: c,
		done:     done,
	}
	go p.run(c, done)
	return p
}

func (p *URLPoller) run(c chan *http.Response, done chan bool) {
	defer close(c)
	for {
		select {
		case <-done:
			verbose("URLPoller stopping for URL %s", p.url)
			return
		case <-p.ticker.C:
			verbose("URLPoller querying URL %s", p.url)
			if res, err := jenkinsRequest(p.url); err == nil {
				verbose("URLPoller calling handler with response for %s", p.url)
				c <- res
				p.ticker.Stop()
				return
			}
		}
	}
}

// Stop stops the URLPoller and cleans up resources
func (p *URLPoller) Stop() {
	p.ticker.Stop()
	select {
	case p.done <- true:
	default:
	}
}
