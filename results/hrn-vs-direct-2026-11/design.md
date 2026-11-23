# hrn versus the direct CLIs: design

Question: on small, real coding tasks, is a run through `hrn run -b <backend>` better than
calling the same CLI directly? "Better" is measured as gate pass, wall time and model tokens,
with the direct CLI as the baseline, per backend.

## Arms

Two arms per backend, four cells per task and repeat:

| arm | claude | codex |
|---|---|---|
| hrn | `hrn run -b claude --mode yolo --dir <wt> --timeout 15m --no-switch --json -` | `hrn run -b codex --mode yolo --dir <wt> --timeout 15m --no-switch --json -` |
| direct | `claude -p --output-format json --permission-mode bypassPermissions` (prompt on stdin, cwd `<wt>`) | `codex exec --json --skip-git-repo-check -C <wt> --dangerously-bypass-approvals-and-sandbox -` (prompt on stdin) |

The direct flags are the ones hrn itself passes for `--mode yolo` (`internal/agent/claude.go`,
`internal/agent/codex.go`), without what hrn adds on top. What hrn adds, and what this
comparison therefore measures as "the hrn layer":

- claude: `--strict-mcp-config` (no MCP servers), `--disable-slash-commands` (no skills),
  `--disallowedTools <guard deny rules>`, stream-json output parsed into a run record.
- codex: `-c mcp_servers.<name>.enabled=false` for every server in the user's `config.toml`.
- both: an allowlisted child environment, the run record under `~/.hrn/runs`, setup hashing,
  usage limit detection (`--no-switch` keeps a limited run on its backend so the pair stays
  paired), the command guard.

Neither arm names a model or an effort, so each CLI uses the account's default; the model that
answered is recorded per cell.

The direct arm's environment is the session's environment minus the desktop app's own
`CLAUDE_CODE_*`/`CLAUDECODE` variables and `ANTHROPIC_BASE_URL` (hrn's allowlist drops these
too; a terminal of the user's would not carry them). `TMP`/`TEMP` point at one short folder on
E: for both arms. No `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` is set anywhere: every run spends
the Claude or Codex plan, never pay-per-token access.

## Tasks

Eight closed harness issues whose merged fix changed one package and came with a test, each
under 120 changed lines. For each, the worktree is at the merge commit's first parent (main
right before the fix), and the gate copies the merged fix's test file(s) into place after the
agent finishes, then runs `go build ./... && go test -count=1 ./<package>/`. Every gate was
checked both ways before the runs: it fails at the base commit and passes at the merge commit
(`validate.log`). Each prompt is written from the issue, names the file and the behaviour the
hidden test pins (two prompts name a helper the hidden test calls), and ends with
"Do not run git on this repository." The same prompt text goes to all four cells of a task.

The hidden test replaces the agent's version of that file, so an agent's own tests in the
same file are not scored; a test the agent adds in another file stays and must still build.

## Cells and order

8 tasks x 2 backends x 2 arms x 2 repeats = 64 cells. Two lanes run at the same time, one per
backend (they draw on different plans and services). Within a lane, each task's four cells run
one after another, alternating arms (r1: hrn, direct; r2: direct, hrn) so that the time of day
and any plan-limit hit fall on both arms alike and a lane that stops early still leaves
complete pairs. A fresh worktree per cell; worktrees are removed after their patch is saved.

Timeout 15 minutes per cell in both arms (hrn's `--timeout`, and the runner kills a direct
CLI's process tree at the same limit). A cell that times out or is refused by a plan limit
counts as a failure and stays in the aggregate.

## Measured per cell

- gate pass (exit 0 of the gate), gate seconds
- wall seconds, measured by the runner around the whole process, both arms alike
- provider-reported tokens: claude input, cache read, cache write, output; codex input
  (which includes cached), cached, output. Summed per cell without weighting.
- cost where the CLI reports one (claude `total_cost_usd`; hrn's codex estimate marked so)
- turns, tool calls, the model that answered, status and error text
- files changed, whether the agent committed, patch size
- hrn extras: run id, refusal flags, warnings on stderr, adapter errors

## Analysis

Per backend, paired by task and repeat: success per arm; the median wall time and tokens per
arm with the direct arm as baseline; the paired difference hrn minus direct on each measure,
with a 95% bootstrap interval over tasks (10,000 resamples, each sampled task keeping both of
its repeats) as the hrn-evals publication gate asks. The gate asks for three repeats; this
comparison has two, by the run budget, and says so.

## Not compared

- `hrn batch`'s worktree setup and gates (this measures `hrn run`; worktrees were made the
  same way for both arms by the runner).
- Interactive use: every cell is a headless, single-prompt run.
- The `api` backend, other CLIs, models or efforts.
- Codex's cost: `codex exec` reports no cost; hrn's figure is a list-price estimate.

## Pins

hrn, Claude Code and Codex versions, Go version and the harness clone's commit are in the
report; each task's merge commit is in `tasks.json`, its base commit in every cell, and the
hidden test files' SHA-256 in `fixtures.sha256`.

## As run

The text above is the design as written before the first run. The run departed from it here:

- **Direct claude output format.** The first direct claude cell (`guard-case`, repeat 1) ran with
  `--output-format json`. The other fifteen ran with `--output-format stream-json --verbose`, so
  that a run killed at the limit keeps the usage of the messages it finished.
- **A run stopped at the limit.** The design counts it as a failure. The gate was run anyway on
  what the run left, and the report gives gate passes and stops separately, with the count
  under the design's rule beside them.
- **Lanes could stop between the two arms of a pair.** The runner reads the plan before every
  cell and stops a lane under a floor (claude 1% of the week, codex 10%), so the design's
  "complete pairs" was not guaranteed. It did not happen: all 64 cells ran.
- **The claude lane was started twice.** After a restart two copies ran side by side for about
  two minutes and both began the same cell; that record was deleted and the cell run again.
  Nothing stops two lanes of one backend from using the same worktree and output paths.
- **One prompt ends differently.** `privacy-check-dedupe` adds, after "Do not run git on this
  repository.", that a test may run git inside its own temporary repository: the hidden test
  does.
- **`agent_committed`** is the runner's own check in both arms (HEAD moved from the base), not
  something hrn reports.
- **What is published.** The public records leave out the error text and stderr along with the
  final answers, and no codex run has a cost, estimated or not.

Known weaknesses of the runner and the validator, left as they ran because they are the record
of this run; none changed a number here:

- The direct arm's environment drops the desktop app's variables but does not remove an
  `ANTHROPIC_API_KEY` or `OPENAI_API_KEY`. Neither was set on the day.
- The runner reads hrn's estimate flag under the name `cost_estimate`; hrn writes
  `cost_is_estimate`, so the flag is empty in every hrn cell. The report draws no conclusion from cost.
- A direct codex run's model is the model name that appears most often in codex's session file,
  not a field read from one known event. All 16 agree with the 16 hrn runs.
- Cells are written in place, not through a temporary file.
- `validate-gates.sh` records an exit code only, goes on after a worktree or a test file fails to
  set up, and prints DONE either way. The saved output of each run was read by hand instead; the
  report's gate table has what it showed.
- Both scripts default to scratch folders on a second drive; `HRN_VS_DIRECT_WORK` and
  `HRN_VS_DIRECT_TMP` move the runner's.
