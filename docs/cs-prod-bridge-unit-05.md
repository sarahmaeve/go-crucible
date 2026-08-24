# Unit 05 Research and Design: Bloom Filters and Approximate Membership

**Status:** foundations lesson and exploration lab implemented; Wheels designed
but not yet implemented

**Research reviewed:** 2026-08-18

**Target toolchain:** Go 1.27.x

## Decision

Build Unit 05 around Bloom filters as **negative lookup accelerators**.

The unit's central claim is:

> A Bloom filter makes absence cheap. A negative answer may skip expensive
> work; a positive answer means only that the exact source still needs to be
> checked.

The first production case is Apache Cassandra. One SSTable stores an immutable
set of partition keys and a `Filter.db` component representing those keys. A
point read can use the filter to avoid an SSTable whose partition-key filter
says no. Cassandra exposes a per-table false-positive target, persists the
filter with the SSTable, rebuilds it when an SSTable is rewritten, and exports
false-positive and memory metrics. Those properties make the whole lifecycle
visible instead of reducing the topic to a formula.

VictoriaLogs supplies a Go-native production workload: word and phrase filters
can skip compressed log blocks that cannot match. Git's changed-path commit
graph supplies a second persisted-format example, including filter versions and
mixed-version compatibility. Go Ethereum supplies the counterexample: a fixed
2,048-bit protocol Bloom can become dense enough that log filtering does much
less useful skipping. VictoriaMetrics `vmagent` is a boundary case in which a
Bloom-backed unique-series limiter intentionally makes the policy approximate.

Kubernetes is not a production example for this unit. A review of the current
Kubernetes core repository and first-party component documentation found no
direct Bloom-filter implementation in the API server, scheduler, controller
manager, kubelet, or the first-party storage path. Software that runs on
Kubernetes does not establish that Kubernetes itself uses the structure.

The lesson should not expand into sketches, approximate cardinality, or a
survey of probabilistic structures. Those topics have different contracts and
different error semantics. A focused unit gives the learner time to reason
about the operational assumptions hidden behind the familiar phrase “no false
negatives.”

## Working title and lesson promise

**Working lesson title:** Bloom Filters: Make Absence Cheap

**Lesson promise:** Given a workload with many expensive negative membership
checks, state the filter contract, size a filter from capacity and an error
budget, place it in front of an exact source, and design its lifecycle so stale
or incompatible metadata cannot turn a modeled one-sided error into a
correctness defect.

## Concept-to-production map

| Concept | Production evidence | Teaching use |
|---|---|---|
| Negative lookup avoidance | Cassandra `Filter.db` stores an SSTable partition-key Bloom filter | Primary case and cost model |
| Tunable false-positive target | Cassandra `bloom_filter_fp_chance` table property | Memory-versus-I/O capacity exercise |
| Filter and data share a lifecycle | Cassandra creates new SSTables during flush/compaction and removes old ones afterward | Generation and rebuild contract |
| Observable effectiveness | Cassandra Bloom false-positive, disk-space, off-heap-memory, and SSTables-per-read metrics | Production dashboard design |
| Block skipping | VictoriaLogs skips blocks without a requested word or phrase | Go-native second domain |
| Persisted algorithm version | Git commit-graph changed-path Bloom filters have format versions | Encoding and compatibility boundary |
| Saturation | Ethereum block headers contain fixed-size log Blooms; Geth still checks exact logs after a possible match | Why capacity is a contract, not a hint |
| Approximate policy | VictoriaMetrics `vmagent` documents approximate cardinality limits backed by its Bloom-filter limiter | Error direction changes product behavior |
| Process-local hashing | Go `hash/maphash.Seed` cannot be serialized or recreated in another process | Ephemeral implementation only |

## Learning outcomes

After the foundations lesson, a learner should be able to:

- distinguish exact membership from approximate membership;
- say `MayContain` rather than `Contains` and interpret each result correctly;
- describe insertion and lookup using a bit array and several derived bit
  positions;
