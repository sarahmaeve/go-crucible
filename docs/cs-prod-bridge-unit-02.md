# Unit 02 Research and Design: Sequences, Sorting, and Ordered Search

**Status:** initial implementation complete  
**Research reviewed:** 2026-08-05  
**Target toolchain:** Go 1.26.x

## Implementation outcome

The selected design is now implemented under
`cs-prod-bridge/02-sequences-sorting-search/` and published through
`cs-prod-bridge/content/02-sequences-sorting-search/`.

- The guide connects sequence representations, Go slice ownership, sorting,
  binary search, merge/intersection, lazy results, Prometheus 3.13.1, and the
  search-versus-ranking boundary.
- The production case traces Datadog's timeseries-index redesign and keeps its
  reported measurements separate from the local model.
- A separate API chapter covers bounded-response motivation, offset and keyset
  trade-offs, total display order, compound cursors, matching datastore access
  paths, response and token contracts, snapshot consistency, Google AIP-158,
  Kubernetes chunked lists, and Loki's same-timestamp handling.
- The runnable lab supplies scan and index paths, sorted-postings
  intersection, deterministic presentation and pagination, individual sorted
  insertion, batch sort-and-merge, correctness tests, operation counters, and
  benchmarks.
- W01 and W02 have green ordinary tests and opt-in deterministic failure tests
  under `csbridgewheel3` and `csbridgewheel4` respectively.

The initial implementation passes the nested module's complete tests and vet,
Hugo 0.164 generation, committed-output comparison, and internal-link checks.
Exercise timing, evidence disclosure, and benchmark fixture sizes remain
ordinary maintenance concerns as the material evolves.

## Decision

Build Unit 02 around one production idea:

> A system pays to establish and preserve an order. In return, it can search,
> intersect, merge, stop early, and present deterministic results without
> repeatedly scanning or sorting everything.

The case-study spine will be Datadog's timeseries-index redesign. It has the
complete production narrative this track requires: unpredictable interactive
queries fell back to full scans, timeouts and manual index maintenance became
operational problems, the team adopted an unconditional inverted index, and
the report states the costs and measured results. The redesign supported
20-times-higher-cardinality queries on the same hardware, reduced query
timeouts by 99%, and made the index nearly 50% cheaper to run.

Prometheus 3.13.1 will be the inspectable implementation companion. Its TSDB
format and Go source expose the exact mechanics that the local lab needs:
monotonically increasing postings, an ordered `Postings` iterator, a
`Seek` operation, intersection, and a list implementation that currently uses
`slices.BinarySearch`.

The exploration lab will implement a small Boolean search service for an SRE
runbook catalog. It will compare scanning all documents with intersecting
sorted postings, and then give matching documents a deterministic display
order. It is intentionally not a relevance engine.

The two Wheel scenarios will be:

1. **The Logarithmic Insert:** binary search finds an insertion point quickly,
   but repeated insertion into the middle of a Go slice shifts a linear suffix
   each time and makes a refresh path quadratic.
2. **The Timestamp-Only Cursor:** an incident-results API sorts by timestamp
   but uses no unique tie-breaker, so pagination skips or repeats records at a
   page boundary.

This choice preserves the provisional curriculum sequence while satisfying
the stronger selection rule in the track plan: Unit 02 now has a first-party
production case, inspectable Go evidence, a runnable investigation, and two
failures modeled on production workloads.

## Why this scope

The proposed unit originally combined arrays, linked structures, sorting,
binary search, time-series windows, merge operations, and ordered APIs. That
is too much unless a single model connects them. The connecting model is not
"learn several algorithms." It is the lifecycle of an ordered collection:

```text
unordered records
    -> establish one comparator and sort order
    -> retain or publish data in that order
    -> binary-seek, intersect, or merge in that same order
    -> apply a total presentation order when results cross an API boundary
```

The important engineering question is where the cost moves. A sorted array
makes reads cheap but does not make insertion cheap. An inverted index avoids
query-time scans but amplifies writes and storage. A lazy iterator can avoid
building a large intermediate result but cannot repair an invalid order.
A stable sort preserves an existing order among equal elements but does not
invent a globally unique cursor key.

### In scope

