// Package util provides generic utility functions.
package util

import "golang.org/x/exp/constraints"

// Number is a constraint for numeric types
type Number interface {
	constraints.Float | constraints.Integer
}

// Avg calculates the arithmetic mean of a slice of numbers.
// Returns 0 if the slice is empty.
func Avg[T Number](data []T) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, v := range data {
		sum += float64(v)
	}
	return sum / float64(len(data))
}

// Ptr returns a pointer to the given value. Useful for converting literals to pointers.
func Ptr[Value any](v Value) *Value {
	return &v
}
