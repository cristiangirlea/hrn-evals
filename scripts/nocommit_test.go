package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// noCommitGate fails a cell whose HEAD moved, which only git does: hrn runs a
// cell's gates before it commits what the agent left, and the worktree's HEAD
// reflog starts at the base. A commit would also blind every gate that checks
// files with git status or git diff against HEAD, so it runs first. It needs
// fresh worktrees: a reused slot's reflog holds earlier tasks' commits.
const noCommitGate = `b=$(git reflog --format=%H HEAD | tail -n 1); [ -n "$b" ] || { echo 'HEAD has no reflog, so a commit by the agent cannot be ruled out'; exit 1; }; if git reflog --format=%H HEAD | grep -qv "$b"; then echo 'the agent moved HEAD with git (a commit, reset or checkout) though the prompt says not to run git; latest moves:'; git reflog --format='%h %gs' HEAD | head -n 5; exit 1; fi; [ "$(git rev-parse -q --verify HEAD)" = "$b" ] || { echo 'HEAD is not at the commit the cell started from though its reflog shows no move: the agent rewrote a ref or the reflog'; exit 1; }`

// Task gates replace the suite's, so the gate starts the list hrn will use.
func TestEveryTaskStartsWithTheNoCommitGate(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "suites", "*", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no suites found: %v", err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var spec struct {
			Worktrees string   `json:"worktrees"`
			Gate      []string `json:"gate"`
			Tasks     []struct {
				ID   string   `json:"id"`
				Gate []string `json:"gate"`
			} `json:"tasks"`
		}
		if err := json.Unmarshal(b, &spec); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if spec.Worktrees != "" && spec.Worktrees != "fresh" {
			t.Errorf("%s: worktrees %q: the no-commit gate needs fresh worktrees", filepath.ToSlash(f), spec.Worktrees)
		}
		for _, task := range spec.Tasks {
			gates := effectiveGates(task.Gate, spec.Gate)
			if len(gates) == 0 || gates[0] != noCommitGate {
				t.Errorf("%s: %s: the first gate is not the no-commit gate in scripts/nocommit_test.go", filepath.ToSlash(f), task.ID)
			}
			for i, g := range gates[min(1, len(gates)):] {
				if g == noCommitGate {
					t.Errorf("%s: %s: the no-commit gate appears again at %d", filepath.ToSlash(f), task.ID, i+1)
				}
			}
		}
	}
}

// effectiveGates picks the list hrn runs: a task's own, even an empty one, else the suite's.
func effectiveGates(task, suite []string) []string {
	if task != nil {
		return task
	}
	return suite
}

