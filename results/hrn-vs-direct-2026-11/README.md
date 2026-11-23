# hrn versus the direct CLIs on eight real harness fixes (2026-11-07)

Does running a small coding task through `hrn run -b claude` or `hrn run -b codex` beat calling
`claude` or `codex` directly with the same prompt, commit and gate? 64 runs were made on
2026-11-07 on the maintainer's Claude (Max) and Codex plans. No `api` backend and no API key.

**Answer: this run established neither an advantage nor an overhead for the hrn layer.** Success
was equal or within one run, and every time and token difference has a 95% task-bootstrap
interval that contains zero. The intervals are wide (claude's wall time: 86 seconds faster to 195
seconds slower), so an effect of that size in either direction is not ruled out. What hrn is for
beyond a single run (routing between backends, batches, plan-limit switching, retrieval) was not
part of this comparison. The page does not meet the publication gate (see the end), so it claims
no saving and no quality difference.

## Results

Per backend: 8 tasks x 2 repeats x 2 arms, paired by task and repeat. Tokens are the provider's
own counts summed without weighting: claude input + cache read + cache write + output; codex
input (which already includes cached reads) + output. In this table the passes and stops are
measured (counted over the 16 runs of the row) and the medians are inferred (the middle of the
16 measured values).

| backend | arm | gates passed | median wall | median tokens | median output tokens | stopped at 15 min |
|---|---|---:|---:|---:|---:|---:|
| claude | direct (baseline) | 15/16 | 391 s | 628k | 4,482 | 4 |
| claude | hrn | 16/16 | 468 s | 587k | 3,863 | 5 |
| codex | direct (baseline) | 16/16 | 196 s | 683k | 2,788 | 0 |
| codex | hrn | 16/16 | 169 s | 764k | 3,007 | 0 |

Paired difference, hrn minus direct, mean over the 16 pairs, with a 95% bootstrap interval over
tasks (10,000 resamples, each sampled task keeping both repeats). These are inferred: computed
from the measured runs by `scripts/aggregate.py`.

| backend | gate pass | wall seconds | total tokens | output tokens |
|---|---|---|---|---|
| claude | +0.06 [0.00, +0.19] | +60 [-86, +195] | -47k [-177k, +88k] | -363 [-2,186, +1,052] |
| codex | 0 [0, 0] | +23 [-38, +101] | +103k [-101k, +337k] | +341 [-253, +1,076] |

- **Success (measured).** Codex passed all 32 runs. Claude passed 31 of 32. The one failure was a
  direct run (`watch-quoted-path`, repeat 2): the hidden test failed on a single-quoted path
  that is never closed, inside a JSON string, ending in an escaped backslash right before the
  string's closing quote, a case the prompt names. One pair in hrn's favour is not a
  difference; its interval touches zero.
- **A run stopped at the limit is not counted as a failure here.** The design said it would be.
  The gate was still run on what each stopped run left behind, and all nine passed, so the table
  reports gate passes and stops separately. By the design's rule both claude arms score 11 of 16
  (inferred: 15 passes less 4 stops, and 16 less 5).
- **Why claude ran into the limit.** The records here show only that the fix was already in
  place when each of the nine runs was stopped. The raw transcripts, read on the day and kept
  off this repository, show the agent running the repository's whole test suite three times
  over at that point, as the repository's AGENTS.md advises before a pull request; that check
  takes ten minutes or more there. Codex never did this. It happened in both arms.
- **Model (measured).** Every claude run in both arms answered with `claude-opus-5-5`, and every
  codex run with `gpt-6.1-terra` (for direct codex runs, the model named most often in codex's
  own session file). Each pair compares the same model.
- **Cost.** Not compared. `results.md` shows the median of the costs Claude Code reported, with
  how many runs reported one (13 of 16 direct, 11 of 16 through hrn); a stopped run usually
  reports none, so the two medians cover different runs. No codex run has a cost.

`results.md` has every run's row and `cells/` each run's record. Each run's raw output, patch and
gate log stay on the maintainer's machine, outside this repository: codex reads the
maintainer's private Codex memory file in every run, through hrn and directly alike, and its
transcripts quote it. For the same reason the records here leave out the agents' final answers,
error text and stderr.

## Plan readings (measured, not attributed)

`plan-readings.txt` has what `hrn backends` said was left of each week at four moments:

| time | Claude week left | Codex week left |
|---|---:|---:|
| 13:39, the codex lane starts | 12% | 57% |
| 14:12, the claude lane starts again | 10% | 56% |
| 15:50, the codex lane ends | 8% | 56% |
| 18:37, the claude lane ends | 5% | 56% |

The claude lane first started at 13:42 and read 12% then; its snapshot was written over when the
lane started again, so only the runner's log has it. Other sessions drew on the same two plans
during those five hours. The fall cannot be divided between these runs and that other use,
so the page gives no cost per run and does not compare the two plans.

## What the hrn arm did differently (measured)

- **Nothing refused.** hrn's guard blocked no command in any of the 32 hrn runs, and raised no
  refusal flags.
- **No adapter errors.** hrn recorded every run, including the five it stopped at its own
  timeout. For those, the tokens come from hrn's per-call usage events. Their output count is a
  lower bound, because Claude Code reports a message's output before the message is complete.
  Three direct claude runs that were stopped have per-message sums with the same limit.
- **Lean start.** hrn started claude without MCP servers, skills and slash commands, and codex
  with each MCP server in its config turned off. The direct arm loaded all of them. That is part
  of what the hrn arm is; the comparison does not separate its effect from the rest.
- **One prompt violation.** One codex run through hrn committed its change although the prompt
  says not to run git. The runner checks this the same way in both arms (`agent_committed`: HEAD
  moved), and the gate still scored the change.
