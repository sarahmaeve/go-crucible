# W01 Debrief

Read this only after completing the handoff.

## Diagnosis

The report established CPU saturation correlated with endpoint count; it did
not establish a Go runtime or GC regression. Allocation evidence weakens the GC
theory. The controlled experiment shows candidate checks growing linearly with
metadata while event count stays fixed.

`HandleBatch` scans metadata for every event. With `e` events and `m` metadata
records, worst-case work is `Theta(e*m)`. Uniform successful lookups average
roughly `m/2` comparisons, which changes the constant but not the growth.

## Bounded repair

Build `map[string]Metadata` once at the beginning of `HandleBatch`, assigning
records from first to last so duplicates remain last-record-wins. Perform one
lookup per event and increment `CandidateChecks` once per lookup. The expected
time becomes `Theta(m+e)` with `Theta(m)` additional space.

Do not move index construction inside the event loop. That preserves the same
bad growth behind map syntax.

## Verification

Run:

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop -v
go test -tags=csbridgewheel1 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop \
  -count=1 -v
```

The ordinary test protects duplicate and missing-key semantics. The tagged
test protects the growth mechanism without relying on a machine-specific
latency threshold. The benchmark and profile confirm the measured consequence.

## Production follow-up

Record endpoint and event cardinality independently in load tests. A single
“batch size” dimension would have missed the scale threshold that activated the
defect.
