# A routed hrn batch versus one plain session on the real-work suite (2026-11-10)

Given the twenty tasks of `suites/routing/real-work.json`, does a routed `hrn batch` (one run per
task, gates, a retry on another model family) finish more of them than one plain Claude Code
session that is handed all twenty in one prompt? Six runs were made on 2026-11-10, three per
arm, on the maintainer's Claude, Codex and Antigravity plans. No `api` backend and no API key.

**Answer: this run did not establish a difference in what the two ways finished.** The batch
passed 60 of 60 gates and the plain session 59 of 60; the interval of the difference touches
zero, which shows neither a gain nor that the two are equal. The table reports time and cost
for each repeat, three per arm, with no interval and no conclusion. The batch reached its 60
only through its gates and retries: its first attempts passed 52 of 60, fewer than the plain
session, and that interval excludes zero. The page misses five of the publication gate's seven
rules (see the end), so these are reported as what was measured, not as a claim about either
way of working.

Numbers read from a run's record are measured; where a number is computed from those, the text
says how, and it is inferred.

## Results

| arm | repeat | gates passed | waiting time | tokens | cost |
|---|---:|---:|---:|---:|---:|
| routed batch | 1 | 20/20 | 5.6 min | 3.45M | $2.90 |
| routed batch | 2 | 20/20 | 4.3 min | 3.36M | $1.31 |
| routed batch | 3 | 20/20 | 4.0 min | 3.16M | $1.20 |
| plain session | 1 | 20/20 | 5.2 min | 2.67M | $2.00 |
| plain session | 2 | 20/20 | 4.8 min | 2.72M | $1.92 |
| plain session | 3 | 19/20 | 5.3 min | 3.13M | $2.08 |

- **Waiting time** is wall time. The batch's includes its gates and retries. The plain session's
  does not include scoring, which took another 0.8 minutes each time.
- **Tokens** are each provider's own counts, summed without weighting (inferred, as that sum):
  claude input + cache read + cache write + output; codex input (which includes cached reads)
  + output; the one agy run's 16,968 tokens as it reported them.
- **Cost** is the sum of what each run reported (inferred, as that sum). Claude's figure is
  Claude Code's own, at list price; codex's is hrn's list-price estimate; the one agy run, in
  batch 3, reported none, so that batch's $1.20 leaves it out. Every run spent a plan, so
  nothing was billed.

Paired difference in pass rate, routed minus plain: per task, the share of the three repeats
that passed in each arm; then the mean over the 20 tasks, with a 95% bootstrap interval over
tasks (10,000 resamples). Both rows are inferred. The tasks are treated as separate cases, but
two of them change the same file: in the batch each works on its own copy, in the plain session
both changes land in one file.

| compared with the plain session | difference |
|---|---|
| the batch's final result | +0.017 [0.000, +0.050] |
| the batch's first attempts only | -0.117 [-0.217, -0.033] |

- **Success.** The plain session's one failure was `plan-budget-guard` in repeat 3: its plan had
  422 words against a limit of 400. Every other gate passed in both arms.
- **First attempts.** The batch's first attempts failed 8 of 60 gates, on six tasks:
  `research-eval-expand` and `research-gate-timeout` twice each, `implement-estimate-mark`,
  `review-idle-age`, `plan-budget-guard` and `plan-unpriced` once each. Five failures were codex
  `gpt-6-luna` at low effort, one codex `gpt-6.1-sol`, two claude `sonnet` plans over their word
  limit.
- **Retries.** All eight failed tasks passed after a retry: seven on the first retry (five codex
  to claude, two claude to codex), one on the second (agy, after claude also failed).
- **Time.** Batch minus plain session, by repeat (inferred, a difference of two wall times):
  +23, -24 and -76 seconds before the plain session's scoring, and -25, -72 and -123 seconds
  with it. The repeats were not run as pairs, so these only set the two columns side by side.
- **Cost.** The batch's twenty first attempts cost $0.79 to $0.90 a repeat. Its retries cost
  $2.01, $0.42 and $0.41: a retry on another backend dropped the routed model and ran that
  backend's default, and in repeat 1 one retry ran on `claude-fable-5-1` and cost $1.11 alone.
- **Models.** The plain session answered with `claude-opus-5-5` every time. The batch ran 33
  first attempts on codex `gpt-6-luna`, 12 on codex `gpt-6.1-sol`, 15 on claude
  `claude-sonnet-5-5`; its retries ran on `claude-opus-5-5` (5), `claude-fable-5-1` (1), codex
  `gpt-6-astra` (2) and agy (1).

`results.txt` has the per-task table and every retry; `results.json` the same as data.

## What each arm loaded and was allowed

- **The plain session** started with the account's whole setup: 250, 208 and 281 tools in the
  three repeats, and 94 MCP servers and 279 skills each time. The batch's claude and codex runs
  start lean.
- **The plain session ran with an allowlist, not with permissions bypassed**: file edits, `go
  test`, `go build`, `go vet`, `gofmt`, `wc`, `ls`, `cat`, `grep`. It was refused 5, 5 and 12
  commands, and passed 59 of 60 without them.
- **The check files were not hidden from an agent.** In the plain sessions 14, 12 and 15 tool
  calls named a file under `fixtures/real/checks`; nine in each were file reads, the others
  shell commands and, once, a write. In the batch, a tool call named one in 6, 8 and 5 of the
  twenty first attempts. These are two different counts, calls in one arm and runs in the
  other, and a call that names a file is not always one that opened it. The gates install those
  files after the agent finishes, but in this run nothing kept an agent from them, in either
  arm.