- Arrays, dynamic arrays, and linked sequences as operation trade-offs
- Go slice backing arrays, length, capacity, aliasing, append, and copying
- A stated sort order tied to one comparator
- Binary search and lower-bound/seek semantics
- Comparison sorting and its construction cost
- Stable sorting versus a total compound order
- Two-pointer merge, union, and intersection of sorted sequences
- Collected results versus lazy ordered iteration
- Index construction, reuse, updates, write amplification, and memory
- Deterministic ordering and cursor boundaries for user-visible results
- Benchmarks, profiles, operation counts, and ordering checks

### Deliberately out of scope

- Relevance scoring, learning-to-rank, embeddings, and approximate nearest
  neighbors
- PageRank or other graph-based authority measures
- Heaps and general k-way merge implementations
- B-trees, tries, skip lists, and general mutable ordered-set implementations
- Full-text tokenization, stemming, language analysis, and typo correction
- Distributed consensus or a general treatment of snapshot isolation

The unit may show those ideas at a production boundary, but it must not teach
them as if they were prerequisites.

## Search, recommendations, and showing results

These subjects overlap, but they are not the same problem.

| Stage | Question | Unit 02 treatment |
|---|---|---|
| Candidate retrieval | Which records satisfy the filters? | Core: dictionary lookup plus sorted postings intersection/union |
| Presentation order | In what deterministic order should matching records be shown? | Core: explicit compound comparator and cursor |
| Relevance ranking | Which matches are most useful to this user? | Boundary only; scoring is not implemented |
| Top-k selection | How can the system retain the best few candidates without sorting all candidates? | Defer to Unit 03 heaps and scheduling |
| Graph authority | Which nodes are important because of their links? | Defer to Unit 04 graphs |
| Learned recommendation | Which item is predicted to maximize a user outcome? | Outside the core algorithms track except as a production context |

