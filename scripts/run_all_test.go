package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAll(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not on PATH")
	}
	script, err := os.ReadFile("run-all.sh")
	if err != nil {
		t.Fatal(err)
	}
	// Resolve only the tools allowed in the sandbox; never inherit a real hrn.
	paths := map[string]string{}
	for _, tool := range []string{"bash", "git", "dirname", "find", "sort", "sha256sum"} {
		out, err := exec.Command(bash, "--noprofile", "--norc", "-c", "type -P "+tool).CombinedOutput()
		if err != nil {
			t.Fatalf("find %s: %v: %s", tool, err, out)
		}
		paths[tool] = strings.TrimSpace(string(out))
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	run := func(t *testing.T, noCodex, dirty, noHash bool) (string, string, int, string) {
		t.Helper()
		root := t.TempDir()
		write := func(name, content string) {
			t.Helper()
			path := filepath.Join(root, name)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0755); err != nil {
				t.Fatal(err)
			}
		}
		write("scripts/run-all.sh", string(script))
		write("fixtures/sample.txt", "fixture\n")
		write("suites/golden/a.json", "{}\n")
		write("suites/golden/b.json", "{}\n")
		shell := "#!" + paths["bash"] + "\n"
		for _, tool := range []string{"git", "dirname", "find", "sort", "sha256sum"} {
			if tool == "sha256sum" && noHash {
				continue
			}
			write("bin/"+tool, shell+"exec "+quote(paths[tool])+" \"$@\"\n")
		}
		write("bin/hrn", shell+`printf '%s\n' "$*" >> calls
if [ "$1" = version ]; then
 printf 'hrn v0.0.0 (commit stub123)\n'
else
 printf 'stub eval\n'
fi
`)
		write("bin/claude", shell+"printf 'claude stub version\\n'\n")
		if !noCodex {
			write("bin/codex", shell+"printf 'codex stub version\\n'\n")
		}
		git := func(args ...string) string {
			t.Helper()
			cmd := exec.Command("git", append([]string{"-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false"}, args...)...)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("git %v: %v: %s", args, err, out)
			}
			return strings.TrimSpace(string(out))
		}
		git("init", "--quiet")
		git("add", ".")
		git("-c", "user.name=Smoke test", "-c", "user.email=smoke@example.invalid", "commit", "--quiet", "-m", "fixture")
		head := git("rev-parse", "HEAD")
		if dirty {
			write("fixtures/sample.txt", "dirty\n")
		}
		// Set PATH inside bash to avoid Windows PATH conversion and exclude host CLIs.
		cmd := exec.Command(bash, "--noprofile", "--norc", "-c", "PATH=\"$PWD/bin\"; export PATH; exec "+quote(paths["bash"])+" scripts/run-all.sh")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "BASH_ENV=", "ENV=")
		out, err := cmd.CombinedOutput()
		code := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		calls, err := os.ReadFile(filepath.Join(root, "calls"))
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		return string(out), string(calls), code, head
	}
	var normalCode int
	t.Run("pins before evals", func(t *testing.T) {
		out, calls, code, head := run(t, false, false, false)
		normalCode = code
		if code != 0 {
			t.Fatalf("exit %d:\n%s", code, out)
		}
		previous := -1
		for _, want := range []string{"pins:\n", "hrn version:\n", "hrn v0.0.0 (commit stub123)\n", "codex version:\n", "codex stub version\n", "suite commit: " + head + "\n", "fixture SHA-256 hashes:\n", fmt.Sprintf("%x  fixtures/sample.txt\n", sha256.Sum256([]byte("fixture\n"))), "=== suites/golden/a.json\n", "stub eval\n"} {
			pos := strings.Index(out, want)
			if pos <= previous {
				t.Fatalf("missing or out of order %q:\n%s", want, out)
			}
			previous = pos
		}
		if want := "version\neval suites/golden/a.json\neval suites/golden/b.json\neval list -n 2\n"; calls != want {
			t.Errorf("calls = %q, want %q", calls, want)
		}
	})
	t.Run("codex absent", func(t *testing.T) {
		out, calls, code, _ := run(t, true, false, false)
		if code != normalCode || !strings.Contains(out, "codex: (not installed)\n") || !strings.HasSuffix(calls, "eval list -n 2\n") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	t.Run("dirty tree", func(t *testing.T) {
		out, calls, code, _ := run(t, false, true, false)
		if code != 1 || strings.Contains(out, "pins:") || calls != "" || !strings.Contains(out, "run-all: commit or stash local changes first; the agents branch from HEAD") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	t.Run("hash tools absent", func(t *testing.T) {
		out, calls, code, _ := run(t, false, false, true)
		if code != 1 || !strings.Contains(out, "run-all: sha256sum or shasum is required for fixture pins") || calls != "version\n" {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
}