## Since this run

Two things this run showed have been changed since. Neither was measured again here.

- The suites now carry a `hide` list that keeps the check files out of an hrn run's worktree
  while the agent works. A plain session is not started by hrn, so nothing hides them from it.
- A routing row can now name a fallback per backend, so a retry on another backend runs a
  chosen model, not that backend's default.

## The two ops gates, checked again on 2026-11-23

The gates of `ops-runner-workflow` and `ops-dockerignore` were tightened on 2026-11-23: the ones
this run used could pass a workflow that names `go vet` and `go test` without running both in
both jobs, and ignore rules whose exceptions do not bring the sources back. The twelve files
this run left for the two tasks, one per task and repeat in each arm, were kept: the batch's on
its branches in the scratch clone, the plain sessions' with each session's output. All twelve
pass the tightened gates, as they passed the gates of the run, so none passed by either route
and no count on this page changes (measured by running the gates on the kept files; no agent
was run again).

## What could not be compared

- **Harder or longer work.** These tasks are small and, but for the two that share a file, stand
  on their own: twenty fit in one session of
  about five minutes and 75 turns. Whether one session holds up when the work no longer fits one
  context was not tested, and that is where a per-task run is expected to help.
- **The same model in both arms.** The batch routes to cheaper models and falls back to the
  expensive ones; the plain session used the account's default. This compares two ways of
  working, not two models.
- **Plan use.** hrn's reading of the Claude week moved from 94% to 81% over the whole
  comparison, 12 points of it during the first batch, while other sessions drew on the same
  plan. It cannot be attributed to either arm. The Codex week moved from 37% to 36%.
- **A plain session with permissions bypassed**, an interactive one, retrieval, a review step,
  and a plan limit reached mid-batch.

## Reproduction

- `design.md` has the method; `specs/` has the inputs (`routed.r1.json` is the batch spec, with
  the scratch clone's folder written as `<clone>`; repeats 2 and 3 differ only in the task ids;
  `plain-prompt.txt` is the plain session's whole prompt); `route.txt` is the routing table in
  force.
- The scripts that started the runs, scored the plain arm and built `results.json` are not
  published. They were written for one machine and have not had the review that the scripts of
  `../hrn-vs-direct-2026-11/` got; `design.md` says in words what they did. The comparison
  tables can be recomputed from `results.json`: a task's first attempt passed when it passed with
  one attempt, and the `details` key has the cost of first attempts and retries, each model's
  runs, every retry, what each plain session loaded and the tool calls that named a check file.
- hrn 0.2.0 at harness `ac82e51`; hrn-evals `f637e30` in a scratch clone, which also fixes the
  fixtures; Claude Code 2.1.284; Codex CLI 0.167.0-alpha.3.2; Go 1.26.4 for the gates.
- Batches `20261110-180332-964f73`, `20261110-182225-9fc1a5` and `20261110-182647-c30e69` under
  `~/.hrn/batches`; every attempt is a run under `~/.hrn/runs`. These are batches, not an
  `hrn eval`, so there is no eval id for `hrn eval diff`.
- Order: batch 1, plain 1, plain 2, batch 2, batch 3, plain 3, one at a time. Other sessions
  were running on the same computer.
- The scorer for the plain arm was checked both ways: it fails all twenty tasks on the untouched
  base, and passes all twenty on what batch 1's tasks changed, as hrn's own gates did.
- No run was dropped or rerun. Raw session output stays on the maintainer's machine.
- `results.json` and `results.txt` were built again on 2026-11-23 from the same records. The
  first build rounded a batch's cost after every run it added, which left the per-backend costs
  two or three cents low, and it printed a retry that failed its gate as a pass when a later
  retry rescued the task. The cost column above uses the corrected sums, and the `details` key
  was added. No pass, time or token count changed.

## Publication gate

Rules 1 and 2 are met. The others are not, so the page claims no saving and no difference in
quality.

- [x] 1. At least six cases, three repetitions per arm, the same tasks in both: twenty tasks,
  three repeats.
- [x] 2. Every run gate-scored; none dropped. The batch by hrn's gates, the plain session by the
  same gates run by the scorer described in `design.md`.
- [ ] 3. Provider-only unweighted token usage. The counts are the providers' own and unweighted,
  but the plain arm's come from Claude Code's output, not from an hrn run record. Cost is as
  reported, with codex's an estimate and agy's missing.
- [ ] 4. Negative and no-op cases. Every failure is in the aggregate, but all twenty tasks ask
  for work: there is no case with nothing to do.
- [ ] 5. The 95% case-bootstrap intervals are reported. The one for the final result touches
  zero, so no difference in success is claimed. The one for first attempts excludes zero.
- [ ] 6. Reproduction pins. Versions, commits, models and batch ids are recorded, and the
  fixtures by the hrn-evals commit; the agy CLI's version and the model of its one run are not
  recorded, and there are no SHA-256 hashes and no eval ids.
- [ ] 7. Every number labelled. The tables and the differences say what is measured and what is
  inferred, but the counts and ranges in the text are not each labelled.

**In one line: on twenty small tasks the routed batch passed 60 of 60 gates and one plain
session 59 of 60, which three repeats cannot tell apart; the batch's cheaper first attempts
failed more often, and its gates and cross-family retries made up the difference.**
