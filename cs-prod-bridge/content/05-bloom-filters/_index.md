+++
title = 'Bloom filters and approximate membership'
description = 'A Bloom filter makes absence cheap, but only when its caller, capacity, and data lifecycle preserve a one-sided contract.'
weight = 5
+++

**Unit 05 · Foundations**

# Bloom filters: make absence cheap

{{< lead >}}A Bloom filter is a small, probabilistic summary of a set. It can
prove that a key is absent. It can only suggest that a key may be present. Put
that one-sided answer in front of an expensive exact lookup, and many negative
queries become cheap. Treat “maybe” as “yes,” or let the summary drift away
from its data, and an optimization becomes a correctness bug.{{< /lead >}}

{{< callout kind="production" title="Incoming reports" >}}
- A Cassandra node consults several immutable SSTables for a partition-key
  read. Each SSTable has a
  [`Filter.db`](https://cassandra.apache.org/doc/latest/cassandra/architecture/storage-engine.html)
  component so the read path can avoid files that definitely lack the key.
- VictoriaLogs says its query path
  [uses Bloom filters to skip blocks](https://docs.victoriametrics.com/victorialogs/faq/)
  without a requested word or phrase. A false positive means unnecessary block
  work, not an incorrect log result.
- Git can write changed-path Bloom filters into its
  [commit graph](https://git-scm.com/docs/git-commit-graph) to accelerate
  `git log -- <path>` in large repositories. The stored format has versions.
- Go Ethereum's range-log filter uses a block header's Bloom to find
  [possibly interesting blocks](https://github.com/ethereum/go-ethereum/blob/master/eth/filters/filter.go),
  then checks their logs exactly. A public performance report showed how a
  fixed-size filter can become
  [too dense to skip much work](https://github.com/ethereum/go-ethereum/issues/25336).
{{< /callout >}}

These systems do not use a Bloom filter because probabilistic answers are
intrinsically desirable. They use one because an exact negative answer can be
expensive: opening another table, unpacking another log block, diffing another
commit, or loading another block's receipts. The filter is cheaper than that
work and most useful when the answer is often “absent.”

Kubernetes is not one of the examples in this unit. A review of the current
Kubernetes core repository and first-party component documentation found no
direct Bloom-filter implementation in the API server, scheduler, controllers,
kubelet, or first-party storage path. A database or telemetry system can run on
Kubernetes without Kubernetes itself using that data structure.

By the end of the unit, you should be able to:

- distinguish an exact set from an approximate-membership filter;
- interpret `MayContain(key) == false` and `MayContain(key) == true` safely;
- build and query a Bloom filter using a bit array and derived hash positions;
- explain why collisions permit false positives;
- state the assumptions hidden inside “no false negatives”;
- size a filter from planned cardinality and a false-positive budget;
- explain why too many keys or too many probes can make a filter ineffective;
- compare filter cost with the exact work it avoids;
- design key encoding, persistence, concurrency, and generation ownership;
- choose metrics that measure useful skipping rather than mere activity; and
- recognize workloads where an exact set or no filter is the better design.

## Start with expensive absence

Imagine a read service with twelve immutable segments. Each segment has a
sorted exact index and a compressed data file. A lookup may need to ask every
segment whether it contains this tenant-scoped key:

~~~text
tenant=acme, key=deploy/8421
~~~

An exact segment check can answer correctly, but it may require index pages,
cache misses, decompression, or storage I/O. If the key appears in only one
segment—or no segment—most of those exact checks discover absence.

A Bloom filter stores no values and cannot return the record. It stands in
front of one exact segment:

~~~text
                         false
query key ──> filter ─────────────> skip this segment
                 │
                 │ true: maybe
                 v
          exact segment lookup ──> present or absent
~~~

The fast path is the upper arrow. A negative filter answer is enough to stop.
A positive filter answer merely preserves the old exact path.

{{< callout kind="contract" title="The complete caller contract" >}}
For a standard Bloom filter built correctly from the same data generation:

- `MayContain(key) == false` means the key is definitely absent from the
  represented exact set.
- `MayContain(key) == true` means the key may be present. Check the exact set.

The method should not be called `Contains`, `Exists`, or `Find`. Those names
hide the most important fact about its result.
{{< /callout >}}

Here is the decision table. “Exact source” is ground truth, not another
prediction:

| Exact source | Filter says | Correct action and outcome |
|---|---|---|
| absent | false | Skip the exact check. This is the useful negative. |
| absent | true | Check exactly, find nothing, and count a false positive. |
| present | true | Check exactly and return the value. |
| present | false | Correctness defect. An assumption outside the pure model failed. |

A false positive wastes work. A false negative can hide real data. That
asymmetry is the design.

## Exact membership and approximate membership answer different questions

An exact set stores enough information to distinguish every member from every
nonmember according to its equality rule. In Go, `map[string]struct{}` is a
common exact set. If the map contains the canonical key, the key is present in
that map.

A Bloom filter stores a compressed pattern shared by many keys. It does not
retain the keys. That buys much lower memory per planned key at the price of a
configurable chance that an absent key happens to look possible.

| Property | Exact map-backed set | Standard Bloom filter |
|---|---|---|
| Negative answer | Exact | Exact under stated construction assumptions |
| Positive answer | Exact | Possible match only |
| Returns a stored value | No, but membership is exact | No |
| Typical space | Stores keys plus table overhead | Fixed bit array plus metadata |
| Add | Yes | Yes |
| Delete one key | Yes | Not safely |
| Grows automatically | Go map does | Classic fixed filter does not |
| Key enumeration | Yes | No |

“Space efficient” therefore needs a workload sentence. The relevant
comparison is not a Bloom filter against zero bytes. It is filter memory plus
remaining exact checks against the exact work performed without the filter.

## Build a filter by setting bits

Start with a bit array of length \(m\), all zero. Choose \(k\) deterministic
hash-derived positions for each key. Adding a key sets those positions to one.

For a tiny example, let \(m=16\) and \(k=3\). Suppose the three positions for
`alpha` are 2, 7, and 13:

~~~text
index:  0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
bits:   0 0 1 0 0 0 0 1 0 0  0  0  0  1  0  0
~~~

Now add `bravo`, whose positions are 4, 7, and 10. Position 7 was already set;
that is normal:

~~~text
index:  0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
bits:   0 0 1 0 1 0 0 1 0 0  1  0  0  1  0  0
~~~

To query a key, calculate the same positions:

1. If any required bit is zero, the key was not added.
2. If all required bits are one, the key may have been added.

Suppose `charlie` maps to 2, 5, and 10. Bit 5 is zero, so `charlie` is
definitely absent. Suppose `delta` maps to 4, 10, and 13. All three bits are
one even though different inserted keys set them. `delta` is a false positive.

~~~text
alpha   set 2 --------+             +-------- set 13
                      |             |
bravo   set 4 ----+   |   +-- set 10|  and shares bit 7
                  |   |   |         |
delta checks      4, 10, 13  -> all one -> maybe
~~~

The filter cannot inspect the bit array and discover which key set a bit. That
loss of identity is where its space saving and its false positives come from.

## Why a negative is conclusive in the model

Adding a key sets every one of its \(k\) positions. Later additions set more
bits; they do not clear those positions. If a later query for that same key
finds a zero at one required position, then the key could not have completed
the modeled insertion.

That argument is simple—and conditional.

{{< callout kind="warning" title="No false negatives is not an unconditional service guarantee" >}}
The standard model assumes all of the following:

- every live key was inserted;
- add and query use exactly the same key bytes;
- both use the same \(m\), \(k\), hash algorithm, seed, and version;
- no operation clears a shared bit;
- a reader observes every completed bit update;
- the bit array is not truncated or corrupt; and
- the filter represents the same exact-data generation the caller will read.

A stale filter, ambiguous key encoding, lost concurrent update, bad restore, or
partial publication can produce an operational false negative even though the
abstract data structure does not.
{{< /callout >}}

The production task is not merely to implement `Add` and `MayContain`. It is to
make those assumptions true for the lifetime of the data.

## Derive the false-positive probability

Name the variables before using the formula:

| Symbol | Workload meaning |
|---|---|
| \(n\) | distinct keys inserted into one filter |
| \(m\) | bits in that filter |
| \(k\) | bit positions examined per add or query |
| \(p\) | probability that an absent queried key passes every bit check |

One hash-derived position chooses one of \(m\) bits. The probability that it
does not choose a particular bit is

\[
1-\frac{1}{m}.
\]

Adding \(n\) keys makes \(kn\) choices. Under the standard independent,
uniform model, the probability that a particular bit remains zero is

\[
\left(1-\frac{1}{m}\right)^{kn}.
\]

Therefore the probability that it is one is

\[
1-\left(1-\frac{1}{m}\right)^{kn}.
\]

An absent query becomes a false positive when all \(k\) of its positions are
one. The common approximation is

\[
p \approx \left(1-e^{-kn/m}\right)^k.
\]

This is a model under hash and query assumptions. It is not a promise that
every batch of 100 absent production queries will contain exactly \(100p\)
false positives.

### More probes are not always better

With \(m\) and \(n\) fixed, an extra probe gives a query another bit that must
match. Initially that helps. But every insertion also sets another bit, making
the array denser. Eventually the added density hurts more than the extra check
helps.

The approximate optimum is

\[
k = \frac{m}{n}\ln 2.
\]

At this point roughly half the bits are set. Choose a nearby positive integer
and evaluate both neighboring choices after rounding the bit-array size.

## Size from a capacity and an error budget

Do not start with “allocate one megabyte” or “use seven hashes.” Start with:

- planned distinct keys \(n\); and
- acceptable false-positive probability \(p\) for absent queries.

Near the optimum, the required bit count is

\[
m = -\frac{n\ln p}{(\ln 2)^2}.
\]

Then calculate \(k\) near \((m/n)\ln 2\).

Representative targets are:

| Target false-positive probability | Approximate bits per planned key | Ideal probes |
|---:|---:|---:|
| 10% | 4.79 | 3.32 |
| 1% | 9.59 | 6.64 |
| 0.1% | 14.38 | 9.97 |

### Worked example: one million keys at one percent

Let \(n=1{,}000{,}000\) and \(p=0.01\):

\[
m = -\frac{1{,}000{,}000\ln(0.01)}{(\ln 2)^2}
  \approx 9{,}585{,}059\text{ bits}.
\]

That is about 1,198,133 bytes, or 1.14 MiB before representation metadata and
alignment. The ideal probe count is

\[
k \approx \frac{9{,}585{,}059}{1{,}000{,}000}\ln 2
  \approx 6.64,
\]

so an implementation would compare the achieved probability around six and
seven probes, commonly choosing seven for this rounded example.

The memory number is not a universal “one million keys cost 1.14 MiB” fact.
It is the result for this capacity, target, classic model, and compact bit
array. A blocked representation may round or reserve space differently.

### Capacity does not expand itself

If the same filter receives ten million distinct keys instead of the planned
one million, \(m\) stays fixed. More bits become one. False positives rise,
possibly until nearly every absent query returns “maybe.” Present keys still
usually test positive, so correctness tests alone can pass while the
optimization has stopped doing useful work.

Treat planned capacity as an operational limit:

- measure or bound source cardinality before construction;
- record planned capacity with the filter;
- monitor insertions or bit density;
- rebuild or roll to a new generation before saturation; and
- do not silently advertise the original \(p\) after exceeding its \(n\).

## Put the filter in an end-to-end cost model

Suppose one logical lookup considers \(s\) immutable segments. Let \(a\) be the
fraction of those segment candidates that truly lack the key. Let \(C_f\) be
one filter-query cost and \(C_e\) one exact segment-check cost.

Without filters, the lookup performs \(s\) exact checks:

\[
\text{cost without filters} \approx sC_e.
\]

With one filter per segment:

- the \(s(1-a)\) candidates that really contain the key still need exact work;
- approximately \(sap\) absent candidates pass as false positives; and
- all \(s\) candidates pay the filter-query cost.

So a rough expected comparison is

\[
\text{cost with filters}
  \approx sC_f + \left(s(1-a)+sap\right)C_e.
\]

This equation explains the workload selection:

- Large \(a\): many candidates are absent, so there is work to skip.
- Large \(C_e/C_f\): each useful negative saves something meaningful.
- Small \(p\): fewer absent candidates leak into exact work, at a memory and
  sometimes CPU cost.
- Large true-hit fraction: the filter cannot remove the exact work.

If the exact set is already a tiny in-memory map, \(C_e\) may be comparable to
\(C_f\). The filter then adds memory, hashing, and complexity without a useful
payoff.

{{< callout kind="note" title="A target probability is not a latency SLO" >}}
The formula estimates false positives under a model. Request latency also
depends on cache state, storage, decompression, segment fan-out, query skew,
contention, true hits, and the cost of reading the filter itself. Measure the
end-to-end work that was skipped.
{{< /callout >}}

## Production case: Cassandra owns one filter per SSTable

Cassandra's
[storage-engine documentation](https://cassandra.apache.org/doc/latest/cassandra/architecture/storage-engine.html)
describes a write path that collects updates in a memtable and flushes them to
an immutable Sorted String Table, usually called an SSTable. Reads may merge
data from memtables and multiple SSTables. Compaction writes combined SSTables
and then allows replaced files to be removed.

An SSTable is a family of components. Among the components documented by
Cassandra are:

- `Data.db`, containing the rows;
- `Index.db`, mapping partition keys to locations in the data file;
- `Summary.db`, sampling the partition index; and
- `Filter.db`, a Bloom filter of partition keys in that SSTable.

That gives the Bloom filter an exact meaning:

> `Filter.db` for SSTable generation G summarizes the partition keys written
> into SSTable generation G.

For a point lookup, a negative answer can remove one SSTable from the more
expensive lookup path. A positive answer cannot prove that the partition is in
the SSTable, so Cassandra still needs its exact indexes and data.

### The tuning knob spends memory to avoid I/O

Cassandra's current
[`CREATE TABLE` reference](https://cassandra.apache.org/doc/latest/cassandra/reference/cql-commands/create-table.html)
documents `bloom_filter_fp_chance` as a table property. A lower value targets
fewer false positives and uses more memory. A value of 1 disables the filter.

This is not “turn accuracy up.” It is a resource trade:

~~~text
lower target p  -> more filter bits -> more memory/disk for filters
                                  -> fewer wasted SSTable checks, if workload fits

higher target p -> fewer filter bits -> less filter memory/disk
                                  -> more absent keys reach exact lookup
~~~

The right value depends on the table's reads, compaction strategy, SSTable
fan-out, storage and cache behavior, and node memory budget. The setting is not
a global truth about Cassandra.

### A configuration change is not a time machine

Cassandra's version-pinned
[Bloom-filter operations guide](https://cassandra.apache.org/doc/3.11/cassandra/operating/bloom_filters.html)
explains a lifecycle detail that follows from the immutable file design. A
filter is calculated when an SSTable is written and persists as that SSTable's
filter component. Changing the table property affects new files. Existing
SSTables retain their existing filters until compaction or another rewrite
regenerates them.

After a change, one table can therefore contain SSTables built to different
targets. The configuration says what future files should do; it does not prove
that every current file already does it.

### Observe usefulness with a denominator

Cassandra's
[documented table metrics](https://cassandra.apache.org/doc/latest/cassandra/managing/operating/metrics.html)
include:

- `BloomFilterFalsePositives`;
- `BloomFilterFalseRatio`;
- `BloomFilterDiskSpaceUsed`;
- `BloomFilterOffHeapMemoryUsed`; and
- `SSTablesPerReadHistogram`.

The first two describe leakage into exact work. The next two describe the
space cost. SSTables per read connects the filter to its production purpose.
A filter can meet a statistical target and still have little effect if most
reads are true hits across a small number of candidate SSTables. Conversely,
a modest false-positive target may be valuable when it removes expensive
negative checks across many cold files.

{{< callout kind="production" title="What Cassandra makes concrete" >}}
The production object is not “a Bloom filter for the database.” It is one
versioned component belonging to one immutable SSTable. Its capacity comes
from that file's partition keys. Its rebuild follows that file's rewrite. Its
memory and false positives are observable alongside the exact read fan-out.
{{< /callout >}}

## Transfer the model: VictoriaLogs skips log blocks

VictoriaLogs documents that it divides stored logs into blocks and uses Bloom
filters to skip blocks without a requested word or phrase. If the filter says
no, the query need not unpack and inspect that block. If it says maybe, the
block is processed and the query predicate still decides which log rows match.

Map this system onto the same variables:

| Bloom model | VictoriaLogs workload |
|---|---|
| exact set | words or phrases represented by one log block |
| exact check | unpack and process the block |
| useful negative | block cannot contain the requested term |
| false positive | block is processed but produces no match for that term |
| capacity/lifecycle | terms represented by one stored block and its metadata |

The domain changed from partition keys to log terms. The caller contract did
not. A possible match is permission to do exact work, not permission to return
the entire block as matching.

### A Go implementation in the same production family

VictoriaMetrics release
[`v1.148.0`](https://github.com/VictoriaMetrics/VictoriaMetrics/releases/tag/v1.148.0)
contains an inspectable `lib/bloomfilter` package. Its
[`filter.go`](https://github.com/VictoriaMetrics/VictoriaMetrics/blob/v1.148.0/lib/bloomfilter/filter.go)
uses four probes, plans 16 bits per item, and atomically updates 64-bit words.
Those constants and concurrency choices describe that released implementation;
they are not part of the abstract Bloom-filter contract.

The package's
[`limiter.go`](https://github.com/VictoriaMetrics/VictoriaMetrics/blob/v1.148.0/lib/bloomfilter/limiter.go)
uses the filter to limit unique items within a refresh interval. The
[`promscrape` caller](https://github.com/VictoriaMetrics/VictoriaMetrics/blob/v1.148.0/lib/promscrape/scrapework.go)
constructs a 24-hour limiter for a target's `series_limit`.

This is deliberately a different product contract. Once the limiter believes
it has reached its cap, a Bloom-positive hash can be treated as an already
known series and admitted. A false positive can therefore let the approximate
count overflow; earlier false positives can also undercount distinct series.
The
[`vmagent` documentation](https://docs.victoriametrics.com/victoriametrics/vmagent/#cardinality-limiter)
states that these limits are approximate and may underflow or overflow by a
small percentage, usually less than one percent.

{{< callout kind="warning" title="Approximation belongs in the product contract" >}}
Using a Bloom filter before an exact read preserves exact results. Using one to
make an admission, quota, billing, authorization, or revocation decision can
change externally visible behavior. That may be an intentional trade-off, as
with a documented approximate limiter. It must not be an accidental reuse of
the lookup-acceleration contract.
{{< /callout >}}

## Persisted filters need a format: Git changed paths

Git's commit graph can store a Bloom filter for paths changed between a commit
and its first parent. The
[`--changed-paths` option](https://git-scm.com/docs/git-commit-graph)
computes this information and documents significant performance gains for
history queries such as:

~~~console
git log -- path/to/large/subtree
~~~

For a commit that definitely did not change a relevant path, history traversal
can avoid more expensive tree-diff work for that candidate. A possible match
keeps the exact path check.

Git's
[`commit-graph-format` documentation](https://git-scm.com/docs/commit-graph-format.html)
describes stored Bloom data and settings. The `commitGraph.changedPathsVersion`
configuration controls which filter versions a Git process reads and writes,
and the documentation warns about compatibility with older Git versions.

This is what a real persistent filter format must settle:

- Which exact paths belong to a commit's represented set?
- How are path bytes normalized and hashed?
- Which algorithm and seed define positions?
- What are \(m\), \(k\), and any maximum-size rule?
- Which producer version wrote the bytes?
- What should a reader do with an unsupported version?
- How are older graph layers regenerated?

A serialized bit array without those answers is not portable metadata.

## Fixed size meets unbounded input: Ethereum log Blooms

The
[Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)
defines a 2,048-bit Bloom over log addresses and topics, which contributes to
the block header's `logsBloom`. Geth's current range-filter path names the
correct sequence in code: its Bloom check decides whether a block is
interesting, and a possible match leads to `checkMatches`, which filters
actual logs.

The filter's bit count is part of a protocol format, not a per-operator memory
setting. As the number of represented log values in a block grows, the fixed
array becomes denser and more absent queries pass.

[Geth issue 25336](https://github.com/ethereum/go-ethereum/issues/25336)
recorded a performance complaint about high false-positive rates and slow log
queries for dense workloads. Later filtering work improved the implementation,
and the issue was closed in 2025, so it is not evidence that current Geth has
the same latency. It is evidence of the durable capacity problem:

> If \(m\) is fixed by a format while \(n\) grows with the workload, the
> original effectiveness cannot remain constant.

An unsaturated correctness test will not find this. Measure bit density,
possible matches, exact matches, exact misses, and the distribution of log
cardinality per block.

## Define a narrow Go contract

Go 1.26 does not provide a standard-library Bloom-filter type. A local API can
make the semantics explicit:

~~~go
// Membership is an approximate, insertion-only view of one exact set.
// A false result proves absence only while the documented construction and
// generation invariants hold. A true result requires an exact lookup.
type Membership interface {
	Add(key []byte)
	MayContain(key []byte) bool
}
~~~

A caller then preserves the fallback:

~~~go
func (s *Segment) Lookup(key []byte) (Record, bool) {
	if !s.filter.MayContain(key) {
		return Record{}, false
	}

	// A possible match is not an answer. The exact index remains authoritative.
	return s.exact.Lookup(key)
}
~~~

The interface still leaves necessary policy outside the methods:

- Is the filter immutable after publication?
- Are caller-owned key bytes read only during the call?
- What is the planned capacity?
- Can it be serialized?
- What exact data generation does it represent?
- Which metrics does a possible-match fallback update?

Document those choices beside the type rather than expecting the bit-array
implementation to reveal them.

### An ephemeral hashing boundary

Go's [`hash/maphash`](https://go.dev/pkg/hash/maphash/) provides seeded hash
functions suitable for byte sequences inside one process. Its official
`Seed` contract also says something decisive: a seed is local to one process
and cannot be serialized or recreated in another process.

That makes `maphash` suitable for an **ephemeral teaching filter** that is
built and queried during one process lifetime. It is not, without a different
format design, suitable for Cassandra-like or Git-like persisted filter bytes.
After restart, a new seed maps the same key to different positions, and the old
bit array no longer represents the query hash.

{{< callout kind="contract" title="Persist the recipe, not only the bits" >}}
A persistent or networked filter needs a stable specification containing at
least the key encoding, hash algorithm and version, seed if applicable, bit
count, probe count, and represented data generation. Reject an unsupported or
incomplete format and use the exact path. Do not guess.
{{< /callout >}}

### Derive probes without hashing the key seven times

The original model speaks of \(k\) hash functions. A production implementation
need not run \(k\) unrelated full-key hashes.

Kirsch and Mitzenmacher's
[“Less Hashing, Same Performance”](https://www.eecs.harvard.edu/~michaelm/postscripts/rsa2008.pdf)
shows that positions can be derived from two hashes:

\[
g_i(x)=h_1(x)+i h_2(x).
\]

Modulo the bit count, those values supply the \(k\) positions without losing
the standard scheme's asymptotic false-positive performance. This is an
implementation technique. The producer and consumer still need identical,
versioned hashing rules.

## Key bytes are part of the schema

A Bloom filter hashes bytes, not the domain concept “tenant and object key.”
These two compound keys must not collapse to the same unframed bytes:

~~~text
(tenant="ab", key="c")
(tenant="a",  key="bc")
~~~

Plain concatenation produces `abc` for both. Use an unambiguous canonical
encoding, such as length-prefixed fields:

~~~text
2:"ab" 1:"c"
1:"a"  2:"bc"
~~~

The format must also settle:

- field order and presence;
- tenant or namespace scoping;
- text normalization and case handling;
- integer width and byte order;
- whether a path includes a trailing separator;
- handling of invalid encodings; and
- schema version.

If the builder inserts one encoding and the reader queries another, a present
domain key can reach different bits. That is an operational false negative
caused by schema disagreement, not a probabilistic false positive.

## Standard Bloom filters do not delete one key

Suppose `alpha` and `bravo` both set bit 7. Clearing bit 7 to delete `alpha`
also makes `bravo` fail its membership test. The filter has no record of how
many keys contributed to that bit.

Safe lifecycle choices include:

- build one immutable filter per immutable data segment;
- rebuild a complete filter after exact data changes;
- create a new generation and atomically publish it with the new exact data;
  or
- choose a different data structure with deletion semantics and pay its
  additional space and complexity.

A counting Bloom filter replaces bits with counters and is a related but
different structure. It introduces counter size, overflow, underflow, and
concurrent-update questions. It does not belong in a patch described merely as
“add delete to the Bloom filter.”

## Publish data and filter as one generation

Consider a service that currently serves generation 41:

~~~text
generation 41 = exact segment 41 + filter 41
~~~

It builds generation 42 in the background. Publishing exact segment 42 while
leaving filter 41 active can hide a key new to generation 42:

~~~text
query new key -> old filter says false -> new exact segment is skipped
~~~

Build and validate the pair off to the side, then swap one pointer or manifest:

~~~text
before: active -> {data: 41, filter: 41}

build:            {data: 42, filter: 42}
validate: every exact key in data 42 tests maybe in filter 42

after:  active -> {data: 42, filter: 42}
~~~

If a filter is missing, corrupt, or from an unsupported version, falling back
to the exact path preserves correctness. Treating the filter as empty does
not—it would say “definitely absent” for everything.

## Make concurrency an ownership decision

A plain Go `[]uint64` filter is not automatically safe for concurrent writes.
Two goroutines can read the same word, set different bits locally, and write
back values that lose one update. A later query can then see zero for a key
whose `Add` returned. A read racing with a write is also a Go data race.

Choose and document one design:

- build the filter on one goroutine, then publish it as immutable;
- synchronize one writer and all readers;
- update words with an atomic compare-and-swap loop; or
- partition ownership so no two writers mutate the same state.

The VictoriaMetrics v1.148.0 implementation is an example of atomic word
updates. The foundations model prefers build-then-publish because immutable
ownership also makes generation swaps and testing easier. Atomic mutation is
not free, and it does not solve capacity or key-encoding drift.

## Measure whether the filter is useful

Useful metrics separate the filter decision from exact truth:

| Measurement | What it tells you |
|---|---|
| filter queries | traffic offered to the filter |
| definite negatives | candidates skipped |
| possible matches | candidates sent to exact lookup |
| exact matches after maybe | true positives |
| exact misses after maybe | false positives |
| exact checks avoided | direct benefit, if the counter's baseline is clear |
| filter bytes | space cost |
| set-bit density | saturation signal |
| planned capacity and inserted count | whether the sizing premise still holds |
| build/data generation and format version | lifecycle agreement |
| exact-check latency or bytes | value of each saved check |

For queried keys known to be absent from the exact source, observed
false-positive ratio is

\[
\frac{\text{possible match followed by exact miss}}
     {\text{all queried exact-source nonmembers}}.
\]

This is not the same as

~~~text
possible matches / all queries
~~~

because possible matches include real members. A hit-heavy workload can make
that second ratio high even when the filter behaves exactly as designed.

### Watch repeated false positives

The probability model commonly describes an absent query drawn under a stated
distribution. Hashing is deterministic. Once a particular absent key is a
false positive, retrying that key normally produces the same false positive.

An average 1% rate does not stop one hot missing key from repeatedly opening an
expensive file. Add a heavy-hitter view for exact misses after “maybe,” cache an
appropriate exact negative where semantics allow, or consider an adaptive
filter if repeated adversarial queries matter. Do not assume retries create
fresh independent chances.

## Test correctness before performance

A filter test suite needs more than a measured false-positive percentage.

### Invariant tests

After construction, every exact member must return `MayContain == true`:

~~~go
for _, key := range exactKeys {
	if !filter.MayContain(key) {
		t.Fatalf("false negative for %q", key)
	}
}
~~~

Run this across empty input, one item, duplicate adds, word boundaries, and
maximum planned capacity. For a persistent format, build and query across the
supported writer/reader versions.

### Caller-contract tests

Construct or search for an absent key that returns “maybe.” Verify that the
caller still consults the exact source and returns absent. This catches the
most dangerous API misuse without relying on a random false positive appearing
during one test run.

### Generation tests

Attempt to pair data generation 42 with filter generation 41. The loader must
reject the pair or disable the filter and use exact lookup. It must not publish
the inconsistent optimization.

### Statistical tests

Insert a fixed, deterministic data set and query a separate fixed absent set.
Compare the measured ratio with a tolerance appropriate to the sample count.
Do not write a flaky test that demands exactly 1% false positives. The
probability describes a distribution, not a fixed quota.

### Scaling tests

Hold \(m\) and \(k\) fixed while increasing insertions above planned \(n\).
Assert deterministic bit density or exact-check counts for the fixture, then
use benchmarks to measure time. Wall-clock timing supports the explanation; it
is not the only failure oracle.

## Know when not to use one

Do not add a Bloom filter merely because the data set is large. It is a poor
fit when:

- most queried candidates are true members, so exact work remains;
- the exact membership structure is already small and memory-resident;
- queries need values, enumeration, ordering, ranges, prefixes, or counts;
- keys must be deleted individually and immutable generations are unavailable;
- cardinality cannot be bounded and no scalable/rebuild scheme exists;
- the filter would be authoritative for a correctness-critical decision;
- key encoding or generation ownership cannot be made stable;
- filter I/O or cache footprint rivals the exact check it is meant to avoid;
  or
- an adversary can concentrate traffic on stable false-positive keys and the
  system has no mitigation.

Sometimes a sorted exact index plus binary search is sufficient. Sometimes a
small exact map is simpler. Sometimes a coarser min/max range check removes
most files. Compare the total design, not just bytes per key.

## Keep source roles separate

| Claim | Kind | Source or evidence |
|---|---|---|
| The original structure trades space for allowable membership errors | CS model | [Bloom's 1970 paper](https://www.cs.princeton.edu/courses/archive/spr05/cos598E/bib/p422-bloom.pdf) |
| The usual \(p\) approximation follows from bit occupancy assumptions | CS model | [CMU 15-853 lecture notes](https://www.cs.cmu.edu/~15853-f19/scribes/lec20.pdf) |
| Two base hashes can derive the probe sequence | CS implementation result | [Kirsch and Mitzenmacher](https://www.eecs.harvard.edu/~michaelm/postscripts/rsa2008.pdf) |
| A `maphash.Seed` is process-local and cannot be recreated elsewhere | Go contract | [`hash/maphash`](https://go.dev/pkg/hash/maphash/) |
| Cassandra stores a partition-key Bloom with each SSTable | Production contract | [Cassandra storage engine](https://cassandra.apache.org/doc/latest/cassandra/architecture/storage-engine.html) |
| VictoriaLogs skips blocks for absent words or phrases | Production contract | [VictoriaLogs FAQ](https://docs.victoriametrics.com/victorialogs/faq/) |
| Git persists versioned changed-path filters | Production format | [Git commit-graph docs](https://git-scm.com/docs/commit-graph-format.html) |
| Geth checks actual logs after a Bloom possible match | Current implementation | [Geth filter source](https://github.com/ethereum/go-ethereum/blob/master/eth/filters/filter.go) |

No one row can substitute for another. A production repository demonstrates a
choice, not the universal probability model. A paper does not define Go's
concurrency contract. A local benchmark describes one workload, not Cassandra
or VictoriaLogs.

## Explain the design to a colleague

A concise production explanation might sound like this:

> Each immutable segment has an exact key index. Most point lookups are absent
> from most segments, and an exact segment check is substantially more
> expensive than several hash probes. I would build one Bloom filter from all
> canonical tenant-and-key bytes in each segment. A negative answer skips that
> segment; a positive always falls through to its exact index.
>
> For planned capacity \(n\) and target false-positive probability \(p\), I
> would size the bit array with
> \(m=-n\ln p/(\ln 2)^2\) and choose an integer \(k\) near
> \((m/n)\ln2\), then verify the achieved rate after representation rounding.
> Construction is \(O(nk)\), a query is \(O(k)\), and the filter uses
> \(O(m)\) bits. Those bounds omit the exact fallback, so I would also model
> absent-query fraction and saved segment-check cost.
>
> The filter and segment would share an immutable generation and be published
> atomically. The format would record canonical key encoding, stable hash
> version, seed, \(m\), \(k\), planned capacity, and data generation. A missing
> or incompatible filter disables the optimization rather than behaving like
> an empty set. I would monitor definite negatives, exact misses after maybe,
> bytes, bit density, inserted cardinality, and exact checks avoided. If the
> workload is hit-heavy or the exact check is already cheap, I would not add
> the filter.

That answer covers semantics, sizing, total cost, lifecycle, observability, and
the rejection condition. Merely describing a bit array would not.

## Reflection

1. Why is `MayContain` a safer method name than `Contains`?
2. Which cell in the decision table is an expected probabilistic outcome, and
   which cell indicates a broken production assumption?
3. In the 16-bit example, how can `delta` pass even though it was never added?
4. Why can another insertion create a false positive but not clear a modeled
   member's bits?
5. For fixed \(m\) and \(n\), why does increasing \(k\) eventually make the
   array less useful?
6. What happens to \(p\) when actual distinct insertions exceed planned \(n\)
   while \(m\) and \(k\) stay fixed?
7. Why must the caller still read Cassandra's exact SSTable structures after a
   Bloom-positive result?
8. Why can Cassandra table configuration and existing SSTable filters disagree
   immediately after an `ALTER TABLE`?
9. What is the expensive exact operation that VictoriaLogs tries to skip?
10. Why does Git need a changed-path filter version rather than only stored
    bits?
11. What does Geth's exact `checkMatches` step protect against?
12. Why is an approximate cardinality limiter a stronger product-policy choice
    than a negative lookup accelerator?
13. Why would persisting a filter built from `hash/maphash` and recreating a new
    seed after restart violate the filter's assumptions?
14. How can a lost concurrent word update create an operational false negative?
15. Why is clearing a bit unsafe when deleting one key?
16. What should a loader do when the filter generation does not match the exact
    data generation?
17. Why is `possible matches / all queries` not a false-positive ratio?
18. How can one repeatedly queried absent key cost more than the average model
    suggests?
19. Name a workload where a small exact set is preferable.
20. For your own system, name \(n\), the exact work \(C_e\), the absent fraction
    \(a\), and the operation a useful negative would skip.

The durable production habit is:

> Let the filter say no. Let the exact source say yes. Version and measure the
> assumptions that keep those roles true.
