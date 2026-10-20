package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runAllCase says how the sandbox for one run of run-all.sh differs from a
// clean checkout with every tool present.
type runAllCase struct {
	noCodex bool
	// hrnCodex says what the stub hrn backends prints for codex: "elsewhere"
	// a path outside PATH (as hrn finds codex's own install on Windows),
	// "both" claude off PATH too, both at paths outside it, "unknown" the
	// not-found line, "fail" nothing and exit 1.
	hrnCodex  string
	dirty     bool   // a tracked file changed after the commit
	untracked bool   // a new file left uncommitted
	hash      string // the hash tool on PATH: "sha256sum" (default), "shasum" or "none"
	failSuite string // the suite whose hrn eval exits 1
	failGit   string // the git subcommand that exits 128, as outside a repository
	// userGit gives the run a user-level git setup in place of the sandbox's
	// isolation, to show that run-all itself does not honour it:
	// "xdg-ignore" lists the untracked file in $XDG_CONFIG_HOME/git/ignore,
	// git's default global ignore; "excludes-file" lists it in a file that
	// ~/.gitconfig names as core.excludesFile; "hide-untracked" sets
	// status.showUntrackedFiles=no in ~/.gitconfig.
	userGit string
	// mark sets "assume-unchanged", "skip-worktree" or both (space-separated)
	// on fixtures/sample.txt (or markFile) after the commit, so that git
	// status no longer reports a change to it.
	mark     string
	markFile string // a committed file to mark instead of fixtures/sample.txt
	// cleared then clears the marks with the commands the refusal names.
	cleared bool
	// ignoredByRepo commits a .gitignore holding .hrn/ and leaves a file
	// under .hrn/, as an eval does.
	ignoredByRepo bool
	// apiSuite commits suites/local/api.json, whose variant runs the api backend.
	apiSuite bool
	// baseURL is ANTHROPIC_BASE_URL for the run; the sandbox drops the user's.
	baseURL string
	args    []string // arguments to run-all.sh
}

func (c runAllCase) markedFile() string {
	if c.markFile != "" {
		return c.markFile
	}
	return "fixtures/sample.txt"
}

