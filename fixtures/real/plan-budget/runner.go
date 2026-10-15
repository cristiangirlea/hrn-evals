// Package runner runs a batch of agent tasks a few at a time.
package runner

import (
	"context"
	"sync"
)

// Task is one agent task of a batch.
type Task struct {
	ID     string
	Prompt string
}

// Usage is what one task's run cost.
type Usage struct {
	CostUSD      float64
	InputTokens  int
	OutputTokens int
}

// RunAll runs every task with run, at most parallel at once, and returns the
// first error any run returned after all of them finished.
func RunAll(ctx context.Context, tasks []Task, parallel int, run func(context.Context, Task) (Usage, error)) error {
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		first error
	)
	slots := make(chan struct{}, max(parallel, 1))
	for _, task := range tasks {
		slots <- struct{}{}
		wg.Add(1)
		go func(task Task) {
			defer wg.Done()
			defer func() { <-slots }()
			if _, err := run(ctx, task); err != nil {
				mu.Lock()
				if first == nil {
					first = err
				}
				mu.Unlock()
			}
		}(task)
	}
	wg.Wait()
	return first
}
