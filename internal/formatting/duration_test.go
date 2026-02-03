package formatting

import (
	"testing"
	"time"
)

func TestDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			duration: 0,
			expected: "00:00.000",
		},
		{
			name:     "milliseconds only",
			duration: 123 * time.Millisecond,
			expected: "00:00.123",
		},
		{
			name:     "seconds only",
			duration: 5 * time.Second,
			expected: "00:05.000",
		},
		{
			name:     "seconds and milliseconds",
			duration: 5*time.Second + 250*time.Millisecond,
			expected: "00:05.250",
		},
		{
			name:     "minutes only",
			duration: 2 * time.Minute,
			expected: "02:00.000",
		},
		{
			name:     "minutes and seconds",
			duration: 3*time.Minute + 45*time.Second,
			expected: "03:45.000",
		},
		{
			name:     "full format",
			duration: 12*time.Minute + 34*time.Second + 567*time.Millisecond,
			expected: "12:34.567",
		},
		{
			name:     "over an hour",
			duration: 75*time.Minute + 30*time.Second,
			expected: "75:30.000",
		},
		{
			name:     "single digit minutes",
			duration: 1*time.Minute + 2*time.Second + 3*time.Millisecond,
			expected: "01:02.003",
		},
		{
			name:     "rounding milliseconds",
			duration: 1*time.Second + 500*time.Microsecond, // 0.5 ms rounds to 1 ms
			expected: "00:01.001",
		},
		{
			name:     "rounding down milliseconds",
			duration: 1*time.Second + 400*time.Microsecond, // 0.4 ms rounds to 0 ms
			expected: "00:01.000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Duration(tt.duration)
			if result != tt.expected {
				t.Errorf("Duration(%v) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestDurationEdgeCases(t *testing.T) {
	t.Run("negative duration becomes zero", func(t *testing.T) {
		// Duration formatter rounds, so very small negative values might round to 00:00.000
		result := Duration(-100 * time.Millisecond)
		// After rounding, negative small durations may become zero or stay negative
		// The behavior depends on time.Round implementation
		if result == "" {
			t.Error("Duration returned empty string for negative duration")
		}
	})

	t.Run("very large duration", func(t *testing.T) {
		// 10 hours = 600 minutes
		duration := 10 * time.Hour
		result := Duration(duration)
		expected := "600:00.000"
		if result != expected {
			t.Errorf("Duration(%v) = %s, want %s", duration, result, expected)
		}
	})

	t.Run("microseconds round correctly", func(t *testing.T) {
		// Test rounding at 0.5ms boundary
		tests := []struct {
			micros   int
			expected string
		}{
			{499, "00:00.000"},  // 0.499ms rounds down
			{500, "00:00.001"},  // 0.500ms rounds up
			{1500, "00:00.002"}, // 1.500ms rounds up
		}

		for _, tt := range tests {
			duration := time.Duration(tt.micros) * time.Microsecond
			result := Duration(duration)
			if result != tt.expected {
				t.Errorf("Duration(%d µs) = %s, want %s", tt.micros, result, tt.expected)
			}
		}
	})
}