- explain how collisions create false positives;
- state the assumptions under which a standard Bloom filter has no false
  negatives;
- derive expected false-positive probability from inserted cardinality \(n\),
  bit count \(m\), and probe count \(k\);
- choose \(m\) and \(k\) from a planned capacity and target probability;
- explain why extra hash probes eventually increase, rather than decrease, the
  false-positive rate;
- compare filter cost with the exact work it may avoid;
- define canonical key bytes, hash algorithm/version, seed, capacity, and data
  generation as part of a persisted-filter format;
- explain why a standard Bloom filter cannot safely delete one key;
- identify over-capacity, stale-generation, concurrency, and repeated-query
  failure modes;
- choose metrics that distinguish a busy filter from a useful filter; and
- reject a Bloom filter when queries are mostly hits, the exact set is already
  cheap, or no exact fallback exists.

## Formal model

Use these workload variables throughout the unit:

| Symbol | Meaning |
|---|---|
| \(n\) | number of distinct keys inserted into one filter |
| \(m\) | number of bits in that filter |
| \(k\) | bit positions examined per add or query |
| \(q\) | membership queries in an observation window |
| \(a\) | fraction of queried candidates that are truly absent |
| \(s\) | immutable segments considered by one logical lookup |
| \(p\) | false-positive probability for an absent queried key under the model |
| \(C_f\) | cost of one filter query |
| \(C_e\) | cost of one exact check after a possible match |

Starting from an all-zero array, each inserted key sets \(k\) positions. A
query returns false if any required position is zero and true otherwise.

After \(n\) insertions, the probability that one bit remains zero is

\[
\left(1 - \frac{1}{m}\right)^{kn}.
\]

Under the usual uniform and approximately independent hashing assumptions, the
false-positive probability is approximately

\[
p \approx \left(1 - e^{-kn/m}\right)^k.
\]

For a fixed \(m/n\), this approximation is minimized near

\[
k = \frac{m}{n}\ln 2.
\]

For planned capacity \(n\) and target false-positive probability \(p\), choose

\[
m = -\frac{n\ln p}{(\ln 2)^2}
\]

bits, then round to the representation boundary and choose an integer \(k\)
near \((m/n)\ln 2\). The implementation must recompute and expose the modeled
false-positive rate after rounding.

At the optimum, representative targets cost approximately:

| Target \(p\) | Bits per planned key | Ideal \(k\) |
|---:|---:|---:|
| 10% | 4.79 | 3.32 |
| 1% | 9.59 | 6.64 |
| 0.1% | 14.38 | 9.97 |

These are modeled values, not guarantees for every implementation or workload.
Blocked filters, correlated inputs, hash choice, rounding, and query
distribution can change observed behavior.

## The semantic contract

Use this truth table early and repeat it in the production cases:

| Exact source | Filter result | Correct action |
|---|---|---|
| absent | false | skip the exact source |
| absent | true | exact check finds no match; count a false positive |
| present | true | exact check returns the value |
| present | false | correctness defect: investigate lifecycle or implementation |

The lower-right cell is impossible only inside the pure model: every live key
was added; adds and queries use identical bytes, configuration, and hashing;
bits were not cleared; reads observe completed writes; storage is not corrupt;
and the filter belongs to the same exact data generation. A production system
must make those conditions true.

The public read contract in teaching code should therefore use `MayContain`,
while construction remains inside a builder:

~~~go
type Membership interface {
	MayContain(key []byte) bool
}

func BuildMembership(keys [][]byte, cfg Config) (*Filter, error)
~~~

It should not use `Contains`, `Exists`, or `Find`, because those names invite a
caller to treat a positive as authoritative.

## Production-source selection

### Primary case: Apache Cassandra SSTables

