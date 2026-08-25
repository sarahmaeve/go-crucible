# W12 Candidate Guide

## State the contract before diagnosing the incident

Write the four possible combinations of whether a key is actually present and
what `MayContain` reports. For each combination, state what the read path
should do: skip the exact index, check it, return a record, or identify a
correctness defect.

Then assess these possible explanations, noting whether each one can actually
produce the reported symptom:

- the new key never reached this process's exact index;
- the writer and reader converted the key to bytes differently;
- the filter and exact index were built from different data generations; and
- the filter returned an ordinary false positive.

For each explanation, name one observation that supports it and one that makes
it less likely. Do not use “Bloom filters are probabilistic” to explain a
false negative. A correctly built Bloom filter can produce false positives,
but it does not reject keys that were added to it.

## Choose evidence

The [evidence index](./evidence/README.md) describes four packets without
showing their contents. Before opening one, write:

1. the question the packet should answer;
2. the different outcomes expected under your remaining explanations; and
3. why this packet is the most useful next step.

You do not need every packet.

## Preserve the complete read path

A repair must preserve all of these outcomes:

- a “definitely absent” answer from a matching filter skips the exact index;
- a filter positive performs an exact check rather than supplying a value;
- a successful reload makes the new exact data and its filter visible
  together;
- a failed reload retains the complete previous snapshot; and
- the snapshot records both the data generation and the filter generation.

Do not rely on a retry, a sleep, or a claim about the expected false-positive
rate. You can check the required behavior with a repeatable test.

## Reproduce the failure

From the `cs-prod-bridge` directory, run the ordinary tests:

```bash
go test ./05-bloom-filters/wheel/12-filter-from-yesterday -count=1 -v
```

Then enable the symptom tests:

```bash
go test -tags=csbridgewheel12 \
  ./05-bloom-filters/wheel/12-filter-from-yesterday \
  -count=1 -v
```

The tagged tests use a controlled membership implementation with no false
positives. It follows the same rule for safe “definitely absent” answers, so
the tests can focus on whether the filter and exact index come from the same
generation rather than on hashing behavior.

After you identify the cause, change only `catalog.go`. Run both commands
again. Finish with a three-minute handoff: explain the first rule the code
violates, the evidence that ruled out an ordinary false positive, why the
repair publishes one matching snapshot, and what happens if preparation
fails.

Then read [DEBRIEF.md](./DEBRIEF.md).
