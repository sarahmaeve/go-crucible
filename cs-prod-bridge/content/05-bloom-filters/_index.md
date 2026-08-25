+++
title = 'Bloom filters and approximate membership'
description = 'Bloom filters reduce the cost of negative lookups by ruling out absent keys before an exact search.'
weight = 5
+++

**Unit 05 · Foundations**

# Bloom filters and negative lookups

{{< lead >}}A Bloom filter is a space-efficient data structure for testing set
membership. It returns either “definitely absent” or “possibly present.” False
positives are possible; the standard data structure does not produce false
negatives.{{< /lead >}}

{{< callout kind="production" title="Examples from production systems" >}}
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
  [possibly interesting blocks](https://github.com/ethereum/go-ethereum/blob/v1.17.3/eth/filters/filter.go),
  then checks their logs exactly. A public performance report showed how a
  fixed-size filter can become
  [too dense to skip much work](https://github.com/ethereum/go-ethereum/issues/25336).
{{< /callout >}}

In each example, a negative result avoids a more expensive operation: opening
a table, unpacking a log block, diffing a commit, or loading receipts. The
exact operation is still used for positive results.

By the end of the unit, you should be able to:

- distinguish an exact set from an approximate-membership filter;
- interpret `MayContain(key) == false` and `MayContain(key) == true` safely;
- build and query a Bloom filter using a bit array and derived hash positions;
- explain why collisions permit false positives;
- state the assumptions hidden inside “no false negatives”;
- size a filter from a planned number of distinct keys and an acceptable
  false-positive probability;
- explain why too many keys or too many probes can make a filter ineffective;
- compare filter cost with the exact work it avoids;
- design key encoding, persistence, concurrency, and generation ownership;
- choose metrics that measure useful skipping rather than mere activity; and
- recognize workloads where an exact set or no filter is the better design.

## Example: lookups across immutable segments

Imagine a read service with twelve immutable segments. Each segment has a
sorted exact index and a compressed data file. A lookup may need to ask every
segment whether it contains this tenant-scoped key:

~~~text
tenant=acme, key=deploy/8421
~~~

An exact segment check can answer correctly, but it may require index pages,
cache misses, decompression, or storage I/O. If the key appears in only one
segment—or no segment—most of those exact checks discover absence.

A Bloom filter cannot return the record because it stores no values. Instead,
it precedes the exact lookup for each segment:

~~~text
                         false
query key ──> filter ─────────────> skip this segment
                 │
                 │ true: maybe
                 v
          exact segment lookup ──> present or absent
~~~

If the filter returns false, skip the segment. If it returns true, search the
segment's exact index. The name `MayContain` reflects these two outcomes more
accurately than `Contains` or `Exists`.

| Key is actually | Filter returns | Action |
|---|---|---|
| absent | false | Skip the exact lookup. |
| absent | true | Search the exact index; this is a false positive. |
| present | true | Search the exact index and return the value. |
| present | false | The filter is incomplete or inconsistent with the data. |

The first three rows are normal outcomes; the final row indicates an
implementation or data-lifecycle error.

## Exact membership and approximate membership answer different questions

An exact set stores enough information to distinguish every member from every
nonmember according to its equality rule. In Go, `map[string]struct{}` is a
common exact set. If the map contains the canonical key, the key is present in
that map.

A Bloom filter stores a compressed pattern shared by many keys rather than the
keys themselves. It can therefore use less memory per planned key than an
exact set, but an absent key has a configurable probability of appearing to be
present.

| Property | Exact map-backed set | Standard Bloom filter |
|---|---|---|
| Negative answer | Exact | Exact under stated construction assumptions |
| Positive answer | Exact | Possible match only |
| Returns a stored value | No, but membership is exact | No |
| Typical space | Stores keys plus table overhead | Fixed bit array plus metadata |
| Add | Yes | Yes |
| Delete one key | Yes | Not safely |
| Accepts more keys without a declared fixed capacity | Yes, until allocation fails | No; exceeding planned capacity raises false positives |
| Key enumeration | Yes | No |

The relevant space comparison depends on the workload: include both the
filter's memory and the exact lookups that still occur, then compare that total
with the exact lookups performed without a filter.

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

1. If any required bit is zero, the key is absent.
2. If all required bits are one, the key may be present.

Suppose `charlie` maps to 2, 5, and 10. Bit 5 is zero, so `charlie` is
definitely absent. Suppose `delta` maps to 4, 10, and 13. All three bits are
one even though different inserted keys set them. `delta` is a false positive.

The filter does not record which keys set a bit, so it cannot resolve a
collision by comparing keys. When all queried bits are one, they may have been
set by other keys; this is a false positive.

### Connections to earlier units

[Unit 01](../01-big-o-hash-tables/) introduced hash collisions. An exact hash
table keeps keys and uses equality to resolve a collision. A Bloom filter
deliberately lets different keys share bits and stores no keys to compare. That
is why a positive Bloom result remains uncertain instead of being resolved to
one exact entry.

[Unit 02](../02-sequences-sorting-search/) established the contract for a
sorted index and binary search. The opening segment's exact index still has
that job. The Bloom filter only decides whether the service can skip calling
the exact search.

[Unit 04](../04-build-dependency-graphs/) showed why an algorithm needs stable
identities. A Bloom-filter builder and reader likewise need to encode the same
logical key as the same byte sequence. The key-encoding section later in this
chapter defines that requirement more precisely.

## Why a negative is conclusive in the model

Adding a key sets every one of its \(k\) positions. Later additions set more
bits; they do not clear those positions. If a later query for that same key
finds a zero at one required position, then the key could not have completed
the modeled insertion.

This conclusion depends on the filter being constructed, published, and read
under the same conditions.

{{< callout kind="warning" title="Conditions required to avoid false negatives" >}}
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

An implementation therefore needs to preserve these conditions for as long as
callers use the filter, in addition to implementing the bit operations.

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

The formula estimates a probability under its hash and query assumptions. It
does not imply that every batch of 100 absent queries will contain exactly
\(100p\) false positives.

### Effect of the probe count

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

## Capacity and false-positive sizing

Sizing begins with two workload requirements:

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

The 1.14 MiB result applies to this capacity, false-positive target, classic
model, and compact bit-array representation. Other representations may round
the size differently or reserve additional space.

### Exceeding planned capacity raises the false-positive rate

If the same filter receives ten million distinct keys instead of the planned
one million, \(m\) stays fixed. More bits become one. False positives rise,
possibly until nearly every absent query returns “maybe.” Tests of inserted
keys can continue to pass during this degradation because those tests do not
measure how often absent keys also pass.

Planned capacity is therefore an operational limit:

- measure or bound the number of distinct source keys before construction;
- record planned capacity with the filter;
- monitor insertions or bit density;
- rebuild or roll to a new generation before saturation; and
- update the reported false-positive target if the filter exceeds its planned
  capacity.

## End-to-end cost model

Suppose one logical lookup considers \(s\) immutable segments. Let \(a\) be the
fraction of those segments that truly lack the key. Let \(C_f\) be one
filter-query cost and \(C_e\) one exact segment-check cost. These symbols use an
average cost for a rough comparison; individual segments may differ.

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

As in Unit 01, the formula names independent workload variables before making a
timing claim. It also treats key hashing as part of \(C_f\). If key length is
not bounded, call it \(\ell\). An implementation that derives \(k\) probes from
two full-key hashes does \(O(\ell+k)\) work per query, not simply \(O(k)\).

This equation shows which workloads benefit most:

- A large absent fraction \(a\) provides more candidate lookups to skip.
- A large ratio \(C_e/C_f\) means that each skipped exact lookup saves
  substantial work.
- A small \(p\) sends fewer absent candidates to the exact lookup, although it
  generally requires more filter memory and may require more probes.
- A large true-hit fraction limits the saving because those candidates still
  require exact lookups.

If the exact set is already a tiny in-memory map, \(C_e\) may be comparable to
\(C_f\). The filter then adds memory, hashing, and complexity without a useful
payoff.

{{< callout kind="note" title="False-positive probability does not determine latency by itself" >}}
The formula estimates false positives, while request latency also depends on
cache state, storage, decompression, the number of segments considered,
traffic distribution, contention, true hits, and the cost of reading the
filter. End-to-end measurements are needed to determine how much work the
filter saves.
{{< /callout >}}

## Using a Bloom filter before database point lookups

The Cassandra case puts a filter inside the database read path. A service can
also keep a filter in front of a database when the exact question is “does this
key exist?” Check a value cache first, and a negative-result cache only under
its own expiry or invalidation contract. After a cache miss, use the filter
only to decide whether the database lookup can be skipped:

~~~text
request -> value or negative-result cache
              |
              | miss
              v
         Bloom filter
          |         |
      false         true: maybe
          |         |
          v         v
   return absent   exact database read -> cache the result when safe
~~~

This pattern is sometimes called protection against *cache penetration*, which
describes repeated cache misses for keys that do not exist. Without another
check, every request reaches the database. The Bloom filter is not another
value cache; it provides a cheaper way to rule out some of those database
lookups. Redis's
[probabilistic-data-structure documentation](https://redis.io/docs/latest/operate/oss_and_stack/stack-with-enterprise/bloom/)
describes the same use of a negative answer to avoid an expensive network or
disk lookup.

Let \(\lambda_p\) be the rate of cache misses for keys that are present and
\(\lambda_a\) the rate for keys that are absent. Without the filter, the
database receives approximately

\[
\lambda_{\text{before}}=\lambda_p+\lambda_a.
\]

With false-positive probability \(p\), it receives approximately

\[
\lambda_{\text{after}}=\lambda_p+p\lambda_a.
\]

The filter therefore avoids approximately \((1-p)\lambda_a\) database calls.
If 80% of these cache misses are for absent keys and \(p=1\%\), the database
receives about 20.8% of the former traffic: all 20% of present-key requests and
0.8% leaked false positives. If only 5% of misses are for absent keys, the same
filter avoids only 4.95% of the former traffic.

{{< callout kind="warning" title="Database traffic and overload control" >}}
A Bloom filter has no information about database capacity. It filters absent
keys; lookups for existing keys still reach the database. A full-key Bloom
filter also provides no information about whether a range contains records.

False positives mean that some absent-key requests also reach the database. At
one million absent requests per second and \(p=1\%\), about 10,000 requests per
second continue to the database.

The bounded queues and admission control from
[Unit 03](../03-queues-heaps-scheduling/), together with concurrency and rate
limits, control work according to available capacity. They solve a different
problem from membership filtering.
{{< /callout >}}

### Synchronization with a mutable database

The service can return “not found” without consulting the database only if the
filter represents every database-visible key. One way to preserve that order
is to add a new key to the filter before making its database row visible. If
the database write fails, the early filter update may cause an unnecessary
database lookup, but it will not hide a record. Making the row visible first
creates a period in which the filter may incorrectly rule it out.

Pre-insertion also creates a short period in which the filter may say “maybe”
before the row is visible. An exact miss during that period must not install a
negative cache entry that survives the later commit. Invalidate that entry on
commit or give negative results an expiry consistent with the write contract.

Distributed writes may not make this ordering visible to every reader. An
outbox can make filter updates durable, but readers still need to bypass the
filter until the consumer has caught up. Other designs route recent writes
around the filter or publish a filter built from a defined database snapshot.
A deletion can leave its bits set because the resulting stale positive only
causes an exact database miss; clearing bits could also affect other keys. When
the filter's completeness is uncertain, the service should use the database
path.

{{< callout kind="production" title="Localytics: a filter before DynamoDB status checks" >}}
A 2018 Localytics engineering account describes building a filter from
opted-out user IDs, storing versions in S3, and loading them into an ingestion
service. A possible match triggered an exact DynamoDB check, whose result was
then cached; a negative avoided that database call. The
[account](https://eng.localytics.com/saving-money-protecting-privacy-with-bloom-filters/)
supports this description of the request path. It does not describe how the
system covered the delay before a newly opted-out user reached the filter, and
its worked request-rate example contains inconsistent arithmetic, so those
details are not used here as evidence.
{{< /callout >}}

## Go API

Go 1.27 does not provide a standard-library Bloom-filter type. A local API can
make the read path explicit while keeping mutation inside a builder:

~~~go
// Membership is an approximate, read-only view of one set.
// False means absent. True means possibly present.
type Membership interface {
	MayContain(key []byte) bool
}

type Config struct {
	PlannedItems            uint64
	TargetFalsePositiveRate float64
	MaxBytes                uint64
}

func BuildMembership(keys [][]byte, cfg Config) (*Filter, error)
~~~

`BuildMembership` validates the configuration and distinct canonical keys,
adds every key, and returns an immutable concrete filter. A consumer can define
the narrow `Membership` interface shown above when it needs substitution. The
segment lookup uses the filter as follows:

~~~go
func (s *Segment) Lookup(key []byte) (Record, bool) {
	if !s.filter.MayContain(key) {
		return Record{}, false
	}

	// The exact index determines whether the key is present.
	return s.exact.Lookup(key)
}
~~~

The interface still leaves necessary policy in the surrounding contract. Use
the following questions as a design checklist, not as questions with universal
answers supplied by the bit-array implementation:

- Are caller-owned key bytes read only during the call?
- What is the planned capacity?
- Can the result be serialized?
- What exact data generation does it represent?
- Which metrics does a possible-match fallback update?

These choices belong in the API documentation because they cannot be inferred
from the bit-array implementation.

The teaching implementation interprets `MaxBytes == 0` as a 64 MiB default and
rejects larger requested filters before allocating their bit arrays. A
production service should set a limit derived from its own memory budget and
configuration trust boundary.

### Process-local hashing and persistent filters

Go's [`hash/maphash`](https://go.dev/pkg/hash/maphash/) provides seeded hash
functions suitable for byte sequences inside one process. Its `Seed` contract
states that a seed is local to one process and cannot be serialized or
recreated in another process.

The chapter's in-memory example can use `maphash` because it builds and queries
the filter during one process lifetime. A persistent filter needs a different,
stable hashing specification. After a restart, a new `maphash` seed maps the
same key to different positions and therefore cannot query the old bit array.

{{< callout kind="contract" title="Persistent filters require format metadata" >}}
A persistent or networked filter needs a stable specification containing at
least the key encoding, hash algorithm and version, seed if applicable, bit
count, probe count, and represented data generation. If a reader cannot
interpret that metadata, it should use the exact path instead of consulting
the filter.
{{< /callout >}}

### Derive multiple probes from two hashes

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
implementation technique. The writer and reader still need identical,
versioned hashing rules.

## Production case: one filter per Cassandra SSTable

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

This association defines the filter's scope:

> `Filter.db` for SSTable generation G summarizes the partition keys written
> into SSTable generation G.

For a point lookup, a negative answer can remove one SSTable from the more
expensive lookup path. A positive answer cannot prove that the partition is in
the SSTable, so Cassandra still needs its exact indexes and data.

### The table setting trades memory for fewer SSTable checks

Cassandra's current
[`CREATE TABLE` reference](https://cassandra.apache.org/doc/latest/cassandra/reference/cql-commands/create-table.html)
documents `bloom_filter_fp_chance` as a table property. A lower value targets
fewer false positives and uses more memory. A value of 1 disables the filter.

The setting controls the following resource trade-off:

~~~text
lower target p  -> more filter bits -> more memory/disk for filters
                                  -> fewer wasted SSTable checks, if workload fits

higher target p -> fewer filter bits -> less filter memory/disk
                                  -> more absent keys reach exact lookup
~~~

The appropriate value depends on the table's read workload, compaction
strategy, number of candidate SSTables, storage and cache behavior, and node
memory budget.

### Configuration changes apply to newly written SSTables

Cassandra's version-pinned
[Bloom-filter operations guide](https://cassandra.apache.org/doc/3.11/cassandra/operating/bloom_filters.html)
explains a lifecycle detail that follows from the immutable file design. A
filter is calculated when an SSTable is written and persists as that SSTable's
filter component. Changing the table property affects new files. Existing
SSTables retain their existing filters until compaction or another rewrite
regenerates them.

After a change, one table can therefore contain SSTables built with different
false-positive targets. The new setting becomes consistent across existing
data only as those SSTables are rewritten.

### Metrics for false positives and SSTables per read

Cassandra's
[documented table metrics](https://cassandra.apache.org/doc/latest/cassandra/managing/operating/metrics.html)
include:

- `BloomFilterFalsePositives`;
- `BloomFilterFalseRatio`;
- `BloomFilterDiskSpaceUsed`;
- `BloomFilterOffHeapMemoryUsed`; and
- `SSTablesPerReadHistogram`.

The first two show how often absent keys still trigger exact work. The next two
show the space cost. SSTables per read connects the filter to its production
purpose. A filter can meet a statistical target and still have little effect if
most reads are true hits across a small number of candidate SSTables.
Conversely, a modest false-positive target may be valuable when it avoids
expensive negative checks across many files that are not already cached.

{{< callout kind="production" title="Filter scope and lifecycle in Cassandra" >}}
Each Cassandra filter is a versioned component of one immutable SSTable. The
partition keys in that file determine the filter's contents and capacity, and
rewriting the file also rebuilds its filter. Cassandra exposes the filter's
memory use and false positives alongside the number of SSTables accessed per
read.
{{< /callout >}}

## Key encoding

The builder and reader must encode each logical key as the same byte sequence.
For a compound key containing a tenant and an object key, simple concatenation
is ambiguous. These two different keys both become `abc` when their fields are
concatenated without boundaries:

~~~text
(tenant="ab", key="c")
(tenant="a",  key="bc")
~~~

A length-prefixed encoding preserves the field boundaries. The notation below
illustrates the idea; a stored format would define the precise byte layout:

~~~text
2:"ab" 1:"c"
1:"a"  2:"bc"
~~~

The encoding specification also needs to define:

- field order and presence;
- tenant or namespace scoping;
- text normalization and case handling;
- integer width and byte order;
- whether a path includes a trailing separator;
- handling of invalid encodings; and
- schema version.

If the builder and reader use different encodings, a present key can map to
different bit positions and produce a false negative. This extends the
identity lesson from Unit 04: both sides of the algorithm need the same
definition of a key.

## Deletion requires rebuilding or a different filter type

Suppose `alpha` and `bravo` both set bit 7. Clearing bit 7 to delete `alpha`
also makes `bravo` fail its membership test. The filter has no record of how
many keys contributed to that bit.

Common lifecycle choices are:

- build one immutable filter per immutable data segment;
- rebuild a complete filter after exact data changes;
- create a new generation and atomically publish it with the new exact data;
  or
- choose a different data structure with deletion semantics and pay its
  additional space and complexity.

A counting Bloom filter replaces bits with counters and is a related but
different structure. It introduces counter size, overflow, underflow, and
concurrent-update questions, so it is outside this foundations chapter.

## Matching data and filter generations

Consider a service that currently serves generation 41:

~~~text
generation 41 = exact segment 41 + filter 41
~~~

It builds generation 42 in the background. Publishing exact segment 42 while
leaving filter 41 active can hide a key new to generation 42:

~~~text
query new key -> old filter says false -> new exact segment is skipped
~~~

Build and validate the pair separately, then swap one pointer or manifest:

~~~text
before: active -> {data: 41, filter: 41}

build:            {data: 42, filter: 42}
validate: every exact key in data 42 tests maybe in filter 42

after:  active -> {data: 42, filter: 42}
~~~

If a filter is missing, invalid, or detectably corrupt, the reader can preserve
correctness by using the exact path. Length, version, and checksum fields help
the reader detect damaged metadata. In contrast, using an empty filter as a
fallback would report every key as absent and is valid only when the exact set
is also empty.

## Concurrency

A plain Go `[]uint64` filter does not support unsynchronized concurrent writes.
For example, two goroutines could read the same word, set different bits in
their local copies, and then overwrite one of the updates. A later query could
find a zero bit for a completed insertion. More generally, the concurrent
access is a Go data race and therefore has no valid synchronization contract.

Four possible designs are:

- build the filter on one goroutine, then publish it as immutable;
- protect all reads and writes with synchronization;
- update words with an atomic compare-and-swap loop; or
- partition ownership so no two writers mutate the same state.

The VictoriaMetrics v1.148.0 implementation uses atomic word updates. The
examples in this chapter use build-then-publish because immutable ownership
also simplifies generation swaps and testing. Atomic updates add
synchronization and cache-coherence work but do not address capacity or key
encoding.

## Production case: block filtering in VictoriaLogs

VictoriaLogs documents that it divides stored logs into blocks and uses Bloom
filters to skip blocks without a requested word or phrase. If the filter says
no, the query need not unpack and inspect that block. If it says maybe, the
block is processed and the query predicate still decides which log rows match.

The correspondence with the earlier segment example is:

| Bloom model | VictoriaLogs workload |
|---|---|
| represented features | indexed terms or other features used to rule out the query for one block |
| exact check | unpack and process the block |
| useful negative | block cannot contain the requested term |
| false positive | block is processed but produces no match for that term |
| capacity/lifecycle | indexed features represented by one stored block and its metadata |

Although this filter represents log terms rather than partition keys, it uses
the same lookup sequence: a negative skips the candidate block, while a
possible match leads to evaluation of the exact query predicate.

## Policy use: an approximate series limit

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

This use affects admission rather than only the cost of an exact lookup. Once
the limiter believes it has reached its cap, it can treat a Bloom-positive hash
as an already known series and admit it. A false positive can therefore allow
the number of admitted distinct series to exceed the configured limit without
increasing the limiter's internal count. Before the cap, a false positive can
also make an unseen series appear to have been counted already. The
[`vmagent` documentation](https://docs.victoriametrics.com/victoriametrics/vmagent/#cardinality-limiter)
states the product-level result directly: the enforced limit is approximate
and may land below or above the configured value by a small percentage, usually
less than one percent.

{{< callout kind="warning" title="Policy decisions have different error consequences" >}}
When a Bloom filter precedes an exact read, a false positive adds work but does
not change the returned result. When it contributes directly to admission,
quota, billing, authorization, or revocation, a false positive can change
externally visible behavior. Such a design therefore needs to state which
errors are permitted, as the VictoriaMetrics documentation does for its
approximate limiter.
{{< /callout >}}

## Persistent filter format: Git changed paths

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

A persistent filter format needs to define the following. These are format
design questions; this chapter identifies the required decisions but does not
specify all of Git's answers:

- Which exact paths belong to a commit's represented set?
- How are path bytes normalized and hashed?
- Which algorithm and, when applicable, seed define positions?
- What are \(m\), \(k\), and any maximum-size rule?
- Which writer version produced the bytes?
- What should a reader do with an unsupported version?
- How are older graph layers regenerated?

These fields allow another reader or software version to interpret the stored
bit array consistently.

## Fixed-size filter example: Ethereum log Blooms

The
[Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)
defines a 2,048-bit Bloom over log addresses and topics, which contributes to
the block header's `logsBloom`. The released
[Geth v1.17.3 range-filter path](https://github.com/ethereum/go-ethereum/blob/v1.17.3/eth/filters/filter.go)
names the correct sequence in code: its Bloom check decides whether a block is
interesting, and a possible match leads to `checkMatches`, which filters actual
logs.

The filter's bit count is part of a protocol format, not a per-operator memory
setting. As the number of represented log values in a block grows, the fixed
array becomes denser and more absent queries pass.

[Geth issue 25336](https://github.com/ethereum/go-ethereum/issues/25336)
recorded a performance complaint about high false-positive rates and slow log
queries for dense workloads. Later filtering work improved the implementation,
and the issue was closed in 2025. The report therefore describes the behavior
of that earlier implementation rather than current Geth, but it illustrates
the effect of increasing \(n\) while keeping \(m\) fixed. More inserted values
set more bits, so a larger fraction of absent queries pass the filter.

Correctness tests performed with a lightly occupied filter do not measure this
degradation. Operational measurements can instead track bit density, possible
matches, exact matches, exact misses, and indexed log values per block.

## Operational metrics

[Unit 03](../03-queues-heaps-scheduling/) distinguished queue activity from
completed work. Bloom-filter metrics need a similar separation between filter
decisions, exact lookup results, and the work avoided:

| Measurement | What it tells you |
|---|---|
| filter queries | traffic offered to the filter |
| definite negatives | candidates skipped |
| possible matches | candidates sent to exact lookup |
| exact matches after maybe | true positives |
| exact misses after maybe | false positives |
| exact checks avoided | direct benefit, if the counter's baseline is clear |
| whole database calls avoided | service-boundary benefit after cache misses |
| filter bypass or fail-open calls | downstream load caused by an unavailable or untrusted filter |
| filter bytes | space cost |
| set-bit density—the fraction of bits that are one | saturation signal |
| planned capacity and inserted count | whether the sizing premise still holds |
| modeled and observed false-positive rates | expected versus workload-specific wasted work |
| configured maximum bytes | whether a sizing request can exceed the allocation budget |
| build/data generation and format version | lifecycle agreement |
| database-to-filter update lag | whether the filter may be missing recent keys |
| exact-check latency or bytes | value of each saved check |

In a controlled test or shadow-read sample where the exact source is checked
for every query, the observed false-positive ratio is

\[
\frac{\text{possible match followed by exact miss}}
     {\text{all queried exact-source nonmembers}}.
\]

The normal production path does not check the exact source after a filter says
no, so it cannot independently observe that denominator. Under the
no-false-negative invariant, it can estimate the same ratio as

\[
\frac{\text{possible match followed by exact miss}}
     {\text{definite negatives} +
      \text{possible matches followed by exact miss}}.
\]

Dashboard documentation should state whether it uses exact shadow evidence or
this invariant-based estimate. Neither ratio is the same as

~~~text
possible matches / all queries
~~~

because possible matches include real members. A hit-heavy workload can make
that second ratio high even when the filter behaves exactly as designed.

### Repeated queries produce repeated false positives

The probability model describes absent queries drawn under a stated
distribution, but hashing is deterministic. Once an absent key produces a
false positive, subsequent queries for that key normally produce the same
result. A global average of 1% can therefore coexist with one frequently
requested missing key repeatedly causing an expensive exact lookup. Systems
can report which absent keys most often produce “maybe” and, where the data
lifecycle permits it, cache the exact negative result.

## Testing requirements

A filter test suite should cover its invariants, its integration with the exact
source, and its behavior as occupancy increases.

### Invariant tests

After construction, every exact member must return `MayContain == true`:

~~~go
for _, key := range exactKeys {
	if !filter.MayContain(key) {
		t.Fatalf("false negative for %q", key)
	}
}
~~~

Run this across empty input, one item, bit positions on both sides of a
machine-word boundary, and maximum planned capacity. If the builder requires a
set of distinct inputs, also verify that it rejects duplicates. For a persistent
format, build and query across the supported writer/reader versions.

Property fuzzing can extend these fixtures with arbitrary binary keys. Preserve
the same two properties for every generated input: inserted keys never return
false, and the complete filtered lookup returns the same record and presence
bit as the exact-only path.

### Caller-contract tests

Construct or search for an absent key that returns “maybe,” then verify that
the caller consults the exact source and returns absent. This directly tests
the positive-result contract without waiting for a random false positive
during the test.

### Generation tests

Attempt to pair data generation 42 with filter generation 41. The expected
behavior is either rejection of the pair or an exact lookup without the
filter.

### Statistical tests

Insert a fixed, deterministic data set and query a separate fixed absent set.
Compare the measured ratio with a tolerance appropriate to the sample count,
rather than requiring an exact percentage from every sample.

### Scaling tests

Hold \(m\) and \(k\) fixed while increasing insertions above planned \(n\).
Assert deterministic bit density or exact-check counts for the fixture, then
use benchmarks to measure the associated change in execution time.

Also benchmark construction and rebuilds. An immutable-generation design pays
for hashing, transient duplicate validation, and bit-array allocation outside
the read path. Report both those temporary allocations and the retained filter
bytes so operators can budget rebuild overlap.

### Concurrency tests

An immutable filter should permit concurrent `MayContain` calls after
publication. Exercise concurrent member and nonmember queries under the race
detector, and make every reader's completion and error observable to the test
goroutine.

## Workloads where a Bloom filter is unsuitable

A large data set alone does not justify a Bloom filter. The structure is a poor
fit when:

- most queried candidates are true members, so exact work remains;
- the exact membership structure is already small and memory-resident;
- queries need values, enumeration, ordering, ranges, or counts;
- queries need full-key prefixes but the represented set contains only complete
  keys; a prefix-aware design would need to insert and define prefixes
  separately;
- keys must be deleted individually and immutable generations are unavailable;
- the number of distinct keys cannot be bounded and no scalable or rebuild
  scheme exists;
- a possible match would be treated as proof of presence, or a policy decision
  cannot tolerate the filter's declared error;
- key encoding or generation ownership cannot be made stable;
- filter I/O or cache footprint rivals the exact check it is meant to avoid;
  or
- traffic can concentrate on stable false-positive keys and repeatedly reach
  the exact source.

A sorted index with binary search may already provide an adequate lookup path.
For smaller sets, an exact map may be simpler, while a min/max range check may
exclude most files in an ordered data set. The comparison should include the
complete lookup path and operational cost rather than only bytes per key.

## Investigate a lifecycle failure

The [Filter from Yesterday Wheel](wheel/) begins with a newly published key
that the ordinary read path reports missing. Work from the incoming report,
keep several explanations alive, and select evidence before opening the
implementation. The exercise isolates the rule that exact data and its
negative-lookup filter must be published as one generation.

## Sources and the claims they support

| Claim | Source role | Source or evidence |
|---|---|---|
| The original structure trades space for allowable membership errors | CS model | [Bloom's 1970 paper](https://www.cs.princeton.edu/courses/archive/spr05/cos598E/bib/p422-bloom.pdf) |
| The usual \(p\) approximation follows from bit occupancy assumptions | CS model | [CMU 15-853 lecture notes](https://www.cs.cmu.edu/~15853-f19/scribes/lec20.pdf) |
| Two base hashes can derive the probe sequence | CS model | [Kirsch and Mitzenmacher](https://www.eecs.harvard.edu/~michaelm/postscripts/rsa2008.pdf) |
| A `maphash.Seed` is process-local and cannot be recreated elsewhere | Go contract | [`hash/maphash`](https://go.dev/pkg/hash/maphash/) |
| Cassandra stores a partition-key Bloom with each SSTable | Production evidence | [Cassandra storage engine](https://cassandra.apache.org/doc/latest/cassandra/architecture/storage-engine.html) |
| A negative can avoid an expensive network or disk lookup | Product documentation | [Redis probabilistic data structures](https://redis.io/docs/latest/operate/oss_and_stack/stack-with-enterprise/bloom/) |
| A service used a filter before exact DynamoDB checks | Production account, with stated limitations | [Localytics engineering account](https://eng.localytics.com/saving-money-protecting-privacy-with-bloom-filters/) |
| VictoriaLogs skips blocks for absent words or phrases | Production evidence | [VictoriaLogs FAQ](https://docs.victoriametrics.com/victorialogs/faq/) |
| Git persists versioned changed-path filters | Production evidence | [Git commit-graph docs](https://git-scm.com/docs/commit-graph-format.html) |
| Geth checks actual logs after a Bloom possible match | Go implementation and production evidence | [Geth v1.17.3 filter source](https://github.com/ethereum/go-ethereum/blob/v1.17.3/eth/filters/filter.go) |

The sources support different kinds of claims. Research papers establish the
probability model, Go documentation defines language and library behavior, and
production documentation records choices made for particular systems. Local
benchmarks describe only the workload and implementation being tested.
