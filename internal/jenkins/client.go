package jenkins

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
)

// Client handles communication with Jenkins APIs
type Client struct {
	host       string
	user       string
	apiKey     string
	httpClient *http.Client
	verbose    func(string, ...any)
}

// Config contains configuration for creating a Jenkins client
type Config struct {
	Host    string
	User    string
	APIKey  string
	Verbose func(string, ...any)
}

// NewClient creates a new Jenkins API client
func NewClient(cfg Config) *Client {
	return &Client{
		host:       cfg.Host,
		user:       cfg.User,
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{},
		verbose:    cfg.Verbose,
	}
}

// log writes a verbose log message if verbose logging is enabled
func (c *Client) log(format string, args ...any) {
	if c.verbose != nil {
		c.verbose(format, args...)
	}
}

// Request makes an authenticated request to the Jenkins API
func (c *Client) Request(method, path string, query ...map[string]string) (*http.Response, error) {
	c.log("Using host [%s]", c.host)
	c.log("Using user [%s] and key [***]", c.user)

	apiKey := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.user, c.apiKey)))
	location := fmt.Sprintf("%s/%s", c.host, path)

	var body io.Reader = nil
	if len(query) > 0 && method != http.MethodGet {
		q := url.Values{}
		for k, v := range query[0] {
			q.Add(k, v)
		}
		data := q.Encode()
		c.log("setting post data [%s]", data)
		body = bytes.NewBuffer([]byte(data))
	}

	req, err := http.NewRequest(method, location, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", apiKey))

	if body != nil {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	if len(query) > 0 && method == http.MethodGet {
		q := req.URL.Query()
		for k, v := range query[0] {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	c.log("Calling jenkins API [%s][%s]", req.Method, req.URL)

	return c.httpClient.Do(req)
}

// GetLatestBuild retrieves the latest build matching the product and branch filters
func (c *Client) GetLatestBuild(pipeline, productFilter, branchFilter string) (*WorkflowRun, error) {
	path := fmt.Sprintf("job/%s/api/json", pipeline)
	query := map[string]string{"tree": "builds[id,fullDisplayName,actions[parameters[name,value]]]"}
	c.log("GetLatestBuild([%s], [%+v])", path, query)

	res, err := c.Request(http.MethodGet, path, query)
	if err != nil {
		c.log("Request error")
		return nil, err
	}
	defer res.Body.Close()

	var job WorkflowJob
	if err := json.NewDecoder(res.Body).Decode(&job); err != nil {
		c.log("JSON decode error")
		return nil, err
	}

	var latestBuild *WorkflowRun
	for _, run := range job.Builds {
		c.log("Build %s", run.ID)
		for _, action := range run.Actions {
			if len(action.Parameters) > 0 {
				bIdx := slices.IndexFunc(action.Parameters, func(p WorkflowParameter) bool {
					return p.Name == "TRYMAX_BRANCH"
				})
				pIdx := slices.IndexFunc(action.Parameters, func(p WorkflowParameter) bool {
					return p.Name == "PRODUCT"
				})
				if bIdx >= 0 && pIdx >= 0 {
					c.log("  %d = %v", bIdx, action.Parameters[bIdx].Value)
					c.log("  %d = %v", pIdx, action.Parameters[pIdx].Value)
					if action.Parameters[pIdx].Value == productFilter && action.Parameters[bIdx].Value == branchFilter {
						latestBuild = &run
						break
					}
				}
			}
		}

		if latestBuild != nil {
			break
		}
	}

	return latestBuild, nil
}

// GetBuildInfo retrieves information about a specific build
func (c *Client) GetBuildInfo(pipeline, buildID string) (*WorkflowRun, error) {
	path := fmt.Sprintf("job/%s/%s/api/json", pipeline, buildID)
	c.log("GetBuildInfo([%s])", path)

	res, err := c.Request(http.MethodGet, path)
	if err != nil {
		c.log("Request error")
		return nil, err
	}
	defer res.Body.Close()

	var run WorkflowRun
	if err := json.NewDecoder(res.Body).Decode(&run); err != nil {
		c.log("JSON decode error for build [%s]", buildID)
		return nil, err
	}

	return &run, nil
}

// GetJobs retrieves the list of recent jobs for a pipeline
func (c *Client) GetJobs(pipeline string) ([]Job, error) {
	path := fmt.Sprintf("job/%s/wfapi/runs", pipeline)
	res, err := c.Request(http.MethodGet, path)
	if err != nil {
		c.log("Request error")
		return nil, err
	}
	defer res.Body.Close()

	var jobs []Job
	if err := json.NewDecoder(res.Body).Decode(&jobs); err != nil {
		c.log("JSON decode error")
		return nil, err
	}

	return jobs, nil
}

// GetJobDetails retrieves detailed information about a specific job
func (c *Client) GetJobDetails(pipeline, jobID string) (*Job, error) {
	path := fmt.Sprintf("job/%s/%s/wfapi/describe", pipeline, jobID)
	res, err := c.Request(http.MethodGet, path)
	if err != nil {
		c.log("Request error")
		return nil, err
	}
	defer res.Body.Close()

	var job Job
	if err := json.NewDecoder(res.Body).Decode(&job); err != nil {
		c.log("JSON decode error")
		return nil, err
	}

	return &job, nil
}

// GetStageLog retrieves the console log for a specific stage
func (c *Client) GetStageLog(logHREF string) (*Node, error) {
	res, err := c.Request(http.MethodGet, logHREF)
	if err != nil {
		c.log("Request error")
		return nil, err
	}
	defer res.Body.Close()

	var node Node
	if err := json.NewDecoder(res.Body).Decode(&node); err != nil {
		c.log("JSON decode error")
		return nil, err
	}

	return &node, nil
}

// TriggerBuild triggers a parameterized build
func (c *Client) TriggerBuild(job string, params map[string]string) (*http.Response, error) {
	path := fmt.Sprintf("job/%s/buildWithParameters", job)
	c.log("TriggerBuild params [%#+v]", params)

	return c.Request(http.MethodPost, path, params)
}
