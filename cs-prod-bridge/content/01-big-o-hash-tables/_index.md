+++
title = 'Big-O and hash tables'
description = 'Use Big-O and Go maps to explain CPU growth, lookup cost, and memory growth in production services.'
weight = 1
+++

**Unit 01 · Foundations**

# Big-O and hash tables

{{< lead >}}Learn when repeated scanning makes CPU work grow, when a Go map can
reduce that work, and why a map can still use too much memory.{{< /lead >}}

{{< callout kind="production" title="Incoming reports" >}}
- An inventory reconciler is fast with 50 test objects. In a cluster with
  50,000 objects, it uses a full CPU core.
- A network collector searches the same endpoint records for every event. This
  work uses most of its CPU time.
- A map-backed cache has fast lookups, but its heap grows all day. Almost every
  request creates a new key.
{{< /callout >}}

The first two reports concern CPU work. Both services repeat more work as their
inputs grow. The third report concerns memory. The cache keeps more keys as new
requests arrive.

Big-O helps explain how the CPU work grows. The number of distinct keys and the
time that each key stays in memory help explain memory growth.

By the end of this unit, you should be able to:

- calculate the work for one scan, repeated scans, and a prebuilt index;
- explain worst-case, expected, and amortized costs;
- state when a hash-table operation has expected constant cost;
- separate what Go guarantees from how Go currently implements maps;
- compare construction cost, update rate, memory use, key count, and ownership;
  and
- use benchmarks and profiles to test a claim about growth.

## Start with a prediction

The running example is a batch enrichment service. Keep track of three input
quantities:

- \(e\) is the number of events in one batch.
- \(m\) is the number of endpoint records in the metadata snapshot.
- \(u\) is the number of distinct endpoint keys, so \(u \le m\). Production
  engineers often call this count **cardinality**.

For each event, the service must find the metadata record with the same IP
address. Before you choose a data structure, answer four questions:

1. If every event scans the metadata slice, how many key comparisons can the
   batch perform?
2. What work and storage does a one-time index add?
3. When might the simpler scan still be faster?
4. What changes if the metadata snapshot is replaced continuously?

{{< callout kind="note" title="Count the operations first" >}}
A scan can find a match after one comparison. A missing key, or a match at the
far end, requires \(m\) comparisons. If this happens for all \(e\) events, the
batch makes \(e \times m\) comparisons.

To build an index, the service examines the \(m\) metadata records once and
stores \(u\) entries. It then does \(e\) index lookups. These counts do not tell
us how long each lookup takes. For that, we need a model of hash-table behavior.
{{< /callout >}}

## How Big-O describes growth

**Asymptotic notation** describes how work or storage changes as an input grows.
It ignores fixed factors and small-input effects. This lets us compare one pass
through the metadata with one pass for every event. It does not predict an
exact running time on a particular machine.

| Notation | Meaning | Plain-language interpretation |
|---|---|---|
| \(O(f(n))\) | An upper bound on growth | For large enough inputs, growth is no faster than this bound, apart from a constant factor. |
| \(\Omega(f(n))\) | A lower bound on growth | For large enough inputs, growth is at least this fast, apart from a constant factor. |
| \(\Theta(f(n))\) | Matching upper and lower bounds | For large enough inputs, this is a tight description of growth, apart from constant factors. |

The worst-case repeated scan is:

\[
\Theta(e \times m)
\]

If event count and metadata count grow together, both can be represented by
one input size, \(n\). In that case, \(\Theta(e \times m)\) becomes
\(\Theta(n^2)\).

In production, these inputs can change independently. More traffic increases
\(e\). Adding a large region can increase \(m\). Keep the variables separate
when the workload does not make them grow together.

Under the conditions in the next section, building the index and enriching the
batch take expected \(\Theta(m + e)\) time. The completed index contains \(u\)
logical entries. Its additional logical storage is \(\Theta(u)\).

{{< callout kind="note" title="A formula is not a timing result" >}}
The formula predicts how work changes when \(e\), \(m\), or \(u\) changes. A
benchmark compares running time for selected input sizes on a particular
machine. A profile shows which code uses the measured CPU time or memory. Each
tool answers a different question.
{{< /callout >}}

## Why an index reduces repeated work

The slice scan revisits metadata records for every event. A hash table examines
the metadata once when it builds an index.

For each key, the table computes a hash. It uses the hash to select a small
group of possible entries. Two different keys can select the same group. This
is a **collision**. When a collision occurs, the table compares keys to find the
correct entry.

Three cost descriptions matter:

- **Expected cost:** The hash function must spread keys across the table, and
  the table must grow before it becomes too full. Under these conditions, a
  lookup examines few entries on average. For bounded-size keys, the expected
  cost is \(\Theta(1)\).
- **Amortized cost:** One insertion can cause the table to grow and move many
  entries. Such costly insertions happen only sometimes. Across a long series
  of insertions, table growth can add constant average work to each insertion.
- **Worst-case cost:** Many keys can collide. One lookup can then compare as
  many as \(\Theta(u)\) entries.

Key size also matters. The time to hash and compare a string grows with the
string's length. This unit uses textual IP addresses with a bounded length. It
therefore treats each hash and comparison as constant-size work.

