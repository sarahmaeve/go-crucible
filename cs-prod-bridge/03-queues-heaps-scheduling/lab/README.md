# Exploration Lab: Keep the Highest-Risk K

A fleet health service examines risk observations for many services but shows
only the `k` highest-risk services on its response page. This Unit 03 lab
isolates top-k selection before later exercises add readiness, retries, and
worker ownership.

The result contract is precise:

- a higher score is better;
- equal scores are ordered by service name, alphabetically;
- return at most `k` candidates in that complete order;
- do not change the input; and
- reject a negative `k` or a score that is `NaN`.

`TopKBySort` and `TopKByHeap` return the same answer. The difference is
the information they retain and the operations they perform while finding it.

## 1. Name the quantities before naming the bound

For the main case, assume `r` candidates and `1 <= k <= r`:

- `r` is the number of candidates examined;
- `k` is the number of winners retained; and
- `h` is the number of candidates after the first `k` that are
  good enough to replace the current cutoff, so `0 <= h <= r-k`.

When `k < r`, an exact method must inspect every candidate. If it skips
one, that unseen candidate could have the highest score and change the winning
set. Selection from unsorted input therefore requires at least linear work in
`r`. When `k = r`, membership is already known because every candidate
wins, but this API still has to copy and order all `r` candidates.

The formulas in this lab count calls to the complete `better` comparison.
Comparing two `float64` scores is fixed-size work. When scores tie, Go
compares service-name strings until it finds different bytes or reaches the
end of one name. We treat service names as bounded in length. If names can grow
with the input, their comparison cost also belongs in the model.

The validation rule is also part of correctness. A comparison involving
`NaN` does not establish the ordering required by either `sort.Slice`
or `container/heap`, so both methods reject it before ordering candidates.

## 2. Count the complete operations

`TopKBySort` validates the input, copies all `r` candidates, sorts the
copy, and returns the first `k`. Its dominant sorting work is:

```text
O(r log r) time
O(r)       working candidate storage
```

The storage bound follows from this API's promise not to change the caller's
slice. A method allowed to reorder that slice would not need the same full
candidate copy.

`TopKByHeap` copies the first `k` candidates and calls
`heap.Init`. Bottom-up heap construction costs `O(k)`. It then
compares each of the remaining `r-k` candidates with the root. Only the
`h` candidates that enter the answer replace the root and require up to
`O(log k)` heap work:

```text
build initial heap:       O(k)
check later candidates:   Θ(r-k)
restore after replacement: O(h log k)
selection total:          O(r + h log k)
working candidate storage: O(k)
```

In the worst case, every later candidate enters, so `h = r-k` and
selection has the familiar `O(r log k)` upper bound for
`k >= 2`. That compact bound is correct, but it hides two useful facts:

- all `r-k` later candidates receive one cutoff comparison; and
- only the `h` candidates that cross the cutoff cause logarithmic heap
  work.

The function promises an ordered result. A heap identifies which candidates
won, but its array is not a fully sorted list. Sorting the `k` survivors
adds `O(k log k)`, so the complete heap-based operation is:

```text
O(r + h log k + k log k)
```

Both lab methods also make a separate validation pass over the slice so they
can reject `NaN`. That pass is `Θ(r)` and does not change either
overall growth class. It does matter at a boundary: with `k = 0`, this
API validates all scores and then returns an empty result, so its actual work
is `Θ(r)`. An API that returned before validation would have a different
contract and a different boundary cost.

## 3. Put the useful cutoff at the root

For the largest `k` candidates, the bounded heap puts the **worst retained
winner** at its root. With score as the only key, that is a min-heap. A new
candidate can enter the answer exactly when it is better than this root.

Using a max-heap of the `k` winners would expose the best winner, which
does not tell us whether a new candidate should replace the weakest winner. To
find the smallest `k` values, reverse the design: keep the largest retained
value at the root of a max-heap.

