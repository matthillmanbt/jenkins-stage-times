package util

import (
	"testing"
)

func TestAvg(t *testing.T) {
	tests := []struct {
		name     string
		data     []int
		expected float64
	}{
		{
			name:     "empty slice",
			data:     []int{},
			expected: 0,
		},
		{
			name:     "single element",
			data:     []int{5},
			expected: 5,
		},
		{
			name:     "multiple elements",
			data:     []int{1, 2, 3, 4, 5},
			expected: 3,
		},
		{
			name:     "negative numbers",
			data:     []int{-10, -5, 0, 5, 10},
			expected: 0,
		},
		{
			name:     "large numbers",
			data:     []int{1000, 2000, 3000},
			expected: 2000,
		},
		{
			name:     "decimal average",
			data:     []int{1, 2, 3},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Avg(tt.data)
			if result != tt.expected {
				t.Errorf("Avg(%v) = %f, want %f", tt.data, result, tt.expected)
			}
		})
	}
}

func TestAvgFloat(t *testing.T) {
	tests := []struct {
		name      string
		data      []float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "floats with decimals",
			data:      []float64{1.5, 2.5, 3.5},
			expected:  2.5,
			tolerance: 0.0001,
		},
		{
			name:      "small decimals",
			data:      []float64{0.1, 0.2, 0.3},
			expected:  0.2,
			tolerance: 0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Avg(tt.data)
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("Avg(%v) = %f, want %f (tolerance %f)", tt.data, result, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestPtr(t *testing.T) {
	t.Run("int pointer", func(t *testing.T) {
		value := 42
		ptr := Ptr(value)
		if ptr == nil {
			t.Fatal("Ptr returned nil")
		}
		if *ptr != value {
			t.Errorf("*Ptr(%d) = %d, want %d", value, *ptr, value)
		}
	})

	t.Run("string pointer", func(t *testing.T) {
		value := "hello"
		ptr := Ptr(value)
		if ptr == nil {
			t.Fatal("Ptr returned nil")
		}
		if *ptr != value {
			t.Errorf("*Ptr(%s) = %s, want %s", value, *ptr, value)
		}
	})

	t.Run("bool pointer", func(t *testing.T) {
		value := true
		ptr := Ptr(value)
		if ptr == nil {
			t.Fatal("Ptr returned nil")
		}
		if *ptr != value {
			t.Errorf("*Ptr(%t) = %t, want %t", value, *ptr, value)
		}
	})

	t.Run("struct pointer", func(t *testing.T) {
		type testStruct struct {
			Name string
			Age  int
		}
		value := testStruct{Name: "Alice", Age: 30}
		ptr := Ptr(value)
		if ptr == nil {
			t.Fatal("Ptr returned nil")
		}
		if ptr.Name != value.Name || ptr.Age != value.Age {
			t.Errorf("*Ptr(%+v) = %+v, want %+v", value, *ptr, value)
		}
	})

	t.Run("zero value", func(t *testing.T) {
		value := 0
		ptr := Ptr(value)
		if ptr == nil {
			t.Fatal("Ptr returned nil")
		}
		if *ptr != 0 {
			t.Errorf("*Ptr(0) = %d, want 0", *ptr)
		}
	})
}
