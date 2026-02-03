package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestConfigError(t *testing.T) {
	err := NewConfigError("host", "must be a valid URL")

	if err.Field != "host" {
		t.Errorf("Field = %s, want host", err.Field)
	}
	if err.Message != "must be a valid URL" {
		t.Errorf("Message = %s, want 'must be a valid URL'", err.Message)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "host") {
		t.Errorf("Error() should contain field name, got: %s", errStr)
	}
	if !strings.Contains(errStr, "must be a valid URL") {
		t.Errorf("Error() should contain message, got: %s", errStr)
	}
}

func TestAPIError(t *testing.T) {
	t.Run("without wrapped error", func(t *testing.T) {
		err := NewAPIError("/job/test", 404, "Not Found", nil)

		if err.URL != "/job/test" {
			t.Errorf("URL = %s, want /job/test", err.URL)
		}
		if err.StatusCode != 404 {
			t.Errorf("StatusCode = %d, want 404", err.StatusCode)
		}

		errStr := err.Error()
		if !strings.Contains(errStr, "404") {
			t.Errorf("Error() should contain status code, got: %s", errStr)
		}
		if !strings.Contains(errStr, "/job/test") {
			t.Errorf("Error() should contain URL, got: %s", errStr)
		}
	})

	t.Run("with wrapped error", func(t *testing.T) {
		innerErr := errors.New("connection refused")
		err := NewAPIError("/job/test", 0, "Failed to connect", innerErr)

		errStr := err.Error()
		if !strings.Contains(errStr, "connection refused") {
			t.Errorf("Error() should contain wrapped error, got: %s", errStr)
		}

		// Test Unwrap
		unwrapped := err.Unwrap()
		if unwrapped != innerErr {
			t.Error("Unwrap() should return the wrapped error")
		}
	})
}

func TestAuthError(t *testing.T) {
	err := NewAuthError("invalid credentials")

	errStr := err.Error()
	if !strings.Contains(errStr, "invalid credentials") {
		t.Errorf("Error() should contain message, got: %s", errStr)
	}
	if !strings.Contains(errStr, "JENKINS_HOST") {
		t.Errorf("Error() should contain helpful hints about env vars, got: %s", errStr)
	}
	if !strings.Contains(errStr, "JENKINS_USER") {
		t.Errorf("Error() should contain JENKINS_USER hint, got: %s", errStr)
	}
	if !strings.Contains(errStr, "JENKINS_KEY") {
		t.Errorf("Error() should contain JENKINS_KEY hint, got: %s", errStr)
	}
}

func TestBuildNotFoundError(t *testing.T) {
	err := NewBuildNotFoundError("1234", "master")

	if err.BuildID != "1234" {
		t.Errorf("BuildID = %s, want 1234", err.BuildID)
	}
	if err.Pipeline != "master" {
		t.Errorf("Pipeline = %s, want master", err.Pipeline)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "1234") {
		t.Errorf("Error() should contain build ID, got: %s", errStr)
	}
	if !strings.Contains(errStr, "master") {
		t.Errorf("Error() should contain pipeline, got: %s", errStr)
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("port", "8080", "must be between 1 and 65535")

	if err.Field != "port" {
		t.Errorf("Field = %s, want port", err.Field)
	}
	if err.Value != "8080" {
		t.Errorf("Value = %s, want 8080", err.Value)
	}
	if err.Message != "must be between 1 and 65535" {
		t.Errorf("Message = %s, want 'must be between 1 and 65535'", err.Message)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "port") {
		t.Errorf("Error() should contain field name, got: %s", errStr)
	}
	if !strings.Contains(errStr, "8080") {
		t.Errorf("Error() should contain value, got: %s", errStr)
	}
}