The lab's tie rule is part of “better.” A lower score is worse. When scores are
equal, an alphabetically later service name is worse. The heap root is the
worst candidate under both fields, not just the candidate with the lowest
score.

Use this input with `k = 3`:

```text
(search, 0.72), (billing, 0.91), (edge, 0.83),
(catalog, 0.91), (worker, 0.40), (api, 0.83)
```

First draw the heap created from the first three candidates by
`heap.Init`. For each remaining candidate, mark the root, decide whether
the candidate crosses the cutoff, and redraw only the parent-child path that
can change after a replacement.

Do not sort the heap array in the drawing. Its promise is narrower: no child
is worse than its parent, so the worst retained candidate is available at
index zero.

This root has a different role from the heap in a dispatcher. The top-k heap
exposes the worst retained winner so checking a new candidate is cheap. A
dispatcher usually exposes the next work item to serve so removal is cheap.
The array mechanics are the same; the supplied comparison defines what the
root means.

## 4. Read phase counters before timing

Run commands from the `cs-prod-bridge` module directory:

```bash
go test ./03-queues-heaps-scheduling/lab -v
```

`SelectionStats` keeps phases separate so one broad asymptotic bound does
not conceal the operations that produced it:

- `CandidatesValidated` counts candidates checked for `NaN`;
- `AllCandidateSortComparisons` counts comparisons while the sort method
  ranks all candidates;
- `CutoffComparisons` counts comparisons between a later candidate and
  the worst retained winner;
- `HeapBuildComparisons` and `HeapBuildSwaps` measure the bottom-up
  construction of the initial size-`k` heap;
- `RootReplacements` is the measured `h`;
- `HeapRestoreComparisons` and `HeapRestoreSwaps` measure work after those
  replacements;
- `WinnerSortComparisons` measures the final ordering of only the retained
  winners; and
- `MaxRetained` records the largest working selection collection. It
  excludes the caller's input and returned copy, so it is a candidate count,
  not a byte-accurate memory measurement.

`TestTopKStatsSeparateWorkPhases` uses two inputs with the same
`r` and `k`. In one, the best candidates arrive first and
`h = 0`. In the other, every later candidate crosses the cutoff and
`h = r-k`. The number of cutoff checks stays the same while the number
of heap restorations changes. This is the distinction hidden by the worst-case
`O(r log k)` summary.

`TestTopKHeapWithOneWinnerNeedsNoHeapReordering` shows the `k = 1`
case. Every later candidate is still checked, but a one-element heap has no
parent-child order to restore and its one winner needs no final sorting.

## 5. Change one workload dimension at a time

The benchmark includes final result ordering for both methods; it never
compares an ordered sort result with an unordered heap array.

```bash
go test ./03-queues-heaps-scheduling/lab \
  -run '^$' -bench BenchmarkTopK -benchmem -count=5
```

Its cases isolate three questions:

- Holding `k = 10` while `r` grows from 1,000 to 100,000 shows the
  unavoidable input scan.
- Holding `r = 10,000` while changing `k` from 1 to 10,000 shows
  when bounded retention stops being much smaller than the input.
- Holding `r = 10,000` and `k = 100` while changing arrival order
  gives the bounded-heap path no later replacements in one case and a
  replacement for every later candidate in the other. Input order may also
  affect the sort implementation's constant factors.

Use operation counts to predict the trend before reading nanoseconds. Then use
the benchmark to find the effect of Go's implementation, memory layout, and
constant factors on this machine.

Read both allocation columns. The heap retains fewer candidate records when
`k` is small, but fewer retained records do not guarantee fewer allocation
events or lower elapsed time. The full sort uses optimized contiguous-slice
code, while the bounded heap performs its own comparisons and swaps. Allocation
counts include slice copies and implementation bookkeeping; they do not count
logical candidates directly. The `O(k)` storage bound describes how
retained candidate storage grows; it does not predict the number of allocations
or which implementation wins at every measured size.

