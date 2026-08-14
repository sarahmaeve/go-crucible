# Unit 01 Wheels of Misfortune

These scenarios start with a production report, not the name of an algorithm.
Use the shared [Wheel worksheet](../../../exercises/wheel/WORKSHEET.template.md)
to record facts, possible causes, and evidence choices.

These scenarios are different from the repository's main Wheel track. They
introduce new CS material and contain everything you need. Their scale checks
use build tags. Normal repository tests therefore stay green, apart from the
existing numbered exercise failures.

The reports and evidence packets are invented teaching examples based on
production systems. They use mechanisms from the unit's production sources,
but they do not reproduce those systems or incidents.

| # | Scenario | Primary idea | Symptom command |
|---|---|---|---|
| [W01](./01-innocent-nested-loop/REPORT.md) | The Innocent Nested Loop | Repeated scan and index construction | `go test -tags=csbridgewheel1 ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop -v` |
| [W02](./02-cache-without-hits/REPORT.md) | The Cache Without Hits | Key equality and the number of retained keys | `go test -tags=csbridgewheel2 ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits -v` |

For each scenario:

1. Read only `REPORT.md`. Write down what you think could cause the problem.
2. Open `CANDIDATE.md`. Choose evidence packets one at a time.
3. Before you open the source, state what the component must do.
4. Run the test that reproduces the symptom.
5. Use benchmarks or a profile to confirm the cause.
6. Make the smallest fix inside your assigned area. Run the tagged and ordinary
   tests again.
7. Write your handoff. Only then compare your reasoning with the debrief.