The current Cassandra storage-engine documentation describes an SSTable as a
set of component files. `Data.db` stores rows, `Index.db` maps partition keys
to data positions, and `Filter.db` stores a Bloom filter of the SSTable's
partition keys. Memtables flush to new SSTables; compaction combines SSTables
and permits old files to be removed after the new file is written.

That makes the filter's ownership precise:

> one filter belongs to one immutable SSTable generation and represents the
> partition keys written into that SSTable.

The current CQL reference documents `bloom_filter_fp_chance` as a per-table
property from 0 to 1. Lower false-positive probability consumes more memory;
1 disables the filter. Older Cassandra operating documentation explains an
important lifecycle detail that remains visible in the file model: changing
the table property changes newly written SSTables, while existing SSTables keep
their old filters until rewrite or compaction.

Use Cassandra metrics to connect the formula to operations:

- `BloomFilterFalsePositives` and `BloomFilterFalseRatio` show wasted positive
  decisions;
- `BloomFilterDiskSpaceUsed` and `BloomFilterOffHeapMemoryUsed` show the space
  side of the trade-off; and
- `SSTablesPerReadHistogram` shows the exact-file fan-out experienced by reads.

The lesson must not present a false-positive target as a direct latency SLO.
Latency also depends on cache state, compaction, key distribution, storage,
true hits, and how many SSTables are candidates.

### Go-native production case: VictoriaMetrics and VictoriaLogs

VictoriaLogs documentation says it uses Bloom filters to skip blocks that do
not contain a requested word or phrase. This is a clean second example: the
expensive exact operation is unpacking and scanning a compressed log block.

The VictoriaMetrics v1.148.0 source also includes a small, inspectable
`lib/bloomfilter` package. Its `filter.go` uses four bit probes and 16 planned
bits per item with atomic word operations. Its `limiter.go` wraps the filter in
a time-windowed unique-item limiter. `promscrape` constructs that limiter for a
target's 24-hour `series_limit`.

The `vmagent` documentation explicitly says its series limits are approximate
and may underflow or overflow, usually by less than one percent. That example
should appear only after the learner understands lookup acceleration. A false
positive in a cache-admission hint wastes work; a false positive in an
approximate policy changes which new series is accepted or rejected. The API
must define that consequence deliberately.

### Persisted-format case: Git changed paths

`git commit-graph write --changed-paths` records Bloom filters for paths changed
between each commit and its first parent. Git documents significant performance
gains for `git log -- <path>` on large repositories. The commit-graph format
records filter settings, and `commitGraph.changedPathsVersion` controls which
versions Git reads and writes.

This is the best compact example of a filter being more than a byte array. Its
consumer and producer must agree on path normalization, hash version, and file
format. Git also demonstrates incremental layers and explicit regeneration of
older filters.

### Saturation case: Go Ethereum logs

Ethereum block headers contain a fixed 2,048-bit log Bloom. Current Geth range
filter code first tests a header's Bloom and, for a possible match, reads and
exactly filters the block's logs. The exact second step preserves correctness.

Geth issue 25336 records a real performance complaint: for dense event
workloads, many blocks may pass the fixed-size Bloom, so `eth_getLogs` gains
little skipping. The issue was closed after later log-filtering work; use it as
a historical workload report, not a claim about current release latency. The
durable point is that a format-fixed \(m\) cannot keep a target \(p\) as \(n\)
grows.

## Go contract and implementation boundary

Go 1.27 has no standard-library Bloom-filter type. The unit therefore defines
its own narrow membership interface and labels the implementation as teaching
code.

`hash/maphash` is attractive for an ephemeral in-process demonstration, but
its official contract says each `Seed` is local to one process and cannot be
serialized or recreated in another process. A filter built with `maphash` must
not be written to disk and interpreted after restart. A persisted filter needs
a specified stable hash and an encoded seed/version.

Kirsch and Mitzenmacher show that implementations can derive the \(k\) probe
positions from two hash values using

\[
g_i(x) = h_1(x) + i h_2(x)
\]

