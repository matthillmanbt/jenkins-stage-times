package cmd

import (
	"encoding/json"
	"fmt"
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
	p.Time = time.Unix(raw, 0)
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