The lab already receives all candidates in a `[]Candidate`. Its
`O(k)` claim therefore describes **additional working candidate storage**.
A streaming API could feed the same bounded-heap operation from an iterator,
channel, file, or network connection without retaining all `r` candidates
first. This particular API does not demonstrate that end-to-end memory use.

## 6. Compare the heap with real alternatives

Top-k is a result contract, not the name of one algorithm. For an exact,
ordered answer, the main choices include:

| Method | Time for ordered top `k` | Working candidate storage in this lab's non-mutating setting | Trade-off |
|---|---:|---:|---|
| Sort all candidates | `O(r log r)` | `O(r)` | Direct and often fast; ranks discarded candidates |
| Build a max-heap of all `r`, then remove `k` roots | `O(r + k log r)` | `O(r)` | Linear bottom-up build and ordered removals; retains every candidate |
| Keep a min-heap of `k` winners | `O(r + h log k + k log k)` | `O(k)` | One-pass selection and bounded state; update work depends on replacements |
| Quickselect, then sort the winners | expected `O(r + k log k)` | `O(r)` here; a mutable input can be partitioned in place | Good batch option; ordinary pivot choices can degrade to `O(r^2)` and it does not maintain a streaming answer |
| Keep a sorted slice of `k` winners | `O(rk)` worst case | `O(k)` | Simple and already ordered; linear insertion becomes expensive as `k` grows |

This lab implements full sorting and a bounded heap because those two methods
make the retained-information difference easy to inspect. It does not establish
that they are the only choices. In an in-memory batch, quickselect or a full
bottom-up heap can have a better time bound than the bounded heap's worst case.
When `k` is tiny, a sorted slice can be competitive despite its worse
asymptotic bound. Measure after matching the algorithm to the actual contract.

These boundary cases should change the choice:

- fixed small `k`: bounded selection grows linearly with `r` and keeps
  `O(k)` working candidates;
- `k = 1`: use the mental model of a maximum scan, even if the generic
  implementation still stores a one-element heap;
- `k` close to `r`: final sorting approaches a full sort, so the heap
  has little to save; and
- `k >= r`: every candidate is returned, so direct sorting is clearer
  than building a bounded heap and sorting the same candidates afterward.

## 7. Follow the production implementation

[Prometheus v3.13.1 `topk`](https://github.com/prometheus/prometheus/blob/v3.13.1/promql/engine.go#L3951-L4085)
uses the same central operation for each aggregation group: retain at most
`k` samples, compare later samples with the smallest retained value, replace
that root when the sample belongs in the result, and call `heap.Fix` when
`k > 1`. It sorts retained instant-query samples before returning them.

The local service names, validation, complete tie rule, counters, and API are
synthetic. They make the operation inspectable; they are not a model of all
PromQL semantics.

If a product asks for the `k` most frequent raw values, remember that the
heap does not compute frequencies. A separate pass must first count each
distinct value, often with the hash-table model from Unit 01. The heap then
selects among those distinct values and their counts.

For the heap invariant, array representation, operations, and proof, use
[MIT 6.006 Lecture 8: Binary Heaps](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/40d4851e550507ca14dc778b9b2266cc_MIT6_006S20_lec8.pdf).
For the exact Go interface used here, read the standard library's
[`container/heap` documentation](https://pkg.go.dev/container/heap).

## 8. Explain the choice without hiding the conditions

Finish with short answers to these questions:

1. Why does finding the largest `k` use the worst retained winner as the
   heap root?
2. Which terms in `O(r + h log k + k log k)` correspond to
   input inspection, changed winners, and final result ordering?
3. Why can two inputs with the same `r` and `k` perform different
   amounts of heap work?
4. What exactly does the bounded heap avoid computing, and what work can it
   never avoid for an exact answer?
5. At which relationship between `k` and `r` would you choose direct
   sorting for this API, and which benchmark evidence would support that
   decision?
6. Why does this slice-based lab prove an `O(k)` additional-storage bound
   but not an end-to-end streaming memory bound?
