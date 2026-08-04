# Unit 01 Wheels of Misfortune

These scenarios begin with production evidence rather than an algorithm name.
Use the shared [Wheel worksheet](../../../exercises/wheel/WORKSHEET.template.md)
to maintain observations, hypotheses, and evidence choices.

Unlike the repository's main Wheel track, these scenarios introduce new CS
subject matter and are self-contained. Their deliberate scale checks use build
tags, so normal repository tests remain green outside the existing numbered
exercise failures.

The reports and evidence packets are synthetic, production-shaped teaching
artifacts. They exercise mechanisms found in the unit's first-party production
sources but do not claim to reproduce those companies' systems or incidents.

| # | Scenario | Primary idea | Symptom command |
|---|---|---|---|
| [W01](./01-innocent-nested-loop/REPORT.md) | The Innocent Nested Loop | Repeated scan and index construction | `go test -tags=csbridgewheel1 ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop -v` |
| [W02](./02-cache-without-hits/REPORT.md) | The Cache Without Hits | Key equality and retained cardinality | `go test -tags=csbridgewheel2 ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits -v` |

For each scenario:

1. Read only `REPORT.md` and write an initial model.
2. Continue to `CANDIDATE.md` and choose evidence packets one at a time.
3. State the expected component contract before opening source.
4. Reproduce the deterministic symptom.
5. Use benchmarks or a profile to confirm the mechanism.
6. Make a bounded repair and rerun both tagged and ordinary tests.
7. Compare your reasoning with the debrief only after writing a handoff.
