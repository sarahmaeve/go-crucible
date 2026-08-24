# Lab: Keep the Highest-Risk K

A fleet-health service examines many risk scores but returns only the `k`
highest-risk services. This lab focuses on top-k selection. Later exercises add
readiness, retries, and worker ownership.

The result must follow these rules:

- a higher score is better;
- equal scores use service name in alphabetical order;
- return at most `k` candidates in that complete order;
- do not change the input; and
- reject a negative `k` or a score that is `NaN`.

`TopKBySort` and `TopKByHeap` return the same answer. They keep different
amounts of information and do different work to find it.

## 1. Name the input sizes before calculating cost

For the main case, use `r` candidates and `1 <= k <= r`:

- `r` is the number of candidates examined;
- `k` is the number of winners retained; and
- `h` is the number of later candidates that replace a current winner, so
  `0 <= h <= r-k`.

An exact method must inspect every candidate. A skipped candidate might have the
highest score. Selection from unsorted input therefore requires at least linear
work in `r`. When `k = r`, every candidate wins, but this API must still copy
and order all `r` candidates.

The formulas count calls to the complete `better` comparison. Comparing two
`float64` scores has a fixed cost. If scores tie, Go compares service-name
bytes until it finds a difference. The model assumes a fixed maximum name
length. If names can grow, include their comparison cost too.

Validation is part of correctness. A `NaN` score cannot provide the ordering
that `sort.Slice` and `container/heap` require, so both methods reject it before
ordering candidates.

## 2. Count every phase

`TopKBySort` validates the input, copies all `r` candidates, sorts the copy, and
returns the first `k`:

```text
O(r log r) time
O(r)       working candidate storage
```

The API promises not to change the caller's slice, so sorting requires a copy.
A method allowed to reorder the input would not need that copy.

`TopKByHeap` copies the first `k` candidates and calls `heap.Init`. This
bottom-up build costs `O(k)`. It compares each later candidate with the root.
Only the `h` new winners replace the root and require up to `O(log k)` heap
work:

```text
build initial heap:       O(k)
check later candidates:   Θ(r-k)
restore after replacement: O(h log k)
selection total:          O(r + h log k)
working candidate storage: O(k)
```

If every later candidate wins, `h = r-k`. This gives the familiar `O(r log k)`
upper bound for `k >= 2`. That short bound hides two useful facts:

- all `r-k` later candidates receive one cutoff comparison; and
- only the `h` candidates that cross the cutoff cause logarithmic heap
  work.

The function promises an ordered result. A heap finds the winners, but its array
is not fully sorted. Sorting the `k` winners adds `O(k log k)`, so the total is:

```text
O(r + h log k + k log k)
```

Both methods make a separate `Θ(r)` validation pass to reject `NaN`. This does
not change the overall growth. It matters when `k = 0`: this API still checks
all scores before returning an empty result, so that case is `Θ(r)`. Another
API could choose to return before validation.

## 3. Put the cutoff at the root

To find the largest `k` candidates, the heap puts the **worst current winner**
at its root. If score is the only key, this is a min-heap. A new candidate wins
only when it is better than the root.

A max-heap would expose the best winner, which cannot tell us whether a new
candidate should replace the weakest winner. To find the smallest `k` values,
reverse the rule and keep the largest current winner at a max-heap root.

The tie rule is part of “better.” A lower score is worse. If scores tie, an
alphabetically later name is worse. The root is the worst candidate under both
fields, not only the lowest score.

Use this input with `k = 3`:

```text
(search, 0.72), (billing, 0.91), (edge, 0.83),
(catalog, 0.91), (worker, 0.40), (api, 0.83)
```

First draw the heap that `heap.Init` creates from the first three candidates.
For each later candidate, mark the root, decide whether the candidate wins, and
redraw only the parent-child path changed by a replacement.

Do not sort the heap array. It promises only that no child is worse than its
parent, which puts the worst winner at index zero.

The root has a different meaning in a dispatcher. A top-k heap exposes the
worst winner so it can check a new candidate. A dispatcher usually exposes the
next work item. The array operations are the same; the comparison defines what
the root means.

## 4. Read operation counts before timing

Run commands from the `cs-prod-bridge` module directory:

```bash
go test ./03-queues-heaps-scheduling/lab -v
```

That ordinary command runs the seed corpus for
`FuzzTopKImplementationsAgree`. To generate more combinations of candidates,
ties, special floating-point scores, and boundary values of `k`, run:

```bash
go test ./03-queues-heaps-scheduling/lab \
  -run '^$' -fuzz '^FuzzTopKImplementationsAgree$' -fuzztime=10s
```

The fuzz property uses full sorting as a reference for the bounded heap. It
also checks result order, retained size, matching validation errors, and that
neither implementation changes the input.

`SelectionStats` counts each phase separately:

- `CandidatesValidated` counts candidates checked for `NaN`;
- `AllCandidateSortComparisons` counts comparisons while sorting all
  candidates;
- `CutoffComparisons` counts comparisons between a later candidate and
  the worst retained winner;
- `HeapBuildComparisons` and `HeapBuildSwaps` count the bottom-up build of the
  initial size-`k` heap;
- `RootReplacements` is the measured `h`;
- `HeapRestoreComparisons` and `HeapRestoreSwaps` count work after replacements;
- `WinnerSortComparisons` counts the final ordering of the retained
  winners; and
