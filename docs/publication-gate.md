# Publication gate

Apply these rules to every results page before claiming a saving or quality difference.

1. **Cases, repetitions and pairing:** use at least six cases and three repetitions per arm, with every arm running the same tasks, paired by case and repetition. This separates a repeatable difference from a lucky run.
2. **Complete scoring:** score the suite's gates for every run and drop no run, including failures and incomplete runs. This prevents selection of only favourable outcomes.
3. **Usage and cost:** use only provider-reported input, output, cache-read and cache-write usage from hrn run records for token claims, summing those fields without price weighting; costs must be provider-reported or explicitly marked estimated. This keeps token counts distinct from pricing assumptions.
4. **Negative and no-op cases:** keep these cases in the aggregate. This includes the overhead when there is nothing to improve or the approach makes things worse.
5. **Uncertainty:** report a 95% bootstrap interval for the claimed paired difference over cases, using 10,000 resamples and retaining each sampled case's paired repetitions; claim a difference only when the interval excludes zero. This tests whether the result survives variation across tasks rather than treating repetitions as independent cases.
6. **Reproduction pins:** pin the hrn version and commit, backend CLI versions, model ids, suite commit, fixture SHA-256 hashes and eval ids usable with `hrn eval diff`. This lets another run identify exactly what was compared.
7. **Number basis:** label every number **measured** (this suite), **inferred** (derived from measured numbers, with the derivation) or **verified** (seen on real runs over a stated window). This states what evidence each number actually has.

Copy this checklist into each results page; tick only fully met rules and explain gaps.

- [ ] 1. At least six cases, three repetitions per arm, paired identical tasks.
- [ ] 2. Every run gate-scored; none dropped.
- [ ] 3. Provider-only unweighted token usage; costs reported or marked estimated.
- [ ] 4. Negative and no-op cases included in the aggregate.
- [ ] 5. 95% case-bootstrap interval, 10,000 resamples, excludes zero.
- [ ] 6. All reproduction pins and eval ids recorded.
- [ ] 7. Every number labelled measured, inferred or verified.