The [Stanford/Cambridge Boolean-retrieval chapter](https://nlp.stanford.edu/IR-book/html/htmledition/processing-boolean-queries-1.html)
provides a clean boundary. Boolean retrieval produces a match set by combining
sorted postings. Ranked retrieval is a separate layer.

LinkedIn's first-party description of its
[feed architecture](https://engineering.linkedin.com/teams/data/artificial-intelligence/feed)
makes the recommendation boundary concrete: multiple first-pass rankers score
their inventories and send top-k results to a second-pass ranker, followed by
a reranker. That is a strong future Unit 03 context, not a reason to turn this
unit into an ML survey.

Showing simple search results still belongs here. A service can retrieve exact
matches and display them by `(updated_at DESC, document_id ASC)` without
claiming that the first result is more relevant. The secondary key makes the
order total and gives pagination an unambiguous continuation point.

## Production-case selection

The candidates were judged on five requirements:

1. A stated production workload and trigger
2. A symptom with operational or user impact
3. A design decision that depends on this unit's CS model
4. Measured evidence and explicit costs or limits
5. Enough technical detail to support a focused Go lab without pretending to
   reproduce the production system

| Candidate | Strength | Limitation | Role in Unit 02 |
|---|---|---|---|
| [Datadog timeseries indexing](https://www.datadoghq.com/blog/engineering/timeseries-indexing-at-scale/) | Full scans and query timeouts, operational toil, inverted index, union/intersection, sorted ID arrays, production experiments, strong outcome and cost data | The article does not specify that its two-set intersection uses the exact local two-pointer implementation | **Selected case-study spine** |
| [Prometheus TSDB](https://github.com/prometheus/prometheus/tree/v3.13.1/tsdb) | Familiar metrics workload, pinned Go source, ordered postings, `Seek`, and a documented on-disk format | The format and source explain the implementation, but not an end-to-end operational redesign | **Go implementation example** |
| [Twitter real-time search](https://blog.x.com/engineering/en_us/topics/infrastructure/2020/reducing-search-indexing-latency-to-one-second) | 15 seconds to 1 second, sorted IDs, unrolled linked lists, cache locality, out-of-order arrival, dark reads, dual pipelines, explicit limits | The final mutable structure is a skip list, which would expand the unit into ordered-set structures and concurrency | Required production extension |
| [GitHub Blackbird code search](https://github.blog/engineering/the-technology-behind-githubs-new-code-search/) | User-visible exact search, sorted postings, lazy iterators, early termination, compaction, commit-level consistency, production scale | Ranking-aware IDs, sharding, sparse grams, and deduplication can obscure the basic model | Optional user-facing extension |
| [Grafana Mimir split-and-merge compactor](https://grafana.com/blog/how-grafana-mimirs-split-and-merge-compactor-enables-scaling-metrics-to-1-billion-active-series/) | One-billion-series workload, block/search pressure, 1.5 TB naive local-disk requirement, TSDB format limits, explicit shard compatibility | Distributed compaction is a larger subject than two ordered sequences | Operational extension |
| [Spotify Sort Merge Bucket](https://engineering.atspotify.com/2021/02/how-spotify-optimized-the-largest-dataflow-job-ever-for-wrapped-2020/) | Prepaid order removes shuffle, roughly 1 PB joined, estimated 50% Dataflow-cost reduction, skew and compatibility limits | Batch-data context is less direct for the primary SRE audience | Optional batch extension |
| [LinkedIn feed](https://engineering.linkedin.com/teams/data/artificial-intelligence/feed) | Real multi-stage recommendation architecture with top-k and reranking | Requires scoring, heaps, and recommendation objectives | Defer to Unit 03 |
| [Google Search indexing incident](https://developers.google.com/search/blog/2019/08/when-indexing-goes-wrong-how-google) | Genuine index rollout incident, missing documents, cross-datacenter rollback, downstream inconsistency | It does not expose enough of the local data structure to drive the lab | Incident and rollout boundary |

Datadog is the best primary case because it begins with an on-call-quality
failure mode and ends with measured system behavior. Prometheus then prevents
the lesson from remaining architectural hand-waving: the learner can read the
ordered iterator contract and current Go implementation directly.

## Learning outcomes

After the unit, a learner should be able to:

- Choose an array, dynamic array, or linked representation from the operations
  and traversal pattern rather than from headline complexity alone.
- Explain why Go slices make indexed access and sequential traversal cheap,
  why subslices can retain or mutate shared storage, and why middle insertion
  shifts elements even when the insertion point came from binary search.
- State the exact comparator and ordering rule required by a binary search,
  merge, intersection, or cursor.
- Derive binary search as `O(log n)` comparisons while keeping construction
  and mutation costs in the same cost ledger.
- Derive two-pointer intersection of postings of lengths `p` and `q` as
  `O(p+q)` advances and explain why one global order is required.
- Distinguish stable sorting from a total order and explain why stable sorting
  alone cannot make separately generated pages deterministic.
- Decide whether to scan, sort once, maintain sorted state, merge a batch, or
  build an inverted index using query count, update rate, selectivity, memory,
  and write amplification.
- Explain when lazy ordered iterators reduce memory or allow early stopping,
  without claiming that laziness improves worst-case running time in every
  case.
- Diagnose a quadratic refresh path and a cursor-tie correctness defect using
  evidence modeled on production workloads.
- Separate the Go API contract, current implementation, production evidence,
  and the local teaching model.

## Cost model

The guide should keep workload variables named:

- `d`: documents or series eligible for a query
- `n`: entries already present in one sorted sequence
- `m`: new entries in an update batch
- `p`, `q`: lengths of two postings lists
- `r`: records collected in a match set
- `k`: records requested for one result page
- `t`: number of query terms or tag matchers
- `Q`: queries served while an index snapshot is reused

Representative modeled costs:

| Operation | Modeled time | Additional logical space | Condition that matters |
|---|---:|---:|---|
| Scan all documents for one fixed-size predicate | `Theta(d)` | `Theta(r)` if matches are collected | No reusable index |
| Merge-sort `n` items | `Theta(n log n)` | `Theta(n)` in the simple model | Comparator is valid |
| Binary search a sorted array | `O(log n)` comparisons | `O(1)` | Search comparator matches sort comparator |
| Insert at slice position `i` | `Theta(n-i)` moved elements | May allocate a new backing array | Contiguous representation |
| Insert `m` items individually near the front | `Theta(mn + m^2)` moved elements | Final sequence plus reallocations | Worst-case positions; becomes quadratic when `m=Theta(n)` |
| Sort an update batch and merge with current data | `Theta(m log m + n + m)` | `Theta(n+m)` for a simple out-of-place merge | Both inputs use the same order and conflict rule |
| Intersect sorted postings | `O(p+q)` advances | Up to `O(min(p,q))` if collected | One global document-ID order |
| Sort a collected result set | `O(r log r)` comparisons | Implementation-dependent | Retrieval order is not presentation order |
| Return `k` items from an already suitable lazy order | At least `Omega(k)` yielded items | Can be `O(1)` iterator state per input | No filter forces traversal of many rejected candidates |

The unit must repeatedly challenge the phrase "binary search makes this
`O(log n)`." It makes the lookup logarithmic. It does not make shifting,
rebuilding, sorting, or publishing the updated collection logarithmic.

### Stable order is not total order

A stable sort preserves the input order of elements for which the comparator
returns equality. This is useful when that input order is meaningful and one
sort observes it. It does not provide a deterministic order across separate
servers, independently constructed pages, map iteration, or changing
snapshots.

For a paginated incident list, `occurred_at DESC` is not total when many
incidents share a timestamp. `(occurred_at DESC, incident_id ASC)` can be a
total order if `incident_id` is unique and immutable. The cursor and search
comparison must contain both components.

### Locality belongs beside Big-O

An array and linked list can both be traversed in linear time, but their memory
behavior is not interchangeable. Twitter's report is a useful production
example: its prepend-only unrolled linked list reduced per-item pointer
overhead and made traversal cache-friendly, while the later skip list accepted
more pointer chasing to support insertion into an ordered collection. The
lesson should describe this as a measured design trade-off, not pretend that
Big-O predicts cache misses.

## Required source spine

### CS model

- [MIT 6.006 Recitation 2: Sequence Interface](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/c08a3b63dfe5f6f6b32257d35f86ae63_MIT6_006S20_r02.pdf)
  compares arrays, linked lists, and dynamic arrays, including amortized append
  and linear arbitrary insertion.
- [MIT 6.006 Lecture 3: Sorting](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/6d1ae5278d02bbecb5c4428928b24194_MIT6_006S20_lec3.pdf)
  covers binary search over a sorted array, linear merge, and merge sort.
- [Introduction to Information Retrieval: Processing Boolean Queries](https://nlp.stanford.edu/IR-book/html/htmledition/processing-boolean-queries-1.html)
  gives the two-pointer postings-intersection algorithm, `O(p+q)` bound, and
  the requirement for a single global postings order.

Optional depth:

- [MIT 6.006 Lecture 5: Linear Sorting](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/78a3c3444de1ff837f81e52991c24a86_MIT6_006S20_lec5.pdf)
  for comparison-sort limits, stability, and field-by-field tuple keys.
- [Faster postings-list intersection via skip pointers](https://nlp.stanford.edu/IR-book/html/htmledition/faster-postings-list-intersection-via-skip-pointers-1.html)
  as an optional preview, not a required implementation.

### Go contract

- [Go specification: slice types](https://go.dev/ref/spec#Slice_types)
- [Go specification: appending to and copying slices](https://go.dev/ref/spec#Appending_and_copying_slices)
- [`slices` package documentation](https://pkg.go.dev/slices), especially
  `BinarySearch`, `BinarySearchFunc`, `Insert`, `SortFunc`, and
  `SortStableFunc`
- [Go Slices: usage and internals](https://go.dev/blog/slices-intro) for the
  backing-array model and aliasing explanation

Claims to make precisely:

- `slices.BinarySearch` requires increasing order and returns the earliest
  match or the insertion position.
- `BinarySearchFunc` must implement the same order used by the slice.
- `slices.Insert` shifts the suffix and documents `O(len(s)+len(v))` time.
- `SortFunc` is not guaranteed stable and requires a strict weak ordering.
- `SortStableFunc` preserves the input order of equal elements.
- `append` may reuse the backing array or allocate a new one. No particular
  capacity growth factor is a language guarantee.

### Current Go implementation

- [Go `slices/sort.go` source](https://go.dev/src/slices/sort.go)
- [Generated comparison-sort implementation](https://go.dev/src/slices/zsortanyfunc.go)

The implementation walkthrough should be pinned to the repository's Go 1.26
line. In that line, `SortFunc` dispatches to pattern-defeating quicksort code
with a heapsort fallback, while the stable path uses insertion-sorted blocks
and symmetric merge. These facts are implementation details, not API
promises. Recheck them when the module's Go version changes.

### Production evidence

Required:

- [Datadog: Timeseries indexing at scale](https://www.datadoghq.com/blog/engineering/timeseries-indexing-at-scale/)
- [Prometheus 3.13.1 index disk format](https://github.com/prometheus/prometheus/blob/v3.13.1/tsdb/docs/format/index.md)
- [Prometheus 3.13.1 postings implementation](https://github.com/prometheus/prometheus/blob/v3.13.1/tsdb/index/postings.go)
- [Twitter: Reducing search indexing latency to one second](https://blog.x.com/engineering/en_us/topics/infrastructure/2020/reducing-search-indexing-latency-to-one-second)

Optional production extensions:

- [GitHub: The technology behind GitHub's new code search](https://github.blog/engineering/the-technology-behind-githubs-new-code-search/)
- [Grafana Mimir's split-and-merge compactor](https://grafana.com/blog/how-grafana-mimirs-split-and-merge-compactor-enables-scaling-metrics-to-1-billion-active-series/)
- [Grafana Mimir runbooks](https://grafana.com/docs/mimir/latest/manage/mimir-runbooks/),
  especially compaction backlog, OOM, and TSDB index-limit diagnoses
- [Spotify Sort Merge Bucket for Wrapped 2020](https://engineering.atspotify.com/2021/02/how-spotify-optimized-the-largest-dataflow-job-ever-for-wrapped-2020/)

### Ordered API and incident boundary

- [Google AIP-158: Pagination](https://google.aip.dev/158) for opaque tokens,
  permitting page-size changes while otherwise retaining query arguments,
  finite defaults, and end-of-collection behavior
- [Kubernetes API concepts: chunked lists](https://kubernetes.io/docs/reference/using-api/api-concepts/#retrieving-large-results-sets-in-chunks)
  for a real continuation token tied to one `resourceVersion` and therefore a
  consistent snapshot
- [Loki HTTP API](https://grafana.com/docs/loki/latest/reference/loki-http-api/)
  for timestamp-ordered log results and direction
- [Loki `logcli` query source](https://github.com/grafana/loki/blob/v3.7.4/pkg/logcli/query/query.go)
  as a concrete example of overlapping a same-timestamp boundary and
  deduplicating because several entries can share that timestamp
- [Google's 2019 Search indexing incident](https://developers.google.com/search/blog/2019/08/when-indexing-goes-wrong-how-google)
  for rollout, rollback, missing-index entries, and downstream consistency

## Case-study design

### Working title

**Making high-cardinality metric filters predictable**

### Production narrative

Datadog's original service stored metric-to-series IDs and series-ID-to-tags.
A filter without a matching generated index required retrieving every series
for the metric and checking its tag set. The number of tag-set lookups grew
linearly with metric cardinality.

The service generated indexes from its live query log. That was space
efficient because Datadog reports that only about 30% of written series were
consistently queried. It worked well for stable programmatic workloads such as
alerts and periodic jobs. Interactive user queries were less predictable and
often missed the generated indexes, producing full scans, timeouts, and poor
user experiences. Changes in programmatic query patterns could also overload
the database, and engineers sometimes had to create or remove indexes by
hand.

The redesign unconditionally indexes every tag. Each `(metric, tag)` key maps
to a set of series IDs. A conjunction retrieves one postings set per tag and
intersects the sets; a disjunction unions them. This makes the worst query
path more predictable because every tag has an index and there is no full-scan
fallback.

The design moves cost rather than removing it:

- A series ID is stored once per tag, often more than ten times, producing
  write and space amplification.
- Some formerly exact generated-index queries now require several lookups and
  are slightly more expensive on average.
- The old service was CPU-bound and underused disk, so the team deliberately
  spent disk capacity to improve the tail and reduce CPU-heavy maintenance.
- Intranode sharding added split/merge overhead but let one query use more
  cores. Production experiments selected eight shards on a 32-core node and
  produced a nearly eight-times performance improvement.
- The report also describes a Go-to-Rust rewrite and language benchmarks. Those
  results belong to Datadog's workload and do not establish a general rule
  about Go. The local unit remains in Go and measures its own workload.

The reported combined result was support for 20-times-higher-cardinality
metrics on the same hardware, a 99% reduction in query timeouts, and nearly
50% lower operating cost.

### Source attribution

The case must distinguish the following:

- **Measured by Datadog:** the workload ratios, production experiment, GC/CPU
  observations, timeout reduction, cardinality increase, and cost change.
- **Described by Datadog:** unconditional per-tag indexes, set union and
  intersection, sorted integer arrays for merge-heavy paths, and sharding.
- **Modeled in the unit:** the exact comparison and advance counts for the
  local two-pointer implementation.
- **Implemented by Prometheus 3.13.1:** ordered `Postings`, `Seek`,
  `Intersect`, `NewListPostings`' ordered-input precondition, and the current
  list `Seek` call to `slices.BinarySearch`.
- **Synthetic locally:** runbook documents, tag distribution, APIs,
  benchmarks, profiles, and all measured numbers from the lab.

The published material should say explicitly which source supports each claim.
The local lab does not reproduce Datadog's system, and the article does not say
that Datadog uses the lab's two-pointer loop. Prometheus's internal
series-reference order also should not be presented as user-facing result
ordering.

### Prometheus deep-read path

The implementation sidebar should trace only a small pinned path:

1. The TSDB block format stores postings as monotonically increasing series
   references.
2. `Postings` exposes iterative access to an ordered list and `Seek` advances
   to a requested reference or greater.
3. `NewListPostings` requires ordered input.
4. The 3.13.1 list implementation currently uses `slices.BinarySearch` for a
   forward seek.
5. `Intersect` combines postings for conjunctive matchers.

This is enough to connect the CS model, Go library, and a production
observability system without teaching the whole Prometheus query engine.

## Exploration-lab design

### Working title

**Searching an SRE runbook catalog**

### Domain and data

Use records with explicit tags so tokenization does not become an accidental
lesson:

```go
type Document struct {
    ID        uint64
    Title     string
    Tags      []string
    UpdatedAt int64
}
```

The immutable index snapshot contains documents by ID and a
`map[string][]uint64` from normalized tag to sorted, duplicate-free document
IDs. The map intentionally reuses Unit 01 exact-key lookup; the sorted slice is
the new structure.

### Variants

1. `ScanAND` checks every document for every requested tag. It is the simple
   correctness reference.
2. `BuildIndex` appends IDs by tag, then sorts and compacts each postings list.
3. `Intersect2` uses two pointers over sorted postings and records comparisons
   and advances.
4. `SearchAND` orders postings by increasing length before repeated
   intersection, following the standard Boolean-query heuristic.
5. `OrderResults` applies `(UpdatedAt DESC, ID ASC)` only after matching.
6. `PageAfter` uses both comparator fields for an immutable snapshot.
7. An update experiment compares individual binary-search-plus-insert with a
   sorted batch plus linear merge. It can share helpers with Wheel 01 without
   revealing that Wheel's source.

### Correctness contract

- Tag normalization is deterministic and documented.
- A document appears at most once in each postings list.
- Every postings list passes `slices.IsSorted`.
- `ScanAND` and `SearchAND` return the same match set.
- Missing tags, repeated query tags, empty queries, and duplicate document
  tags have explicit behavior.
- Presentation sorting does not change membership.
- Pagination over one immutable snapshot returns every result exactly once,
  including timestamp ties.

### Measurements

Benchmarks should vary dimensions independently:

- `d`: 1,000, 10,000, and 100,000 documents
- postings density: rare/rare, rare/common, and common/common
- overlap: disjoint, partial, and nearly identical
- `Q`: one query versus many queries reusing an index
- update distribution: append-heavy, uniform, and front-heavy

Deterministic counters are primary evidence:

- document/tag predicate checks for the scan
- comparisons and pointer advances for intersection
- elements shifted for individual slice insertion
- elements read and written for batch sort/merge
- matches collected before presentation

Wall-clock benchmarks and CPU/allocation profiles support those counts. They
must not be the sole pass/fail check.

### Predictions required before execution

- Which dimensions change scan work?
- Which dimensions change intersection work?
- When does index construction dominate because `Q` is small?
- Why can a rare/common query benefit from seeking or binary search even when
  the general two-pointer bound remains useful?
- When does collecting then sorting `r` results dominate retrieval?
- How does update position change shift count even when lookup count does not?

## Wheel of Misfortune designs

### W01: The Logarithmic Insert

**Incoming report:** A controller's configuration snapshot refresh is fast in
tests but burns CPU and delays readiness in large fleets. Queries against the
published snapshot remain fast. The regression appeared after the team changed
the snapshot builder to preserve sorted order during ingestion.

**Hidden first local cause:** For every incoming record, the builder calls
`slices.BinarySearchFunc` and then `slices.Insert`. The search takes logarithmic
comparisons, but the insertion shifts the suffix. Front-heavy or random input
performs `Theta(mn + m^2)` moves across a batch.

**Why fixtures miss it:** Test data is small and already arrives mostly in
output order, so inserts occur near the end and move few elements.

**Evidence packets:**

1. Refresh CPU and readiness symptoms while lookup latency remains normal
2. Operation counts by existing size, batch size, and insertion-position
   distribution
3. A CPU profile dominated by copying/shifting rather than comparison
4. The builder source and sortedness/conflict-policy tests

**Repair:** Normalize and validate the incoming batch, sort and compact
it once, and linearly merge it with the current snapshot using the documented
duplicate/update rule. Publish one immutable result.

**Deterministic verification:** Assert identical records and order, count
moves/reads rather than elapsed time, and show the front-heavy series changing
from quadratic growth to the stated batch-sort-plus-merge bound.

### W02: The Timestamp-Only Cursor

**Incoming report:** An incident-search UI shows duplicate events on one page
and omits others when an alert fan-out creates many events with the same
timestamp. Retrying can produce a different boundary even though every event
exists in storage.

**Hidden first local cause:** Results are ordered only by `OccurredAt`, and the
cursor stores only that timestamp. The page boundary cannot identify which
equal-timestamp record was last returned. A stable sort is not sufficient
because separate requests do not share one meaningful input order.

**Evidence packets:**

1. Customer report and page-by-page IDs
2. Timestamp-frequency evidence showing a large tie at the boundary
3. Request/response traces with the same timestamp cursor
4. Comparator, cursor, and snapshot code

**Repair:** Define a total order such as
`(OccurredAt DESC, IncidentID ASC)`, carry both fields in an opaque cursor, and
apply the same comparison in sorting and continuation. Bind the exercise to an
immutable snapshot; explain separately how Kubernetes binds continuation to a
`resourceVersion`.

**Deterministic verification:** Generate tied timestamps and assert that
concatenating all pages returns every expected ID exactly once in the declared
order. Include a failing property-style or fuzz seed in the evidence.

Loki provides a useful production comparison without sharing the exercise's
invented defect. Its API explicitly sorts log entries by timestamp, and the
`logcli` source handles a batch that ends inside a same-timestamp group by
overlapping the next request and suppressing entries it has already printed.

## Guide shape

The published guide should use this order:

1. Incoming reports: slow refresh, unpredictable metric query, and missing
   boundary results
2. Predict the costs using named workload variables
3. Sequence operations and representation trade-offs
4. Go slice contract, backing arrays, and aliasing
5. Establishing an order: comparator, sort, stability, and total keys
6. Binary search as a preconditioned operation
7. Linear merge and Boolean postings intersection
8. Collected results, laziness, and early termination
9. Datadog production case
10. Prometheus 3.13.1 implementation example
11. Showing exact-match results without pretending to rank them
12. Lab and Wheels
13. Interview translation and reflection

The opening signals should be recognizable to an SRE:

- A label-filter query scans every series for a high-cardinality metric.
- A compactor or index refresh falls behind and makes the read path do more
  work.
- A binary-searched slice is cheap to query but expensive to update.
- A log or incident UI loses results that share a timestamp at a page edge.

## Interview translation

The unit should end with an answer shaped like this:

> I would separate candidate retrieval from presentation. If the service has
> `d` runbooks and repeatedly filters by tags, scanning costs `Theta(d)` per
> fixed-size predicate. An inverted index pays construction and storage once.
> For two sorted postings of lengths `p` and `q`, a two-pointer intersection
> takes `O(p+q)` advances under one document-ID order. Binary search can seek in
> `O(log n)` comparisons, but inserting into a Go slice still shifts a linear
> suffix, so I would batch, sort, and merge updates when refreshes are large.
> After retrieval I would sort matches by a total key such as update time plus
> stable ID and put both fields in the cursor. I would verify match-set
> equivalence, sortedness, shift/advance counts, and then benchmark the real
> selectivity and update distribution.

## Implementation layout

```text
cs-prod-bridge/
  content/02-sequences-sorting-search/
    _index.md
    case-study.md
    result-ordering.md
    lab/_index.md
    wheel/
      _index.md
      worksheet.md
      01-logarithmic-insert/
        _index.md
        candidate.md
        debrief.md
        evidence/
      02-timestamp-only-cursor/
        _index.md
        candidate.md
        debrief.md
        evidence/
  02-sequences-sorting-search/
    README.md
    CASE-STUDY.md
    lab/
      README.md
      search.go
      search_test.go
      search_bench_test.go
    wheel/
      README.md
      01-logarithmic-insert/
        REPORT.md
        CANDIDATE.md
        DEBRIEF.md
        index.go
        index_test.go
        scale_test.go
        evidence/
      02-timestamp-only-cursor/
        REPORT.md
        CANDIDATE.md
        DEBRIEF.md
        results.go
        results_test.go
        scale_test.go
        evidence/
```

Reuse Unit 01's shortcodes, hidden debrief/evidence navigation, report-first
flow, and opt-in symptom tags. Do not add shared abstractions until repetition
is visible in the second completed unit.

## Implementation checklist

The implementation includes:

- Every sorted operation names its comparator and validates its precondition.
- Contract, model, implementation, and measurement claims are visibly
  separated.
- The Datadog case includes workload, failure, decision, trade-offs, measured
  result, and provenance limits.
- Prometheus implementation claims use pinned 3.13.1 links.
- The lab's scan and indexed variants have match-set-equivalence tests.
- Benchmarks vary document count, postings density, overlap, reuse, and update
  order independently.
- At least one deterministic check bounds intersection advances.
- At least one deterministic check counts slice elements shifted.
- Pagination tests cover more equal-timestamp records than fit on one page.
- The API chapter connects cursor correctness to a matching datastore query and
  index, and keeps the decoded cursor model separate from the public token
  contract.
- The Wheels fail only under their documented build tags.
- Normal `go test ./...`, formatting, site generation, and committed-output
  checks pass.
- The guide never describes Boolean matching as relevance ranking and never
  presents stable sorting as a substitute for a total cursor order.

## Research conclusions to carry forward

- **Prometheus belongs in Unit 02**, but as pinned implementation evidence,
  not as the sole case narrative.
- **Simple search belongs in Unit 02** when it means exact candidate retrieval
  with an inverted index and ordered postings.
- **Showing results belongs in Unit 02** when it means deterministic sorting,
  ties, cursors, and snapshot boundaries.
- **Recommendation engines do not belong at the center of Unit 02.** Their
  retrieval, scoring, top-k, second-pass ranking, and reranking stages are too
  large to treat as "sorting." Save the top-k and scheduling bridge for Unit
  03; use graph ranking in Unit 04 only if a production case justifies it.
- **The strongest SRE story is cost movement, not a magic data structure.**
  Datadog traded writes and disk for predictable queries and lower operational
  toil. Twitter traded cache-friendly prepend-only lists for a mutable ordered
  structure and used dark reads and response comparison to manage rollout
  risk. Mimir spends compaction resources so queries touch fewer blocks.
- **Correct ordering is a system contract.** It spans producer, stored index,
  iterator, merge/intersection, result comparator, cursor, and snapshot. A
  mismatch at any boundary can be slow, incorrect, or both.