- `MaxRetained` records the largest working collection. It excludes the input
  and returned copy, so it counts candidates rather than bytes.

`TestTopKStatsSeparateWorkPhases` uses two inputs with the same `r` and `k`.
In one, the best candidates arrive first and `h = 0`. In the other, every later
candidate wins and `h = r-k`. Both inputs perform the same cutoff checks but a
different number of heap repairs. The worst-case `O(r log k)` summary hides
this difference.

`TestTopKHeapWithOneWinnerNeedsNoHeapReordering` covers `k = 1`. Every later
candidate is checked, but a one-element heap needs no repair or final sort.

## 5. Change one input size at a time

Both benchmarks include the final result sort. They do not compare an ordered
sort result with an unordered heap array.

```bash
go test ./03-queues-heaps-scheduling/lab \
  -run '^$' -bench BenchmarkTopK -benchmem -count=5
```

The cases ask three questions:

- Keep `k = 10` while `r` grows from 1,000 to 100,000 to show the required
  input scan.
- Keep `r = 10,000` while `k` grows from 1 to 10,000 to find when the retained
  set is no longer much smaller than the input.
- Keep `r = 10,000` and `k = 100` while changing input order. One case causes
  no later replacements; the other replaces the root for every later
  candidate. Input order can also affect the sort's machine-specific costs.

Predict the operation counts before reading the run times. Then use the
benchmark to see how Go's implementation and memory layout affect this machine.

Read both allocation columns. When `k` is small, the heap retains fewer
candidates. That does not guarantee fewer allocations or a faster run. Full
sorting uses optimized slice code; the heap performs its own comparisons and
swaps. Allocation counts include copies and internal records, not logical
candidates. The `O(k)` bound describes retained candidate storage. It does not
predict allocation count or the faster implementation at every size.

The lab already receives all candidates in a `[]Candidate`. Its `O(k)` claim
therefore describes **additional working candidate storage**. A streaming API
could feed the heap from an iterator, channel, file, or network connection.
This slice-based API does not demonstrate total streaming memory use.

## 6. Compare the available methods

Top-k describes the required result, not one algorithm. For an exact, ordered
answer, the choices include:

| Method | Time for ordered top `k` | Working candidate storage when the input must stay unchanged | Trade-off |
|---|---:|---:|---|
| Sort all candidates | `O(r log r)` | `O(r)` | Direct and often fast; ranks discarded candidates |
| Build a max-heap of all `r`, then remove `k` roots | `O(r + k log r)` | `O(r)` | Linear bottom-up build and ordered removals; retains every candidate |
| Keep a min-heap of `k` winners | `O(r + h log k + k log k)` | `O(k)` | One-pass selection that stores `k` candidates; update work depends on replacements |
| Quickselect, then sort the winners | expected `O(r + k log k)` | `O(r)` here; a mutable input can be partitioned in place | Good batch option; ordinary pivot choices can degrade to `O(r^2)` and it does not maintain a streaming answer |
| Keep a sorted slice of `k` winners | `O(rk)` worst case | `O(k)` | Simple and already ordered; linear insertion becomes expensive as `k` grows |

The lab implements full sorting and a bounded heap because they make the
storage difference easy to inspect. They are not the only choices. For an
in-memory batch, quickselect or a full bottom-up heap can have a better time
bound for some values of `r` and `k`. When `k` is very small, a sorted slice can
be competitive despite worse growth. Match the method to the required result,
then measure it.

Pay special attention to these boundaries:

- fixed small `k`: bounded selection grows linearly with `r` and keeps `O(k)`
  working candidates;
- `k = 1`: use the mental model of a maximum scan, even if the generic
  implementation still stores a one-element heap;
- `k` close to `r`: final sorting approaches a full sort, so the heap
  has little to save; and
- `k >= r`: every candidate is returned, so direct sorting is clearer than
  building a heap and sorting the same candidates afterward.

## 7. Compare with a production implementation

[Prometheus v3.13.1 `topk`](https://github.com/prometheus/prometheus/blob/v3.13.1/promql/engine.go#L3951-L4085)
uses the same central operation for each group. It retains at most `k` samples,
compares later samples with the smallest retained value, replaces that root for
a new winner, and calls `heap.Fix` when `k > 1`. It sorts the retained
instant-query samples before returning them.

The local service names, validation, tie rule, counters, and API are invented
for this lab. They make the operation easy to inspect; they do not represent
all PromQL behavior.

If a product asks for the `k` most frequent raw values, the heap does not count
them. First count each distinct value, often with a hash table. Then use the
heap to select among those values and counts.

For the heap rule, array representation, operations, and proof, use
[MIT 6.006 Lecture 8: Binary Heaps](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/40d4851e550507ca14dc778b9b2266cc_MIT6_006S20_lec8.pdf).
For the exact Go interface used here, read the standard library's
[`container/heap` documentation](https://pkg.go.dev/container/heap).

## 8. Explain the choice plainly

Finish with short answers to these questions:

1. Why does a largest-`k` heap keep the worst current winner at its root?
2. Which terms in `O(r + h log k + k log k)` correspond to
   input inspection, changed winners, and final result ordering?
3. Why can two inputs with the same `r` and `k` perform different
   amounts of heap work?
4. Which work does the bounded heap avoid, and which work must every exact
   method still do?
5. When would you choose direct sorting for this API, and which benchmark
   result would support that choice?
6. Why does this slice-based lab show `O(k)` additional storage but not total
   `O(k)` memory use?
