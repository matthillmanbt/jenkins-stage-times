package cmd

import (
	"fmt"
)

// Error types for different failure scenarios

// ConfigError represents configuration-related errors
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("configuration error for %s: %s", e.Field, e.Message)
}

// NewConfigError creates a new configuration error
func NewConfigError(field, message string) *ConfigError {
	return &ConfigError{Field: field, Message: message}
}

// APIError represents Jenkins API-related errors
type APIError struct {
	URL        string
	StatusCode int
	Message    string
	Err        error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("API error [%s] (status %d): %s - %v", e.URL, e.StatusCode, e.Message, e.Err)
	}
	return fmt.Sprintf("API error [%s] (status %d): %s", e.URL, e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// NewAPIError creates a new API error
func NewAPIError(url string, statusCode int, message string, err error) *APIError {
	return &APIError{
		URL:        url,
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

// AuthError represents authentication-related errors
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication error: %s", e.Message)
}

// NewAuthError creates a new authentication error with helpful context
func NewAuthError(message string) *AuthError {
	return &AuthError{
		Message: message + "\n\nPlease ensure you have set the following environment variables:\n" +
			"  JENKINS_HOST - Your Jenkins server URL\n" +
			"  JENKINS_USER - Your Jenkins username\n" +
			"  JENKINS_KEY  - Your Jenkins API key\n\n" +
			"Or configure them in ~/.jenkins.yaml",
	}
}

// BuildNotFoundError represents errors when a build cannot be found
type BuildNotFoundError struct {
	BuildID  string
	Pipeline string
}

func (e *BuildNotFoundError) Error() string {
	return fmt.Sprintf("build %s not found in pipeline %s", e.BuildID, e.Pipeline)
}

// NewBuildNotFoundError creates a new build not found error
func NewBuildNotFoundError(buildID, pipeline string) *BuildNotFoundError {
	return &BuildNotFoundError{BuildID: buildID, Pipeline: pipeline}
}

// ValidationError represents validation errors for user input
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s='%s': %s", e.Field, e.Value, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{Field: field, Value: value, Message: message}
}
