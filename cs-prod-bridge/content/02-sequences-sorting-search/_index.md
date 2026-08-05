+++
title = 'Sequences, sorting, and ordered search'
description = 'Sorting records can make later searches and merges cheaper, provided updates and readers use the same ordering rules.'
weight = 2
+++

**Unit 02 · Foundations**

# Sequences, sorting, and ordered search

{{< lead >}}Sorting is rarely the end of the job. Systems keep data ordered so
later operations can avoid starting from scratch: binary search can discard
half of the remaining records, two sorted lists can be merged in one pass, and
a page token can identify where the next page begins. This unit examines the
cost of creating that order and the failures that occur when different parts
of a system disagree about what “ordered” means.{{< /lead >}}

{{< callout kind="production" title="Incoming reports" >}}
- A label-filter query is fast for ordinary metrics but times out when it has
  to examine every series for a high-cardinality metric.
- Reading a configuration snapshot is fast, but publishing a new snapshot
  drives one CPU core to 100 percent because the builder inserts incoming
  records into the middle of a sorted slice one at a time.
- An incident endpoint returns HTTP 200 and normal latency for every page, but
  omits records when many incidents share the boundary timestamp.
{{< /callout >}}

These incidents involve ordering at different stages. The metric query needs
an index that narrows the records examined. The snapshot builder has made reads
cheap by doing too much work during updates. The incident endpoint sorts its
records but cannot identify an exact page boundary when timestamps tie.

We will treat those problems separately. Postings lists may use increasing
document IDs because that order supports efficient intersection, while the
same service may arrange final results by descending update time because that
is what a user expects to see.

By the end of this unit, you should be able to:

- choose a sequence representation from its operations and traversal pattern;
- state the comparator and sortedness precondition behind a binary search,
  merge, intersection, or cursor;
- separate logarithmic search from linear movement in a contiguous slice;
- derive the work for scanning, sorting, batch merging, and postings
  intersection;
- distinguish stable sorting from a total order;
- keep exact matching separate from presentation order and relevance;
- account for index construction, reuse, writes, storage, and updates; and
- use correctness tests, operation counts, benchmarks, and profiles to explain
  how an implementation behaves as its inputs grow.

## Start with workload variables

To estimate the cost, distinguish the size of the stored collection from the
size of one update or query. This unit uses:

- `d`: documents or series eligible for a query
- `n`: entries already in a sorted sequence
- `m`: entries in one update batch
- `p`, `q`: lengths of two postings lists
- `r`: records that match retrieval, before presentation sorting
- `k`: records requested on one page
- `Q`: queries served from the same index snapshot

These quantities lead to different design decisions. A service with a million
stored records and ten updates does not have the same refresh problem as a
million-record initial load. An indexed query also need not inspect all `d`
documents: its work is often determined by the postings lengths `p` and `q`
and by how much those lists overlap.

| Operation | Modeled time | Condition behind the claim |
|---|---:|---|
| Scan all documents for one constant-cost check | `Θ(d)` | No reusable index |
| Merge-sort `n` items | `Θ(n log n)` | Comparisons are treated as constant cost |
| Binary-search a sorted slice | `O(log n)` comparisons | Search and sort use the same order |
| Insert at slice position `i` | `Θ(n-i)` moves | Elements are contiguous |
| Insert `m` items at or near the front one by one | `Θ(mn+m²)` moves | Each insertion shifts almost the entire current slice |
| Sort a batch, then merge | `O(m log m+n+m)` | Both inputs use one order and conflict rule |
| Intersect two postings lists | `O(p+q)` advances | Both lists use the same document-ID order |
| Merge-sort `r` results for display | `Θ(r log r)` comparisons | Retrieval order is not presentation order |

These bounds describe how work grows with the inputs. They do not determine
elapsed time, cache behavior, or allocation counts on a particular machine;
those require measurement.

## Choose a representation from the operations you need

Arrays, slices, and linked lists can all store a sequence of values, but they
make different operations cheap. Choose among them by asking which operations
dominate this workload, rather than which structure is universally fastest.

| Representation | Indexed access | Insert/delete near middle | Traversal behavior |
|---|---:|---:|---|
| Fixed array | constant | linear movement | contiguous and cache-friendly |
| Dynamic array / Go slice | constant | linear movement | contiguous; usually constant work per append over a long series of appends, with occasional copying |
| Singly linked list | linear | constant after locating predecessor | pointer chasing; no random access |

The complexity of one operation is not enough to choose a representation. A
linked list can insert in constant time only after the caller has found the
predecessor; finding that position still takes a linear walk. A dynamic array
must move elements for a middle insertion, but it often traverses quickly
because adjacent elements occupy adjacent memory and carry no per-element
link.

