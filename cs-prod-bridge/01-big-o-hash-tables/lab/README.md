# Lab: Scan or Index?

The lab contains two implementations that return the same results but do
different amounts of work:

- `EnrichWithScan` searches the metadata slice again for every event.
- `MetadataIndex.Enrich` looks up each exact key in a prebuilt map.

Both versions use a last-record-wins rule for duplicate IP metadata. Both use
`Found == false` when an event has no matching metadata.

## 1. Check the results

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/lab -v
```

Read the table tests. Which checks concern correct results? Do any checks
concern how the work grows?

## 2. Predict how the work will grow

The benchmark names show the number of events and metadata records separately.
Before you run the benchmark, predict how much each implementation will change
from `small` to `medium` to `large`.

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/lab \
  -run '^$' -bench . -benchmem -count=5
```

Each workload has three benchmarks:

- `scan` searches metadata for every event.
- `index-end-to-end` builds an index and uses it for one batch.
- `index-reused` uses an index that is already built.

The reused-index benchmark measures only lookups. It does not include the time
or memory needed to build the index. Do not report it as the total cost.

If you have `benchstat`, save two runs and compare them. The official
[`testing` benchmark documentation](https://pkg.go.dev/testing#hdr-Benchmarks)
explains how the benchmark tool works. The
[`benchstat` documentation](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
explains how to compare several measurements instead of relying on one run.

## 3. Change one input at a time

In `enrichment_bench_test.go`, keep the event count fixed and double only the
metadata count. Then keep the metadata count fixed and double only the event
count. Compare each result with `Theta(e*m)` and expected `Theta(e+m)`.

Try one or two metadata records. Is the scan faster? Big-O predicts growth. It
does not predict the input size at which one design becomes faster or the size
of fixed costs.

## 4. Find where the scan uses CPU time

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/lab \
  -run '^$' -bench 'BenchmarkEnrichment/large/scan$' \
  -benchtime=2s -cpuprofile /tmp/cs-bridge-scan.pprof

go tool pprof -top /tmp/cs-bridge-scan.pprof
```

Before you open the profile, predict which function will use the most CPU time.
The profile shows where the program does the work. The complexity model
explains why that work grows.

## 5. Test the limits of the index

Answer without changing code:

- If the service rebuilds the metadata index for every event, how does that
  change the total work?
- If the map key includes a timestamp, what determines how many keys stay in
  memory?
- If callers need every IP in a prefix, can an exact-key map answer that query
  directly?
- If reads and updates happen at the same time, who owns updates? How does the
  service synchronize access?
