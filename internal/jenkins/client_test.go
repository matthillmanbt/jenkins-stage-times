package jenkins

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// mockVerbose is a test verbose function that does nothing
func mockVerbose(format string, args ...any) {}

func TestNewClient(t *testing.T) {
	cfg := Config{
		Host:    "https://jenkins.example.com",
		User:    "testuser",
		APIKey:  "testkey",
		Verbose: mockVerbose,
	}

	client := NewClient(cfg)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.host != cfg.Host {
		t.Errorf("client.host = %s, want %s", client.host, cfg.Host)
	}
	if client.user != cfg.User {
		t.Errorf("client.user = %s, want %s", client.user, cfg.User)
	}
	if client.apiKey != cfg.APIKey {
		t.Errorf("client.apiKey = %s, want %s", client.apiKey, cfg.APIKey)
	}
}

func TestClientRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("missing Authorization header")
		}

		// Check method and path
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/test/path" {
			t.Errorf("path = %s, want /test/path", r.URL.Path)
		}

		// Return test response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		Host:    server.URL,
		User:    "testuser",
		APIKey:  "testkey",
		Verbose: mockVerbose,
	})

	resp, err := client.Request(http.MethodGet, "test/path")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestClientGetJobs(t *testing.T) {
	// Load fixture
	jobsJSON, err := os.ReadFile("../../testdata/job.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	// Create test server that returns a list of jobs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job/master/wfapi/runs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return an array with one job
		w.Write([]byte("["))
		w.Write(jobsJSON)
		w.Write([]byte("]"))
	}))
	defer server.Close()

	client := NewClient(Config{
		Host:    server.URL,
		User:    "test",
		APIKey:  "test",
		Verbose: mockVerbose,
	})

	jobs, err := client.GetJobs("master")
	if err != nil {
		t.Fatalf("GetJobs failed: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	if jobs[0].ID != "1234" {
		t.Errorf("job.ID = %s, want 1234", jobs[0].ID)
	}
}

func TestClientGetBuildInfo(t *testing.T) {
	// Load fixture
	workflowJSON, err := os.ReadFile("../../testdata/workflow_run.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job/master/1234/api/json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(workflowJSON)
	}))
	defer server.Close()

	client := NewClient(Config{
		Host:    server.URL,
		User:    "test",
		APIKey:  "test",
		Verbose: mockVerbose,
	})

	build, err := client.GetBuildInfo("master", "1234")
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}

	if build.ID != "1234" {
		t.Errorf("build.ID = %s, want 1234", build.ID)
	}
	if build.Result != "SUCCESS" {
		t.Errorf("build.Result = %s, want SUCCESS", build.Result)
	}
}

func TestClientGetLatestBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return raw JSON to avoid Timestamp marshaling issues in the test
		responseJSON := `{
			"_class": "org.jenkinsci.plugins.workflow.job.WorkflowJob",
			"builds": [
				{
					"id": "1235",
					"timestamp": 1704067200000,
					"actions": [
						{
							"parameters": [
								{"name": "PRODUCT", "value": "ingredi"},
								{"name": "TRYMAX_BRANCH", "value": "origin/master"}
							]
						}
					]
				},
				{
					"id": "1234",
					"timestamp": 1704067100000,
					"actions": [
						{
							"parameters": [
								{"name": "PRODUCT", "value": "other"},
								{"name": "TRYMAX_BRANCH", "value": "origin/master"}
							]
						}
					]
				}
			]
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(Config{
		Host:    server.URL,
		User:    "test",
		APIKey:  "test",
		Verbose: mockVerbose,
	})

	build, err := client.GetLatestBuild("master", "ingredi", "origin/master")
	if err != nil {
		t.Fatalf("GetLatestBuild failed: %v", err)
	}

	if build == nil {
		t.Fatal("expected build, got nil")
	}

	if build.ID != "1235" {
		t.Errorf("build.ID = %s, want 1235", build.ID)
	}
}

func TestClientGetJobDetails(t *testing.T) {
	jobJSON, err := os.ReadFile("../../testdata/job.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job/master/1234/wfapi/describe" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jobJSON)
	}))
	defer server.Close()

	client := NewClient(Config{
		Host:    server.URL,
		User:    "test",
		APIKey:  "test",
		Verbose: mockVerbose,
	})

	job, err := client.GetJobDetails("master", "1234")
	if err != nil {
		t.Fatalf("GetJobDetails failed: %v", err)
	}

	if job.ID != "1234" {
		t.Errorf("job.ID = %s, want 1234", job.ID)
	}
	if len(job.Stages) != 2 {
		t.Errorf("len(job.Stages) = %d, want 2", len(job.Stages))
	}
}

func TestClientTriggerBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/job/build-site/buildWithParameters" {
			t.Errorf("path = %s, want /job/build-site/buildWithParameters", r.URL.Path)
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.Form.Get("PROJECT_NAME") != "master" {
			t.Errorf("PROJECT_NAME = %s, want master", r.Form.Get("PROJECT_NAME"))
		}

		w.Header().Set("Location", "https://jenkins.example.com/queue/item/123/")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(Config{
		Host:    server.URL,
		User:    "test",
		APIKey:  "test",
		Verbose: mockVerbose,
	})

	params := map[string]string{
		"PROJECT_NAME": "master",
		"BUILD_NUMBER": "1234",
	}

	resp, err := client.TriggerBuild("build-site", params)
	if err != nil {
		t.Fatalf("TriggerBuild failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status code = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}
