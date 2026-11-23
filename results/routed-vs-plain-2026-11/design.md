# A routed hrn batch versus one plain session: design

Question: given the twenty tasks of `suites/routing/real-work.json`, does a routed `hrn batch`
(one run per task, gates, a retry on another model family) finish more of them than one plain
Claude Code session that is handed all twenty at once? Measured: gates passed out of twenty,
waiting time, tokens, and hrn's reading of each plan's week before and after each run.

This is the comparison the 2026-11-07 page (`../hrn-vs-direct-2026-11/`) left open: that one
measured a single `hrn run` against the same CLI; this one measures the orchestration.

## Arms

| | routed batch | plain session |
|---|---|---|
| unit | 20 tasks, each its own run in its own fresh worktree | 1 session, 1 worktree, all 20 tasks in one prompt |
| backend and model | by purpose, from the routing table in force (`route.txt`): codex `gpt-6-luna` low for implement, refactor, research and test; codex `gpt-6.1-sol` medium for ops and review; claude `sonnet` medium for debug and plan | `claude -p`, the account's default model |
| purpose | declared on each task, from its id's prefix | none |
| limits per attempt | `mode: yolo`, `max_turns: 25`, `timeout: 15m`, as the suite sets them | 120 minutes for the whole session, no turn limit |
| after a failed gate | retried on the other family (codex to claude, claude to codex), then on agy, told what failed | nothing: the session ends when it says it is done |
| parallel | 3 tasks at a time | one session |
| what it loads | hrn's lean start (no MCP servers, no skills) | the account's whole Claude Code setup |
| permissions | hrn's `yolo` with its command guard | `--permission-mode acceptEdits` and an allowlist: file edits, `go test`, `go build`, `go vet`, `gofmt`, `wc`, `ls`, `cat`, `grep` |

The plain arm does not bypass permissions: the tool the runs were started from is not allowed to
start an agent that does. The tasks need only file edits and `go`, so the allowlist covers them;
any command the plain session was refused is counted per run.

Knowledge retrieval is off in both arms. hrn refuses a task that has retrieval on and a retry on
another backend, and no knowledge project exists for this repository.

## Tasks and gates

The suite's twenty tasks, prompts and gates, unchanged, from hrn-evals `f637e30` in a scratch
clone. The plain session's prompt is one opening line ("There are 20 separate tasks below, all
in this repository. Do every one of them; each stands on its own.") followed by the twenty
prompts word for word, each under `Task n (id):`.

Two tasks change the same file (`fixtures/real/events/client.go`). In the batch each works on
its own copy; in the plain session both changes land in one file. That is part of what is
being compared.

Scoring:

- routed batch: hrn runs each task's gates in the task's worktree, as in any batch.
- plain session: every file the session changed is copied, byte for byte, onto a fresh worktree
  at the base, once per task, and that task's gates run there in order with `bash -c`, stopping
  at the first failure. One task's gate therefore never sees another gate's leftovers.
- The scorer was checked both ways: on the untouched base all twenty gates fail, each with its
  own message; on what each passing batch task changed, the scorer passes the same task.

## Repeats and order

Three repeats per arm, run one after another and never at the same time, in the order batch,
plain, plain, batch, batch, plain, so the time of day falls on both arms alike.

## Measured

- gates passed out of 20 per repeat, and per task over the three repeats
- for the batch: first-attempt passes, tasks retried, which backend passed them
- waiting time: the batch's wall time (gates and retries included); the session's wall time,
  and that plus its scoring time
- tokens as each provider reports them, unweighted; cost only where reported, marked estimate
- `hrn backends` readings of the Claude and Codex week before and after each run
- for the plain session: model, tool calls, subagents started, commands refused

## Analysis

Tasks are the cases (20). Per task, the pass rate over three repeats in each arm; the paired
difference routed minus plain, with a 95% bootstrap interval over tasks (10,000 resamples).

## Not compared

- The same model in both arms. The batch routes most tasks to cheaper models and falls back to
  claude; the plain session uses the account's default model. This compares the two ways of
  working, not two models.
- A plain session with permissions bypassed, or an interactive one with a person answering.
- Retrieval, the `api` backend, a review step.