func TestAnEmptyTaskGateListReplacesTheSuites(t *testing.T) {
	var spec struct {
		Gate  []string `json:"gate"`
		Tasks []struct {
			Gate []string `json:"gate"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(`{"gate": ["suite"], "tasks": [{"gate": []}, {}]}`), &spec); err != nil {
		t.Fatal(err)
	}
	if got := effectiveGates(spec.Tasks[0].Gate, spec.Gate); len(got) != 0 {
		t.Errorf("a task with \"gate\": [] runs %q, want no gates as hrn runs it", got)
	}
	if got := effectiveGates(spec.Tasks[1].Gate, spec.Gate); len(got) != 1 || got[0] != "suite" {
		t.Errorf("a task without gates runs %q, want the suite's", got)
	}
}

// The gate is checked both ways on a cell built as hrn builds one: a branch
// at the base, a worktree on it, the agent's work, then the gate through
// bash -c from Go, before anything commits.
func TestNoCommitGateBothWays(t *testing.T) {
	bash, ok := gateBash()
	if !ok {
		t.Skip("no bash that hrn would use")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	cases := []struct {
		name     string
		noReflog bool     // the repository writes no reflogs
		agent    []string // bash run in the worktree as the agent
		fail     string   // "" when the gate must pass, else a part of its message
	}{
		{name: "nothing changed"},
		{name: "changes left uncommitted", agent: []string{"echo changed > a.txt && echo new > b.txt"}},
		{name: "changes staged but not committed", agent: []string{"echo changed > a.txt && git add a.txt"}},
		{name: "read-only git", agent: []string{"git status --porcelain && git log --oneline -1 && git diff"}},
		{name: "a new branch, nothing committed", agent: []string{"git checkout -q -b other"}},
		{name: "a commit on the task's branch", agent: []string{"echo changed > a.txt && git commit -qam work"}, fail: "moved HEAD"},
		{name: "a commit then a reset back", agent: []string{"echo changed > a.txt && git commit -qam work && git reset -q --hard HEAD~1"}, fail: "moved HEAD"},
		{name: "a commit on a detached HEAD", agent: []string{"git checkout -q --detach && echo changed > a.txt && git commit -qam work"}, fail: "moved HEAD"},
		{name: "an amended base", agent: []string{"git commit -q --amend -m reworded"}, fail: "moved HEAD"},
		{name: "a commit on a new branch", agent: []string{"git checkout -q -b other && echo changed > a.txt && git commit -qam work"}, fail: "moved HEAD"},
		{name: "a commit, then its reflog entry deleted", agent: []string{"echo changed > a.txt && git commit -qam work && git reflog delete HEAD@{0}"}, fail: "not at the commit the cell started from"},
		{name: "a branch rewritten without a reflog entry", agent: []string{`c=$(git commit-tree -p HEAD -m sneaky "HEAD^{tree}") && printf '%s\n' "$c" > "$(git rev-parse --git-common-dir)/refs/heads/hrn/cell"`}, fail: "not at the commit the cell started from"},
		{name: "no reflogs, nothing committed", noReflog: true, fail: "cannot be ruled out"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wt := hrnCell(t, c.noReflog)
			env := isolatedGit(t)
			for _, step := range c.agent {
				cmd := exec.Command(bash, "-c", step)
				cmd.Dir, cmd.Env = wt, env
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("agent step %q: %v\n%s", step, err, out)
				}
			}
			cmd := exec.Command(bash, "-c", noCommitGate)
			cmd.Dir, cmd.Env = wt, env
			out, err := cmd.CombinedOutput()
			switch {
			case c.fail == "" && err != nil:
				t.Errorf("the gate failed a cell that committed nothing: %v\n%s", err, out)
			case c.fail != "" && err == nil:
				t.Errorf("the gate passed a cell that moved HEAD:\n%s", out)
			case c.fail != "" && !strings.Contains(string(out), c.fail):
				t.Errorf("the gate failed without saying why: want %q in\n%s", c.fail, out)
			}
		})
	}
}

// hrnCell makes a repository with one commit and a worktree on a branch made
// at that commit, as hrn's addWorktree does, and returns the worktree.
func hrnCell(t *testing.T, noReflog bool) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo")
	wt := filepath.Join(filepath.Dir(root), "cell")
	env := isolatedGit(t)
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	git(root, "init", "-q")
	if noReflog {
		git(root, "config", "core.logAllRefUpdates", "false")
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(root, "add", "a.txt")
	git(root, "commit", "-qm", "base")
	base := git(root, "rev-parse", "HEAD")
	git(root, "branch", "--", "hrn/cell", base)
	git(root, "worktree", "add", "-q", "--", wt, "hrn/cell")
	return wt
}

// isolatedGit keeps the user's and the system's git setup out of the test.
func isolatedGit(t *testing.T) []string {
	t.Helper()
	home := t.TempDir()
	var env []string
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); !strings.HasPrefix(strings.ToUpper(k), "GIT_") {
			env = append(env, kv)
		}
	}
	return append(env, "HOME="+home, "XDG_CONFIG_HOME="+home, "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=agent", "GIT_AUTHOR_EMAIL=agent@example.invalid",
		"GIT_COMMITTER_NAME=agent", "GIT_COMMITTER_EMAIL=agent@example.invalid")
}
