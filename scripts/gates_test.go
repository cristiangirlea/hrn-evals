package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// suiteGates returns every gate command of every suite under ../suites,
// keyed by "file: task" for messages.
func suiteGates(t *testing.T) map[string][]string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "suites", "*", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no suites found: %v", err)
	}
	gates := map[string][]string{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var spec struct {
			Gate  []string `json:"gate"`
			Tasks []struct {
				ID   string   `json:"id"`
				Gate []string `json:"gate"`
			} `json:"tasks"`
		}
		if err := json.Unmarshal(b, &spec); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		name := filepath.ToSlash(f)
		gates[name+": (top level)"] = spec.Gate
		for _, task := range spec.Tasks {
			gates[name+": "+task.ID] = task.Gate
		}
	}
	return gates
}

// hrn runs a gate as bash -c <command> through os/exec. On Windows the
// command line Go builds is parsed again by Git Bash's runtime, which drops
// one of two backslashes inside double quotes, so a gate can reach bash
// changed. Every gate is passed the same way here and echoed back.
func TestGatesReachBashIntact(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not on PATH")
	}
	// The separator travels as an argument too, so the script needs no escapes.
	sep := string(rune(31))
	echo := `sep=$1; shift; for a; do printf '%s%s' "$a" "$sep"; done`
	for where, gates := range suiteGates(t) {
		// A few gates per call keeps each command line far below Windows' limit.
		for start := 0; start < len(gates); start += 8 {
			chunk := gates[start:min(start+8, len(gates))]
			args := append([]string{"-c", echo, "_", sep}, chunk...)
			out, err := exec.Command(bash, args...).Output()
			if err != nil {
				t.Fatalf("%s: bash: %v", where, err)
			}
			got := strings.Split(strings.TrimSuffix(string(out), sep), sep)
			for i, gate := range chunk {
				if i >= len(got) || got[i] != gate {
					t.Errorf("%s: a gate reaches bash changed; write the backslash another way (a bracket expression such as [.] or [\\]):\nsent %q\ngot  %q", where, gate, got[min(i, len(got)-1)])
				}
			}
		}
	}
}

// GNU grep 3.0 in Git for Windows aborts on -i with -F when no UTF-8 locale
// is set, as in a gate.
var grepIgnoreCaseFixed = regexp.MustCompile(`\bgrep\s+(-[A-Za-z]*i[A-Za-z]*F|-[A-Za-z]*F[A-Za-z]*i|-[A-Za-z]*i[A-Za-z]*\s+-[A-Za-z]*F|-[A-Za-z]*F[A-Za-z]*\s+-[A-Za-z]*i)`)

func TestGatesAvoidGrepIgnoreCaseFixed(t *testing.T) {
	for where, gates := range suiteGates(t) {
		for _, gate := range gates {
			if m := grepIgnoreCaseFixed.FindString(gate); m != "" {
				t.Errorf("%s: %q aborts without a UTF-8 locale on Git for Windows; drop -F (or -i):\n%s", where, m, gate)
			}
		}
	}
}

func TestGrepIgnoreCaseFixedPattern(t *testing.T) {
	for gate, want := range map[string]bool{
		"grep -qiF 'x' f":   true,
		"grep -qFi 'x' f":   true,
		"grep -i -F 'x' f":  true,
		"grep -F -qi 'x' f": true,
		"grep -qi 'x' f":    false,
		"grep -qF 'x' f":    false,
		"grep -qiE 'x' f":   false,
	} {
		if got := grepIgnoreCaseFixed.MatchString(gate); got != want {
			t.Errorf("%q: match = %v, want %v", gate, got, want)
		}
	}
}
