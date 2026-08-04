+++
title = 'Indexing a network-enrichment stream'
description = 'A production case study of replacing repeated metadata scans with a reusable Go map index.'
weight = 2
+++

**Unit 01 · Production case study**

# Indexing a network-enrichment stream

{{< lead >}}Building one IP index per metadata snapshot removes a repeated
scan. It also requires explicit rules for duplicates, missing records, updates,
memory, and concurrent access.{{< /lead >}}

## Production source and teaching boundary

Dropbox's
[NetFlash engineering report](https://dropbox.tech/infrastructure/netflash-tracking-dropbox-network-traffic-in-real-time-with-elasticsearch)
describes Go collectors that process billions of network-log entries per day
and enrich those entries with metadata associated with IP addresses. This case
borrows that event-to-metadata relationship.

{{< callout kind="note" title="What is real, and what is synthetic" >}}
The NetFlash report supplies the production setting. The batch API,
`Metadata` type, owner and region fields, duplicate policy, fixtures, and
benchmarks in this repository are synthetic. They neither reproduce Dropbox's
implementation nor claim its performance results.
{{< /callout >}}

## The workload

The teaching service receives network events and one metadata snapshot. Each
event contains a source IP address and a byte count. Before aggregation and
storage, the service adds the endpoint's owner and region.

```text
event:    {source_ip: 10.20.3.7, bytes: 8192}
metadata: {ip: 10.20.3.7, owner: payments, region: us-west}
```

The cost model uses three quantities:

- \(e\) is the number of events in the batch.
- \(m\) is the number of metadata records in the snapshot.
- \(u\) is the number of distinct IP keys in those records, so \(u \le m\).

The operation is an exact match: find the metadata record whose IP equals the
event's source IP.

The lab uses canonical textual IPv4 addresses, so key length is bounded. A
production system accepting IPv6 text must canonicalize equivalent spellings
or use a comparable address value such as
[`netip.Addr`](https://pkg.go.dev/net/netip#Addr). Otherwise two strings can
denote the same address while behaving as different map keys.

{{< callout kind="production" title="Representative activation condition" >}}
Suppose a batch contains 100,000 events and a newly expanded metadata snapshot
contains 50,000 records. A worst-case scan can perform five billion key
comparisons. The example is synthetic, but the arithmetic identifies the
workload size that activates the design problem.
{{< /callout >}}

## First design: scan the slice

The simplest implementation scans the metadata slice from newest to oldest
until it finds the event's IP. The code is direct, allocates no index, and can
be fastest when a small snapshot is searched only a few times.

A match in the first position needs one comparison. A missing key or a match in
the final position needs \(m\) comparisons. Repeating that worst case for
\(e\) events gives:

\[
\Theta(e \times m)
\]

If \(e\) and \(m\) both grow in proportion to a common size \(n\), this
becomes \(\Theta(n^2)\). Keeping the variables separate avoids assuming that
event traffic and metadata size grow together.

## Second design: build one index

The scan becomes expensive because every event revisits metadata examined for
earlier events. The alternative builds one `map[string]Metadata` from the
snapshot, then reuses it for all events in the batch.

| Stage | Expected time | Additional storage |
|---|---:|---:|
| Build the index | \(\Theta(m)\) | \(\Theta(u)\) logical entries, where \(u \le m\) |
| Enrich all events | \(\Theta(e)\) | Result storage common to both designs |
| Complete batch | \(\Theta(m + e)\) | \(\Theta(u)\) beyond the scan |

These expected time bounds assume that hashing distributes keys well and that
hashing and comparing one key are constant-size operations. The bounded textual
IP keys in this teaching workload satisfy the key-size assumption. The Go
language itself does not guarantee these time bounds for maps.

The index constructor examines all \(m\) records even when some IPs are
duplicates. The completed map contains \(u\) entries. The lab also passes
`m` as the map's capacity hint, so the runtime may reserve capacity based on
the record count rather than the final number of distinct keys.

Constant factors still determine the measured crossover. Allocation, string
hashing, CPU-cache behavior, and garbage collection can make the scan faster
for a small snapshot used only once. Big-O predicts how the alternatives grow;
a benchmark shows where one wins for a particular workload and machine.

{{< callout kind="warning" title="Include index construction" >}}
A benchmark of a prebuilt index measures lookup cost but excludes construction.
If the service builds an \(m\)-record map inside the event loop, it repeats
that construction \(e\) times and returns to expected
\(\Theta(e \times m)\) work.
{{< /callout >}}

## Define equivalent results before comparing speed

The index is an optimization only if it returns the same result as the scan.
Both implementations therefore need explicit rules for duplicate IPs, missing
metadata, and snapshot lifetime.

### Duplicate IP addresses

In Go, `index[record.IP] = record` replaces the current value for that key.
The lab uses **last record wins**: the map processes records from first to last,
and the scan searches from last to first.

That rule is meaningful only if input order is stable and “last” has a
documented meaning, such as newest version. A production service could instead
reject a snapshot containing conflicting records. What matters is that both
implementations apply the same conflict rule deliberately.

### Missing metadata

A lookup can also find no record. The lab returns `Found == false`, keeping
absence distinct from metadata whose owner or region happens to be an empty
string.

A production collector must then decide what to do with the unenriched event:
retain it, drop it, retry after a metadata refresh, or emit an error metric.
The map cannot choose that service behavior.

### Snapshot replacement and ownership

So far, both implementations assume that the metadata snapshot remains fixed
throughout the batch. A continuously running collector needs a replacement
policy. It can publish a new immutable index, confine all mutation to one
goroutine, or protect readers and writers with synchronization.

Publishing immutable indexes gives every batch a consistent snapshot, but the
service must define when new batches observe a replacement and when old indexes
can be reclaimed.

### Memory and distinct-key count

One index retains \(u\) logical entries. The measured heap also includes map
bookkeeping, reserved capacity, strings, and any old index still reachable by a
reader.

Memory can grow even while lookup remains fast. The current batch design bounds
one index by the current snapshot, but a process that accidentally retains
every replaced snapshot can keep several full indexes alive. A metadata source
that continually adds IPs without removing retired endpoints can also increase
\(u\) from one snapshot to the next.

### Concurrent access

The same ordinary Go map may be read by multiple goroutines if none of them
writes to it. If one goroutine modifies the index while others read it, the
service needs synchronization. Publishing a completed immutable index avoids
in-place writes; an in-place update design must coordinate every reader and
writer.

## Measure the decision

The growth model gives a prediction, not a timing result. For the representative
batch above, the scan can perform five billion comparisons. The indexed design
examines 50,000 records once and then performs 100,000 expected constant-time
lookups. Those counts justify testing the index; they do not supply a latency
number.

The [enrichment lab](../lab/) contains
correctness tests and benchmarks at several event and metadata sizes. Before
running it, predict:

- what changes when \(e\) doubles while \(m\) stays fixed;
- what changes when \(m\) doubles while \(e\) stays fixed;
- how construction appears in the end-to-end index benchmark; and
- why the prebuilt-index benchmark measures lookup cost but excludes
  construction.

Use three kinds of evidence:

1. **Derive comparison counts.** For the scan, calculate how key comparisons
   change with \(e\) and \(m\). This prediction does not depend on machine
   speed.
2. **Run a benchmark series.** Measure the scan, an index built per batch, and a
   reused index across several workload sizes.
3. **Capture a profile.** Confirm which code consumes the measured CPU time or
   retained memory.

One timing says which implementation won for one input on one machine. A
series shows whether the measured growth matches the prediction. A profile
helps explain a result that does not.

## A production hash-table failure

A local benchmark can show when this index becomes faster than the repeated
scan. It cannot prove that every hash-table implementation will remain fast
with production keys.

Dropbox's report on
[automated performance regression detection](https://dropbox.tech/infrastructure/keeping-sync-fast-with-automated-performance-regression-detectio)
describes a third-party map with a weak hash function. Related entries landed
in the same buckets, changing insertion and lookup from constant to linear
time. The failure was in that implementation with that key workload; it is not
a claim that Go's built-in map has the same defect.

The incident demonstrates the scope of an expected-cost statement: it depends
on the hash function, the keys, and the table implementation. Application code
using a Go map does not select the runtime's probing algorithm, but it does
control key representation, canonicalization, distinct-key count, entry
lifetime, and whether exact-key lookup is the operation the service needs.

## Decision and limits

For the intended workload—a large batch that reuses one fixed metadata
snapshot—the growth analysis makes one index per snapshot the design to
benchmark and use by default.

{{< callout kind="contract" title="Decision for this teaching service" >}}
Accept one metadata snapshot, build one index with the documented
last-record-wins rule, and reuse it for every event in the batch. Keep the scan
as a correctness reference and as a comparison for small, one-use workloads.
Use the benchmark to find the measured crossover rather than claiming that the
index wins at every size.
{{< /callout >}}

Revisit the decision if:

- a new snapshot arrives after only a small number of lookups, so construction
  dominates the work;
- range or prefix queries replace exact-IP lookup;
- retained snapshots or endpoint cardinality exceed the memory budget;
- duplicate, missing-key, or address-canonicalization rules change; or
- profiles show allocation, garbage collection, or synchronization dominating.

The map changes the lookup cost. The service is still responsible for key
identity, conflict handling, snapshot ownership, memory bounds, and evidence
from its own workload.

Continue with the [unit guide](../), the
[scan-versus-index lab](../lab/), or the
[two Wheel scenarios](../wheel/).
