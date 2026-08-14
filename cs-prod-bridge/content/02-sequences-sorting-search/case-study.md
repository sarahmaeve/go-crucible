+++
title = 'Making high-cardinality metric filters predictable'
description = 'How Datadog replaced unpredictable full-metric scans with an index for every tag.'
weight = 2
+++

**Unit 02 · Production case study**

# Making high-cardinality metric filters predictable

{{< lead >}}Datadog's timeseries index was fast when an existing index covered
a query. Otherwise, the service used a slow fallback. The team replaced that
fallback with an index for every tag. The new design wrote and stored more data,
but it made interactive queries more predictable and reduced timeouts.{{< /lead >}}

This case study follows Datadog's report,
[Timeseries indexing at scale](https://www.datadoghq.com/blog/engineering/timeseries-indexing-at-scale/).
The article explains which queries used the slow path, how users and operators
were affected, what the team changed, and what the new system cost. All
production measurements below come from that article. The complexity analysis
and runbook lab are smaller teaching examples. The later Prometheus section
examines a separate Go implementation.

## Why repeated queries were fast

Datadog's service receives a metric name and tag filters. It returns the
matching timeseries. A metric's **cardinality** is its number of distinct
timeseries. The original index stored two relationships:

```text
metric name -> series IDs
series ID   -> tags
```

Without a more specific index, a query retrieved every series ID for the metric
and checked the tags for each series. The work grew with metric cardinality,
even when few series matched.

The service did not index every possible tag. It watched its query log and
built indexes for filter patterns that callers had used before. This saved
storage. Datadog reports that callers regularly queried only about 30 percent
of the series being written. The design worked well for monitors and scheduled
jobs because they tend to repeat queries.

## New queries fell back to a full scan

Interactive users changed their queries more often. A new combination of tags
might not have an index. The service then retrieved every series for the metric
and checked the tags for each one.

Metrics with many series made the fallback expensive. Datadog reports full
scans, timeouts, and poor interactive use. A small query change could miss the
old index and add substantial database load. Engineers sometimes had to create
or remove indexes by hand. The service saved disk space for familiar queries,
but CPU use and latency became unpredictable for unfamiliar queries.

## The new design indexed every tag

The new design creates an inverted relationship for every tag:

```text
(metric, tag) -> set of series IDs
```

For an AND query, the service retrieves one series-ID set for each tag and
finds the IDs shared by all sets. For an OR query, it combines the sets. A
stored sequence of IDs is often called a **postings list**. The report also
describes sorted integer arrays in parts of the service that do many merges.

The lab finds shared IDs in two sorted postings lists of lengths `p` and `q`.
Two positions can find the shared IDs in `O(p+q)` advances because every
comparison advances at least one position. For several tags, starting with the
shortest lists often keeps the temporary result small. Datadog's article
describes set operations and sorted integer arrays. It does not say that this
exact loop handled the production queries.

The important change was that every tag had an index. A query could no longer
miss the available indexes and fall back to scanning every series for the
metric.

{{< callout kind="contract" title="Two properties the index must preserve" >}}
Each `(metric, tag)` entry must contain all matching series IDs. Otherwise, a
query can silently miss data. If the service stores those IDs in sorted order,
every merge and intersection must use the same order. Sorting does not change
which IDs belong to the set. It does determine whether the ordered algorithm
returns the correct set.
{{< /callout >}}

## Predictable queries cost more writes and storage

The new inverted index writes a series ID once for every tag on that series.
Datadog reports that one ID can therefore be stored more than ten times. This
increases write traffic and disk use.

Some queries previously used one index built for that exact query. They now
require several postings lookups and set operations. Datadog reports that these
queries became slightly more expensive on average. The main improvement was
more predictable behavior and fewer very slow queries, not a faster result for
every request.

This trade fit the system because CPU was the limit and disk capacity was still
available. The new index used more disk to avoid full scans and reduce manual
index work.

| Cost added while indexing | Benefit during queries |
|---|---|
| One series ID recorded for each tag | Every tag query has an index |
| More storage and write I/O | Fewer full metric scans |
| Several postings reads and set operations | Tag filters can be combined without a full scan |
| Index construction and maintenance | Less manual index creation/removal |

## The team tested sharding with production traffic

The team also split each node's RocksDB indexes into independent shards. A query
could search the shards in parallel and use more CPU cores. The service then
had to coordinate that work and merge partial results. The best shard count
depended on the workload and machine.

Datadog tested the choices with production traffic. On a 32-core node, the team
selected eight shards and reported that performance improved by almost a
factor of eight. This shard count was the measured choice for that system. It
is not a general rule of one shard for every four cores.

The larger project also rewrote the service from Go to Rust. For this workload,
Datadog measured approximately 30 percent of the Go version's CPU time in
garbage collection. This helps explain why the rewrite contributed to the
result. It does not mean that every inverted index should avoid Go. The local
lab stays in Go so you can measure its allocations and CPU use.

## What Datadog measured

Datadog reports that the combined redesign supported metrics with 20 times more
series on the same hardware. It reduced query timeouts by 99 percent and made
the index almost 50 percent cheaper to operate.

Those figures describe the combined result of the new index, storage format,
sharding, language rewrite, and production rollout. The small two-position loop
in the lab did not produce those results by itself.

The following questions apply to other systems:

1. What fallback makes the work unpredictable, and what causes it?
2. What index would remove that fallback, and how will the service maintain it?
3. Which write, memory, and average-query costs does the change add?
4. How should the team test parallel work and storage with representative traffic?
5. Do user-visible timeouts and operating cost improve, not only a small
   benchmark?

## Optional: how Prometheus exposes ordered postings in Go

Datadog's article explains why the architecture changed. It does not show the
Go interface behind its postings operations. Prometheus provides related
production Go code that we can inspect. Its index supports ordered postings,
seeking, and intersection.

In the [Prometheus 3.13.1 index format](https://github.com/prometheus/prometheus/blob/v3.13.1/tsdb/docs/format/index.md),
postings contain increasing series references. In the pinned
[`tsdb/index/postings.go`](https://github.com/prometheus/prometheus/blob/v3.13.1/tsdb/index/postings.go):

- `Postings` iterates an ordered list;
- `Seek` advances to a requested reference or a greater one;
- `NewListPostings` requires ordered input;
- the list seek currently delegates to `slices.BinarySearch`; and
- `Intersect` combines ordered postings for conjunction.

`NewListPostings` requires ordered input, and the iterator preserves that
order. These facts make its binary search valid. The query does not sort
arbitrary input before every seek. The increasing references are an internal
index order, not a ranking shown to users.

## Test the same trade-off in the lab

The [runbook-search lab](../lab/) applies the same ideas to a smaller data set.
One implementation scans every runbook. The other maps each tag to a sorted
list of document IDs. After finding the matching IDs, both versions use a
separate rule to order the runbooks for display.

Before running it, predict:

- whether index construction uses most of the work when a snapshot serves only
  one query;
- how rare and common tags change the postings lengths;
- whether disjoint and heavily overlapping postings require the same number of
  position advances;
- when sorting the displayed matches costs more than retrieving them;
- how many writes and how much storage one postings entry per document tag adds;
  and
- whether each snapshot serves enough queries to repay its build cost.

Before you compare speed, check that the scan and index return the same IDs for
empty, missing, repeated, and differently cased tags. Then use operation counts
and benchmarks to find when reuse repays the initial index build. Also measure
how postings length and overlap change the cost.

## Questions for a design review

- Is the original fallback still reachable?
- Which input causes it: total documents, metric cardinality, fraction of
  documents in each postings list, overlap, or update rate?
- Where does the system verify that an input claimed to be ordered actually is?
- How many queries reuse the constructed index?
- How many writes and stored entries does each source record add?
- Does the system build temporary sets or produce matches as needed?
- Which production measurements would show that users actually benefited?
- For every reported result, is its source clear?

Continue with the [unit guide](../), the [lab](../lab/), or
[result ordering and cursor design](../result-ordering/).
