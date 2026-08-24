# Unit 04 Wheels of Misfortune

Each Wheel begins with an incoming report. Before you open the source code,
list several possible causes. Then choose evidence that can tell them apart.

Use the shared [investigation worksheet](../../../exercises/wheel/WORKSHEET.template.md)
to record what nodes and edges mean, possible explanations, and evidence
choices.

| Wheel | Symptom | Public source | Failure-test build tag |
|---|---|---|---|
| [W01: The Build That Loops Back](./01-build-that-loops-back/REPORT.md) | Validation accepts a small stage graph, then recursive conversion repeats the same stages | [BuildKit PR #999](https://github.com/moby/buildkit/pull/999) and [PR #4567](https://github.com/moby/buildkit/pull/4567) | `csbridgewheel8` |
| [W02: Unknown Is Not Independent](./02-unknown-is-not-independent/REPORT.md) | Work runs at the same time after dependency analysis could not reach a conclusion | [Prometheus PR #15560](https://github.com/prometheus/prometheus/pull/15560) | `csbridgewheel9` |
| [W03: Two Resources, One URN](./03-two-resources-one-urn/REPORT.md) | A deletion preview removes a dependency before the resource that still refers to it | [Pulumi PR #19179](https://github.com/pulumi/pulumi/pull/19179) | `csbridgewheel10` |

From the `cs-prod-bridge` directory, first run the ordinary tests:

```bash
go test ./04-build-dependency-graphs/wheel/... -count=1
```

Each build tag turns on tests for one reported failure. Keep your changes
inside that Wheel. Preserve the required behavior and make the tagged tests
pass. Do not change the examples to remove the difficult graph.

The local services, inputs, measurements, and traces are invented. The debriefs
say which facts come from each public change and which details belong only to
the exercise.
