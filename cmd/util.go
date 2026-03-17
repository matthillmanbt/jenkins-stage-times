package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"
)

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
			res, err := jenkinsClient.Request(http.MethodGet, p.url)
			if err != nil {
				verbose("URLPoller request error for %s: %v", p.url, err)
				continue
			}
			if res.StatusCode < 200 || res.StatusCode >= 300 {
				verbose("URLPoller non-success status %d for %s", res.StatusCode, p.url)
				res.Body.Close()
				continue
			}
			verbose("URLPoller calling handler with response for %s", p.url)
			c <- res
		}
	}
}

var queueNumberRE = regexp.MustCompile(`/queue/item/(\d+)`)

// QueueNumberFromPath extracts the queue item number from a Jenkins queue path.
// Returns the queue number string, or empty string if not found.
func QueueNumberFromPath(path string) string {
	match := queueNumberRE.FindStringSubmatch(path)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

// Stop stops the URLPoller and cleans up resources
func (p *URLPoller) Stop() {
	p.ticker.Stop()
	select {
	case p.done <- true:
	default:
	}
}
