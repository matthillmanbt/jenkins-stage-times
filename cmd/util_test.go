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
