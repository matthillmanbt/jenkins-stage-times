package cmd

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/spf13/viper"
)

type Link struct {
	HREF string
}

type ResultLink struct {
	Self      Link
	Artifacts *Link `json:"omitempty"`
}

type Stage struct {
	Links         ResultLink `json:"_links"`
	ID            string
	Name          string
	ExecNode      string
	Status        string
	StartTime     Timestamp `json:"startTimeMillis"`
	Duration      int       `json:"durationMillis"`
	PauseDuration int       `json:"pauseDurationMillis"`
}

type Job struct {
	Links         ResultLink `json:"_links"`
	ID            string
	Name          string
	Status        string
	StartTime     Timestamp `json:"startTimeMillis"`
	EndTime       Timestamp `json:"endTimeMillis"`
	Duration      int       `json:"durationMillis"`
	QueueDuration int       `json:"queueDurationMillis"`
	PauseDuration int       `json:"pauseDurationMillis"`
	Stages        []Stage
}

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

type WorkflowAction struct {
	Class      string `json:"_class"`
	Parameters []WorkflowParameter
}

type WorkflowParameter struct {
	Class string `json:"_class"`
	Name  string
	Value any
}

type WorkflowJob struct {
	Class  string `json:"_class"`
	Builds []WorkflowRun
}

func getLatestBuild(productFilter string, branchFilter string) (*WorkflowRun, error) {
	url := fmt.Sprintf("job/%s/api/json", viper.Get("pipeline"))
	query := map[string]string{"tree": "builds[id,fullDisplayName,actions[parameters[name,value]]]"}
	res, err := jenkinsRequest(url, query)
	if err != nil {
		verbose("Request error")
		return nil, err
	}
	defer res.Body.Close()

	var job WorkflowJob
	if err := json.NewDecoder(res.Body).Decode(&job); err != nil {
		verbose("JSON decode error")
		return nil, err
	}

	var latestBuild *WorkflowRun
	for _, run := range job.Builds {
		vVerbose("Build %s", run.ID)
		for _, action := range run.Actions {
			if len(action.Parameters) > 0 {
				bIdx := slices.IndexFunc(action.Parameters, func(p WorkflowParameter) bool { return p.Name == "TRYMAX_BRANCH" })
				pIdx := slices.IndexFunc(action.Parameters, func(p WorkflowParameter) bool { return p.Name == "PRODUCT" })
				vVerbose("  %d = %v", bIdx, action.Parameters[bIdx].Value)
				vVerbose("  %d = %v", pIdx, action.Parameters[pIdx].Value)
				if action.Parameters[pIdx].Value == productFilter && action.Parameters[bIdx].Value == branchFilter {
					latestBuild = &run
					break
				}
			}
		}

		if latestBuild != nil {
			break
		}
	}

	return latestBuild, nil
}

func getBuildInfo(buildID string) (*WorkflowRun, error) {
	url := fmt.Sprintf("job/%s/%s/api/json", viper.Get("pipeline"), buildID)
	res, err := jenkinsRequest(url)
	if err != nil {
		verbose("Request error")
		return nil, err
	}
	defer res.Body.Close()

	var run WorkflowRun
	if err := json.NewDecoder(res.Body).Decode(&run); err != nil {
		verbose("JSON decode error for build [%s]", buildID)
		return nil, err
	}

	return &run, nil
}