without losing the asymptotic false-positive performance of the standard
scheme. This is optional implementation reading, not required for the basic
contract.

Concurrency must be explicit. A plain `[]uint64` implementation is not safe
for concurrent `Add` calls or an `Add` racing with `MayContain`. Lost bit
updates can create an apparent false negative. Valid designs include:

- building an immutable filter before publication;
- one writer with synchronized readers;
- atomic word updates; or
- sharding with documented ownership.

The foundations code and lab use build-then-publish because it keeps the model
visible. A future extension may add an atomic variant.

## Lifecycle rules the lesson must teach

### Capacity is part of correctness for the performance promise

A filter does not automatically grow like a Go map. Inserting beyond planned
\(n\) sets more bits and raises \(p\). It does not normally make present keys
disappear, but it can eliminate most useful negative answers.

Store or expose:

- planned item capacity;
- approximate inserted count or source cardinality;
- bit count and probe count;
- observed bit density; and
- rebuild reason and generation; and
- a maximum allocation derived from the service memory budget.

### A standard Bloom filter cannot delete one key

Several keys can set the same bit. Clearing that bit to remove one key may
make another inserted key test false. Safe choices are immutable generations,
full rebuilds, or a different structure whose deletion semantics are designed
and paid for. A counting Bloom filter is a different data structure and is out
of scope for the foundations implementation.

### The filter and exact data move together

Publishing new exact data with an old filter can hide a newly present key.
Publishing a new filter early is safe for lookup correctness only if a possible
match still reaches an exact source containing the represented data; otherwise
the generations are inconsistent in the other direction. The simple rule is
to package data and filter under one immutable generation and atomically swap
the pair.

### Key bytes are schema

The filter hashes bytes, not domain values. Producers and consumers must agree
on field order, length framing, normalization, case, Unicode handling, numeric
endianness, and tenant scoping. Ambiguous concatenation such as `"ab" + "c"`
versus `"a" + "bc"` must not define a compound key.

## Cost model for a production lookup

Suppose a logical lookup considers \(s\) immutable segments. Let \(a\) be the
fraction of segment candidates that truly lack the key. Without filters it
performs \(s\) exact checks. With one filter per segment, the expected count of
exact checks is approximately

\[
s(1-a) + sap.
\]

The first term is true-positive work that a Bloom filter cannot remove. The
second is false-positive work. Filter CPU adds approximately \(sC_f\), so a
rough per-lookup comparison is

\[
\text{without} \approx sC_e
\]

and

\[
\text{with} \approx sC_f + \left(s(1-a)+sap\right)C_e.
\]

This makes the selection rule explicit. Bloom filters are attractive when
absence is common, exact checks are expensive, and filter lookup is much
cheaper. They help less when most candidates contain the key, the exact index
is memory-resident and cheap, or the filter bytes themselves cause expensive
cache or I/O traffic.

## Observability design

The lab exposes runtime counters and sizing state with unambiguous denominators:

- total filter queries;
- definite-negative results;
- possible-match results;
- exact true matches after possible matches;
- exact misses after possible matches;
- exact checks avoided;
- filter bytes and bit density;
- planned capacity and inserted count; and
- modeled and correctly-denominated observed false-positive rates.

A persistent production implementation should additionally expose its
data/filter generation and hash-format version.

For absent queries whose truth was verified, the observed false-positive ratio
is

\[
\frac{\text{possible match followed by exact miss}}
     {\text{all exact-source nonmembers queried}}.
\]

`possible_matches / all_queries` is not the false-positive rate because it
includes true positives. A dashboard that omits the denominator can make a
healthy hit-heavy workload look like a broken filter.

Repeated false-positive keys also deserve a heavy-hitter view. The classic
probability describes a fresh absent query under hashing assumptions. The same
absent key normally maps to the same set bits on every retry, so one hot false
positive can waste far more work than a uniform average suggests.

