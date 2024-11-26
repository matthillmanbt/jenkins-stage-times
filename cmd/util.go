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

func Ptr[Value any](v Value) *Value {
	return &v
}

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

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	d -= s * time.Second
	return fmt.Sprintf("%02d:%02d.%03d", m, s, d.Milliseconds())
}

var childEnv = "__IS_CHILD"

func SpawnBG(args ...string) *exec.Cmd {
	return Spawn(os.Args[0], args...)
}

func Spawn(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("%v=%v", childEnv, 1))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	verbose("spawning command [%v]", cmd.Args)
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	return cmd
}

// func getNumFileDescriptors() (int, error) {
// 	pid := os.Getpid()
// 	fds, err := os.Open(fmt.Sprintf("/proc/%d/fd", pid))

// 	if err != nil {
// 		return 0, err
// 	}
// 	defer fds.Close()

// 	files, err := fds.Readdirnames(-1)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return len(files), nil
// }

type URLPoller struct {
	ticker   *time.Ticker
	url      string
	Response <-chan *http.Response
}

func NewURLPoller(url string) *URLPoller {
	c := make(chan *http.Response, 1)
	p := &URLPoller{
		ticker:   time.NewTicker(time.Second * 3),
		url:      url,
		Response: c,
	}
	go p.run(c)
	return p
}

func (p *URLPoller) run(c chan *http.Response) {
	for ; true; <-p.ticker.C {
		verbose("URLPoller querying URL %s", p.url)
		if res, err := jenkinsRequest(p.url); err == nil {
			verbose("URLPoller calling handler with response for %s", p.url)
			c <- res
			p.ticker.Stop()
		}
	}
}
