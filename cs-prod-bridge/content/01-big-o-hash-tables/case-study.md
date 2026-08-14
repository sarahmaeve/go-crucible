+++
title = 'Indexing a network-enrichment stream'
description = 'Compare repeated metadata scans with one reusable Go map index.'
weight = 2
+++

**Unit 01 · Production case study**

# Indexing a network-enrichment stream

{{< lead >}}One IP index can remove repeated metadata scans. Before using the
index, define how the service handles duplicates, missing records, updates,
memory, and concurrent access.{{< /lead >}}

## What comes from production

Dropbox's
[NetFlash engineering report](https://dropbox.tech/infrastructure/netflash-tracking-dropbox-network-traffic-in-real-time-with-elasticsearch)
describes Go collectors that process billions of network-log entries each day.
The collectors add IP metadata to those entries. This case study uses that
event-to-metadata relationship.

{{< callout kind="note" title="What is real, and what is synthetic" >}}
The NetFlash report provides the production setting. This repository provides
an invented batch API, `Metadata` type, duplicate policy, test data, and
benchmarks. These materials do not reproduce Dropbox's implementation or its
performance results.
{{< /callout >}}

## What the service must do

The service receives a batch of network events and one metadata snapshot. Each
event has a source IP address and a byte count. The service finds the matching
endpoint and adds its owner and region before it stores the event.

```text
event:    {source_ip: 10.20.3.7, bytes: 8192}
metadata: {ip: 10.20.3.7, owner: payments, region: us-west}
```

The cost model uses three input quantities:

- \(e\) is the number of events in the batch.
- \(m\) is the number of metadata records in the snapshot.
- \(u\) is the number of distinct IP keys in those records, so \(u \le m\).

The service needs an exact match. It must find the metadata record whose IP is
equal to the event's source IP.

The lab uses IPv4 addresses in one standard text form, so the key length has a
fixed bound. A production system can receive equivalent IPv6 addresses with
different spellings. It must convert them to one standard form or use a
comparable address value such as
[`netip.Addr`](https://pkg.go.dev/net/netip#Addr). Otherwise, two strings can
refer to the same address but act as different map keys.

{{< callout kind="production" title="Example: when the scan becomes expensive" >}}
Suppose that a batch contains 100,000 events and its metadata snapshot contains
50,000 records. A worst-case scan can do five billion key comparisons. The
numbers are invented, but they show the input size that makes the design a
problem.
{{< /callout >}}

## First design: scan the slice

The simplest design scans the metadata slice from newest to oldest. It stops
when it finds the event's IP. This code is direct and does not allocate an
index. It can be faster when the snapshot is small and the service searches it
only a few times.

A match in the first position needs one comparison. A missing key or a match in
the final position needs \(m\) comparisons. Repeating that worst case for
\(e\) events gives:

\[
\Theta(e \times m)
\]

If \(e\) and \(m\) grow together, this becomes \(\Theta(n^2)\). In production,
traffic and metadata can grow independently, so keep \(e\) and \(m\) separate.

## Second design: build one index

The scan becomes expensive because each event revisits metadata from earlier
searches. The alternative builds one `map[string]Metadata` from the snapshot.
The service then reuses this map for all events in the batch.

| Stage | Expected time | Additional storage |
|---|---:|---:|
| Build the index | \(\Theta(m)\) | \(\Theta(u)\) logical entries, where \(u \le m\) |
| Enrich all events | \(\Theta(e)\) | Result storage common to both designs |
| Complete batch | \(\Theta(m + e)\) | \(\Theta(u)\) beyond the scan |

These expected time bounds depend on two conditions. The hash function must
distribute keys well. Each key must also take bounded time to hash and compare.
The IP keys in this lab have a bounded length, so they meet the second
condition. The Go language does not guarantee these time bounds for maps.

The index constructor examines all \(m\) records, including duplicate IPs. The
completed map contains \(u\) entries. The lab also gives `m` to the map as a
capacity hint. The runtime can therefore reserve space based on the record
count, not the final number of distinct keys.

For small inputs, fixed costs still matter. Allocation, string hashing, CPU
cache behavior, and garbage collection can make a one-use scan faster. Big-O
predicts how each design grows. A benchmark shows which design is faster for a
specified workload and machine.

{{< callout kind="warning" title="Include index construction" >}}
A benchmark of a prebuilt index measures only lookup cost. It does not include
the cost to build the index. If the service builds an \(m\)-record map inside
the event loop, it builds that map \(e\) times. The work returns to expected
\(\Theta(e \times m)\).
{{< /callout >}}

## Check correctness before speed

The index is an improvement only if it returns the same results as the scan.
Both designs need clear rules for duplicate IPs, missing metadata, and snapshot
lifetime.

### Duplicate IP addresses

In Go, `index[record.IP] = record` replaces the current value for that key. The
lab uses a **last record wins** rule. The map processes records from first to
last, and the scan searches from last to first.

This rule is useful only when input order is stable and “last” has a documented
meaning, such as the newest version. A production service could instead reject
a snapshot with conflicting records. In either case, both designs must use the
same rule.

### Missing metadata

A lookup can find no record. The lab returns `Found == false`. This keeps a
missing record separate from a record with an empty owner or region.

The production collector must decide what to do with an event that has no
metadata. It can keep the event, drop it, try again after an update, or record
an error metric. The map cannot make this service decision.

### Snapshot replacement and ownership

Both designs assume that the metadata snapshot does not change during a batch.
A continuous service needs a rule for replacement. It can publish a new
read-only index, give one goroutine sole ownership of updates, or synchronize
all readers and writers.

Publishing read-only indexes gives each batch a consistent snapshot. The
service must still decide when a new batch uses the replacement and when it can
release the old index.

### Memory and distinct-key count

One index keeps \(u\) logical entries. The measured heap also includes the map's
internal data, reserved capacity, strings, and old indexes that readers can
still reach.

Memory can grow while lookups remain fast. In the current design, the snapshot
size limits one index. A process that keeps every old snapshot can hold several
full indexes in memory. A metadata source can also increase \(u\) over time if
it adds IPs but does not remove retired endpoints.

### Concurrent access

Multiple goroutines can read the same ordinary Go map when none of them writes
to it. If one goroutine changes the index while others read it, the service must
synchronize access. A completed read-only index avoids in-place writes. A
design with in-place updates must coordinate every reader and writer.

## Test the decision

The growth model gives a prediction, not a running time. In the example above,
the scan can do five billion comparisons. The indexed design examines 50,000
records once and then does 100,000 lookups with expected constant cost. These
counts give us a reason to test the index. They do not predict latency.

The [enrichment lab](../lab/) contains
correctness tests and benchmarks at several event and metadata sizes. Before
running it, predict:

- what changes when \(e\) doubles while \(m\) stays fixed;
- what changes when \(m\) doubles while \(e\) stays fixed;
- how construction appears in the end-to-end index benchmark; and
- why the prebuilt-index benchmark measures lookup cost but excludes
  construction.

Use three kinds of evidence:

1. **Calculate comparisons.** For the scan, calculate how the number of key
   comparisons changes with \(e\) and \(m\). This result does not depend on
   machine speed.
2. **Run a series of benchmarks.** Measure the scan, an index built for each
   batch, and a reused index at several workload sizes.
3. **Capture a profile.** Find which code uses the measured CPU time or memory.

One timing result compares two designs for one input on one machine. A series
shows whether measured growth matches the prediction. A profile helps explain
a result that does not match.

## Optional: a production hash-table failure

A local benchmark can show when this index is faster than the repeated scan. It
cannot prove that every hash-table implementation will stay fast for production
keys.

Dropbox's report on
[automated performance regression detection](https://dropbox.tech/infrastructure/keeping-sync-fast-with-automated-performance-regression-detectio)
describes a third-party map with a weak hash function. Related entries went into
the same buckets. Insertions and lookups changed from constant to linear time.
That failure occurred in one implementation with one set of keys. It does not
show that Go's built-in map has the same defect.

The incident shows why expected cost depends on the hash function, the keys,
and the table implementation. Code that uses a Go map does not select the
runtime's probing algorithm. It does control the key representation, the number
and lifetime of keys, and the operation that the service asks the map to do.

## Decision for this service

The intended workload is a large batch that reuses one fixed metadata snapshot.
For this workload, benchmark one index per snapshot and use it by default if
the measurements support the prediction.

{{< callout kind="contract" title="Decision for this teaching service" >}}
Accept one metadata snapshot. Build one index with the documented
last-record-wins rule, and reuse it for every event in the batch. Keep the scan
to check correctness and to compare small, one-use workloads. Use benchmarks to
find the input size at which the index becomes faster. Do not claim that the
index is faster at every size.
{{< /callout >}}

Revisit the decision if:

- a new snapshot arrives after only a few lookups, so most work goes into
  building indexes;
- range or prefix queries replace exact-IP lookup;
- old snapshots or the number of endpoint keys exceed the memory budget;
- the rules for duplicate keys, missing keys, or IP address formats change; or
- profiles show that allocation, garbage collection, or synchronization uses
  most of the resources.

The map changes the lookup cost. It does not decide what makes two keys equal,
how to handle conflicts, who owns a snapshot, or how much memory the service can
use. The team must test the decision with its own workload.

Continue with the [unit guide](../), the
[scan-versus-index lab](../lab/), or the
[two Wheel scenarios](../wheel/).
