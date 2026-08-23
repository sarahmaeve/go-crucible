# Lab: Skip the Cold Segment

This lab puts one Bloom filter in front of each immutable segment's exact
sorted-key index. Its contract is deliberately narrow:

```text
false: the segment definitely lacks the key; skip its exact index
true:  the segment may contain the key; ask its exact index
```

The filter never returns a record. `LookupSegments` stops only when the sorted
index confirms a match.

## 1. Predict before running

Open [WORKSHEET.md](./WORKSHEET.md) and record directional predictions for:

- all-hit versus mostly-absent workloads;
- cheap versus expensive exact checks;
- 10%, 1%, and 0.1% false-positive targets;
- 1x, 2x, and 10x planned capacity; and
- 1,000 distinct absent keys versus one engineered false-positive absent key
  retried 1,000 times.

Name the variables rather than calling all of them `n`:

- `s`: candidate segments visited by a lookup;
- `n`: distinct keys inserted in each filter;
- `m`: bits in each filter;
- `k`: maximum bit probes per filter query;
- `a`: fraction of candidate segments that truly lack the key; and
- `p`: false-positive probability for an absent candidate under the model.

Without filters, a lookup performs one exact check for each candidate segment
it visits. With filters, expected exact checks are approximately
`s(1-a)+sap`. Every candidate still pays the filter cost.

## 2. Run the experiment matrix

From the `cs-prod-bridge` module directory:

```bash
go run ./05-bloom-filters/lab/cmd/experiment
```

The runner prints operation counts rather than hiding them behind passing test
assertions. Read each column as follows:

- `ABSENT`: fraction of logical queries absent from every segment;
- `TARGET`: requested sizing target;
- `MODELED-P`: modeled false-positive probability after representation rounding,
  using the actual distinct insertions in each filter; it is not a guarantee for
  a finite sample;
- `OBS-P`: observed false positives divided by all exact-source nonmembers
  queried; true hits are excluded from this denominator;
- `LOAD`: actual distinct keys divided by planned capacity;
- `DENSITY` and `BYTES`: fraction of bits set and total filter storage;
- `UNIQUE-ABS`: distinct absent query keys;
- `PROBES`: bit positions actually inspected, including early exits;
- `EXACT`, `AVOIDED`, and `FP`: exact checks, skipped checks, and exact misses
  after possible matches; and
- `COST`: `filter-path/exact-only` work under the stated exact-check cost.

The cost model charges one unit per inspected bit position and either 1 or 100
units per exact check. It does not attempt to price the two full-key hashes and
is not elapsed time. It makes the selection boundary visible: a filter can
lose when exact checks are cheap and win on the same operation counts when
exact checks are expensive.

Compare these rows in order:

1. `single-segment-all-hits` has no negative candidate to skip. The filter can
   only add work.
2. The `mix-*` rows hold candidate segments constant while changing query
   mix. Notice that a successful lookup in the final segment still has seven
   absent segment candidates. Query hit rate and candidate absence rate are
   related but not identical.
3. The `target-*` rows trade filter bytes and probes for fewer false positives.
4. The `capacity-*` rows insert the same actual keys into filters sized for
   fewer keys. At 10x load, present keys remain correct while avoidance
   collapses.
5. The retry rows use the same filter. Uniform absent traffic spreads false
   positives across keys; one engineered false-positive key repeats its exact
   work on every retry.

The matrix uses a fixed educational hash so its counts reproduce across runs.
`BuildMembership`, the normal API, uses process-local `hash/maphash` seeds.
Neither choice silently defines a persistent format: a real format must name
and version its key encoding, hash, dimensions, and represented generation.

## 3. Change one workload variable

Use `-custom` to run your own workload. This all-hit, one-segment case has
nothing for the filter to rule out:

```bash
go run ./05-bloom-filters/lab/cmd/experiment \
  -custom -segments=1 -absent=0 -exact-cost=100
```

Now change only `-absent=1`. Then compare cheap and expensive exact work by
changing only `-exact-cost`:

