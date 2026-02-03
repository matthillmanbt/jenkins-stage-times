package jenkins

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestTimestampUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected time.Time
		wantErr  bool
	}{
		{
			name:     "valid timestamp",
			jsonStr:  "1704067200000",
			expected: time.Unix(1704067200, 0),
			wantErr:  false,
		},
		{
			name:     "zero timestamp",
			jsonStr:  "0",
			expected: time.Unix(0, 0),
			wantErr:  false,
		},
		{
			name:    "invalid json",
			jsonStr: "not-a-number",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts Timestamp
			err := json.Unmarshal([]byte(tt.jsonStr), &ts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !ts.Time.Equal(tt.expected) {
				t.Errorf("timestamp = %v, want %v", ts.Time, tt.expected)
			}
		})
	}
}

func TestJobUnmarshal(t *testing.T) {
	data, err := os.ReadFile("../../testdata/job.json")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		t.Fatalf("failed to unmarshal job: %v", err)
	}

	// Verify basic fields
	if job.ID != "1234" {
		t.Errorf("job.ID = %s, want 1234", job.ID)
	}
	if job.Name != "Build #1234" {
		t.Errorf("job.Name = %s, want Build #1234", job.Name)
	}
	if job.Status != "SUCCESS" {
		t.Errorf("job.Status = %s, want SUCCESS", job.Status)
	}
	if job.Duration != 120000 {
		t.Errorf("job.Duration = %d, want 120000", job.Duration)
	}

	// Verify timestamp parsing
	expectedStart := time.Unix(1704067200, 0)
	if !job.StartTime.Time.Equal(expectedStart) {
		t.Errorf("job.StartTime = %v, want %v", job.StartTime.Time, expectedStart)
	}

	// Verify stages
	if len(job.Stages) != 2 {
		t.Fatalf("len(job.Stages) = %d, want 2", len(job.Stages))
	}

	// Check first stage
	stage1 := job.Stages[0]
	if stage1.ID != "1" {
		t.Errorf("stage1.ID = %s, want 1", stage1.ID)
	}
	if stage1.Name != "Checkout" {
		t.Errorf("stage1.Name = %s, want Checkout", stage1.Name)
	}
	if stage1.ExecNode != "node1" {
		t.Errorf("stage1.ExecNode = %s, want node1", stage1.ExecNode)
	}

	// Check second stage
	stage2 := job.Stages[1]
	if stage2.ID != "2" {
		t.Errorf("stage2.ID = %s, want 2", stage2.ID)
	}
	if stage2.Name != "Build" {
		t.Errorf("stage2.Name = %s, want Build", stage2.Name)
	}
	if len(stage2.ParentNodes) != 1 || stage2.ParentNodes[0] != "1" {
		t.Errorf("stage2.ParentNodes = %v, want [1]", stage2.ParentNodes)
	}
}

func TestWorkflowRunUnmarshal(t *testing.T) {
	data, err := os.ReadFile("../../testdata/workflow_run.json")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	var run WorkflowRun
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatalf("failed to unmarshal workflow run: %v", err)
	}

	// Verify basic fields
	if run.ID != "1234" {
		t.Errorf("run.ID = %s, want 1234", run.ID)
	}
	if run.Result != "SUCCESS" {
		t.Errorf("run.Result = %s, want SUCCESS", run.Result)
	}
	if run.Building {
		t.Error("run.Building = true, want false")
	}
	if run.Duration != 120000 {
		t.Errorf("run.Duration = %d, want 120000", run.Duration)
	}

	// Verify actions and parameters
	if len(run.Actions) != 1 {
		t.Fatalf("len(run.Actions) = %d, want 1", len(run.Actions))
	}

	action := run.Actions[0]
	if len(action.Parameters) != 2 {
		t.Fatalf("len(action.Parameters) = %d, want 2", len(action.Parameters))
	}

	// Check parameters
	productParam := action.Parameters[0]
	if productParam.Name != "PRODUCT" {
		t.Errorf("param[0].Name = %s, want PRODUCT", productParam.Name)
	}
	if productParam.Value != "ingredi" {
		t.Errorf("param[0].Value = %v, want ingredi", productParam.Value)
	}

	branchParam := action.Parameters[1]
	if branchParam.Name != "TRYMAX_BRANCH" {
		t.Errorf("param[1].Name = %s, want TRYMAX_BRANCH", branchParam.Name)
	}
	if branchParam.Value != "origin/master" {
		t.Errorf("param[1].Value = %v, want origin/master", branchParam.Value)
	}
}

func TestStageLinks(t *testing.T) {
	jsonStr := `{
		"_links": {
			"self": {"href": "/job/test/1/node/5/"},
			"log": {"href": "/job/test/1/node/5/wfapi/log"}
		},
		"id": "5",
		"name": "Test Stage",
		"status": "SUCCESS",
		"startTimeMillis": 1704067200000,
		"durationMillis": 1000,
		"pauseDurationMillis": 0,
		"execNode": "test-node",
		"parentNodes": [],
		"stageFlowNodes": []
	}`

	var stage Stage
	if err := json.Unmarshal([]byte(jsonStr), &stage); err != nil {
		t.Fatalf("failed to unmarshal stage: %v", err)
	}

	if stage.Links.Self.HREF != "/job/test/1/node/5/" {
		t.Errorf("stage.Links.Self.HREF = %s, want /job/test/1/node/5/", stage.Links.Self.HREF)
	}
	if stage.Links.Log.HREF != "/job/test/1/node/5/wfapi/log" {
		t.Errorf("stage.Links.Log.HREF = %s, want /job/test/1/node/5/wfapi/log", stage.Links.Log.HREF)
	}
}
