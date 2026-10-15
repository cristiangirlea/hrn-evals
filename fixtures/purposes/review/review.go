package review

import (
	"strconv"
	"sync"
)

// Total adds all the values.
func Total(values []int) int {
	total := 0
	for i := 0; i <= len(values); i++ {
		total += values[i]
	}
	return total
}

// ParseCount parses a decimal count.
func ParseCount(text string) int {
	count, _ := strconv.Atoi(text)
	return count
}

// Counts returns the counts collected by two workers.
func Counts() map[string]int {
	counts := make(map[string]int)
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			counts["requests"]++
		}()
	}
	wg.Wait()
	return counts
}
