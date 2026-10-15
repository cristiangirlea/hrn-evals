package refactor

import "errors"

// Sum returns the sum, rejecting empty input.
func Sum(values []int) (float64, error) {
	if len(values) == 0 {
		return 0, errors.New("values must not be empty")
	}
	total := 0.0
	for _, value := range values {
		total += float64(value)
	}
	return total, nil
}

// Mean returns the arithmetic mean, rejecting empty input.
func Mean(values []int) (float64, error) {
	if len(values) == 0 {
		return 0, errors.New("values must not be empty")
	}
	total := 0.0
	for _, value := range values {
		total += float64(value)
	}
	return total / float64(len(values)), nil
}