- **Worktrees.** The runner made a fresh worktree for every run in both arms, so worktree setup
  is not part of the difference. `hrn run` does not make one; `hrn batch` would.

## What could not be compared

- **The orchestration hrn exists for.** Routing a task to the backend that does it best, batches
  with gates and retries, moving a run to another backend at a plan limit, and project
  knowledge. Each run here was one headless prompt to one backend.
- **Harder tasks.** These tasks were easy enough that 63 of 64 runs passed, so success could not
  separate the arms. The hrn-evals real-work suite, where runs do fail, would.
- **Cost and plan use**, for the reasons above.
- **Other backends.** agy, gemini and opencode were not run.

## Defects found on the way

- **hrn's Claude plan reading lags Claude Code's.** At one point `hrn backends` said 7% of the week
  was left while Claude Code itself showed about 13% (seen once, not recorded). hrn uses that reading to decide when to
  switch backends at a plan limit.
- **Direct claude ignored the repository's permission allowlist.** Claude Code said it was
  ignoring the clone's allow rules because the workspace was not trusted. It did not matter
  here, since both arms bypass permissions.

## Gate validation

Each task's gate was run both ways before the task's first run: the merged fix's test file
copied onto the commit before the fix, and onto the merge. `validate.log` has the exit codes;
the reason each one failed before the fix was read from the saved output, which is not
published:

| task | issue | before the fix | at the merge |
|---|---:|---|---|
| guard-case | 211 | `TestGuardBlocks` fails | passes |
| gc-idle-zero | 245 | the test file does not build: `ageAt` is undefined | passes |
| watch-dest-document | 228 | `TestPublicAccessAllowsPageNavigationAfterLoginAndKeepsOtherGuards` fails | passes |
| remote-oversized-line | 164 | `TestWatchStopsOnOversizedEventLine` fails | passes |
| watch-quoted-path | 237 | `TestPathsInAWindowsRunAreShortenedToTheWorkspaceHomeAndProgramNames` and its Unix twin fail | passes |
| codex-reconnect-keeps-error | 349 | `TestCodexAReconnectNoticeKeepsTheErrorBeforeIt` fails | passes |
| durable-partial-usage | 149 | `TestDurableFailedStreamKeepsPartialUsage` fails | passes |
| privacy-check-dedupe | 648 | `TestTheRunReportsEachFindingOnceInAConflictedFile` fails | passes |

`ageAt` is the helper the `gc-idle-zero` prompt names. `validate-2026-11-23.log` is a second run
of the eight with the validator as it is now, which first runs the commit before the fix with
its own tests: all eight pass there, fail with the copied tests in the same way as in the
table, and pass at the merge. `validate.log` also lists four more
candidates (issues 429, 607, 348 and 122) that passed the same check and were not used: the
comparison was sized for eight tasks, and `tasks.json` was fixed ten minutes before the first run.
`fixtures.sha256` has the SHA-256 of each hidden test file at its merge commit.

## Reproduction

- Design and runs: `design.md` (with a closing section on where the run departed from it),
  `tasks.json`, `prompts/` and `scripts/`.
- The scripts were changed after the run: weaknesses found in review are closed in
  `scripts/runner.py` and `scripts/validate-gates.sh` (`design.md`, "The scripts after the run",
  lists each), and `scripts/publish.py` and `scripts/aggregate.py` were reworked for this page:
  `publish.py` writes `cells/` and `plan-readings.txt` from the runner's records through a list
  of allowed fields, and `python scripts/aggregate.py cells` rebuilds `results.md` from `cells/`.
- hrn 0.2.0 at harness `dc7179d`; the harness clone at main `9dcee21`; Claude Code 2.1.284; Codex
  CLI 0.166.0-alpha.6; Go 1.26.10 for the gates. Each cell has its base commit; `tasks.json` has
  each merge commit.
- One claude run was discarded and run again. After a restart, a second copy of the claude lane
  ran beside the first for about two minutes, and both started `guard-case` direct repeat 2.
  That record was deleted unscored, and the run was made again once a single lane was left. No
  other run was dropped.
- `hrn_run_id` in each hrn cell names its record under `~/.hrn/runs`. These are `hrn run`
  records, not an `hrn eval`, so there is no eval id for `hrn eval diff`.

## Publication gate

No rule is fully met, so this page describes a run and claims no difference.

- [ ] 1. At least six cases, three repetitions per arm. Eight cases, but **two** repetitions, by
  the run budget.
- [ ] 2. Every run gate-scored; none dropped. All 64 kept runs are gate-scored, but one run was
  discarded without a score and made again (see Reproduction).
- [ ] 3. Provider-only unweighted token usage. The counts are the providers' own and unweighted,
  but the direct arm's come from the CLI's output and not from an hrn run record, and eight
  stopped claude runs have per-call sums whose output count is a lower bound. No cost is claimed.
- [ ] 4. Negative and no-op cases. The failure and the nine stopped runs are in the aggregate,
  but all eight tasks ask for a fix: there is no case with nothing to improve.
- [ ] 5. The 95% case-bootstrap intervals (10,000 resamples) are reported. None excludes zero.
- [ ] 6. Reproduction pins. Versions, commits, models, run ids and the hidden tests' SHA-256 are
  recorded; there are no eval ids, since these are not `hrn eval` runs.
- [ ] 7. Every number labelled. The results are labelled measured or inferred, but the two
  plan readings under "Defects found on the way" were seen once on the day and are in no record.

**In one line: on these eight small fixes the run showed neither a gain nor a cost from the hrn
layer, within wide intervals; hrn's case rests on the orchestration this comparison did not
test.**
