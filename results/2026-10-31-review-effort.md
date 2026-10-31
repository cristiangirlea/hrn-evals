# Review effort: default against medium (a small run, no claim)

Date: 2026-10-31. Run on the owner's request (harness #722), on the owner's Claude Code and Codex
plans. **This run is below the publication gate (rules 1 and 5), so it supports no claim of a
quality or cost difference.** It is a first look; the gate's checklist is at the end.

## Pins

- hrn `0.2.0` at harness commit `dda6719`.
- Claude Code `2.1.284`, model `claude-opus-5-5` in every claude cell (the CLI's default; read
  from the cells' run records).
- codex-cli `0.165.0-alpha.1.2`, model `gpt-6.1-terra` in every codex cell (the CLI's default;
  read from the cells' run records).
- Suite `suites/routing/review-effort.json` at commit `1fef46a`.
- Eval `6778d`, batch `20261031-080154-1377fc`, 24 cells.
- Fixture SHA-256:
  - `fixtures/purposes/review/review.go` `2bc49730f628eb7ece8d73dc913380f4190a17c31f40156d5d6d3f1a3baa4811`
  - `fixtures/real/review-idle/idle.go` `57c3e459f5f6e3bd0b690e9c1862b3ca0a3833031b97a044a6766d005dcaa3f7`
  - `fixtures/real/review-retry/watch.go` `2770e88112e57af3b60046b94ccd9047e9dfc6e79fa1869edcb9bc77c1977aab`

## Results (all **measured**, this suite)

| Variant | Gates passed | Median cost | Median context |
|---|---|---|---|
| claude, default effort | 5/6 | $0.19 (provider-reported) | 113,439 |
| claude, medium | 6/6 | $0.21 (provider-reported) | 116,081 |
| codex, default effort | 6/6 | not reported by the CLI | 114,951 |
| codex, medium | 6/6 | not reported by the CLI | 116,686 |

- The one failure was `review-planted-defects` on claude at the default effort (1 of 2).
- One of the six codex-medium cells committed on its own although every prompt says "Do not run
  git."; its gates still passed and it is counted like the others.
- No run was dropped.

## What it does and does not show

- With three tasks and two repeats per arm, a one-cell difference (5/6 against 6/6) is within
  chance, and no bootstrap interval was computed: **no quality difference is claimed** either way.
- Median context barely moved with effort on these three small tasks (**measured**, above). It
  does not show why the run records' medium-effort reviews were cheaper; task size was not
  measured here.
- Codex at the default effort also passed 6/6, so this run does not show that medium is needed,
  only that it found every planted defect here.

**Decision (owner's issue #722):** no change to the routing table. Review stays on codex at medium
effort, as set from eval `7bd77` on 2026-10-19. A claim about effort and review quality or cost
needs a run that meets the gate: at least six review tasks and three repeats, with an interval.

## Publication gate

- [ ] 1. At least six cases, three repetitions per arm, paired identical tasks. *Three cases, two
      repetitions; the arms are paired.*
- [x] 2. Every run gate-scored; none dropped.
- [ ] 3. Provider-only unweighted token usage; costs reported or marked estimated. *Costs above are
      provider-reported for claude, missing for codex; no token totals are claimed.*
- [x] 4. Negative and no-op cases included in the aggregate.
- [ ] 5. 95% case-bootstrap interval, 10,000 resamples, excludes zero. *Not computed; no
      difference is claimed.*
- [x] 6. All reproduction pins and eval ids recorded.
- [x] 7. Every number labelled measured, inferred or verified.