```bash
go run ./05-bloom-filters/lab/cmd/experiment \
  -custom -segments=8 -absent=1 -exact-cost=1

go run ./05-bloom-filters/lab/cmd/experiment \
  -custom -segments=8 -absent=1 -exact-cost=100
```

Capacity is a contract. Keep 200 actual keys per segment but size for only 20:

```bash
go run ./05-bloom-filters/lab/cmd/experiment \
  -custom -keys=200 -planned=20 -absent=1
```

To expose query shape, compare the default uniform absent workload with:

```bash
go run ./05-bloom-filters/lab/cmd/experiment \
  -custom -absent=1 -repeat-absent
```

For each pair, explain the change using counts before looking at nanoseconds.

## 4. Follow the implementation contract

`BuildMembership` requires distinct keys and derives standard dimensions from
planned distinct items and a target false-positive probability:

```text
m = -n ln(p) / (ln 2)^2
k = round((m/n) ln 2)
```

Storage rounds to whole 64-bit words. The code derives `k` positions from two
hash values. `Filter.query` may stop at the first zero bit, so `FilterProbes`
counts actual positions inspected while `ProbeCount` is the maximum. Filter
statistics recompute the modeled false-positive probability from the rounded
dimensions, probe count, and actual distinct insertions.

`Config.MaxBytes` bounds the filter allocation before construction. Zero uses
the lab's 64 MiB default. Production callers should choose a smaller limit from
their memory budget rather than allowing untrusted sizing inputs to request an
unbounded allocation. Custom experiments expose the same setting as
`-max-bytes`; for example, `-custom -keys=200 -planned=200 -max-bytes=8`
fails before allocating the requested filter.

When every required bit is set, `LookupSegments` calls `lookupExact`; its
binary search owns the answer. Run the correctness and accounting tests:

```bash
go test ./05-bloom-filters/lab -count=1 -v
```

The package also contains executable examples, binary-key fuzz properties, and
a concurrent immutable-read test. Normal `go test` executes the examples and
fuzz seed corpus. Exercise additional generated inputs and verify read safety
with:

```bash
go test ./05-bloom-filters/lab -fuzz=FuzzBuildMembershipNoFalseNegatives -fuzztime=5s
go test ./05-bloom-filters/lab -fuzz=FuzzFilteredAndExactSegmentsAgree -fuzztime=5s
go test -race ./05-bloom-filters/lab -count=1
```

For an all-absent workload, verify these identities:

```text
ExactChecks + ExactChecksAvoided = CandidateSegments
FalsePositives = PossibleMatches = ExactChecks
```

In a mixed workload, `PossibleMatches / all queries` is not a false-positive
rate because possible matches include true hits.

## 5. Benchmark after explaining the counts

```bash
go test ./05-bloom-filters/lab -run '^$' -bench . -benchmem -count=5
```

`BenchmarkSegmentLookups` compares filtered and exact all-absent lookups as
segment and key counts grow. The exact index is in memory; this does not
simulate disk, network, decompression, or cache misses, so timings are not a
claim about Cassandra or another storage engine.

`BenchmarkBuildMembership` measures the time and temporary allocations needed
to build or rebuild immutable filters at several cardinalities. Its
`filter-bytes` metric reports the resulting long-lived bit-array size; benchmark
allocation totals also include transient duplicate validation and hashing work.

`BenchmarkFilterCapacity` holds planned capacity fixed while inserting one,
two, and ten times that count. It reports density beside time and allocations.

## 6. State the production boundary

`Segment` constructs exact data and its filter together. A real system must
publish them under one generation. Reading new exact data through an old
filter can create an operational false negative even though the abstract data
structure has no false negatives.

The toy implementation cannot supply universal answers to the production
policy questions below. Answer them for the system you are evaluating, using
the experiment counts where they apply:

1. What exact work does a negative avoid?
2. What policy will enforce distinct-key capacity and trigger a rebuild?
3. If the filter is persisted, which key encoding, hash version, dimensions,
   and represented-data generation travel with it?
4. What does a reader do when it cannot understand that metadata?
5. Which measurements show whether absent candidates and exact checks are
   expensive enough to repay filtering?
