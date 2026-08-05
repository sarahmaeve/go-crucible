+++
title = 'Making high-cardinality metric filters predictable'
description = 'How Datadog replaced unpredictable metric-wide scans with an index that covered every tag.'
weight = 2
+++

**Unit 02 · Production case study**

# Making high-cardinality metric filters predictable

{{< lead >}}Datadog's timeseries index was fast when a query matched an index
that the service had already generated, and slow when it did not. The team
replaced that unpredictable fallback with an index for every tag. The new
design stored and wrote more data, but it made interactive queries more
predictable and sharply reduced timeouts.{{< /lead >}}

This case follows Datadog's first-party report,
[Timeseries indexing at scale](https://www.datadoghq.com/blog/engineering/timeseries-indexing-at-scale/).
The article explains which queries triggered the slow path, how those queries
affected users and operators, what the team changed, and what the redesigned
system cost to run. All production measurements below come from that article.
The complexity analysis and runbook lab are simplified teaching examples; the
later Prometheus section examines a separate Go implementation.

## Why the original design worked for repeated queries

Datadog's service receives a metric name and a set of tag filters, then returns
the matching timeseries. A metric's **cardinality** is the number of distinct
timeseries recorded under that metric name. The original index maintained two
basic relationships:

```text
metric name -> series IDs
series ID   -> tags
```

Without a more specific index, a query could retrieve all series IDs for a
metric and check each series' tags. The query-time work therefore grew with
the metric's cardinality, even if only a small fraction matched.

Rather than index every possible tag, the service watched its query log and
generated indexes for filter patterns that callers had used before. This saved
storage: Datadog reports that callers consistently queried only about 30
percent of the series being written. It also worked well for monitors and
scheduled jobs, which tend to repeat the same queries.

## Interactive queries triggered the fallback

Interactive users did not repeat queries so predictably. When a user combined
tags in a new way, no generated index necessarily covered that combination.
The service then retrieved every series for the metric and checked the tags on
each one.

High-cardinality metrics made that fallback expensive. Datadog describes full
scans, timeouts, and a poor interactive experience. Even a small change to a
recurring query could miss its old generated index and place substantial load
on the database, and engineers sometimes had to create or remove indexes by
hand. The service saved disk space by indexing familiar queries, but the cost
was unpredictable CPU use and latency for unfamiliar ones.

## The redesign indexed every tag

The new design creates an inverted relationship for every tag:

```text
(metric, tag) -> set of series IDs
```

For a conjunction, the service retrieves one series-ID set per tag and
intersects them. For a disjunction, it unions them. A stored sequence of IDs is
often called a **postings list**. The report also describes sorted integer
arrays in merge-heavy parts of the service.

The lab illustrates intersection with two sorted postings lists of lengths `p`
and `q`. Two pointers can find their common IDs in `O(p+q)` advances because
every comparison consumes at least one input. For a query with several tags,
starting with the shortest postings lists will often keep the temporary result
small. Datadog's article describes set operations and sorted integer arrays,
but it does not specify that this exact loop handled its queries.

The exact intersection loop mattered less than the fact that every tag now had
an index. A query could no longer miss the generated-index set and fall back to
scanning every series for the metric.

{{< callout kind="contract" title="Two properties the index must preserve" >}}
First, every `(metric, tag)` entry must contain all of its matching series IDs;
otherwise a query can silently miss data. Second, if those IDs are stored in
sorted order for merging or intersection, every operation must agree on the
same ID order. Sorting does not change which IDs belong to the set, but it does
determine whether the ordered algorithm returns that set correctly.
{{< /callout >}}

## Predictability required more writes and storage

An unconditional inverted index writes a series ID once for every tag attached
to that series. Datadog reports that one ID may consequently be stored more
than ten times. That repetition increased both write traffic and disk use.

Some queries that previously matched one purpose-built generated index now
require several postings lookups and set operations. The report says these
queries became slightly more expensive on average. The improvement appeared
in predictability and in the slowest queries, rather than in every individual
request.

This was an attractive trade because the old system was limited by CPU while
disk capacity remained available. The new index used more of that disk to
avoid full scans and reduce manual index maintenance.

| Cost added while indexing | Benefit during queries |
|---|---|
| One series ID recorded for each tag | Every tag is queryable without a generated-index miss |
| More storage and write I/O | Fewer full metric scans |
| Several postings reads and set operations | Tag filters can be combined without a full scan |
| Index construction and maintenance | Less manual index creation/removal |

## Sharding had to be measured under production traffic

The team also split each node's RocksDB indexes into several independent
shards. A query could then search those shards in parallel and use more CPU
cores, although the service had to coordinate the work and merge the partial
results. The best number of shards depended on the workload and the machine.

Datadog tested this with production traffic. On a 32-core node, the team
selected eight shards and reported nearly an eightfold performance
improvement. The number of shards is an observed design point for that system,
not a portable rule of “one shard per four cores.”

The larger project also rewrote this service from Go to Rust. In this
particular workload, Datadog measured roughly 30 percent of the Go version's
CPU time in garbage collection. That explains why the rewrite contributed to
their result; it does not imply that every inverted index should avoid Go. The
local lab stays in Go so that its own allocations and CPU profile remain
visible.

## Reported production outcome

Datadog reports the combined redesign supported queries over metrics with 20
times higher cardinality on the same hardware, reduced query timeouts by 99
percent, and made the index nearly 50 percent cheaper to operate.

Those figures describe the combined result of the new indexing strategy,
storage representation, sharding, language rewrite, and production rollout.
They should not be attributed to the small two-pointer loop in the lab alone.

For an SRE, the sequence of questions is more reusable than any one of those
numbers:

1. identify an unpredictable fallback and what triggers it;
2. build and maintain the index coverage that removes the fallback;
3. name the write, memory, and average-query costs moved elsewhere;
4. experiment with parallelism and representation using representative traffic;
5. measure user-visible tail failures and operating cost, not only microbenchmarks.

## How Prometheus exposes ordered postings in Go

Datadog's article explains why the architecture changed, but it does not show
the Go interface behind its postings operations. Prometheus provides a related
example that we can inspect directly: production Go code whose index exposes
ordered postings, seeking, and intersection.

In the [Prometheus 3.13.1 index format](https://github.com/prometheus/prometheus/blob/v3.13.1/tsdb/docs/format/index.md),
postings contain increasing series references. In the pinned
[`tsdb/index/postings.go`](https://github.com/prometheus/prometheus/blob/v3.13.1/tsdb/index/postings.go):

- `Postings` iterates an ordered list;
- `Seek` advances to a requested reference or a greater one;
- `NewListPostings` requires ordered input;
- the list seek currently delegates to `slices.BinarySearch`; and
- `Intersect` combines ordered postings for conjunction.

`NewListPostings` requires its input to be ordered, and the iterator preserves
that order. Those two facts make its binary search valid; the query path does
not sort arbitrary input before every seek. The increasing references are an
internal index order, not a ranking shown to Prometheus users.

## Explore the same trade-off in the local lab

The [runbook-search lab](../lab/) applies the same ideas to a much smaller data
set. One implementation scans every runbook; the other maps each tag to a
sorted list of document IDs. After retrieving the matching IDs, both paths use
a separate comparison to arrange the runbooks for display.

Before running it, predict:

- whether index construction dominates when a snapshot serves only one query;
- how rare and common tags change the postings lengths;
- whether disjoint and heavily overlapping postings require the same number of
  pointer advances;
- when sorting the displayed matches costs more than retrieving them;
- how much write and storage growth one postings entry per document tag adds;
  and
- whether each snapshot serves enough queries to repay its build cost.

Before comparing speed, confirm that the scan and the index return the same
IDs for empty, missing, repeated, and differently cased tags. Operation counts
and benchmarks can then show when the up-front index build is repaid and how
the cost changes with postings length and overlap.

## Design review checklist

- Is the original fallback still reachable?
- What workload dimension triggers it: total documents, metric cardinality,
  postings density, overlap, or update rate?
- Where does the system verify that an input claimed to be ordered actually is?
- How many queries reuse the constructed index?
- What is the per-record write and storage amplification?
- Does the system build intermediate sets or iterate lazily?
- Which production measurements would show that users actually benefited?
- For every reported result, is its source clear?

Continue with the [unit guide](../), the [exploration lab](../lab/), or
[result ordering and cursor design](../result-ordering/).
