# W01 Candidate Guide

## What you may change

Change only the function that applies an incoming batch to the current
snapshot. A successful refresh must:

- return records with unique keys in increasing string order;
- keep the last incoming record when a key appears more than once;
- leave both input slices unchanged; and
- return a slice with its own backing array.

Do not change update delivery, readiness policy, or query serving.

Until you have narrowed the cause, do not open `index.go`, either test file,
or `DEBRIEF.md`.

## Choose the next evidence packet

The [evidence index](./evidence/README.md) names available packets without
revealing their contents. Before opening one, write down:

1. the possible cause or input characteristic it examines;
2. what each likely result would tell you; and
3. why it is the best next packet to read.

You do not need every packet.

## Before opening the source

Open the source after you can explain:

- `n`, the number of records already in the snapshot;
- `m`, the number of incoming records;
- whether the location of incoming keys in the sorted result affects refresh
  time;
- whether the measurements point to comparisons, allocation, or copying;
- the ordering that binary search requires; and
- the combined cost of finding an insertion point and opening space for the
  record.

From the `cs-prod-bridge` directory, run ordinary correctness tests:

```bash
go test ./02-sequences-sorting-search/wheel/01-logarithmic-insert -v
```

Run the large front-insertion case:

```bash
go test -tags=csbridgewheel3 \
  ./02-sequences-sorting-search/wheel/01-logarithmic-insert \
  -run TestRefreshWriteGrowthOnFrontHeavyBatch -count=1 -v
```

If you need timing and a CPU profile:

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

Change `Refresh` so that a large batch of keys near the front no longer causes
repeated copying of the existing suffix. Preserve the ordering, duplicate-key
behavior, and unchanged inputs described above.

The tests also check these counters:

- `Inserted` counts an incoming record when its key has not appeared in the
  current snapshot or earlier in the incoming batch;
- `Replaced` counts every other incoming record, including later duplicates
  within the batch;
- `Inserted + Replaced` therefore equals the incoming batch size; and
- `SnapshotWrites` counts the record writes used to construct the returned
  snapshot: initial copies, replacements, inserted records, and records shifted
  to open a gap. An implementation that constructs the final output in one
  pass counts one write for each output record.

Verify both test modes, then write a three-minute handoff before reading
[DEBRIEF.md](./DEBRIEF.md).
