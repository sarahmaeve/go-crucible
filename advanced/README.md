# Advanced Concept Explorations

This track sits between the numbered debugging exercises and
[Computer Science Through Production Systems](../cs-prod-bridge/README.md).
It is for implementation choices that deserve runnable evidence but are not
well represented by either a planted bug or a full computer-science unit.

An exploration starts with a production question, makes the relevant behavior
observable in small programs, and ends with a decision that depends on the
learner's measurements. The backing code is correct on the initial checkout:
there is no intended failing test and no canonical one-line repair.

## Explorations

| Exploration | Question |
|---|---|
| [JSON codecs](./json-codecs/README.md) | When does replacing `encoding/json` with ByteDance Sonic produce enough end-to-end value to justify the compatibility and maintenance cost? |

## How these differ from the other tracks

| Track | Starting point | Primary outcome |
|---|---|---|
| Numbered crucible | A reproducible defect | Diagnose and repair it |
| Advanced exploration | A plausible engineering choice | Measure, test assumptions, and make a bounded recommendation |
| CS-production bridge | A transferable CS model | Explain and apply the model across production systems |

## Exploration quality bar

Every exploration should:

- state a question rather than advertise a preferred implementation;
- include representative counterexamples where the proposed optimization may
  not help;
- separate API compatibility from performance measurements;
- include passing correctness tests and reproducible benchmarks;
- measure an end-to-end path in addition to an isolated primitive;
- identify cold-start, dependency, deployment, and upgrade costs where they
  apply; and
- finish with a falsifiable recommendation tied to a named workload.

Elapsed time is evidence, not a correctness assertion. Tests must not fail
because one implementation happens to be faster on a particular machine.

From the repository root, verify or benchmark the JSON exploration with:

```bash
make advanced-json-check
make advanced-json-check-legacy
make advanced-json-bench
make advanced-json-bench-legacy
make advanced-json-cold
make advanced-json-retention
```

The JSON targets default to Go 1.27.0. The `-legacy` targets rebuild the same
workloads with `GOEXPERIMENT=nojsonv2`, providing a temporary comparison with
the pre-1.27 `encoding/json` engine. Override `JSON_LAB_TOOLCHAIN` deliberately
when investigating another toolchain.
