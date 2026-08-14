+++
title = 'Sequences, sorting, and ordered search'
description = 'Use ordered data to make searches and merges cheaper, and keep every reader and update on the same ordering rule.'
weight = 2
+++

**Unit 02 · Foundations**

# Sequences, sorting, and ordered search

{{< lead >}}Systems keep data ordered to make later work cheaper. Binary search
can discard half of the remaining records. Two sorted lists can merge in one
pass. A page token can mark where the next page starts. This unit explains the
cost of creating that order and the failures that occur when parts of a system
use different ordering rules.{{< /lead >}}

{{< callout kind="production" title="Incoming reports" >}}
- A label-filter query is fast for ordinary metrics. It times out when it must
  examine every series for a metric with many distinct series.
- Reading a configuration snapshot is fast. Publishing a new one uses a full
  CPU core while the builder inserts records into a sorted slice one at a time.
- An incident endpoint returns HTTP 200 with normal latency for every page. It
  omits records when many incidents share the timestamp at a page boundary.
{{< /callout >}}

Each report concerns order at a different stage. The metric query needs an
index that reduces the number of records it examines. The snapshot builder has
made reads cheap by doing too much work during updates. The incident endpoint
sorts its records but cannot identify one exact boundary when timestamps tie.

These stages can use different orders for different purposes. A postings list
can use increasing document IDs because that order makes intersection cheap.
The same service can show final results by descending update time because users
expect to see new records first.

By the end of this unit, you should be able to:

- choose a sequence representation from the operations it must support;
- state the comparison rule and required input order for a binary search,
  merge, intersection, or cursor;
- separate logarithmic search from linear movement in a contiguous slice;
- calculate the work for scanning, sorting, batch merging, and postings
  intersection;
- explain the difference between stable sorting and a total order;
- keep exact matching separate from display order and relevance;
- include index construction, reuse, writes, storage, and updates in a design
  decision; and
- use correctness tests, operation counts, benchmarks, and profiles to explain
  what happens as the inputs grow.

## Start with the input sizes

Do not treat every input as one value called “size.” Separate the stored
collection from one update or query. This unit uses:

- `d`: documents or series that a query could examine
- `n`: entries already in a sorted sequence
- `m`: entries in one update batch
- `p`, `q`: entries in two postings lists
- `r`: matching records before display sorting
- `k`: records requested on one page
- `Q`: queries that reuse the same index snapshot

These quantities lead to different decisions. A service with one million
stored records and ten updates does different work from an initial load of one
million records. An indexed query also does not always examine all `d`
documents. Its work often depends on `p`, `q`, and how much the two postings
lists overlap.

| Operation | Modeled time | Condition behind the claim |
|---|---:|---|
| Scan all documents for one constant-cost check | `Θ(d)` | No reusable index |
| Merge-sort `n` items | `Θ(n log n)` | Comparisons are treated as constant cost |
| Binary-search a sorted slice | `O(log n)` comparisons | Search and sort use the same order |
| Insert at slice position `i` | `Θ(n-i)` moves | Elements are contiguous |
| Insert `m` items at or near the front one by one | `Θ(mn+m²)` moves | Each insertion shifts almost the entire current slice |
| Sort a batch, then merge | `O(m log m+n+m)` | Both inputs use one order and conflict rule |
| Intersect two postings lists | `O(p+q)` advances | Both lists use the same document-ID order |
| Merge-sort `r` results for display | `Θ(r log r)` comparisons | Index order is not display order |

These bounds describe how work grows with the inputs. They do not predict
running time, how well the data uses CPU caches, or allocation counts on a
particular machine. Measure those values.

## Choose storage from the operations you need

Arrays, slices, and linked lists can all store a sequence of values, but they
make different operations cheap. First ask which operations the service does
most often. No representation is fastest for every operation.

| Representation | Indexed access | Insert/delete near middle | Iteration behavior |
|---|---:|---:|---|
| Fixed array | constant | linear movement | contiguous and cache-friendly |
| Dynamic array / Go slice | constant | linear movement | contiguous; usually constant work per append over a long series of appends, with occasional copying |
| Singly linked list | linear | constant after locating predecessor | pointer chasing; no random access |

