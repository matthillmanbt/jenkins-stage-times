// Package jenkins provides types and client functionality for interacting with Jenkins APIs.
package jenkins

import (
	"encoding/json"
	"fmt"
	"time"
)

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

	// 2. Parse the unix timestamp (Jenkins uses milliseconds)
	p.Time = time.Unix(raw/1000, 0)
	return nil
}

// Link represents a hyperlink in Jenkins API responses
type Link struct {
	HREF string
}

// ResultLink contains navigation links for a Jenkins resource
type ResultLink struct {
	Self Link
	Log  Link
}

// Base contains common fields shared by Job and Stage types
type Base struct {
	Links     ResultLink `json:"_links"`
	ID        string
	Name      string
	Status    string
	StartTime Timestamp `json:"startTimeMillis"`
	Duration  int       `json:"durationMillis"`
}

// Stage represents a pipeline stage in a Jenkins job
type Stage struct {
	Base

	ExecNode      string
	PauseDuration int `json:"pauseDurationMillis"`

	ParentNodes    []string
	StageFlowNodes []Stage
}

// Job represents a complete Jenkins pipeline job execution
type Job struct {
	Base

	EndTime Timestamp `json:"endTimeMillis"`

	QueueDuration int `json:"queueDurationMillis"`
	PauseDuration int `json:"pauseDurationMillis"`
	Stages        []Stage
}

// WorkflowRun represents metadata about a workflow run from Jenkins API
type WorkflowRun struct {
	Class   string `json:"_class"`
	Name    string `json:"fullDisplayName"`
	ID      string
	Actions []WorkflowAction

	Duration          int
	EstimatedDuration int
	FullDisplayName   string
	DisplayName       string
	Result            string
	Timestamp         Timestamp
	URL               string
	Description       string
	Building          bool
}

// WorkflowAction represents an action taken during a workflow run
type WorkflowAction struct {
	Class      string `json:"_class"`
	Parameters []WorkflowParameter
}

// WorkflowParameter represents a parameter passed to a workflow
type WorkflowParameter struct {
	Class string `json:"_class"`
	Name  string
	Value any
}

// WorkflowJob represents a Jenkins workflow job with multiple builds
type WorkflowJob struct {
	Class  string `json:"_class"`
	Builds []WorkflowRun
}

// Node represents a node in the Jenkins execution graph with console output
type Node struct {
	ID         string `json:"nodeId"`
	Status     string `json:"nodeStatus"`
	Length     int
	HasMore    bool
	Text       string
	ConsoleURL string
}

// ExecutableItem represents an executable item in the Jenkins queue
type ExecutableItem struct {
	Class  string `json:"_class"`
	Number int
	URL    string
}

// QueueItem represents an item in the Jenkins build queue
type QueueItem struct {
	ID         string
	Executable ExecutableItem
}
