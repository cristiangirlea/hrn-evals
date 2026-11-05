//go:build ignore

// Explicitly included by stream-eval.sh; keep its entry point separate from
// where-it-loses so go build ./... and go test ./scripts/ still work.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := streamEval(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "stream-eval:", err)
		os.Exit(1)
	}
}
