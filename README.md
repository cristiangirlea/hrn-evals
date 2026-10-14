# hrn-evals

Eval suites for [hrn](https://github.com/cristiangirlea/harness), kept as data apart from
the engine. Each suite is an `hrn eval` spec: a task set, a matrix of agents to compare, and
the gates that decide what passing means. Running the same suites again after a change (a
new model, a reworded prompt, a harness release) and diffing the two runs is how a
regression gets caught instead of noticed later.

## Layout

| Path | What it holds |
|---|---|
| `suites/golden/` | Unambiguous tasks with a known right answer. A failure here is a regression. `gobench.json` holds four Go tasks (memoized Fibonacci, a REST handler, a unit test, a short explanation), taken from the prompts of the 2025 `llm-agent-benchmark` and given gates. |
| `suites/regression/` | Tasks that once went wrong, kept so they stay fixed. |
| `suites/adversarial/` | Tasks with a trap in the input, such as instructions hidden in a file the agent reads. The gates check that the trap was not followed. |
| `suites/routing/` | The tasks the other suites prove, run across backends, models and efforts so `hrn route from-eval` can build the routing table. `codex-first-table.json` is the first, codex only (`results/2026-10-14-routing.md`). |
| `fixtures/` | The files the suites work on. They are committed, so every run starts from the same state. |
| `scripts/run-all.sh` | Runs every suite, or the ones named, then lists the saved evals. |
| `results/` | The results pages. Every page from now on meets [the publication gate](docs/publication-gate.md), the rule for when a saving or quality claim may be made. |
| `results/where-it-loses.md` | The standing page of where each backend and model loses, regenerated from the saved evals by `scripts/where-it-loses.sh`; failures stay when a later run passes, with a note. |

This repository is its own sandbox: each task runs in a git worktree of it on a branch of
its own, and the worktrees are removed when the eval ends. `fixtures/buggy` has a failing
test on purpose; `go test ./...` in this repository fails until an agent fixes it in a
worktree.

## Running

```bash
scripts/run-all.sh                                   # every suite
scripts/run-all.sh suites/adversarial/*.json         # one kind
hrn eval diff <earlier-id> <later-id>                # what changed between two runs
```

Records go to `~/.hrn/evals`, as for any `hrn eval`. The matrix in each suite compares
Claude Code (Sonnet) with Codex, the two agents in daily use; add a variant to compare
another backend or model.

## Adding a suite

1. Put what the task needs under `fixtures/<name>/` and commit it. A check the agent should
   not write to its own taste goes in as `*.go.txt` (see `fixtures/gobench/checks/`), which
   the gate copies into place after the agent finishes and which a gate verifies was not
   changed. The agent can still read it; the gate only stops it from editing it. Write such a
   test as an external `_test` package, so its names cannot collide with the agent's code.
2. Write `suites/<kind>/<name>.json`. Keep `"dir": "../.."` so the agents and the gates run
   at the repository root, and end every prompt with "Do not run git": the harness commits
   whatever the agent leaves.
3. Make every gate print why it failed (`|| { echo '...'; exit 1; }`); that text is what a
   diff and a retry show.
4. Check every gate both ways: it must pass a correct solution and fail a plausible wrong
   one. A gate that cannot fail measures nothing. For a task that asks for tests, mutation
   testing works well: break the code under test in two different ways and require the
   agent's tests to fail on both (`gobench`'s `go-unit-test-add`). Gates run in Git Bash on
   Windows, where there is no gcc, so `go test -race` is not available to them.
5. Run it once on each variant before committing it, and write down the result in the PR.

## Scheduling

Not on GitHub Actions: the agents need a logged-in Claude Code and Codex (or an API key),
which a hosted runner does not have. The plan is a nightly job on the local cluster from the
`infra` repository, with its results compared against the previous night's by
`hrn eval diff`.

Never put credentials, private data or client code into a suite or a fixture.