## Foundations lesson sequence

1. Begin with an absent partition key crossing many immutable files.
2. Compare an exact set's answer with a Bloom filter's two safe outcomes.
3. Build and query a tiny bit-array example by hand.
4. Derive the false-positive probability from bit occupancy.
5. Size a million-key filter for 1% false positives.
6. Add exact-fallback cost and derive when the filter pays for itself.
7. Read Cassandra's SSTable file model, table setting, and metrics.
8. Transfer the model to VictoriaLogs block skipping.
9. Examine Git persistence and Geth saturation as lifecycle boundaries.
10. Define an ephemeral Go 1.27 interface and explain `maphash`'s persistence
    limitation.
11. Design generation swaps, capacity monitoring, tests, and dashboards.
12. End with a production explanation rather than an implementation recital.

## Implemented exploration lab: Skip the Cold Segment

The Go lab creates immutable segments containing exact sorted key indexes and
optional Bloom filters. Its repeatable matrix and custom runner vary:

- segment count;
- keys per segment;
- absent-query fraction;
- target false-positive probability;
- actual insertions divided by planned capacity;
- uniform versus repeated absent keys; and
- in-memory versus simulated expensive exact checks.

The lab reports deterministic counts before timing:

- filter probes;
- exact segment checks;
- true hits;
- false positives; and
- exact checks avoided.

Benchmarks then add allocations and nanoseconds for filter construction,
filtered and exact lookups, and over-capacity lookups. Executable examples,
property fuzzing, and a race-enabled concurrent-read test exercise the caller
and immutability contracts. The learner predicts the direction of each change
in a worksheet before running the matrix. A variant that lies about capacity
demonstrates saturation without changing correctness, and a cost model compares
cheap and expensive exact checks without presenting modeled work as elapsed
time. The builder rejects allocations above a configurable byte limit.

## Planned Wheels of Misfortune

### Wheel 11: Maybe Is Not Present

**Incoming report:** a negative-cache optimization occasionally returns a
record that never existed, or suppresses a required exact check.

**First local cause:** caller interprets `MayContain == true` as proof of
membership.

**Deterministic symptom:** a searched absent key is an engineered false
positive; the buggy API returns present while the exact source returns absent.

### Wheel 12: Filter from Yesterday

**Incoming report:** a newly written key is intermittently reported missing
after a configuration reload.

**First local cause:** the exact segment and filter were published under
different generations.

**Deterministic symptom:** every inserted key exists in the exact new segment,
but one key maps to a zero bit in the old filter.

### Wheel 13: Ten Times Planned Cardinality

**Incoming report:** storage reads and latency rise while the Bloom filter
reports no errors and consumes its usual amount of memory.

**First local cause:** actual distinct insertions exceed planned capacity by an
order of magnitude.

**Deterministic symptom:** bit density and exact checks rise predictably while
all present-key correctness checks continue to pass.

The Wheels should keep ordinary tests passing and use build tags only for the
reported symptoms, following the existing track convention.

## Source audit

### Required CS sources

