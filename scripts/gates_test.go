package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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

// gateBash picks bash as hrn's tools.Shell does: the one on PATH unless it is
// WSL's launcher in System32, else Git Bash at its install path.
func gateBash() (string, bool) {
	if path, err := exec.LookPath("bash"); err == nil && !(runtime.GOOS == "windows" && strings.Contains(strings.ToLower(filepath.ToSlash(path)), "/system32/")) {
		return path, true
	}
	if runtime.GOOS == "windows" {
		for _, p := range []string{`C:\Program Files\Git\bin\bash.exe`, `C:\Program Files\Git\usr\bin\bash.exe`} {
			if _, err := os.Stat(p); err == nil {
				return p, true
			}
		}
	}
	return "", false
}

// hrn runs a gate as bash -c <command> through os/exec. On Windows Go quotes
// the whole argument and Git Bash's runtime parses it again, which turns any
// doubled backslash into one, so a gate can reach bash changed. Every gate is
// passed the same way here and echoed back.
func TestGatesReachBashIntact(t *testing.T) {
	bash, ok := gateBash()
	if !ok {
		t.Skip("no bash that hrn would use")
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
// is set, as in a gate. A grep or fgrep call runs to the next pipe, list
// operator or closing parenthesis; git grep has its own engine and is left out.
var grepCall = regexp.MustCompile(`(^|[\s|;&(` + "`" + `])(git\s+)?(f?grep)(\s[^|;&)]*)`)

func grepIgnoreCaseFixed(gate string) (string, bool) {
	for _, m := range grepCall.FindAllStringSubmatch(gate, -1) {
		if m[2] != "" {
			continue
		}
		ignoreCase, fixed := false, m[3] == "fgrep"
		for _, word := range strings.Fields(m[4]) {
			switch {
			case word == "--ignore-case":
				ignoreCase = true
			case word == "--fixed-strings":
				fixed = true
			case len(word) > 1 && word[0] == '-' && word[1] != '-':
				ignoreCase = ignoreCase || strings.Contains(word[1:], "i")
				fixed = fixed || strings.Contains(word[1:], "F")
			}
		}
		if ignoreCase && fixed {
			return strings.TrimSpace(m[0]), true
		}
	}
	return "", false
}

func TestGatesAvoidGrepIgnoreCaseFixed(t *testing.T) {
	for where, gates := range suiteGates(t) {
		for _, gate := range gates {
			if m, bad := grepIgnoreCaseFixed(gate); bad {
				t.Errorf("%s: %q aborts without a UTF-8 locale on Git for Windows; drop -F (or -i):\n%s", where, m, gate)
			}
		}
	}
}

func TestGrepIgnoreCaseFixedPattern(t *testing.T) {
	for gate, want := range map[string]bool{
		"grep -qiF 'x' f":                   true,
		"grep -qFi 'x' f":                   true,
		"grep -i -F 'x' f":                  true,
		"grep -F -qi 'x' f":                 true,
		"grep -E -i -F x f":                 true,
		"grep -q -i -F x f":                 true,
		"grep -F -e x -i f":                 true,
		"grep --ignore-case -F x f":         true,
		"grep -i --fixed-strings x f":       true,
		"fgrep -i x f":                      true,
		"tr a b | grep -qiF 'x'":            true,
		"echo $(grep -iF x f)":              true,
		"grep -qi 'x' f":                    false,
		"grep -qF 'x' f":                    false,
		"grep -qiE 'x' f":                   false,
		"fgrep x f":                         false,
		"grep -qi x f | grep -qF y":         false,
		"git grep -iF x":                    false,
		"cat f | grep -q x; grep -F -- y f": false,
	} {
		if _, got := grepIgnoreCaseFixed(gate); got != want {
			t.Errorf("%q: flagged = %v, want %v", gate, got, want)
		}
	}
}
