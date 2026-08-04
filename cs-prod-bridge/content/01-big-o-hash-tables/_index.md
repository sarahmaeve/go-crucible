+++
title = 'Big-O and hash tables'
description = 'A production-oriented introduction to Big-O, hash tables, and Go maps.'
weight = 1
+++

**Unit 01 · Foundations**

# Big-O and hash tables

{{< lead >}}Learn to spot three production signals: code that repeatedly scans
the same data, workloads dominated by exact-key lookups, and maps whose retained
keys grow without a bound.{{< /lead >}}

{{< callout kind="production" title="Incoming reports" >}}
- An inventory reconciler is fast with 50 test objects but consumes a full CPU
  core in a 50,000-object cluster.
- A network collector spends most of its CPU time matching each event to an
  endpoint record it has already searched many times.
- A map-backed cache performs fast lookups, yet its heap grows throughout the
  day because almost every request creates a new key.
{{< /callout >}}

All three failures involve growth. The reconciler and collector repeat more
work as their inputs grow. The cache retains more state as new keys arrive.
Big-O describes how the work grows; distinct-key count and entry lifetime
explain the retained state.

This unit connects those ideas to Go maps. By the end, you should be able to:

- derive the work performed by one scan, repeated scans, and a prebuilt index;
- distinguish worst-case, expected, and amortized costs;
- state the assumptions behind expected constant-time hash-table operations;
- separate Go's map contract from its current implementation;
- include construction cost, update rate, memory, distinct-key count, and
  ownership in a design decision; and
- test a scaling claim with benchmarks and profiles.

## Start with a prediction

The running example is a batch enrichment service. Its workload has three
relevant quantities:

- \(e\) is the number of events in one batch.
- \(m\) is the number of endpoint records in the metadata snapshot.
- \(u\) is the number of distinct endpoint keys, so \(u \le m\).

Each event asks one exact-key question: which metadata record has this IP
address? Before choosing a data structure, answer four questions:

1. If every event scans the metadata slice, how many key comparisons can the
   batch perform?
2. What work and storage does a one-time index add?
3. When might the simpler scan still be faster?
4. What changes if the metadata snapshot is replaced continuously?

{{< callout kind="note" title="Count the operations first" >}}
A scan can find a match after one comparison, but a missing key or a match at
the far end requires \(m\) comparisons. Repeating that worst case for \(e\)
events produces \(e \times m\) comparisons.

Building an index examines the \(m\) metadata records once and stores \(u\)
entries. Enriching the batch then performs \(e\) index lookups. This count
does not yet prove how long a lookup takes; that requires the hash-table model
and its assumptions.
{{< /callout >}}

## What the notation means

**Asymptotic notation** describes how resource use grows as inputs become
large. It ignores constant factors and small-input effects, so it can
distinguish one pass through the metadata from a metadata pass repeated for
every event. It does not predict milliseconds on a particular machine.

| Notation | Meaning | Plain-language interpretation |
|---|---|---|
| \(O(f(n))\) | An upper bound on growth | For sufficiently large inputs, growth is no faster than this bound, apart from a constant factor. |
| \(\Omega(f(n))\) | A lower bound on growth | For sufficiently large inputs, growth is at least this fast, apart from a constant factor. |
| \(\Theta(f(n))\) | Matching upper and lower bounds | This describes the growth tightly, apart from constant factors. |

The worst-case repeated scan is:

\[
\Theta(e \times m)
\]

If event count and metadata count both grow in proportion to a common input
size \(n\), then \(\Theta(e \times m)\) becomes \(\Theta(n^2)\). Keeping
\(e\) and \(m\) separate avoids assuming that traffic and metadata grow
together. A traffic increase changes \(e\); onboarding a large region may
change \(m\).

Under the assumptions described in the next section, building the index and
enriching the batch take expected \(\Theta(m + e)\) time. The completed index
contains \(u\) logical entries, so its additional logical storage is
\(\Theta(u)\).

{{< callout kind="note" title="A formula is not a timing result" >}}
The formula predicts how work changes when \(e\), \(m\), or \(u\) changes.
A benchmark finds the input sizes at which one implementation becomes faster
than another on a particular machine. A profile identifies the code consuming
the measured CPU time or memory. These are different questions.
{{< /callout >}}

## How a hash table changes the work

Each event makes a slice scan revisit metadata records examined for earlier
events. A hash table examines the metadata once while building an index. For
each key, it computes a hash and uses that hash to select a small group of
candidate entries. Different keys can select the same group; this is a
**collision**, and the table must compare candidate keys to find the right one.

Three cost descriptions matter:

- **Expected cost:** if the hash function spreads keys across the table and the
  table is resized before it becomes too full, a lookup examines only a small
  number of candidate entries on average. For bounded-size keys, this is
  expected \(\Theta(1)\).
- **Amortized cost:** one insertion may trigger a resize and move many entries.
  Across a long sequence of insertions, those occasional resizes can still add
  only constant average work per insertion.
- **Worst-case cost:** if many keys collide, one lookup may compare against as
  many as \(\Theta(u)\) entries in the index.