1. Burton H. Bloom,
   [“Space/Time Trade-offs in Hash Coding with Allowable Errors”](https://www.cs.princeton.edu/courses/archive/spr05/cos598E/bib/p422-bloom.pdf),
   *Communications of the ACM*, 1970. Original problem and data structure.
2. Carnegie Mellon 15-853,
   [Algorithms in the Real World, Lecture 20](https://www.cs.cmu.edu/~15853-f19/scribes/lec20.pdf).
   Accessible probability derivation with named \(M\), \(N\), and \(k\).
3. Adam Kirsch and Michael Mitzenmacher,
   [“Less Hashing, Same Performance: Building a Better Bloom Filter”](https://www.eecs.harvard.edu/~michaelm/postscripts/rsa2008.pdf),
   2008. Optional source for double hashing.

### Required Go source

1. [`hash/maphash` package documentation](https://go.dev/pkg/hash/maphash/),
   especially the `Seed` process-local and non-serializable contract.

There is no Go 1.27 standard-library Bloom-filter contract. The lesson's
interface is local teaching code, and current production implementations are
identified by repository and release.

### Required production sources

1. Apache Cassandra,
   [Storage Engine](https://cassandra.apache.org/doc/latest/cassandra/architecture/storage-engine.html):
   current SSTable components and compaction lifecycle.
2. Apache Cassandra,
   [`CREATE TABLE`](https://cassandra.apache.org/doc/latest/cassandra/reference/cql-commands/create-table.html):
   current `bloom_filter_fp_chance` property.
3. Apache Cassandra,
   [Bloom Filters](https://cassandra.apache.org/doc/3.11/cassandra/operating/bloom_filters.html):
   version-pinned operational explanation of rewrite behavior. Do not copy its
   version-specific defaults into a current configuration recommendation.
4. Apache Cassandra,
   [Metrics](https://cassandra.apache.org/doc/latest/cassandra/managing/operating/metrics.html):
   Bloom-filter and SSTables-per-read measurements.
5. VictoriaLogs,
   [FAQ](https://docs.victoriametrics.com/victorialogs/faq/): documented block
   skipping for word and phrase filters.
6. VictoriaMetrics v1.148.0,
   [`lib/bloomfilter/filter.go`](https://github.com/VictoriaMetrics/VictoriaMetrics/blob/v1.148.0/lib/bloomfilter/filter.go),
   [`lib/bloomfilter/limiter.go`](https://github.com/VictoriaMetrics/VictoriaMetrics/blob/v1.148.0/lib/bloomfilter/limiter.go),
   and
   [`promscrape/scrapework.go`](https://github.com/VictoriaMetrics/VictoriaMetrics/blob/v1.148.0/lib/promscrape/scrapework.go):
   released Go implementation and its caller.
7. VictoriaMetrics,
   [`vmagent` cardinality limiter](https://docs.victoriametrics.com/victoriametrics/vmagent/#cardinality-limiter):
   documented approximation and metrics.
8. Git,
   [`git-commit-graph`](https://git-scm.com/docs/git-commit-graph) and
   [commit-graph format](https://git-scm.com/docs/commit-graph-format):
   changed-path construction, speedup role, persistence, and versions.
9. Go Ethereum,
   [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf):
   protocol definition of the 2,048-bit logs Bloom.
10. Go Ethereum,
   [`eth/filters/filter.go`](https://github.com/ethereum/go-ethereum/blob/master/eth/filters/filter.go):
   current exact-fallback read path.
11. Go Ethereum,
    [issue 25336](https://github.com/ethereum/go-ethereum/issues/25336):
    historical report about fixed-size filter effectiveness under dense log
    workloads.

## Claims deliberately not made

- A target false-positive probability is not a latency guarantee.
- “No false negatives” is not unconditional production truth.
- More hash functions are not always better.
- A Bloom filter is not a cache and does not return values.
- A Bloom filter does not answer range, prefix, count, or similarity queries.
- A classic filter does not support safe per-key deletion.
- The Go standard library does not provide this data structure in the target
  toolchain.
- A first-party implementation's constants are not universal recommendations.
- Running a Bloom-filter-using database or observability service on Kubernetes
  does not mean Kubernetes uses a Bloom filter.

## Completion criteria

The foundations chapter is ready when:

- every positive production claim links to a primary source;
- the false/true decision table appears before formulas;
- the formula names assumptions and rounding;
- the million-key sizing example is reproducible;
- Cassandra's filter is tied to one SSTable and its lifecycle;
- the Go section distinguishes ephemeral `maphash` from persistence;
- VictoriaLogs, Git, and Geth each teach a different boundary rather than
  repeating the Cassandra case;
- no production measurement is presented as a guarantee; and
- a Hugo production build and clean rebuild comparison pass.
