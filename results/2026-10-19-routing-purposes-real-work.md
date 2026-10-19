# 2026-10-19: the purposes and real-work suites, and a routing row for every purpose

The owner gave the go on 2026-10-19 for the first runs of `suites/routing/purposes.json` and
`suites/routing/real-work.json`. They are the only suites with tasks for the `review`,
`refactor`, `plan` and `ops` purposes, which the table of [2026-10-18](2026-10-18-routing.md) had
no row for. Both ran with Haiku named by its id, `claude-haiku-5-5`, for the first time (PR #62).
Every number on this page is **measured** on these suites unless marked otherwise.

Both evals ran from one clean clone of `main` at `1962cfc`, through `scripts/run-all.sh`, which
refused nothing. No variant was left out: the one plan limit hrn remembered (`claude`, until
2026-10-15) had expired.

| eval | suite | tasks | cells | passed | wall time |
|---|---|---:|---:|---:|---:|
| `4c210` | `routing/purposes` | 4 | 48 | 45 | 21m13s |
| `7bd77` | `routing/real-work` | 20 | 240 | 190 | 3h7m1s |

## What each variant did

| variant | model that answered | effort | `4c210` pass | `7bd77` pass | `7bd77` median time | `7bd77` median cost |
|---|---|---|---:|---:|---:|---:|
| sol-medium | gpt-6.1-sol | medium | 12/12 | 50/60 | 57 s | $0.044 estimated |
| sonnet-medium | claude-sonnet-5-5 | medium | 12/12 | 52/60 | 3m1s | $0.17 estimated |
| luna-low | gpt-6-luna | low | 10/12 | 43/60 | 51 s | $0.0030 estimated |
| haiku-low | claude-haiku-5-5 | low | 11/12 | 45/60 | 2m54s | $0.30 estimated |

In `4c210` the medians were 46 s, 1m47s, 38 s and 49 s, at $0.040, $0.15, $0.0026 and $0.28
(all estimated), in the same order.

Haiku is now Haiku 5.5. Every Claude cell's record names `claude-haiku-5-5` or
`claude-sonnet-5-5`, and every codex rollout names `gpt-6.1-sol` or `gpt-6-luna`. By Claude
Code's own figure, Haiku 5.5 cost more per cell than Sonnet on both suites.

### `4c210`, `routing/purposes`

| task | purpose | sol-medium | sonnet-medium | luna-low | haiku-low |
|---|---|---|---|---|---|
| review-planted-defects | review | 3/3 | 3/3 | 2/3 | 3/3 |
| refactor-duplication | refactor | 3/3 | 3/3 | 3/3 | 3/3 |
| plan-rate-limiting | plan | 3/3 | 3/3 | 2/3 | 2/3 |
| ops-ci-workflow | ops | 3/3 | 3/3 | 3/3 | 3/3 |

### `7bd77`, `routing/real-work`

| task | purpose | sol-medium | sonnet-medium | luna-low | haiku-low |
|---|---|---|---|---|---|
| implement-variant-name | implement | 3/3 | 3/3 | 3/3 | 3/3 |
| implement-event-line-limit | implement | 3/3 | 3/3 | 3/3 | 2/3 |
| implement-estimate-mark | implement | 3/3 | 3/3 | 3/3 | 3/3 |
| implement-shown-path | implement | 3/3 | 3/3 | 3/3 | 3/3 |
| debug-system32-case | debug | 3/3 | 3/3 | 3/3 | 3/3 |
| debug-idle-zero | debug | 0/3 | 3/3 | 3/3 | 3/3 |
| debug-quoted-path | debug | 3/3 | 3/3 | 0/3 | 3/3 |
| test-moved-cache | test | 3/3 | 3/3 | 3/3 | 3/3 |
| test-path-separator | test | 3/3 | 3/3 | 3/3 | 3/3 |
| test-cell-escape | test | 0/3 | 0/3 | 0/3 | 0/3 |
| research-eval-expand | research | 0/3 | 0/3 | 0/3 | 0/3 |
| research-gate-timeout | research | 3/3 | 1/3 | 3/3 | 1/3 |
| review-idle-age | review | 3/3 | 3/3 | 0/3 | 2/3 |
| review-event-retry | review | 3/3 | 3/3 | 1/3 | 3/3 |
| refactor-suite-key | refactor | 3/3 | 3/3 | 3/3 | 3/3 |
| refactor-request-headers | refactor | 3/3 | 3/3 | 3/3 | 3/3 |
| plan-budget-guard | plan | 2/3 | 3/3 | 3/3 | 0/3 |
| plan-unpriced | plan | 3/3 | 3/3 | 2/3 | 1/3 |
| ops-runner-workflow | ops | 3/3 | 3/3 | 2/3 | 3/3 |
| ops-dockerignore | ops | 3/3 | 3/3 | 2/3 | 3/3 |

The purpose column is the category hrn gave each prompt (`purpose.Classify` at harness
`883fe02`), which is what `hrn route from-eval` groups by. It agrees with every task id's prefix.

## Two tasks no variant could pass

`test-cell-escape` and `research-eval-expand` scored 0 of 12. In both, a gate fails whatever the
agent wrote, when hrn runs it on Windows ([#64](https://github.com/cristiangirlea/hrn-evals/issues/64)):

- **`test-cell-escape`.** hrn runs a gate as `bash -c <command>` from Go. On the way through the
  Windows command line, Git Bash turns the line-break mutation's `\\n` into `\n`. sed reads that
  as a newline, so the mutation changes nothing and the step reports "the tests still pass when
  a line break is left as it is". Reproduced locally on Sonnet's `r1` commit `377ebe4`, whose
  tests do check `Cell("a\nb") == "a<br>b"`: the recorded command run through Go's `os/exec`
  into Git Bash, as hrn runs a gate, fails the step; the same step run from a script mutates
  `Cell`, and the tests fail on it as the gate intends. 11 of the 12 failures are this; Luna's
  `r1` wrote no test file, a real failure.
- **`research-eval-expand`.** The answer 3 gate's `grep -qiF` aborts: every one of the 12 gate
  outputs shows grep "Aborted", and the gate's own `|| { ...; exit 1; }` records exit 1.
  Reproduced locally, Git for Windows' grep 3.0 aborts on this pattern (exit 134) with no UTF-8
  locale (`LC_ALL=C`, or `LANG` unset as in a shell started without a profile) and exits 0 with
  `C.UTF-8`; that hrn's gate shell starts without one is inferred from the outputs. The answers read, Sonnet's `r1` and Sol's `r1`, quote the
  message correctly.

The scores stay as recorded (gate rule 2). The routing rows below do not depend on these two
tasks (**inferred** from the per-task table): without them `test` is a four-way 6/6 tie and
`research` is Sol and Luna at 3/3, so the same rows would win.

## The other failures

Each with the last line its gate printed.

- `4c210 review-planted-defects luna-low r3`: "REVIEW.md is missing or empty".
- `4c210 plan-rate-limiting luna-low r3`: "PLAN-ANSWER.md is missing".
- `4c210 plan-rate-limiting haiku-low r3`: "too long: 407 words" (limit 400).
- `debug-idle-zero sol-medium` r1-r3: "a cache written just now is still not taken, or another
  decision changed".
- `debug-quoted-path luna-low` r1-r3: "Program still misreads a quoted path, or a case that worked
  broke".
- `implement-event-line-limit haiku-low r3`: the agent hit the 15-minute cell timeout; no gate ran.
- `research-gate-timeout` sonnet-medium r2, r3 and haiku-low r2, r3: "answer 3 does not say the
  task wins".
- `review-idle-age luna-low`: r1 "the review does not find the same negative age at line 31", r2
  and r3 "fixtures/real/review-idle/REVIEW.md is missing or empty"; `haiku-low r3`: "the review
  does not find the same negative age at line 31".
- `review-event-retry luna-low` r2, r3: "fixtures/real/review-retry/REVIEW.md is missing or empty".
- `plan-budget-guard haiku-low` r1-r3: "too long" at 527, 491 and 495 words; `sol-medium r2`:
  "the plan does not say what happens to the tasks already running".
- `plan-unpriced haiku-low` r1, r2: "too long" at 310 and 334 words; `luna-low r1`:
  "fixtures/real/plan-unpriced/PLAN-ANSWER.md is missing or empty".
- `ops-runner-workflow luna-low r3`: "ci.yml points into fixtures/: the module is the checkout root".
- `ops-dockerignore luna-low r2`: ".dockerignore is missing or empty".

Luna's nine "missing" failures (two in `4c210`, seven in `7bd77`, `test-cell-escape r1`
included) are runs that ended `ok` without writing the file they were asked for: the failure mode of 2026-10-14 and 2026-10-15, which
did not occur in `fbc00`.

## Passes that changed nothing, and git

hrn counts 4 passes in `4c210` and 8 in `7bd77` that changed nothing: Sonnet 2 and 3, Luna 1 and
4, Haiku 1 and 1. In 11 of the 12 the agent's own tool calls ran `git add` and `git commit`,
though every prompt ends "Do not run git.", so hrn found nothing left to commit. The
twelfth (`4c210 ops-ci-workflow luna-low r3`) shows no git call in its log. The work is in place
in each and the gates passed it; the gates do not check the instruction.

One cell wrote outside its worktree. In `7bd77 implement-shown-path haiku-low r1` (run
`20261019-124026-d02425`, a pass) Haiku copied `shownpath.go` and the hidden check into a
`shownchk/` folder at the root of the clone the eval ran from, to run them apart. That folder is
untracked in the clone, and the cells branch from its committed HEAD, so it reached no other cell. The cells run in `yolo` mode,
which does not fence writes to the worktree.

## Tokens and cost

Provider-reported usage summed over each variant's cells, the four fields as the batch records
hold them, unweighted. Codex counts its cache reads inside `input` (inclusive semantics); Claude
counts them apart. Codex costs are hrn's list-price estimate. Claude's are Claude Code's own
list-price figure, which the run records carry as reported. Neither plan bills them, so both are
marked estimated here.

`4c210`, 12 cells per variant:

| variant | input | output | cache read | cache write | total cost |
|---|---:|---:|---:|---:|---:|
| sol-medium | 1,050,115 | 8,079 | 873,088 | 0 | $0.52 estimated |
| sonnet-medium | 90 | 18,035 | 2,071,964 | 335,227 | $1.94 estimated |
| luna-low | 1,122,352 | 8,732 | 941,440 | 0 | $0.03 estimated |
| haiku-low | 98 | 28,404 | 2,725,360 | 267,518 | $3.25 estimated |

`7bd77`, 60 cells per variant:

| variant | input | output | cache read | cache write | total cost |
|---|---:|---:|---:|---:|---:|
| sol-medium | 8,158,287 | 60,490 | 7,218,304 | 0 | $3.21 estimated |
| sonnet-medium | 536 | 138,295 | 13,132,831 | 1,658,115 | $10.64 estimated |
| luna-low | 9,663,019 | 63,802 | 8,614,528 | 0 | $0.22 estimated |
| haiku-low | 544 | 236,681 | 15,729,437 | 1,434,005 | $20.03 estimated |

The two evals' totals, $5.74 and $34.10, are those `hrn eval list` shows.

## The table

The table of 2026-10-18 was saved first, unchanged (it is byte for byte
[`route-2026-10-18.json`](route-2026-10-18.json)). Then `hrn route from-eval 4c210` and `hrn route
from-eval 7bd77` ran, in that order, with the default margin 0.05. from-eval replaces the row of
every purpose an eval's tasks fall into, so `7bd77`, with two tasks per new purpose, has the last
word over `4c210`'s one. Pooling the two evals by hand gives the same four rows (**inferred**:
review Sol and Sonnet 9/9, refactor all 9/9, plan Sonnet 9/9 alone within the margin, ops Sol,
Sonnet and Haiku 9/9). The result is in `~/.hrn/route.json` on the machine that ran the evals and
copied to [`route-2026-10-19.json`](route-2026-10-19.json).

| purpose | backend | model | effort | eval | why |
|---|---|---|---|---|---|
| debug | claude | sonnet | medium | `7bd77` | Sonnet and Haiku 9/9, Sol and Luna 6/9; Sonnet's median cost is lower than Haiku's |
| implement | codex | gpt-6-luna | low | `7bd77` | Sol, Sonnet and Luna 12/12, Haiku 11/12 (outside the margin); Luna cheapest |
| ops | codex | gpt-6.1-sol | medium | `7bd77` | Sol, Sonnet and Haiku 6/6, Luna 4/6; Sol cheapest of the three |
| plan | claude | sonnet | medium | `7bd77` | Sonnet 6/6, Sol and Luna 5/6 (outside the margin), Haiku 1/6 |
| refactor | codex | gpt-6-luna | low | `7bd77` | all four 6/6; Luna cheapest |
| research | codex | gpt-6-luna | low | `7bd77` | Sol and Luna 3/6, Sonnet and Haiku 1/6; Luna cheaper |
| review | codex | gpt-6.1-sol | medium | `7bd77` | Sol and Sonnet 6/6, Haiku 5/6, Luna 1/6; Sol cheaper |
| test | codex | gpt-6-luna | low | `7bd77` | all four 6/9; Luna cheapest |
| unknown | codex | gpt-6-luna | low | `fbc00` | unchanged: neither suite has a task of this purpose |

The cost compared is each variant's median over the whole eval, not per purpose, as from-eval
works. The `sonnet` model is stored as the suite names it, the alias, which these runs resolved to
`claude-sonnet-5-5`.

**What changed against `fbc00`.** Four purposes have a row for the first time. Of the five that
had one, only `debug` changes model: Luna low at `fbc00`, where `fix-sum` was its only debug task,
to Sonnet medium here, on three debug tasks where Luna missed `debug-quoted-path` 0 of 3.
`implement`, `research` and `test` keep Luna low, now from `7bd77`, which has more tasks for each
(4, 2 and 3 against 2, 1 and 1), so no row rests on less evidence than before. `unknown` keeps its
`fbc00` row. The rest of the setup is the same as for `fbc00` (hrn `883fe02`, Claude Code
2.1.284, codex-cli 0.162.0-alpha.17.2), except that Haiku is 5.5, not 4.5, and the suites are at
`1962cfc`, not `2906e51`. Newer harness commits change how from-eval treats harness variants, of
which these specs have none, and cells that never acted (ended `ok` with no tool call, no second
turn and no change), which newer hrn leaves out of the pass rate. One cell here is such a cell:
`7bd77 plan-unpriced luna-low r1` (run `20261019-142633-dbfdb7`), one turn, no tool call, no
commit, counted above as a missing-file failure. The saved records carry no such count, so a
newer `from-eval` run on them builds the same rows. An eval recorded by newer hrn would score
Luna's `plan` 5/5, a tie with Sonnet that Luna wins on cost, so `plan` would go to Luna low;
pooled with `4c210` Luna would be 7/8 and outside the margin (**inferred** from the diff of
`internal/route`, `internal/eval` and `cmd/hrn/route.go` since `883fe02`).

**Read this before relying on it.** Eight of the nine rows are ties at the top broken by cost;
only `plan` is decided by pass rate alone, and it rests on the single Luna cell that never
acted. The cost tie-break favours codex: Luna's median cell is $0.0030 against Sonnet's $0.17 in
`7bd77`. `debug` and `plan` are the first rows to pick Claude, because in both the codex variants
fell below the bar. Two
tasks per purpose (three for debug and test, four for implement) is enough to separate a variant
that fails a task outright, as Luna did on `debug-quoted-path` and `review-idle-age`, not to rank
variants that pass most of them. Luna's "missing file" runs are the reason to keep watching its
rows on real work.

## Against the publication gate

This page claims no saving and no quality difference. It records the evals and the table built
from them. Checked against [the publication gate](../docs/publication-gate.md):

- [ ] 1. **Cases, repetitions and pairing:** met for `7bd77`, not for `4c210`. `7bd77` has 20
  cases, three repetitions per arm, every arm on the same tasks, paired by case and repetition.
  `4c210` has four cases with three repetitions each; it is reported as gate results and feeds the
  table only through the hand pooling above, which agrees with `7bd77` alone.
- [x] 2. **Complete scoring:** met. All 288 cells are in their records and no run is dropped.
  Every cell's gates are scored except `7bd77 implement-event-line-limit haiku-low r3`, which hit
  the 15-minute timeout before its gates ran and counts as a failure. The 24 cells of the two tasks whose gates are broken stay in the
  tables as scored; the section above says why they failed.
- [x] 3. **Usage and cost:** met. The token tables sum the four provider-reported fields from the
  batch records without price weighting. Every cost is marked estimated: codex's are hrn's
  estimate, Claude's are Claude Code's own list-price figure.
- [x] 4. **Negative and no-op cases:** met. Every failure is in the aggregate (45 of 48, 190 of
  240), and the 12 passes that changed nothing are counted and explained.
- [ ] 5. **Uncertainty:** not met and not needed, since no paired difference is claimed.
- [x] 6. **Reproduction pins:** met.
  - hrn 0.2.0 built from harness `883fe02ee946`.
  - codex-cli 0.162.0-alpha.17.2, from codex's own session files: all 144 rollouts of the two
    evals' codex cells carry it. `run-all.sh` printed "codex: (not installed)" because codex is
    not on Git Bash's `PATH`, while hrn runs it from its install path.
  - Claude Code 2.1.284, as `run-all.sh` printed it.
  - Models: `claude-sonnet-5-5` and `claude-haiku-5-5` as Claude Code reported them in the batch
    records (`model_reported` set on all 144 Claude cells); `gpt-6.1-sol` and `gpt-6-luna` from
    codex's own rollout files, one per codex cell. Codex reports no model to hrn, so the batch
    records' codex `model` is only the id hrn asked for.
  - Suite commit `1962cfc5b50ee517608461b248ef7829c68149cb`, both batch records' `base`
    (batches `20261019-111134-10aefc` and `20261019-113313-340c9a`).
  - Fixture SHA-256, as `run-all.sh` printed them, for every file under `fixtures/purposes/` and
    `fixtures/real/`, the folders these suites and their gates read (hidden checks included).
    Each matches the file at `1962cfc`:
    - `fixtures/purposes/ops/add.go` `09b196c046e9d3e27f7dd59f9e6da10f8ac1d1775e83793e5c84b61d17de7c8d`
    - `fixtures/purposes/ops/add_test.go` `d919831c345d29249b964029b1f603529d8f516aefbb4c360c3f99f519c19525`
    - `fixtures/purposes/ops/go.mod` `fbfa2535a2c1e65635ee1e668b6ea0c410964333f712c2884e4d5cdea8a7a2c8`
    - `fixtures/purposes/plan/handler.go` `40028ad4a7016cf2a51d1469f63d0fcbc14145d119ed996f34d9a1722d854cd7`
    - `fixtures/purposes/refactor/checks/dup_test.go.txt` `46e9defe51c253902cc4ee39b1c87794908a09ae27befe6a2200026f525919b0`
    - `fixtures/purposes/refactor/dup.go` `3b0e486f1fe8801c744cd3ab4de1d95265423d49804e699a8f6821f4871c001d`
    - `fixtures/purposes/review/review.go` `2bc49730f628eb7ece8d73dc913380f4190a17c31f40156d5d6d3f1a3baa4811`
    - `fixtures/real/cacheage/cacheage.go` `afb8dbdb3f903364bb281f933fba9dd053a78cbb1d4c30b8afef02ea17bf29e5`
    - `fixtures/real/cellescape/cell.go` `c6a0ce24769333e174b1726ecb24abc5387efccdc2895f33ab49f1f597928c3a`
    - `fixtures/real/checks/cacheage_test.go.txt` `7f0040da48d76580049dbf232ed908e6d1234c54c649941ee53a126128c96061`
    - `fixtures/real/checks/estimate_test.go.txt` `20697ad87fb44074255a7fbce85905f5c1d1f297594a9d62bb7c98636fbad8dd`
    - `fixtures/real/checks/events_headers_test.go.txt` `0abca1f5b07309d1fccbbe63a0713c4165827568174216ac7d9886e641a0c662`
    - `fixtures/real/checks/events_limit_test.go.txt` `55193be448b05ce05fa201f2f8bf5dc74c97de5121973a78413139596d6872fb`
    - `fixtures/real/checks/quotedpath_test.go.txt` `f37a7fdd8fe85aa2f9f70a008e0d183f2ded6ef2f2ee51356744f82aa70b6aac`
    - `fixtures/real/checks/shownpath_test.go.txt` `36245f4bdc467a8f643ed73d3709c8fdb8348d10bb94b95659a8cade05aed359`
    - `fixtures/real/checks/suitekey_test.go.txt` `8f84b5f79d4a26f31a583782d2dde75da6e753286aa9c63cfa44bf202799167f`
    - `fixtures/real/checks/variantname_test.go.txt` `9420068a0c6e5d29a109c214f808d06c5441d58a9f3bb23d8c331625673c09e7`
    - `fixtures/real/checks/wslpath_test.go.txt` `ec0ff4f3c7b6399da846cd9ae4c943ebf06d4b86858af6f554c624fc56068c50`
    - `fixtures/real/estimate/cost.go` `f56a089a0e41d1f5a24b8e4ba6b5c82c75f00de5dc3a40edad0caaa027b99c8f`
    - `fixtures/real/events/ISSUE.md` `463be8a7f7ad100d3362d7c2f3de75e3fcd2ce8abc680cf45b475bbabfd8df28`
    - `fixtures/real/events/client.go` `e04f5ad107f0cfac85c2dcbfef647017c2b2f10115b5ba32e7f789ea79d41215`
    - `fixtures/real/movedcache/movedcache.go` `7b4171f7d6d98b767089740604c3e486ec86d1ae6839aa56de397039c32fdee1`
    - `fixtures/real/ops-docker/.env.example` `5ee87d58e52d4b11340b3129c9ec410f3069546e03cdc64a4f1dfb23eed9415b`
    - `fixtures/real/ops-docker/Dockerfile` `2f78c2abfea4778654a9f9b4e227d131132a05e0c04c1e6cf7505ce2289864e4`
    - `fixtures/real/ops-docker/cmd/server/main.go` `6f0b3e04943e54e3e7faeb092a7612132e5d457454c29bf8fa04adc90dca8da9`
    - `fixtures/real/ops-docker/docs/NOTES.md` `25fdb5d5553b47386c50ad4d3907ace76440add717a738a9178667bfac62abe7`
    - `fixtures/real/ops-docker/go.mod` `79c84bfb7597b65473d0fd2c73694c7a477ca2eb6c80b7fc4320c6e5f269a0bd`
    - `fixtures/real/ops-docker/go.sum` `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
    - `fixtures/real/ops-docker/internal/greet/greet.go` `ab4ba37f26bf1e8356d36beed22f3da23c54aae2a7ba355f0cadfda394a049cb`
    - `fixtures/real/ops-docker/scripts/dev.sh` `ba203bd42012536664597c489c178ec7d1c80cd4bc8a2093ca3bf29cd7329400`
    - `fixtures/real/ops-runner/RUNNERS.md` `43e4879e6d8ca67ad96ce1cb328a174a3039162497e8a246e9e20ca7e4db6ab0`
    - `fixtures/real/ops-runner/add.go` `2f49c3ddb40dc6fc8680932ae94c947ebcdb037a4518d5551a824d1825e87f13`
    - `fixtures/real/ops-runner/add_test.go` `6205a91718dc4f299a029750ce99d7d8c78b50746e4496f1728629d4cbf1341f`
    - `fixtures/real/ops-runner/go.mod` `fa41b92becf3cd16785dfd42c2ac28d74df879fa6b5d8356c0768311a1c8a129`
    - `fixtures/real/pathsep/pathsep.go` `844a76e3dd455a70a93e2f3bb88961ee15da54dd88e70976cf258dee518c0995`
    - `fixtures/real/plan-budget/CONSTRAINTS.md` `4c5a0cc1329c44e3a9c36fb2467562d62dcd2c421a83ff0be98c2175eed681a2`
    - `fixtures/real/plan-budget/runner.go` `d13914373accd7559d75d47ea9a3ea0eab172902c769db8fef7bdf227a00ec09`
    - `fixtures/real/plan-unpriced/compare.go` `d2420fe0bcc626f44c6c868be3c618bb83b11b87a468c4fd23058db11ee4a896`
    - `fixtures/real/quotedpath/quotedpath.go` `bd23b0592e678ca6efc33cde772e285b92afcfea30d929cdb960f08f7e5f8b4f`
    - `fixtures/real/research/QUESTIONS-EVAL.md` `d2c5ed02c7057a56a2d7e61d21cb83b2793afe95706365420e591c2ff3a67839`
    - `fixtures/real/research/batch.go.txt` `a9e90b26cecae7348864463f62efa78f62e7f03813d59c8261d9fa887e19c1b4`
    - `fixtures/real/research/eval.go.txt` `b843117abab60966974ac66bd78ace13afea1951dbf9ebc1d6db6897fc62df36`
    - `fixtures/real/review-idle/idle.go` `57c3e459f5f6e3bd0b690e9c1862b3ca0a3833031b97a044a6766d005dcaa3f7`
    - `fixtures/real/review-retry/watch.go` `2770e88112e57af3b60046b94ccd9047e9dfc6e79fa1869edcb9bc77c1977aab`
    - `fixtures/real/shownpath/shownpath.go` `1ee55b33bdb5121c3d834c8e2ec699ce5dbd1bd49aa1c80b742efe8ad2c73b2a`
    - `fixtures/real/suitekey/suitekey.go` `14650093b67f68f0c9dd6eb6feef99a4bad6337dd7a6a6a072c0dcd71ca90226`
    - `fixtures/real/variantname/REQUIREMENTS.md` `20953a949e5bcf1c2ac1a965917af3a66ee27738fba0f2fe15aca82378ebafca`
    - `fixtures/real/variantname/name.go` `54dba3a35302cb7a650a73c52ebf0c8f6b0b920fce4b65fe4c191eaf584c3556`
    - `fixtures/real/wslpath/wslpath.go` `17774db67f422d92b2dac5827af7463bf8531aa92d46c3fb6f20b0277fcff517`
  - Eval ids `4c210` and `7bd77` for `hrn eval diff`. Neither suite has run before, so there is
    no earlier eval to diff against.
- [x] 7. **Number basis:** met. Every number above is measured on these suites except those
  marked inferred (the rows without the two broken tasks, the pooled rows and what newer hrn would
  do, each with its derivation) and the two gate replays, which are local reproductions, not
  eval records. The
  table rows are inferred from the measured numbers by `hrn route from-eval`, with the
  derivation stated per row.
