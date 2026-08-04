# Exploration Lab: Scan or Index?

The lab contains two correct enrichment implementations:

- `EnrichWithScan` searches the metadata slice for every event.
- `MetadataIndex.Enrich` performs exact-key lookups in a prebuilt map.

Both use an explicit last-record-wins policy for duplicate IP metadata. Missing
metadata is represented by `Found == false`.

## 1. Confirm semantics

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/lab -v
```

Read the table tests. Identify which assertions concern correctness and which,
if any, concern scalability.

## 2. Predict growth

The benchmarks name event and metadata cardinality separately. Before running
them, predict the relative change for each implementation from `small` to
`medium` to `large`.

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/lab \
  -run '^$' -bench . -benchmem -count=5
```

There are three measurements per workload:

- `scan` performs the event-to-metadata nested work.
- `index-end-to-end` builds an index and uses it once.
- `index-reused` measures event enrichment with an already-built index.

The reused benchmark is useful but incomplete. Quoting it alone would hide
construction and retention costs.

If `benchstat` is installed, save two runs and compare them. The official
[`testing` benchmark documentation](https://pkg.go.dev/testing#hdr-Benchmarks)
explains the harness; [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
compares distributions rather than relying on one run.

## 3. Change one dimension

In `enrichment_bench_test.go`, hold events constant and double only metadata.
Then hold metadata constant and double only events. Relate each result to
`Theta(e*m)` versus expected `Theta(e+m)`.

Try a metadata size of one or two. Does the simpler scan win? Big-O predicts
growth, not the crossover point or constant factors.

## 4. Profile the large scan

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/lab \
  -run '^$' -bench 'BenchmarkEnrichment/large/scan$' \
  -benchtime=2s -cpuprofile /tmp/cs-bridge-scan.pprof

go tool pprof -top /tmp/cs-bridge-scan.pprof
```

Predict which function will dominate before opening the profile. A profile
localizes work; the complexity model explains why that work grows.

## 5. Challenge the index

Answer without changing code:

- If the metadata snapshot is rebuilt for every single event, what happened to
  the advantage?
- If a timestamp is added to the map key, what controls retained cardinality?
- If callers need every IP in a prefix, is exact-key hashing still the natural
  operation?
- If updates and reads occur concurrently, what ownership or synchronization
  contract is missing?

