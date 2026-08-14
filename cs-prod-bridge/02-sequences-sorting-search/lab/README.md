# Lab: Scan, Intersect, and Order

This lab searches a small catalog of SRE runbooks. Each runbook already has
tags. The exercises therefore focus on indexes, sorted lists, and pagination,
not text parsing.

Every document must have a unique ID. `BuildIndex` returns an error for a
duplicate ID. `ScanAND` requires the same valid input. The lab has two ways to
answer an AND query:

- `ScanAND` examines every document and checks all requested tags.
- `Index.SearchAND` gets the sorted IDs for each tag, starts with the shortest
  list, and finds IDs shared by all lists.

Both methods return matching IDs in increasing numeric order. `OrderResults`
then loads the documents and sorts them for display. Newer documents come first.
For equal update times, the smaller ID comes first. The lab does not calculate
relevance scores.

## 1. Check results and ordering rules

Run commands in this lab from the `cs-prod-bridge` module directory.

```bash
go test ./02-sequences-sorting-search/lab -v
```

Before you read the implementation, write the result you expect for:

- an empty query;
- a missing tag;
- repeated or differently-cased query tags;
- duplicate tags on one document;
- duplicate document IDs; and
- several documents with the same update timestamp.

Find the tests that check two rules: every postings list is sorted, and the scan
and index return the same IDs.

## 2. Predict how search work will grow

Use the following names when describing the workload:

- `d`: number of documents
- `p` and `q`: number of IDs in two postings lists
- `r`: number of matching documents that must be sorted for display
- `Q`: number of queries answered using the same index

Before you run the benchmarks, predict how `d`, `p`, `q`, `r`, and `Q` affect
each stage. Consider the document scan, index build, postings intersection, and
display sort separately.

```bash
go test ./02-sequences-sorting-search/lab \
  -run '^$' -bench 'BenchmarkSearch|BenchmarkIndexReuse|BenchmarkSelectivity' \
  -benchmem -count=5
```

`index-reused` measures only search after the index build. It is not a complete
measurement. `BenchmarkIndexReuse` includes the build and then runs 1, 10, or
100 queries on that index. Compare it with `index-end-to-end`. Pay special
attention to the one-query case, where there is no reuse to repay the build.

**Selectivity** describes how much a filter reduces the possible matches. These
benchmarks keep document count fixed while changing postings-list size and
overlap. Before you compare times, identify the shorter list and count the
matching IDs in each case. Explain why document count alone does not predict
the cost of an indexed query.

## 3. Count comparisons and position advances

`IntersectSorted` records comparisons and position advances. Each pass advances
at least one input. The upper bound therefore depends on the combined postings
length, not the product of their lengths.

Change the two lists in `TestIntersectSortedCountsBoundedWork` to be:

- disjoint and interleaved;
- identical;
- one very short and one very long; and
- equal in length but with one match at the end.

Running time varies across machines. Comparison and position-advance counts let
you compare the work without depending on a particular CPU.

## 4. Separate matching from display order

`OrderResults` sorts by update time, newest first. If two documents have the
same update time, it sorts them by ID:

```text
UpdatedAt descending, then document ID ascending
```

`PageAfter` expects documents in this order and records both fields in the
cursor. Add enough documents with the same timestamp to span two pages, then
confirm that following every cursor returns every ID once.

`PageAfter` models the ordered boundary after the server decodes and validates
a production page token. It does not model an HTTP response, token encoding,
authorization, expiration, or the original query. The API chapter covers those
separate rules. Passing this slice-based lab does not prove that a complete
endpoint is correct.

Next, temporarily remove the ID tie-breaker. `PageAfter` should reject the input
because two documents now occupy the same position. W02 provides the related
debugging exercise: [The Timestamp-Only Cursor](../wheel/02-timestamp-only-cursor/).

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

In the front-heavy test data, each insert moves many existing records. In the
append-heavy data, it does not. Binary search uses logarithmic comparisons in
both cases. The insertion performs the movement.

## 6. Find where insertion uses CPU time

```bash
go test ./02-sequences-sorting-search/lab \
  -run '^$' -bench 'BenchmarkSortedUpdates/front/individual$' \
  -benchtime=2s -cpuprofile /tmp/cs-bridge-ordered-insert.pprof

go tool pprof -top /tmp/cs-bridge-ordered-insert.pprof
```

Before you open the profile, predict whether comparisons, allocation, or
copying will use the most CPU time. Then compare the profile with
`UpdateStats.Shifted`. The profile shows where the program spent its time. The
counter shows how movement changes as the input grows.

## 7. Test the limits of the design

- When is scanning preferable to building an index?
- How many queries must reuse an index before the saved search work repays its
  build cost?
- How many index entries does each document add, and how much additional memory
  does that require?
- If documents can be added while a client reads several pages, what should the
  API promise?
- When would the service need to rank matches or retain only the best `k`,
  rather than sorting every match?

The benchmark results describe only this small runbook catalog. Use the Datadog
and Prometheus sections of the guide for claims about those production systems.
