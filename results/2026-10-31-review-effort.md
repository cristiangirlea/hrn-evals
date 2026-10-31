# Review effort: default against medium

Date: 2026-10-31. Eval `6778d` (batch `20261031-080154-1377fc`), suite `suites/routing/review-effort.json`,
hrn at `dda6719`. Cells run on the owner's Claude Code and Codex plans with each CLI's default
model; no judge, the gates decide (each task has a planted defect the review must name with its
line).

| Variant | Planted defect found | Median cost | Median context |
|---|---|---|---|
| claude, default effort | 5/6 (83%) | $0.19 | 113k |
| claude, medium | 6/6 | $0.21 | 116k |
| codex, default effort | 6/6 | not reported | 115k |
| codex, medium | 6/6 | not reported | 117k |

The one miss was `review-planted-defects` on claude at the default effort (1 of 2).

What it settles (harness #722): medium effort found every planted defect on both backends, so the
review route's medium effort (codex, `gpt-6.1-sol`, set from eval `7bd77` on 2026-10-19) stays.
On these small review tasks effort barely changed what a review costs (about 115k of context
either way), so the much lower recorded medians for medium-effort reviews came mostly from smaller
tasks, not from the effort: lowering effort is not the token lever the run records suggested.
