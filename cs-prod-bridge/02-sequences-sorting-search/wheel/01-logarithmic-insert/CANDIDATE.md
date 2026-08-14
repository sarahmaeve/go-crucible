# W01 Candidate Guide

## What you may change

Change only the function that applies an incoming batch to the current
snapshot. The returned snapshot must:

- return records with unique keys in increasing string order;
- keep the last incoming record when a key appears more than once;
- leave both input slices unchanged; and
- return a slice with its own backing array.

Do not change update delivery, readiness policy, or query serving.

Do not open `index.go`, either test file, or `DEBRIEF.md` until the evidence
points to this component.

## Choose evidence

The [evidence index](./evidence/README.md) names the available packets without
showing their contents. Before you open one, write:

1. the possible cause or input property that it can test;
2. what each likely result would mean; and
3. why this is the most useful packet to read next.

You do not need every packet.

## When to open the code

Open the source after you can answer these questions:

- What are `n`, the number of current records, and `m`, the number of incoming
  records?
- Does the position of incoming keys in the sorted result affect refresh time?
- Does the evidence point to comparisons, allocation, or record movement?
- Which order does binary search require?
- What is the total cost of finding a position and putting a record there?

From the `cs-prod-bridge` directory, run the ordinary tests for correct results:

```bash
go test ./02-sequences-sorting-search/wheel/01-logarithmic-insert -v
```

Run the large front-insertion case:

```bash
go test -tags=csbridgewheel3 \
  ./02-sequences-sorting-search/wheel/01-logarithmic-insert \
  -run TestRefreshWriteGrowthOnFrontHeavyBatch -count=1 -v
```

If you need running-time measurements and a CPU profile:

```bash
go test -tags=csbridgewheel3 \
  ./02-sequences-sorting-search/wheel/01-logarithmic-insert \
  -run '^$' -bench BenchmarkRefreshFrontHeavy -benchmem

go test -tags=csbridgewheel3 \
  ./02-sequences-sorting-search/wheel/01-logarithmic-insert \
  -run '^$' -bench BenchmarkRefreshFrontHeavy -benchtime=2s \
  -cpuprofile /tmp/cs-bridge-wheel3.pprof
go tool pprof -top /tmp/cs-bridge-wheel3.pprof
```

Make the smallest change that satisfies the tagged growth check. Preserve the
ordering, duplicate-key behavior, and unchanged inputs described above.

The tests also define these counters:

- `Inserted` counts an incoming record if its key has not appeared in the
  current snapshot or earlier in the batch;
- `Replaced` counts every other incoming record, including later duplicates
  within the batch;
- `Inserted + Replaced` therefore equals the incoming batch size; and
- `SnapshotWrites` counts writes that construct the returned snapshot. This
  includes initial copies, replacements, inserted records, and movement that
  opens a gap. If code constructs the final output in one pass, it counts one
  write for each output record.

Run both test modes again. Then write a three-minute handoff before you read
[DEBRIEF.md](./DEBRIEF.md).
