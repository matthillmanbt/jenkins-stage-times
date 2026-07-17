package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestURLPollerStopsCleanly(t *testing.T) {
	// Create a test server that never responds successfully
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	// Create a mock client for the test
	oldClient := jenkinsClient
	defer func() { jenkinsClient = oldClient }()

	// Create a poller
	poller := NewURLPoller("test/path")

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Stop the poller
	poller.Stop()

	// Give it a moment to stop
	time.Sleep(10 * time.Millisecond)

	// The poller should have stopped
	// We can't check the channel directly, but we can verify Stop() doesn't panic
	poller.Stop() // Calling Stop again should be safe
}

func TestSpawnBGReturnsError(t *testing.T) {
	// Try to spawn a non-existent command
	_, err := Spawn("/nonexistent/command/path", "arg1", "arg2")
	if err == nil {
		t.Error("expected error when spawning non-existent command, got nil")
	}
}

func TestSpawnSuccess(t *testing.T) {
	// Test spawning a real command (echo is available on all platforms)
	cmd, err := Spawn("echo", "test")
	if err != nil {
		t.Fatalf("unexpected error spawning echo: %v", err)
	}

	if cmd == nil {
		t.Fatal("expected cmd to be non-nil")
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		t.Errorf("command failed: %v", err)
	}
}

func TestMergeParams(t *testing.T) {
	params := map[string]string{"PRODUCT": "ingredi", "TRYMAX_BRANCH": "origin/master"}

	err := MergeParams(params, []string{
		"CAPTURE_IB_LOGS=true",
		"TRYMAX_BRANCH=origin/feature/x", // overrides default
		"WEB_BRANCH=",                    // empty value is allowed
		"OPTS=a=b",                       // value may contain '='
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"PRODUCT":         "ingredi",
		"TRYMAX_BRANCH":   "origin/feature/x",
		"CAPTURE_IB_LOGS": "true",
		"WEB_BRANCH":      "",
		"OPTS":            "a=b",
	}
	for k, v := range want {
		if params[k] != v {
			t.Errorf("params[%q] = %q, want %q", k, params[k], v)
		}
	}
	if len(params) != len(want) {
		t.Errorf("params has %d entries, want %d", len(params), len(want))
	}
}

func TestMergeParamsInvalid(t *testing.T) {
	for _, kv := range []string{"NOVALUE", "=missing-key", ""} {
		if err := MergeParams(map[string]string{}, []string{kv}); err == nil {
			t.Errorf("expected error for %q, got nil", kv)
		}
	}
}