func TestRunAll(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not on PATH")
	}
	// From PowerShell, WSL's bash comes first on PATH; it drops the sandbox's
	// environment (only WSLENV crosses) and cannot run Windows paths. The
	// script runs under Git Bash, so take the one installed beside git.
	uname, _ := exec.Command(bash, "--noprofile", "--norc", "-c", "uname -s").Output()
	if !keepBash(runtime.GOOS, string(uname)) {
		git, _ := exec.LookPath("git")
		if bash = gitBashFor(git); bash == "" {
			t.Skipf("the first bash on PATH is not Git Bash, and there is no Git for Windows bash beside git (%q); run-all.sh runs under Git Bash", git)
		}
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
			if up := strings.ToUpper(kv); !strings.HasPrefix(up, "GIT_") && !strings.HasPrefix(up, "ANTHROPIC_") {
				env = append(env, kv)
			}
		}
		return append(env, "BASH_ENV=", "ENV=", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "XDG_CONFIG_HOME="+t.TempDir())
	}
	run := func(t *testing.T, c runAllCase) (string, string, int, string) {
		t.Helper()
		root := t.TempDir()
		env := sandboxEnv(t)
		if c.userGit != "" {
			xdg, home := t.TempDir(), t.TempDir()
			gitconfig := filepath.Join(home, ".gitconfig")
			var ignore, config string
			switch c.userGit {
			case "xdg-ignore":
				ignore = filepath.Join(xdg, "git", "ignore")
			case "excludes-file":
				ignore = filepath.Join(home, "global-ignore")
				config = "[core]\n\texcludesFile = " + filepath.ToSlash(ignore) + "\n"
			case "hide-untracked":
				config = "[status]\n\tshowUntrackedFiles = no\n"
			default:
				t.Fatalf("unknown userGit %q", c.userGit)
			}
			if ignore != "" {
				if err := os.MkdirAll(filepath.Dir(ignore), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(ignore, []byte("new.txt\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(gitconfig, []byte(config), 0644); err != nil {
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
		if c.markFile != "" {
			write(c.markFile, "marked\n")
		}
		write("suites/golden/a.json", "{}\n")
		write("suites/golden/b.json", "{}\n")
		if c.apiSuite {
			write("suites/local/api.json", `{"matrix": [{"name": "local", "backend": "api", "model": "m"}]}`+"\n")
		}
		if c.baseURL != "" {
			env = append(env, "ANTHROPIC_BASE_URL="+c.baseURL)
		}
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
		// hrn backends prints its lines as hrn does, with codex at a native
		// path outside PATH (backslashes on Windows) for "elsewhere".
		backends := "true"
		switch c.hrnCodex {
		case "elsewhere":
			backends = "printf '%s\\n' " + quote("  ✗ api       disabled: set ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN)") + " " +
				quote("  ✓ claude    enabled (default): /nowhere/claude") + " " +
				quote("  ✓ codex     enabled: "+filepath.Join(root, "opt", "codex")) + " " +
				quote("              limit reached until 12:00")
			write("opt/codex", shell+"printf 'codex hrn-path version\\n'\n")
		case "both":
			// Neither backend on PATH: claude is the default, at a path with a
			// space on a CRLF line, and codex's --version fails.
			backends = "printf '%s\\r\\n' " + quote("  ✓ claude    enabled (default): "+filepath.Join(root, "opt", "my claude", "claude")) +
				"; printf '%s\\n' " + quote("  ✓ codex     disabled: "+filepath.Join(root, "opt", "codex"))
			write("opt/my claude/claude", shell+"printf 'claude hrn-path version\\n'\n")
			write("opt/codex", shell+"exit 3\n")
		case "unknown":
			backends = "printf '%s\\n' " + quote("  ✓ claude    enabled (default): /nowhere/claude") + " " +
				quote("  ✗ codex     enabled: codex not found on PATH or in known install locations")
		case "fail":
			backends = "exit 1"
		case "":
		default:
			t.Fatalf("unknown hrnCodex %q", c.hrnCodex)
		}
		write("bin/hrn", shell+`printf '%s\n' "$*" >> calls
if [ "$1" = version ]; then
 printf 'hrn v0.0.0 (commit stub123)\n'
elif [ "$1" = backends ]; then
 `+backends+`
elif [ "$1 $2" = `+quote("eval "+c.failSuite)+` ]; then
 printf 'stub eval failed\n'
 exit 1
else
 printf 'stub eval\n'
fi
`)
		if c.hrnCodex != "both" {
			write("bin/claude", shell+"printf 'claude stub version\\n'\n")
		}
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
		// One mark per update-index call, as git applies only one of them.
		for _, mark := range strings.Fields(c.mark) {
			git("update-index", "--"+mark, c.markedFile())
		}
		// Clear the marks with exactly the commands the refusal names.
		if c.cleared {
			git("update-index", "--no-assume-unchanged", c.markedFile())
			git("update-index", "--no-skip-worktree", c.markedFile())
		}
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
		args := ""
		for _, a := range c.args {
			args += " " + quote(a)
		}
		cmd := exec.Command(bash, "--noprofile", "--norc", "-c", "PATH=\"$PWD/bin\"; export PATH; exec "+quote(paths["bash"])+" scripts/run-all.sh"+args)
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
		for _, want := range []string{"pins:\n", "hrn version:\n", "hrn v0.0.0 (commit stub123)\n", "codex version:\n", "codex stub version\n", "suite commit: " + head + "\n", "\nmodel ids: a spec's variant asks for a model or for the backend's default (sonnet is an alias); in the batch record (~/.hrn/batches/<id>.json) tasks[].attempts[].result.model is the id the backend reported when result.model_reported is true (claude), otherwise the model asked for; codex reports none, so a codex variant without a model ran codex's configured default, which is not recorded\n", "fixture SHA-256 hashes:\n", fixtureHash, "=== suites/golden/a.json\n", "stub eval\n"} {
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
	// The api backend bills per token unless it points at a local server.
	t.Run("api suite left out of the default run", func(t *testing.T) {
		needHash(t)
		out, calls, code, _ := run(t, runAllCase{apiSuite: true, baseURL: "http://127.0.0.1:8081"})
		if code != 0 || !strings.Contains(out, "run-all: leaving out suites/local/api.json") {
			t.Fatalf("exit %d:\n%s", code, out)
		}
		if want := "version\neval suites/golden/a.json\neval suites/golden/b.json\neval list -n 2\n"; calls != want {
			t.Errorf("calls = %q, want %q", calls, want)
		}
	})
	for _, base := range []string{"", "https://api.anthropic.com", "http://127.0.0.1:8081@api.anthropic.com", "http://127.0.0.1.example.com:8081", "http://localhost:8081/../x"} {
		t.Run("named api suite refused with base "+base, func(t *testing.T) {
			out, calls, code, _ := run(t, runAllCase{apiSuite: true, baseURL: base, args: []string{"suites/local/api.json"}})
			if code != 1 || calls != "" || !strings.Contains(out, "run-all: suites/local/api.json runs the api backend, which bills per token") {
				t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
			}
		})
	}
	for _, base := range []string{"http://127.0.0.1:8081", "http://localhost:8081/", "http://[::1]:8081"} {
		t.Run("named api suite runs against "+base, func(t *testing.T) {
			needHash(t)
			out, calls, code, _ := run(t, runAllCase{apiSuite: true, baseURL: base, args: []string{"suites/local/api.json"}})
			if want := "version\neval suites/local/api.json\neval list -n 1\n"; code != 0 || calls != want {
				t.Fatalf("exit %d, calls %q, want %q:\n%s", code, calls, want, out)
			}
		})
	}
	// hrn runs codex from outside PATH; the pin is that binary's version.
	t.Run("codex only where hrn finds it", func(t *testing.T) {
		needHash(t)
		out, calls, code, _ := run(t, runAllCase{noCodex: true, hrnCodex: "elsewhere"})
		if code != 0 || strings.Contains(out, "(not installed)") || !strings.Contains(out, "claude version:\nclaude stub version\n") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
		if want := "version\nbackends\neval suites/golden/a.json\neval suites/golden/b.json\neval list -n 2\n"; calls != want {
			t.Errorf("calls = %q, want %q", calls, want)
		}
		if !strings.Contains(out, "codex version (hrn backends: ") || !strings.Contains(out, filepath.Join("opt", "codex")+"):\ncodex hrn-path version\n") {
			t.Fatalf("no codex pin from the path hrn prints:\n%s", out)
		}
	})
	// hrn backends is asked once for both; the matched line's state has a
	// space, its path a space and a CR, and a failing --version still pins.
	t.Run("both backends only where hrn finds them", func(t *testing.T) {
		needHash(t)
		out, calls, code, _ := run(t, runAllCase{noCodex: true, hrnCodex: "both"})
		if code != 0 || strings.Contains(out, "(not installed)") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
		if want := "version\nbackends\neval suites/golden/a.json\neval suites/golden/b.json\neval list -n 2\n"; calls != want {
			t.Errorf("calls = %q, want %q", calls, want)
		}
		if !strings.Contains(out, "claude version (hrn backends: ") || !strings.Contains(out, filepath.Join("opt", "my claude", "claude")+"):\nclaude hrn-path version\n") ||
			!strings.Contains(out, "codex: version unavailable\n") {
			t.Fatalf("missing claude pin or codex fallback:\n%s", out)
		}
	})
	t.Run("codex unknown to PATH and hrn", func(t *testing.T) {
		needHash(t)
		out, calls, code, _ := run(t, runAllCase{noCodex: true, hrnCodex: "unknown"})
		if code != 0 || !strings.Contains(out, "codex: (not installed)\n") || !strings.HasPrefix(calls, "version\nbackends\n") || !strings.HasSuffix(calls, "eval list -n 2\n") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	// A failing hrn backends must stop the run before any eval, not pin
	// "(not installed)" for a backend hrn would run.
	t.Run("hrn backends fails", func(t *testing.T) {
		out, calls, code, _ := run(t, runAllCase{noCodex: true, hrnCodex: "fail"})
		if code == 0 || calls != "version\nbackends\n" || strings.Contains(out, "codex: (not installed)") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	t.Run("codex absent", func(t *testing.T) {
		needHash(t)
		// hrn backends prints nothing for codex.
		out, calls, code, _ := run(t, runAllCase{noCodex: true})
		if code != 0 || !strings.Contains(out, "codex: (not installed)\n") || !strings.HasPrefix(calls, "version\nbackends\n") || !strings.HasSuffix(calls, "eval list -n 2\n") {
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
	// Real use: nothing in the user's own git setup hides an untracked file
	// from the dirty check, since the agents, branching from HEAD, would not
	// see it. One case per way a user's setup can hide it.
	for _, userGit := range []string{"xdg-ignore", "excludes-file", "hide-untracked"} {
		t.Run("untracked file under the user's "+userGit, func(t *testing.T) {
			out, calls, code, _ := run(t, runAllCase{untracked: true, userGit: userGit})
			if code != 1 || calls != "" || !strings.Contains(out, "run-all: commit or stash local changes first; the agents branch from HEAD") {
				t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
			}
		})
	}
	// A marked file hides its change from git status; the mark itself is
	// refused, changed or not, and the file is named.
	for _, c := range []struct {
		name string
		c    runAllCase
	}{
		{"changed file marked assume-unchanged", runAllCase{mark: "assume-unchanged", dirty: true}},
		{"changed file marked skip-worktree", runAllCase{mark: "skip-worktree", dirty: true}},
		{"unchanged file marked skip-worktree", runAllCase{mark: "skip-worktree"}},
		// Both marks show as the lowercase tag s.
		{"changed file with both marks", runAllCase{mark: "skip-worktree assume-unchanged", dirty: true}},
		// Named as typed, not as an octal escape.
		{"non-ASCII file name", runAllCase{mark: "skip-worktree", markFile: "fixtures/țară.txt"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			out, calls, code, _ := run(t, c.c)
			if code != 1 || calls != "" || !strings.Contains(out, "marked assume-unchanged or skip-worktree") || !strings.Contains(out, "  "+c.c.markedFile()+"\n") ||
				!strings.Contains(out, "  git update-index --no-assume-unchanged <file>\n  git update-index --no-skip-worktree <file>\n") {
				t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
			}
		})
	}
	t.Run("marks cleared with the commands the refusal names", func(t *testing.T) {
		needHash(t)
		out, calls, code, _ := run(t, runAllCase{mark: "skip-worktree assume-unchanged", cleared: true})
		if code != 0 || !strings.Contains(out, "pins:") || !strings.HasSuffix(calls, "eval list -n 2\n") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
		}
	})
	t.Run("a user's git setup does not hide the repository's own ignores", func(t *testing.T) {
		// .hrn/ and the other entries of the repository's .gitignore still
		// keep a clean tree clean: the run goes on to the pins and the evals.
		needHash(t)
		out, calls, code, _ := run(t, runAllCase{userGit: "excludes-file", ignoredByRepo: true})
		if code != 0 || !strings.Contains(out, "pins:") || !strings.HasSuffix(calls, "eval list -n 2\n") {
			t.Fatalf("exit %d, calls %q:\n%s", code, calls, out)
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
	t.Run("git ls-files fails", func(t *testing.T) {
		out, calls, code, _ := run(t, runAllCase{failGit: "ls-files"})
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

// keepBash says whether the first bash on PATH can run the tests: any bash
// off Windows, and on Windows only Git Bash (uname -s MINGW* or MSYS*), not
// WSL's, whose uname says Linux, or prints nothing when no distro is set up.
func keepBash(goos, unameS string) bool {
	if goos != "windows" {
		return true
	}
	s := strings.TrimSpace(unameS)
	return strings.HasPrefix(s, "MINGW") || strings.HasPrefix(s, "MSYS")
}

// gitBashFor returns Git for Windows' wrapper bash (<root>\bin\bash.exe, which
// puts the MSYS tools on PATH) for a git.exe at <root>\cmd, <root>\bin or
// <root>\mingw64\bin, or "" when there is none.
func gitBashFor(git string) string {
	if git == "" {
		return ""
	}
	dir := filepath.Dir(git)
	for range 3 {
		if p := filepath.Join(dir, "bin", "bash.exe"); fileExists(p) {
			return p
		}
		dir = filepath.Dir(dir)
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestKeepBash(t *testing.T) {
	for _, c := range []struct {
		goos, uname string
		want        bool
	}{
		{"linux", "Linux\n", true}, // a Linux host, WSL's included
		{"darwin", "Darwin\n", true},
		{"windows", "MINGW64_NT-10.0-26200\n", true},
		{"windows", "MSYS_NT-10.0-26200\n", true},
		{"windows", "Linux\n", false}, // WSL's bash from Windows
		{"windows", "", false},        // WSL's bash with no distro
	} {
		if got := keepBash(c.goos, c.uname); got != c.want {
			t.Errorf("keepBash(%q, %q) = %v, want %v", c.goos, c.uname, got, c.want)
		}
	}
}

func TestGitBashFor(t *testing.T) {
	touch := func(t *testing.T, path string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		name, git string
		bashes    []string // bash.exe files to create, relative to the root
		want      string   // relative to the root, "" for none
	}{
		{"git in cmd", "cmd/git.exe", []string{"bin/bash.exe"}, "bin/bash.exe"},
		{"git in bin", "bin/git.exe", []string{"bin/bash.exe"}, "bin/bash.exe"},
		{"git in mingw64/bin", "mingw64/bin/git.exe", []string{"bin/bash.exe"}, "bin/bash.exe"},
		{"only the MSYS bash", "cmd/git.exe", []string{"usr/bin/bash.exe"}, ""},
		{"scoop shim", "scoop/shims/git.exe", nil, ""},
		{"no bash near git", "cmd/git.exe", nil, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			git := filepath.Join(root, filepath.FromSlash(c.git))
			touch(t, git)
			for _, b := range c.bashes {
				touch(t, filepath.Join(root, filepath.FromSlash(b)))
			}
			want := ""
			if c.want != "" {
				want = filepath.Join(root, filepath.FromSlash(c.want))
			}
			if got := gitBashFor(git); got != want {
				t.Errorf("gitBashFor(%q) = %q, want %q", git, got, want)
			}
		})
	}
	if got := gitBashFor(""); got != "" {
		t.Errorf(`gitBashFor("") = %q, want ""`, got)
	}
}
