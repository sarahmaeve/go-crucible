# W01 Debrief

Read this only after completing the handoff.

## Cause

`HandleBatch` scans the metadata for every event. This repeated scan causes the
growth problem.

The report shows that CPU use changes with endpoint count. It does not show a
Go runtime or garbage-collection regression. The allocation evidence makes the
garbage-collection explanation less likely. In the controlled experiment,
candidate checks grow with metadata count while event count stays fixed.

With `e` events and `m` metadata records, the worst-case work is `Theta(e*m)`.
If successful lookups are spread evenly across the metadata, each scan checks
approximately `m/2` records on average. This changes the fixed factor, but it
does not change how the work grows.

## Smallest repair

Build one `map[string]Metadata` at the start of `HandleBatch`. Add records from
first to last so duplicate keys still use the last record. Do one map lookup
for each event, and increment `CandidateChecks` once for each lookup.

The expected time becomes `Theta(m+e)`. The map stores one logical entry for
each distinct metadata key. Its capacity hint is `m`, the number of metadata
records.

Do not build the index inside the event loop. That would repeat the index work
for every event and keep the same growth problem.

## Check the repair

Run:

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop -v
go test -tags=csbridgewheel1 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop \
  -count=1 -v
```

The ordinary test checks the rules for duplicate and missing keys. The tagged
test checks how the work grows without using a machine-specific latency limit.
The benchmark and profile show the effect on running time and CPU work.

## What to measure in production

Record endpoint count and event count separately in load tests. One “batch
size” measurement would hide which input caused the change at scale.
