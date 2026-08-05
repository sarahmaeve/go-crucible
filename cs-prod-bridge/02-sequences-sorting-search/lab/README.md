# Exploration Lab: Scan, Intersect, and Order

This lab searches a small catalog of SRE runbooks. Each runbook already has a
set of tags, so the exercises can concentrate on indexing, sorted lists, and
pagination rather than parsing text.

Every document must have a unique ID. `BuildIndex` returns an error when it
finds a duplicate; `ScanAND` expects the caller to provide the same valid
input. The lab offers two ways to answer an AND query:

- `ScanAND` examines every document and checks whether it contains every
  requested tag.
- `Index.SearchAND` retrieves the sorted list of IDs for each tag, starts with
  the shortest list, and intersects the lists.

Both methods return matching IDs in increasing numerical order.
`OrderResults` then loads the documents and sorts them for display, newest
first. When two documents have the same update time, the smaller ID comes
first. The lab does not calculate relevance scores.

## 1. Confirm behavior and ordering rules

Run commands in this lab from the `cs-prod-bridge` module directory.

```bash
go test ./02-sequences-sorting-search/lab -v
```

Before reading the implementation, write down the behavior you expect for:

- an empty query;
- a missing tag;
- repeated or differently-cased query tags;
- duplicate tags on one document;
- duplicate document IDs; and
- several documents with the same update timestamp.

Find the tests that check that every postings list is sorted and that the scan
and index return the same IDs.

## 2. Predict retrieval growth

Use the following names when describing the workload:

- `d`: number of documents
- `p` and `q`: number of IDs in two postings lists
- `r`: number of matching documents that must be sorted for display
- `Q`: number of queries answered using the same index

Before running the benchmarks, write down how you expect `d`, `p`, `q`, `r`,
and `Q` to affect each stage: scanning the documents, building the index,
intersecting two postings lists, and sorting the matches.

```bash
go test ./02-sequences-sorting-search/lab \
  -run '^$' -bench 'BenchmarkSearch|BenchmarkIndexReuse|BenchmarkSelectivity' \
  -benchmem -count=5
```

`index-reused` measures search after the index has already been built, so it is
not an end-to-end measurement. `BenchmarkIndexReuse` includes the build and
then runs 1, 10, or 100 queries against that index. Compare it with
`index-end-to-end`, especially in the one-query case, where the build has no
opportunity to pay for itself.

The selectivity benchmarks keep the number of documents fixed while changing
the sizes and overlap of the postings lists. Before comparing their timings,
identify the shorter list and the number of matching IDs in each case. Use
those differences to explain why document count alone does not predict the
cost of an indexed query.

## 3. Count comparisons and pointer advances

`IntersectSorted` records comparisons and pointer advances. Each loop advances
at least one input, so the operation is bounded by the combined postings
length rather than their product.

Change the two lists in `TestIntersectSortedCountsBoundedWork` to be:

- disjoint and interleaved;
- identical;
- one very short and one very long; and
- equal in length but with one match at the end.

Elapsed time varies across machines. The comparison and pointer-advance counts
let you compare the algorithm's work without depending on a particular CPU.

## 4. Separate matching from presentation

`OrderResults` sorts by update time, newest first. If two documents have the
same update time, it sorts them by ID:

```text
UpdatedAt descending, then document ID ascending
```

`PageAfter` expects documents in this order and records both fields in the
cursor. Add enough documents with the same timestamp to span two pages, then
confirm that following every cursor returns every ID once.

`PageAfter` models the ordered boundary after a production page token has been
decoded and validated. It does not model an HTTP response, token encoding,
authorization, expiry, or query binding. The API chapter treats those as a
separate contract so success in this slice-based lab is not mistaken for a
complete endpoint design.

Next, temporarily remove the ID tie-breaker. `PageAfter` should reject the
input because two documents now occupy the same position in the ordering. W02
contains a paginator that accepts timestamp-only ordering and demonstrates how
that design skips records: [The Timestamp-Only Cursor](../wheel/02-timestamp-only-cursor/).

## 5. Find the cost hidden behind binary search

Both update variants maintain the same sorted set:

- `InsertIndividually` binary-searches and inserts every ID into a slice.
- `SortAndMergeBatch` sorts and compacts the delta, then performs one linear
  union.

Run:

```bash
go test ./02-sequences-sorting-search/lab \
  -run '^$' -bench BenchmarkSortedUpdates -benchmem -count=5
```

The front-heavy fixture makes every individual insert shift a large suffix.
The append-heavy fixture does not. Binary search performs logarithmic
comparisons in both cases; it does not move the suffix.

## 6. Check where insertion time goes

```bash
go test ./02-sequences-sorting-search/lab \
  -run '^$' -bench 'BenchmarkSortedUpdates/front/individual$' \
  -benchtime=2s -cpuprofile /tmp/cs-bridge-ordered-insert.pprof

go tool pprof -top /tmp/cs-bridge-ordered-insert.pprof
```

Before opening the profile, predict whether comparisons, allocation, or copying
will use the most CPU. Then compare the profile with `UpdateStats.Shifted`.
The profile identifies where the program spent its time; the counter shows how
the amount of copying changes as the input grows.

## 7. Challenge the design

- When is scanning preferable to building an index?
- How many queries must use an index before the saved search work repays the
  build?
- How many index entries does each document add, and how much additional memory
  does that require?
- If documents can be added while a client is following page cursors, what
  behavior should the API promise?
- When would the service need to rank matches or retain only the best `k`,
  rather than sorting every match?

The benchmark results describe this small runbook catalog. Use the Datadog and
Prometheus sections of the guide for the behavior of those production systems.