Thus, “hash lookup is \(O(1)\)” is not a complete claim. State that the lookup
has expected constant cost for bounded-size keys. Also state the conditions
about hashing, collisions, and table growth.

### Read the CS model

Read the language-neutral model before you treat one language implementation as
a general guarantee:

- [MIT 6.006: asymptotic complexity notes](https://live.ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/c6d8f06c6f11e3342633dec85498f551_MIT6_006S20_r01.pdf) — formal definitions and comparison rules.
- [MIT 6.006: hashing lecture](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/ce9e94705b914598ce78a00a70a1f734_MIT6_006S20_lec4.pdf) — collisions, table occupancy, chaining, open addressing, and expected cost.
- [Open Data Structures: Hash Tables](https://opendatastructures.org/ods-python/5_Hash_Tables.html) — a second free treatment of collision handling and resizing.

## What Go guarantees about maps

The CS model describes hash tables in general. The
[Go specification](https://go.dev/ref/spec#Map_types) defines what a Go program
can rely on. It defines map behavior, but it gives no time bound for lookup,
insertion, or deletion.

{{< callout kind="contract" title="Language and API contract" >}}
- Each key identifies at most one map entry.
- The key type must be *comparable*. Its values must support `==` and `!=`.
- Slices, maps, and functions are not comparable. They cannot be map keys.
- A nil map can be read; assigning an entry to it panics.
- Iteration order is unspecified and may differ from one iteration to the next.
- The comma-ok lookup form distinguishes a missing key from a stored zero value.
{{< /callout >}}

The official [Go maps in action](https://go.dev/blog/maps) article explains zero
values, struct keys, and concurrent access. Multiple goroutines can read the
same map when no goroutine writes to it. If a goroutine writes to the map, the
program must synchronize access or give one goroutine sole ownership of writes.

{{< callout kind="warning" title="Not guaranteed by Go" >}}
The Go specification does not guarantee that every map lookup takes constant
time.
{{< /callout >}}

## Optional: how Go currently implements maps

Go 1.24 replaced the earlier built-in map implementation with Swiss Tables. The
Go team's
[Swiss Table implementation article](https://go.dev/blog/swisstable) explains
control bytes, groups of candidate slots, probing, table growth, and the
constraints created by Go's iteration rules.

The examples in this repository target Go 1.27. Before you interpret a profile,
run `go version`. Runtime details can change between releases. Swiss Tables are
an **implementation detail**, not part of the language contract.

The current
[`internal/runtime/maps` source](https://go.dev/src/internal/runtime/maps/map.go)
is an optional deep read after the higher-level article.

## Use maps for the operations they support

A map is useful here because the service repeatedly looks up one exact key in
the same collection. Examples include:

- Which owner and region correspond to this IP address?
- Have we already seen this event ID?
- What counter belongs to this bounded label set?
- Which cached object has this UID?

Maps do not directly answer order, range, prefix, or minimum-value queries.
Those queries can require different data structures. Choose a structure that
supports the operations your service performs.

### Common design mistakes

Exact-key lookup can be the correct operation while the surrounding design is
still wrong. Check for these mistakes:

- A batch scans the same collection once for every input item.
- Code rebuilds the index for every lookup instead of once per snapshot.
- A cache key includes a request ID or timestamp. Almost every request then
  creates an entry that stays in memory.
- Entries have no owner, expiration rule, or other removal policy.
- The scan and index implement different duplicate-key behavior.
- A benchmark uses key sizes or distributions unlike production traffic.
- The map reduces CPU work but increases memory and garbage-collection work.
- Multiple goroutines access the same ordinary map while one of them writes.

For a live map or cache, **cardinality** means the number of distinct keys kept
in memory, either now or during a specified time. Request count does not tell
you how many entries remain reachable. Key cardinality does.

## Do the exercises

The reading explains the model. The exercises let you observe the predicted
growth. Make each prediction before you run the measurement:

1. Run the [enrichment lab](lab/). Compare a
   scan, an index built for one batch, and an index reused across batches.
2. Read the [network-enrichment case study](case-study/). Check its assumptions
   about workload and correctness.
3. Start each [Wheel scenario](wheel/) from its incoming report. Do not open the
   source code first.

## Explain the decision to a colleague

Do not stop at “use a hash map for \(O(1)\) lookup.” Name the inputs, the cost,
the conditions behind that cost, and the added storage:

> The batch contains \(e\) events and \(m\) metadata records. Repeated scanning
> does up to \(\Theta(e \times m)\) comparisons. An index examines the \(m\)
> records once and stores \(u\) distinct keys. With bounded-size keys and normal
> hashing behavior, building the index takes expected \(\Theta(m)\) time. The
> event pass takes expected \(\Theta(e)\) time, and the index uses
> \(\Theta(u)\) logical storage. Before choosing it, I would check how often the
> service reuses the index, how often metadata changes, how it handles duplicate
> keys, and how many keys it can retain. I would then benchmark the real
> workload.

Finish with the measurement that could show the decision is wrong. This turns
the notation into an engineering argument, not a memorized answer.
