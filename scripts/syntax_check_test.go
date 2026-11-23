package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// bash -n reads one script and hands any further path to it as an argument,
// so a call that lists several checks the first alone. CI, the check skill
// and AGENTS.md each state the check and must give every script its own call.
func TestEveryScriptHasItsOwnSyntaxCheck(t *testing.T) {
	scripts, err := filepath.Glob("*.sh")
	if err != nil || len(scripts) == 0 {
		t.Fatalf("no shell scripts found: %v", err)
	}
	call := regexp.MustCompile(`(?m)^[ \t]*(?:- run:[ \t]*)?bash -n[ \t]+(\S+)(.*)$`)
	for _, f := range []string{"../.github/workflows/ci.yml", "../.claude/skills/check/SKILL.md", "../AGENTS.md"} {
		b, err := os.ReadFile(filepath.FromSlash(f))
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimPrefix(f, "../")
		checked := map[string]bool{}
		for _, m := range call.FindAllStringSubmatch(string(b), -1) {
			if rest := strings.TrimSpace(m[2]); rest != "" {
				t.Errorf("%s: bash -n checks %s only; %q reaches it as arguments and is never read", name, m[1], rest)
			}
			checked[m[1]] = true
		}
		for _, s := range scripts {
			if !checked["scripts/"+s] {
				t.Errorf("%s: no bash -n call of its own for scripts/%s", name, s)
			}
		}
	}
}
