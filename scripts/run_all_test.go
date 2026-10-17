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

// runAllCase says how the sandbox for one run of run-all.sh differs from a
// clean checkout with every tool present.
type runAllCase struct {
	noCodex   bool
	dirty     bool   // a tracked file changed after the commit
	untracked bool   // a new file left uncommitted
	hash      string // the hash tool on PATH: "sha256sum" (default), "shasum" or "none"
	failSuite string // the suite whose hrn eval exits 1
	failGit   string // the git subcommand that exits 128, as outside a repository
	// globalIgnore gives the run a user-level git setup that ignores the
	// untracked file both ways (core.excludesFile in ~/.gitconfig and
	// $XDG_CONFIG_HOME/git/ignore) instead of the sandbox's isolation, to show
	// that run-all itself does not honour it.
	globalIgnore bool
	// ignoredByRepo commits a .gitignore holding .hrn/ and leaves a file
	// under .hrn/, as an eval does.
	ignoredByRepo bool
}

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
	lookup := func(tool string) string {
		out, err := exec.Command(bash, "--noprofile", "--norc", "-c", "type -P "+tool).CombinedOutput()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	paths := map[string]string{}
	for _, tool := range []string{"bash", "git", "dirname", "find", "sort"} {
		if paths[tool] = lookup(tool); paths[tool] == "" {
			t.Fatalf("%s is not on bash's PATH", tool)
		}
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	// The real SHA-256 tool behind both stubs: sha256sum where it exists (Git
	// Bash, Linux), shasum -a 256 where only that does (macOS).
	realHash := ""
	if p := lookup("sha256sum"); p != "" {
		realHash = quote(p)
	} else if p := lookup("shasum"); p != "" {
		realHash = quote(p) + " -a 256"
	}
	// Only the cases whose run reaches the fixture hashes need a real tool;
	// the others stop before it, so they run on any host.
	needHash := func(t *testing.T) {
		t.Helper()
		if realHash == "" {
			t.Skip("neither sha256sum nor shasum is on PATH")
		}
	}
	// Nothing of the user's git setup may decide what counts as dirty in the
	// sandbox. Every inherited GIT_ variable goes (GIT_DIR and GIT_WORK_TREE
	// from a hook would point git at another repository; GIT_CONFIG_COUNT,
	// GIT_CONFIG_PARAMETERS and the like would add settings). GIT_CONFIG_GLOBAL
	// stops ~/.gitconfig and $XDG_CONFIG_HOME/git/config, GIT_CONFIG_NOSYSTEM
	// the system config, and XDG_CONFIG_HOME pointing at an empty directory
	// the default ignore file (an empty value would fall back to
	// ~/.config/git/ignore). Read at each run, so a test can set the user's
	// side first.
	sandboxEnv := func(t *testing.T) []string {
		var env []string
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(strings.ToUpper(kv), "GIT_") {
				env = append(env, kv)
			}
		}
		return append(env, "BASH_ENV=", "ENV=", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "XDG_CONFIG_HOME="+t.TempDir())
	}
	run := func(t *testing.T, c runAllCase) (string, string, int, string) {
		t.Helper()
		root := t.TempDir()
		env := sandboxEnv(t)
		if c.globalIgnore {
			xdg, home := t.TempDir(), t.TempDir()
			ignore := filepath.Join(xdg, "git", "ignore")
			if err := os.MkdirAll(filepath.Dir(ignore), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(ignore, []byte("new.txt\n"), 0644); err != nil {
				t.Fatal(err)
			}
			gitconfig := filepath.Join(home, ".gitconfig")
			if err := os.WriteFile(gitconfig, []byte("[core]\n\texcludesFile = "+filepath.ToSlash(ignore)+"\n"), 0644); err != nil {
				t.Fatal(err)
			}
			var kept []string
			for _, kv := range env {
				if !strings.HasPrefix(kv, "GIT_CONFIG_GLOBAL=") && !strings.HasPrefix(kv, "XDG_CONFIG_HOME=") {
					kept = append(kept, kv)
				}
			}
			env = append(kept, "GIT_CONFIG_GLOBAL="+filepath.ToSlash(gitconfig), "XDG_CONFIG_HOME="+xdg)
		}
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
		if c.ignoredByRepo {
			write(".gitignore", ".hrn/\n")
		}
		shell := "#!" + paths["bash"] + "\n"
		for _, tool := range []string{"dirname", "find", "sort"} {
			write("bin/"+tool, shell+"exec "+quote(paths[tool])+" \"$@\"\n")
		}
		// The subcommand is the first argument after any -c options.
		write("bin/git", shell+`sub=$1
if [ "$sub" = -c ]; then sub=$3; fi
if [ "$sub" = `+quote(c.failGit)+` ]; then
 echo "fatal: not a git repository" >&2
 exit 128
fi
exec `+quote(paths["git"])+` "$@"
`)
		switch c.hash {
		case "", "sha256sum":
			if realHash != "" {
				write("bin/sha256sum", shell+"exec "+realHash+" \"$@\"\n")
			}
		case "shasum":
			// Refuses anything but -a 256, so a passing run shows the script
			// asked for SHA-256.
			write("bin/shasum", shell+`if [ "$1" = -a ] && [ "$2" = 256 ]; then shift 2; else exit 2; fi
exec `+realHash+` "$@"
`)
		}
		write("bin/hrn", shell+`printf '%s\n' "$*" >> calls
if [ "$1" = version ]; then
 printf 'hrn v0.0.0 (commit stub123)\n'
elif [ "$1 $2" = `+quote("eval "+c.failSuite)+` ]; then
 printf 'stub eval failed\n'
 exit 1
else
 printf 'stub eval\n'
fi
`)
		write("bin/claude", shell+"printf 'claude stub version\\n'\n")
		if !c.noCodex {
			write("bin/codex", shell+"printf 'codex stub version\\n'\n")
		}
		git := func(args ...string) string {
			t.Helper()
			cmd := exec.Command("git", append([]string{"-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false"}, args...)...)
			cmd.Dir = root
			cmd.Env = env
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
		if c.dirty {
			write("fixtures/sample.txt", "dirty\n")
		}
		if c.ignoredByRepo {
			write(".hrn/worktrees/x/file.txt", "eval leftover\n")
		}
		if c.untracked {
			write("fixtures/new.txt", "new\n")
		}
		// Set PATH inside bash to avoid Windows PATH conversion and exclude host CLIs.
		cmd := exec.Command(bash, "--noprofile", "--norc", "-c", "PATH=\"$PWD/bin\"; export PATH; exec "+quote(paths["bash"])+" scripts/run-all.sh")
		cmd.Dir = root
		cmd.Env = env
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
	fixtureHash := fmt.Sprintf("%x  fixtures/sample.txt\n", sha256.Sum256([]byte("fixture\n")))
	t.Run("pins before evals", func(t *testing.T) {
		needHash(t)
		out, calls, code, head := run(t, runAllCase{})
		if code != 0 {
			t.Fatalf("exit %d:\n%s", code, out)
		}
		previous := -1
		for _, want := range []string{"pins:\n", "hrn version:\n", "hrn v0.0.0 (commit stub123)\n", "codex version:\n", "codex stub version\n", "suite commit: " + head + "\n", "fixture SHA-256 hashes:\n", fixtureHash, "=== suites/golden/a.json\n", "stub eval\n"} {
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
		needHash(t)
		out, calls, code, _ := run(t, runAllCase{noCodex: true})
		if code != 0 || !strings.Contains(out, "codex: (not installed)\n") || !strings.HasSuffix(calls, "eval list -n 2\n") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	for _, c := range []struct {
		name string
		c    runAllCase
	}{
		{"dirty tree", runAllCase{dirty: true}},
		{"untracked file", runAllCase{untracked: true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			out, calls, code, _ := run(t, c.c)
			if code != 1 || strings.Contains(out, "pins:") || calls != "" || !strings.Contains(out, "run-all: commit or stash local changes first; the agents branch from HEAD") {
				t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
			}
		})
	}
	t.Run("untracked file under a global ignore", func(t *testing.T) {
		// The user's side ignores the untracked file both ways git looks
		// for a global ignore; the sandbox must still see it.
		xdg, home := t.TempDir(), t.TempDir()
		ignore := filepath.Join(xdg, "git", "ignore")
		if err := os.MkdirAll(filepath.Dir(ignore), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ignore, []byte("new.txt\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[core]\n\texcludesFile = "+filepath.ToSlash(ignore)+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("XDG_CONFIG_HOME", xdg)
		t.Setenv("HOME", home)
		out, calls, code, _ := run(t, runAllCase{untracked: true})
		if code != 1 || calls != "" || !strings.Contains(out, "run-all: commit or stash local changes first; the agents branch from HEAD") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	t.Run("untracked file the user's global ignore matches", func(t *testing.T) {
		// Real use: the user's global excludes do not hide a file from the
		// dirty check, since the agents, branching from HEAD, would not see it.
		out, calls, code, _ := run(t, runAllCase{untracked: true, globalIgnore: true})
		if code != 1 || calls != "" || !strings.Contains(out, "run-all: commit or stash local changes first; the agents branch from HEAD") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	t.Run("a global ignore does not hide the repository's own ignores", func(t *testing.T) {
		// .hrn/ and the other entries of the repository's .gitignore still
		// keep a clean tree clean.
		out, _, code, _ := run(t, runAllCase{globalIgnore: true, ignoredByRepo: true})
		if strings.Contains(out, "commit or stash local changes first") {
			t.Fatalf("exit %d: an ignored file counted as dirty:\n%s", code, out)
		}
	})
	t.Run("untracked file under git variables from a hook", func(t *testing.T) {
		// A git hook running go test would hand over GIT_DIR and friends;
		// GIT_CONFIG_COUNT can add a core.excludesFile that hides the file.
		outer, ignore := t.TempDir(), filepath.Join(t.TempDir(), "ignore")
		if err := os.WriteFile(ignore, []byte("new.txt\n"), 0644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GIT_DIR", filepath.Join(outer, ".git"))
		t.Setenv("GIT_WORK_TREE", outer)
		t.Setenv("GIT_CONFIG_COUNT", "1")
		t.Setenv("GIT_CONFIG_KEY_0", "core.excludesFile")
		t.Setenv("GIT_CONFIG_VALUE_0", filepath.ToSlash(ignore))
		out, calls, code, _ := run(t, runAllCase{untracked: true})
		if code != 1 || calls != "" || !strings.Contains(out, "run-all: commit or stash local changes first; the agents branch from HEAD") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	t.Run("shasum fallback", func(t *testing.T) {
		needHash(t)
		out, calls, code, _ := run(t, runAllCase{hash: "shasum"})
		if code != 0 || !strings.Contains(out, fixtureHash) || !strings.HasSuffix(calls, "eval list -n 2\n") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	t.Run("hash tools absent", func(t *testing.T) {
		out, calls, code, _ := run(t, runAllCase{hash: "none"})
		if code != 1 || !strings.Contains(out, "run-all: sha256sum or shasum is required for fixture pins") || calls != "version\n" {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	// git failing must stop the run, not read as a clean tree or an empty
	// suite commit; no eval may start.
	t.Run("git status fails", func(t *testing.T) {
		out, calls, code, _ := run(t, runAllCase{failGit: "status"})
		if code == 0 || calls != "" || strings.Contains(out, "pins:") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	t.Run("git rev-parse fails", func(t *testing.T) {
		out, calls, code, _ := run(t, runAllCase{failGit: "rev-parse"})
		if code == 0 || calls != "version\n" || strings.Contains(out, "suite commit:") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	t.Run("a failing eval", func(t *testing.T) {
		needHash(t)
		// The other suites still run, and the list comes last, so the failed
		// run's id is there to read; the script exits 1.
		out, calls, code, _ := run(t, runAllCase{failSuite: "suites/golden/a.json"})
		if want := "version\neval suites/golden/a.json\neval suites/golden/b.json\neval list -n 2\n"; code != 1 || calls != want {
			t.Fatalf("exit %d, calls %q, want exit 1 and %q:\n%s", code, calls, want, out)
		}
	})
}