[MIT 6.006 Recitation 2](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/c08a3b63dfe5f6f6b32257d35f86ae63_MIT6_006S20_r02.pdf)
works through these trade-offs formally for arrays, linked lists, and dynamic
arrays.

{{< callout kind="production" title="A production example from Twitter" >}}
[Twitter's real-time search indexing report](https://blog.x.com/engineering/en_us/topics/infrastructure/2020/reducing-search-indexing-latency-to-one-second)
describes an index that initially used a cache-friendly, prepend-only unrolled
linked list. When the service later needed to insert records that arrived out
of order, the team moved to a skip list. That change accepted more pointer
traversal in exchange for an operation the previous representation handled
poorly. We will not implement a skip list here, but the example shows why the
required update pattern matters as much as the lookup complexity.
{{< /callout >}}

## Go slices share backing arrays

The [Go specification](https://go.dev/ref/spec#Slice_types) defines a slice as
a descriptor over a contiguous segment of an underlying array. Its length is
the number of accessible elements; its capacity is the size of the segment
from the first accessible element to the end of that array.

```go
base := []string{"a", "b", "c", "d"}
view := base[1:3] // "b", "c"; shares base's backing array
view[0] = "B"    // base[1] is now "B"
```

Creating `view` is cheap because Go does not copy the strings. The consequence
is that `base` and `view` refer to the same memory: changing one can change the
other. A function that promises to publish an immutable snapshot therefore
cannot assume that returning a slice makes the data immutable. It must either
keep every alias under its control or copy the data before publishing it. A
small subslice can also keep a much larger backing array alive.

`append` exposes the same issue. When there is enough capacity, it may reuse
the existing backing array, so another slice that shares that array can observe
the writes. Otherwise it allocates a new array. The specification deliberately
does not promise a particular capacity growth factor. The built-in `copy` can
copy between overlapping slices, which is useful when elements must move
within one array.

The [`slices` package](https://pkg.go.dev/slices) provides convenient generic
operations, but convenience does not change the representation cost.
`slices.Insert` must open a gap and shift a suffix; its documented running time
is `O(len(s)+len(v))`.

{{< callout kind="warning" title="Binary search finds; insertion moves" >}}
Finding position `i` in `O(log n)` comparisons does not insert an item into a
contiguous slice in `O(log n)` time. The suffix of length `n-i`
still moves. Repeating front-heavy inserts can produce quadratic movement.
{{< /callout >}}

The official [Go slices article](https://go.dev/blog/slices-intro) illustrates
the relationship between a slice descriptor and its backing array. Its sample
growth code is explanatory, however; production code may rely on the behavior
specified by Go, but not on a runtime growth strategy that can change between
releases.

## Establish one explicit order

Representation determines the cost of storing and moving records. A
comparison rule determines whether algorithms that rely on sorted input return
the right answer.

“Sorted” is incomplete without a comparison rule. For incident records, these
are different possible orders:

```text
OccurredAt ascending
OccurredAt descending
OccurredAt descending, then IncidentID ascending
```

Code that sorts the incident records and code that later searches or merges
them must use the same rule. That rule must also be transitive: if `a` sorts
before `b`, and `b` sorts before `c`, then `a` must sort before `c`. Without
that consistency, sorting and binary search cannot divide the records into
dependable ranges. The `slices.SortFunc` documentation calls the complete
comparator requirement a **strict weak ordering**.

### Stable order is not total order

A stable sort keeps equal records in the relative order in which it received
them. That is useful when the input order is meaningful, but it does not make
ties reproducible across map iterations, replicas, independently built result
sets, or later snapshots.

A **total order** distinguishes every pair of records. If incident IDs are
unique and immutable, `(OccurredAt DESC, IncidentID ASC)` supplies such an
order: incidents with different timestamps sort by time, while incidents with
the same timestamp sort by ID. A cursor can carry both values and identify the
last record returned from a fixed snapshot.

[MIT 6.006 Lecture 3](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/6d1ae5278d02bbecb5c4428928b24194_MIT6_006S20_lec3.pdf)
covers binary search, merge, and merge sort. The optional
[linear-sorting lecture](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/78a3c3444de1ff837f81e52991c24a86_MIT6_006S20_lec5.pdf)
develops stability and field-by-field tuple ordering further.

## Binary search only works on the order it expects

Binary search compares a target with the middle item and discards one half of
the remaining range. Because each comparison halves the range, it needs at
most `O(log n)` comparisons to find the target or the position where that
target could be inserted.

This works only when the slice is already sorted by the same comparison used
during the search. If the slice is in descending order and the search assumes
ascending order, the function can return a plausible index that is still
wrong. Binary search neither checks the entire slice nor accounts for the work
required to sort or update it.

Go's `slices.BinarySearch` requires an increasing sorted slice and returns the
earliest equal position or the insertion position. `BinarySearchFunc` requires
its comparison function to implement the same order as the slice.

Ask four questions when code says “we use binary search”:

1. Which function sorts the data?
2. Where does the program verify that incoming data is already sorted?
3. Does update work include movement or rebuilding after the search?
4. Is the query frequent enough to repay sorting or index construction?

## Merge updates instead of inserting them individually

Suppose a published snapshot has `n` records and an incoming batch has `m`.
In the worst case, every new key sorts before every item currently in the
slice and the batch order keeps putting the next key at the front. The inserts
then move `n`, `n+1`, `n+2`, and so on. The sum is:

\[
mn + \frac{m(m-1)}{2} = \Theta(mn+m^2)
\]

Instead of modifying the published sequence after every record, the builder
can process the incoming batch as a unit:

1. validate the incoming records;
2. sort them in `O(m log m)`;
3. resolve repeated keys using the service's replacement rule; and
4. linearly merge the two sorted inputs in `O(n+m)`.

During the merge, compare the next unconsumed record from each input. Copy the
record with the smaller key and advance that input. When the keys match, apply
the replacement rule and advance both inputs. Because each pass through the
loop consumes at least one record, the merge is linear in the combined input
size. Building a new slice requires extra memory, but it avoids reopening a
gap in the old slice for every incoming record and ensures that the returned
records do not share mutable backing storage with either input.

[W01: The Logarithmic Insert](wheel/01-logarithmic-insert/) turns this mistake
into a debugging exercise.

## Boolean search with sorted postings

An inverted index maps each term or tag to a **postings list** of matching
document IDs:

```text
database -> [2, 5, 8, 13]
latency  -> [1, 5, 8, 21]
```

To evaluate `database AND latency`, keep one position in each list:

- equal IDs are added to the result and both pointers advance;
- the smaller ID cannot appear later in the other sorted list, so its pointer
  advances;
- the operation ends when either input is exhausted.

The result is `[5, 8]`. Each comparison advances at least one position, so
lists of lengths `p` and `q` require at most `O(p+q)` advances. This reasoning
depends on both lists using the same increasing document-ID order. With several
terms, intersecting shorter postings first often keeps the temporary result
small.

The Stanford/Cambridge
[Boolean retrieval chapter](https://nlp.stanford.edu/IR-book/html/htmledition/processing-boolean-queries-1.html)
derives this algorithm and the shorter-list-first heuristic. It also makes a
useful distinction: Boolean retrieval determines which documents match a
query; it does not decide which matching document is most useful.

### Scanning can still be correct

Building an index spends time, memory, and write bandwidth. A scan can be the
right choice when the collection is small, the query is rare, the snapshot is
replaced before many queries reuse it, or predicates cannot be indexed
economically.

Reuse is the central trade-off. If building an index costs `B`, each indexed
query costs `C`, and `Q` queries share one snapshot, the average work per query
is approximately `B/Q + C`. The lab's `index-reused` benchmark measures only
`C`, after construction has already finished, so it does not describe the
end-to-end cost when `Q` is small. The lengths of real postings lists matter as
well: a common tag may leave most documents to examine, while a rare tag can
reduce the candidate set immediately.

## Collect all matches or produce them lazily

After matching begins, the implementation still has a choice: collect every
match in a slice, or return an iterator that produces one match at a time.

Collecting all matches is simple and lets callers revisit the result, but it
allocates space proportional to the number of matches. A lazy intersection can
instead retain only the positions of its input iterators. Although this saves
memory, early stopping is correct only when the consumer wants the iterator's
order. The local index yields increasing document IDs, so its first `k` matches
are not necessarily the `k` most recent runbooks. Filters can also require the
iterator to inspect many rejected candidates before it produces `k` accepted
ones.

[GitHub's Blackbird code-search report](https://github.blog/engineering/the-technology-behind-githubs-new-code-search/)
describes sorted postings, lazy iterators, early termination, and index
maintenance in a user-facing production search system. Blackbird can use early
termination because its identifiers contain information about result rank.
The local lab's IDs contain no such information; copying the technique would
return low-numbered documents, not the best or newest documents.

## Candidate retrieval is not result ranking

The unit's runbook search has two explicit stages:

```text
exact tag query -> matching IDs -> fixed display order
```

The first stage answers “Which records have every requested tag?” The second
answers “In what order should the user see those records?” Sorting the matches
by `(UpdatedAt DESC, DocumentID ASC)` puts recently updated runbooks first and
uses the document ID to resolve ties. It does not attempt to estimate which
runbook best answers the user's underlying question.

Recommendation systems go further: they define an objective, compute features,
score candidates, and often run several ranking passes. **Top-k** means keeping
the best `k` candidates according to such a score. LinkedIn's
[feed architecture](https://engineering.linkedin.com/teams/data/artificial-intelligence/feed)
provides a concrete production example: several initial rankers send candidates
to a more expensive ranking stage, followed by a final pass that applies
product rules. Later units will cover the heaps used for top-k selection and
the graphs used by algorithms such as PageRank. Here we stop after exact
matching and a fixed display order.

## Ordered postings in Prometheus

Prometheus provides a real Go example of the same underlying mechanics. Its
metrics index exposes postings through an iterator that promises increasing
series references. The links below are pinned to version 3.13.1 so the source
does not change underneath the explanation:

1. The [Prometheus 3.13.1 index format](https://github.com/prometheus/prometheus/blob/v3.13.1/tsdb/docs/format/index.md)
   stores postings as monotonically increasing series references.
2. [`Postings`](https://github.com/prometheus/prometheus/blob/v3.13.1/tsdb/index/postings.go)
   is an ordered iterator and `Seek` advances to a reference greater than or
   equal to a target.
3. `NewListPostings` requires ordered input.
4. In this pinned version, the list implementation uses
   `slices.BinarySearch` to seek within the remaining ordered values.
5. `Intersect` combines postings for AND queries.

{{< callout kind="note" title="What carries over to the lab" >}}
Prometheus shows why an iterator can implement `Seek` efficiently when its
input is ordered. The lab uses ordinary slices to make those operations easy
to inspect and count; its benchmark numbers describe only the lab. In both
systems, the increasing internal ID order serves index operations and says
nothing about the order in which a user should see results.
{{< /callout >}}

## Check correctness before timing

Begin with tests that establish the expected result, then add measurements that
explain its cost:

1. **Behavior tests:** the scan and index return the same match set; sorting
   does not change membership; pagination returns every ID exactly once.
2. **Ordering checks:** postings are duplicate-free and increasing; inputs to
   binary search and merge satisfy the declared order.
3. **Operation counts:** pointer advances and comparisons are bounded by
   postings lengths; the insertion counter records how many existing elements
   move when a gap is opened.
4. **Benchmarks and profiles:** vary document count, selectivity, overlap,
   index reuse, and update position, then identify where CPU time and
   allocations occur.

Elapsed time still matters, but it varies with the machine and its current
load. In a fixed test case, a count of four million shifted records explains
why the work grows even if the test happens to finish quickly. A CPU profile
can then confirm that the process is spending its time moving those records.

The [exploration lab](lab/) compares scanning with indexed queries, both with
and without index construction included. It also exercises postings overlap,
presentation sorting, pagination, and sorted updates. The
[two Wheel scenarios](wheel/) ask you to diagnose the slice-update and
pagination failures from incident reports and gradually revealed evidence.

For cursor design and snapshot behavior, continue to
[Paginating ordered results](result-ordering/).

## Interview translation

A concise answer might reason through the design this way:

> I would first ask how many queries reuse each snapshot. For repeated tag
> queries, an inverted index can replace a scan over all `d` runbooks with an
> intersection of the relevant postings lists. If two lists have lengths `p`
> and `q`, and both are sorted by document ID, a two-pointer intersection takes
> `O(p+q)` advances. If the snapshot serves only one query, however, building
> the index may cost more than scanning it.
>
> I would treat updates separately. Binary search can find an insertion point
> in `O(log n)` comparisons, but inserting into a Go slice still shifts the
> following records. For a large batch, I would sort the batch once and merge
> it with the current snapshot.
>
> After retrieval, I would sort the matches by the order promised by the API—for
> example, descending update time followed by a unique ID—and place both fields
> in the cursor. Tests would compare indexed results with a scan and verify
> pagination over tied timestamps. Benchmarks would use the service's actual
> tag frequencies, overlap, update positions, and queries per snapshot.

## Reflection

- Which operation must be fast, and how often does it run?
- Where is the order established, and which code ensures that updates preserve
  it?
- Which input distribution triggers the bad path?
- What does the service spend on construction, writes, memory, and snapshot
  replacement to make reads cheaper?
- Does the cursor contain every field used to break ties?
- Which claim came from a specification, current source, a production report,
  or a calculation made in this unit?