Key size is part of the assumption. Hashing and comparing an arbitrary string
takes time proportional to the string's length. This unit uses textual IP
addresses, whose length is bounded, so it treats hashing and comparison as
constant-size work.

This is why “hash lookup is \(O(1)\)” is incomplete. The useful claim is
expected constant time for bounded-size keys under assumptions about hashing,
collisions, and table growth.

### Required CS sources

Read the language-neutral model before treating a particular language
implementation as a general guarantee:

- [MIT 6.006: asymptotic complexity notes](https://live.ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/c6d8f06c6f11e3342633dec85498f551_MIT6_006S20_r01.pdf) — formal definitions and comparison rules.
- [MIT 6.006: hashing lecture](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/ce9e94705b914598ce78a00a70a1f734_MIT6_006S20_lec4.pdf) — collisions, table occupancy, chaining, open addressing, and expected cost.
- [Open Data Structures: Hash Tables](https://opendatastructures.org/ods-python/5_Hash_Tables.html) — a second free treatment of collision handling and resizing.

## What Go promises about maps

The CS model describes hash tables in general. The
[Go specification](https://go.dev/ref/spec#Map_types) separately defines what
Go programs may rely on. It specifies map behavior but promises no time bound
for lookup, insertion, or deletion.

{{< callout kind="contract" title="Language and API contract" >}}
- Each key identifies at most one map entry.
- The key type must be *comparable*: its values must support `==` and `!=`.
- Slices, maps, and functions are not comparable and therefore cannot be map
  keys.
- A nil map can be read; assigning an entry to it panics.
- Iteration order is unspecified and may differ from one iteration to the next.
- The comma-ok lookup form distinguishes a missing key from a stored zero value.
{{< /callout >}}

The official [Go maps in action](https://go.dev/blog/maps) article covers zero
values, struct keys, and concurrent access. The same map may be read by
multiple goroutines if none of them writes to it. Access involving a write
requires synchronization or a design in which one goroutine owns mutation.

{{< callout kind="warning" title="Not guaranteed by Go" >}}
The Go specification does not guarantee that every map lookup takes constant
time.
{{< /callout >}}

## Go 1.24 and Swiss Tables

Go 1.24 replaced the earlier built-in map implementation with Swiss Tables.
The Go team's
[Swiss Table implementation article](https://go.dev/blog/swisstable) explains
control bytes, groups of candidate slots, probing, table growth, and the
constraints created by Go's iteration rules.

The examples in this repository target Go 1.26. Check `go version` when
interpreting a profile because runtime implementation details can change
between releases. Swiss Tables remain an **implementation detail**, not part
of the language contract.

The current
[`internal/runtime/maps` source](https://go.dev/src/internal/runtime/maps/map.go)
is an optional deep read after the higher-level article.

## Where maps fit—and where they do not

A map is useful here because the service repeatedly looks up one exact key in
the same collection:

- Which owner and region correspond to this IP address?
- Have we already seen this event ID?
- What counter belongs to this bounded label set?
- Which cached object has this UID?

Order, range, prefix, and minimum-value queries need different operations and
may need different structures. Choose the structure from the queries the
service performs, not from familiarity.

### Failure modes to check

Even when exact-key lookup is appropriate, the surrounding design can erase
the CPU benefit or move the failure to memory:

- A batch scans the same collection once for every input item.
- Code rebuilds the index for every lookup instead of once per snapshot.
- A cache key includes a request ID or timestamp, creating a new retained entry
  for almost every request.
- Entries have no owner, expiry rule, or other removal policy.
- The scan and index implement different duplicate-key behavior.
- A benchmark uses key sizes or distributions unlike production traffic.
- Lower CPU use is offset by retained memory and garbage-collection work.
- Multiple goroutines access the same ordinary map while one of them writes.

For a live map or cache, **cardinality** means the number of distinct keys
currently retained, or retained within a stated time window. That count is
often more useful than request count because it determines how many entries
remain reachable.

## Work the unit

The reading supplies the vocabulary; the exercises make the predicted growth
observable. Preserve the opportunity to predict before measuring:

1. Run the [enrichment lab](lab/). Compare a
   scan, an index built for one batch, and an index reused across batches.
2. Read the [network-enrichment case study](case-study/) and challenge its
   workload and correctness assumptions.
3. Attempt the [Wheel scenarios](wheel/)
   from their incoming reports before opening source.

## Translate the reasoning into an interview answer

Avoid stopping at “use a hash map for \(O(1)\) lookup.” Name the workload,
the cost claim, its assumptions, and the added storage:

> With \(e\) events and \(m\) metadata records, repeated scanning performs
> worst-case \(\Theta(e \times m)\) comparisons. Building an index examines
> the \(m\) records once and stores \(u\) distinct keys. For bounded-size
> keys and normal hashing behavior, construction takes expected
> \(\Theta(m)\), the event pass takes expected \(\Theta(e)\), and the
> index uses \(\Theta(u)\) logical storage. I would clarify reuse, update
> rate, duplicate handling, cardinality bounds, and then benchmark the relevant
> workload.

Finish by saying which measurement could disprove the choice. That turns the
notation into an engineering argument rather than a memorized answer.