The cost of one operation is not enough to choose a representation. A linked
list can insert in constant time after the caller finds the preceding node.
Finding that node can still require a linear walk. A dynamic array must move
elements for a middle insertion. It can still traverse quickly because adjacent
elements occupy adjacent memory and do not store a link for each element.

[MIT 6.006 Recitation 2](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/c08a3b63dfe5f6f6b32257d35f86ae63_MIT6_006S20_r02.pdf)
works through these trade-offs formally for arrays, linked lists, and dynamic
arrays.

{{< callout kind="production" title="A production example from Twitter" >}}
[Twitter's real-time search indexing report](https://blog.x.com/engineering/en_us/topics/infrastructure/2020/reducing-search-indexing-latency-to-one-second)
describes an index that first used a prepend-only unrolled linked list. Each
node held several records together, which helped it use CPU caches well. Later,
the service had to insert records that arrived out of order.
The team moved to a skip list. The new structure required the service to follow
more pointers, but it handled out-of-order insertion better. This unit does not
implement a skip list. The example shows why the update pattern matters as much
as lookup cost.
{{< /callout >}}

## Go slices share backing arrays

The [Go specification](https://go.dev/ref/spec#Slice_types) defines a slice as a
description of a continuous part of an underlying array. The slice length is
the number of elements you can access. Its capacity extends from the first
accessible element to the end of the array.

```go
base := []string{"a", "b", "c", "d"}
view := base[1:3] // "b", "c"; shares base's backing array
view[0] = "B"    // base[1] is now "B"
```

Creating `view` is cheap because Go does not copy the strings. Both slices refer
to the same memory. A change through one slice can therefore appear through the
other.

A function can promise a read-only snapshot, but returning a slice does not
enforce that promise. The function must control every other slice that shares
the array, or it must copy the data before publication. A small subslice can
also keep a much larger array in memory.

`append` has the same sharing risk. When the slice has enough capacity, `append`
can reuse the existing array. Another slice that shares the array can then see
the writes. When capacity is insufficient, `append` allocates a new array. The
specification does not promise a particular capacity growth rule. The built-in
`copy` function supports overlapping slices, so code can use it to move elements
within one array.

The [`slices` package](https://pkg.go.dev/slices) provides generic operations.
Those operations still pay the cost of the underlying representation.
`slices.Insert` must open a gap and move the elements after it. Its documented
running time is `O(len(s)+len(v))`.

{{< callout kind="warning" title="Binary search finds; insertion moves" >}}
Binary search can find position `i` in `O(log n)` comparisons. Inserting there
is a separate operation. A contiguous slice must still move the `n-i` elements
after that position. Many inserts near the front can cause quadratic movement.
{{< /callout >}}

The official [Go slices article](https://go.dev/blog/slices-intro) illustrates
how a slice refers to its underlying array. Its sample growth code is an
explanation, not a language guarantee. Production code can rely on the Go
specification. It must not rely on a runtime growth strategy that can change
between releases.

## Write down one comparison rule

The representation determines the cost of storing and moving records. The
comparison rule determines whether algorithms that require sorted input return
the correct answer.

“Sorted” is incomplete without a comparison rule. For incident records, these
are different possible orders:

```text
OccurredAt ascending
OccurredAt descending
OccurredAt descending, then IncidentID ascending
```

The code that sorts, searches, and merges the incident records must use the same
rule. The rule must also be transitive. If `a` comes before `b`, and `b` comes
before `c`, then `a` must come before `c`. Without this consistency, sorting and
binary search cannot divide records into reliable ranges. The
`slices.SortFunc` documentation calls this comparator requirement a **strict
weak ordering**.

### A stable sort does not create a total order

A stable sort keeps equal records in their input order. This helps when the
input order has meaning. It does not make ties reproducible across map
iterations, replicas, separately built results, or later snapshots.

A **total order** gives every record a unique position. If incident IDs are
unique and do not change, `(OccurredAt DESC, IncidentID ASC)` creates a total
order. Incidents with different timestamps sort by time. Incidents with the
same timestamp sort by ID. A cursor can carry both values and identify the last
record returned from a fixed snapshot.

[MIT 6.006 Lecture 3](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/6d1ae5278d02bbecb5c4428928b24194_MIT6_006S20_lec3.pdf)
covers binary search, merge, and merge sort. The optional
[linear-sorting lecture](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/78a3c3444de1ff837f81e52991c24a86_MIT6_006S20_lec5.pdf)
develops stability and field-by-field tuple ordering further.

## Binary search requires the expected order

Binary search compares the target with the middle item. It then discards half
of the remaining range. It needs at most `O(log n)` comparisons to find the
target or the position where the target could be inserted.

This works only when the slice is already sorted by the comparison used for the
search. If the slice is descending and the search assumes ascending order, the
function can return a plausible but incorrect index. Binary search does not
check the entire slice. It also does not include the work required to sort or
update the slice.

Go's `slices.BinarySearch` requires an increasing sorted slice and returns the
earliest equal position or the insertion position. `BinarySearchFunc` requires
its comparison function to implement the same order as the slice.

When code says “we use binary search,” ask four questions:

1. Which function sorts the data?
2. Where does the program verify that incoming data is already sorted?
3. Does update work include movement or rebuilding after the search?
4. Is the query frequent enough to repay sorting or index construction?

## Merge a batch instead of inserting one record at a time

Suppose the published snapshot has `n` records and the incoming batch has `m`.
In the worst case, every new key belongs before the current records. Each insert
opens another gap near the front. The inserts move `n`, `n+1`, `n+2`, and so on.
The sum is:

\[
mn + \frac{m(m-1)}{2} = \Theta(mn+m^2)
\]

The builder can process the complete incoming batch instead:

1. validate the incoming records;
2. sort them in `O(m log m)`;
3. resolve repeated keys using the service's replacement rule; and
4. linearly merge the two sorted inputs in `O(n+m)`.

During the merge, compare the next unused record from each input. Copy the
record with the smaller key and advance that input. When the keys match, apply
the replacement rule and advance both inputs. Each pass consumes at least one
record, so the merge is linear in the combined input size.

The new slice requires extra memory. It avoids opening a new gap for every
incoming record, and it does not share writable array storage with either
input.

[W01: The Logarithmic Insert](wheel/01-logarithmic-insert/) turns this mistake
into a debugging exercise.

## Find shared IDs in sorted postings lists

An inverted index maps each term or tag to a **postings list** of matching
document IDs. For example:

```text
database -> [2, 5, 8, 13]
latency  -> [1, 5, 8, 21]
```

To evaluate `database AND latency`, keep a position in each list:

- If the IDs are equal, add the ID to the result and advance both positions.
- If the IDs differ, advance the position with the smaller ID. That ID cannot
  appear later in the other sorted list.
- Stop when either list has no more IDs.

The result is `[5, 8]`. Each comparison advances at least one position. Lists
of lengths `p` and `q` therefore require at most `O(p+q)` advances. Both lists
must use the same increasing document-ID order. For several terms, start with
the shorter postings lists to keep the temporary result small.

The Stanford/Cambridge
[Boolean retrieval chapter](https://nlp.stanford.edu/IR-book/html/htmledition/processing-boolean-queries-1.html)
derives this algorithm and explains why it helps to start with shorter lists.
It also separates two questions. Boolean retrieval determines which documents
match. It does not decide which matching document is most useful.

### A scan can still be the right choice

Building an index uses time, memory, and write bandwidth. A scan can be the
right choice when the collection is small or the query is rare. It can also be
right when the service replaces a snapshot before many queries reuse it, or
when an index would cost too much to maintain.

Reuse is the central trade-off. Suppose building an index costs `B`, each
indexed query costs `C`, and `Q` queries share one snapshot. The average work
per query is approximately `B/Q + C`.

The lab's `index-reused` benchmark measures only `C`. It starts after index
construction, so it does not show the full cost when `Q` is small. The lengths
of the postings lists matter too. A common tag can leave most documents to
examine. A rare tag can reduce the possible matches immediately.

## Collect matches now or produce them as needed

The implementation can collect every match in a slice. It can instead return
an iterator that produces one match at a time.

Collecting all matches is simple and lets callers read the results again. It
uses memory in proportion to the number of matches. A lazy intersection can
keep only the positions of its input iterators.

This saves memory, but stopping early is correct only when the caller wants the
iterator's order. The local index returns increasing document IDs. Its first
`k` matches are not necessarily the `k` newest runbooks. A filter can also make
the iterator examine many rejected records before it produces `k` results.

[GitHub's Blackbird code-search report](https://github.blog/engineering/the-technology-behind-githubs-new-code-search/)
describes sorted postings, lazy iterators, early stopping, and index maintenance
in a production search system. Blackbird can stop early because its identifiers
contain information about result rank. The local lab's IDs do not. Using the
same technique here would return low-numbered documents, not the best or newest
documents.

## Matching records is different from ranking them

The runbook search has two stages:

```text
exact tag query -> matching IDs -> fixed display order
```

The first stage asks which records have every requested tag. The second asks
which order the user should see. `(UpdatedAt DESC, DocumentID ASC)` puts recently
updated runbooks first and uses the document ID to resolve ties. It does not
estimate which runbook best answers the user's question.

Recommendation systems do more. They define a goal, calculate features, score
possible results, and often rank them more than once. **Top-k** means keeping
the best `k` results under that score. LinkedIn's
[feed architecture](https://engineering.linkedin.com/teams/data/artificial-intelligence/feed)
provides a production example. Several fast scoring stages send results to a
more expensive ranking stage. A final pass applies product rules. Later units cover
heaps for top-k selection and graphs for algorithms such as PageRank. This unit
stops after exact matching and a fixed display order.

## Optional: ordered postings in Prometheus

Prometheus provides a Go example of the same operations. Its metrics index
provides postings through an iterator that promises increasing series
references. These links use version 3.13.1 so that the source stays fixed:

1. The [Prometheus 3.13.1 index format](https://github.com/prometheus/prometheus/blob/v3.13.1/tsdb/docs/format/index.md)
   stores postings as series references that always increase.
2. [`Postings`](https://github.com/prometheus/prometheus/blob/v3.13.1/tsdb/index/postings.go)
   is an ordered iterator and `Seek` advances to a reference greater than or
   equal to a target.
3. `NewListPostings` requires ordered input.
4. In this pinned version, the list implementation uses
   `slices.BinarySearch` to seek within the remaining ordered values.
5. `Intersect` combines postings for AND queries.

{{< callout kind="note" title="What carries over to the lab" >}}
Ordered input lets an iterator implement `Seek` efficiently. The lab uses
ordinary slices so you can inspect and count the operations. Its benchmark
numbers apply only to the lab. In both systems, increasing internal IDs support
index operations. They do not determine the order shown to users.
{{< /callout >}}

## Check correctness before timing

First, use tests to establish the correct result. Then measure its cost:

1. **Behavior tests:** Check that the scan and index return the same matches.
   Check that sorting keeps the same members and pagination returns each ID
   once.
2. **Ordering checks:** Check that postings have no duplicates and increase.
   Check that binary search and merge receive data in the declared order.
3. **Operation counts:** Bound position advances and comparisons by the postings
   lengths. Count the elements moved when an insertion opens a gap.
4. **Benchmarks and profiles:** Change document count, the fraction that
   matches, overlap, index reuse, and update position. Find where CPU time and
   allocations occur.

Running time still matters, but it changes with the machine and its current
load. In a fixed test, four million moved records explain why the work grows,
even if the test finishes quickly. A CPU profile can confirm that the process
spends its time moving those records.

The [exploration lab](lab/) compares scanning with indexed queries, both with
and without index construction included. It also exercises postings overlap,
display sorting, pagination, and sorted updates. The
[two Wheel scenarios](wheel/) ask you to diagnose the slice-update and
pagination failures from incident reports and gradually revealed evidence.

For cursor design and snapshot behavior, continue to
[Paginating ordered results](result-ordering/).

## Explain the decision to a colleague

One explanation could be:

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

## Questions to review

- Which operation must be fast, and how often does it run?
- Where is the order established, and which code ensures that updates preserve
  it?
- Which input pattern causes the expensive path?
- What does the service spend on construction, writes, memory, and snapshot
  replacement to make reads cheaper?
- Does the cursor contain every field used to break ties?
- For each claim, did it come from a specification, current source, production
  report, or calculation in this unit?
