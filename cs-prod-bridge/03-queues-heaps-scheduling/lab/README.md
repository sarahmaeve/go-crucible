# Exploration Lab: Keep the Highest-Risk K

A fleet health service examines risk observations for many services but shows
only the `k` highest-risk services on its response page. This first Unit 03
exercise isolates the heap operation before later exercises add readiness,
retry, and worker ownership.

The result contract is precise:

- a higher score is better;
- equal scores are ordered by service name, alphabetically;
- return at most `k` candidates in that complete order;
- do not change the input; and
- reject a negative `k` or a score that is `NaN`.

`TopKBySort` and `TopKByHeap` return the same result. They retain different
amounts of information while finding it.

## 1. Start with the question the product asks

If the caller needs every candidate in ranked order, sorting every candidate
directly produces that answer. This caller asks for only `k` winners from
`r` candidates. The relative order of the other `r-k` candidates will never
be observed.

`TopKByHeap` keeps at most `k` current winners. The root is deliberately the
*worst* of those winners. For every later candidate, the program therefore
asks one useful question: is this candidate better than the weakest candidate
currently included? If not, it can be discarded. If so, it replaces the root
and `heap.Fix` restores the heap's parent-child order.

The heap does not return the winners in final display order. It identifies the
winners while retaining only `k` candidates; the function then sorts those
`k` survivors. Its work is therefore:

```text
scan and retain: O(r log(k+1))
order winners:  O(k log k)
retained space: O(k)
```

Sorting every candidate takes `O(r log r)` time and retains a copy of all
`r` candidates. When `k` is close to `r`, the heap has little to save and the
plain sort may be the clearer choice.

## 2. Confirm the result and tie rule

Run commands from the `cs-prod-bridge` module directory:

```bash
go test ./03-queues-heaps-scheduling/lab -v
```

Before opening `topk.go`, write the expected top three for the fixture in
`TestTopKImplementationsAgree`. Two scores tie. Explain which field decides
their order and why a comparison based only on score would leave the result
contract incomplete.

The tests also reject `NaN`. Comparisons with `NaN` do not say that it is
higher than, lower than, or equal to an ordinary score, so accepting it would
leave the result order undefined.

## 3. Draw the information kept by the heap

Use this input with `k = 3`:

```text
(search, 0.72), (billing, 0.91), (edge, 0.83),
(catalog, 0.91), (worker, 0.40), (api, 0.83)
```

After each candidate:

1. draw the retained heap as an array;
2. mark the root;
3. say whether the new candidate can enter the answer; and
4. if the root changes, identify the one path whose parent-child order may
   need to be restored.

Do not expect the heap array to be fully sorted. Its promise is narrower: no
child is worse than its parent, so the worst retained candidate is at the
root.

## 4. Read the counters before timing

`SelectionStats` separates four kinds of evidence:

- `ScanComparisons` counts comparisons between a later candidate and the
  current root;
- `HeapComparisons` and `HeapSwaps` count work used to preserve heap order;
- `FinalSortComparisons` counts work used to put the returned winners in
  display order; and
- `MaxRetained` records how many candidates were in the method's working
  selection collection. It excludes the caller's input and the result copy;
  it is a logical count, not a byte-accurate memory measurement.

Add a table-driven test for `r = 100, 1_000, 10_000` with `k = 10`. Record
each counter. Then keep `r` fixed and increase `k`. Explain which counts follow
`r`, which follow `k`, and why elapsed time is not required to establish the
retained-space difference.

## 5. Compare the two complete operations

The benchmark includes result ordering for both methods; it does not compare a
fully ordered result with an unordered heap.

```bash
go test ./03-queues-heaps-scheduling/lab \
  -run '^$' -bench BenchmarkTopK -benchmem -count=5
```

Use the benchmark to test, not replace, the operation-count explanation. Try a
`k` close to the candidate count. A bounded heap is a workload choice, not an
automatic improvement over sorting.

Also read both allocation columns. The heap keeps fewer candidate records, but
Go's `container/heap` passes pushed values through `any`, and the growing heap
has its own allocations. It may report more individual allocations while
allocating fewer total bytes. The `O(k)` space bound describes how retained
candidate storage grows; it does not promise a particular allocation count.

## 6. Follow the production implementation

[Prometheus v3.13.1 `topk`](https://github.com/prometheus/prometheus/blob/v3.13.1/promql/engine.go#L3951-L4085)
uses the same central operation for each aggregation group: retain at most
`k` samples, compare later samples with the smallest retained value, replace
that root when the sample belongs in the result, and call `heap.Fix`. It sorts
the retained instant-query samples before returning them.

The local service names, validation, tie rule, counters, and API are synthetic.
They make the operation inspectable; they are not a model of all PromQL
semantics.

For the underlying data structure and proof, use
[MIT 6.006 Lecture 8: Binary Heaps](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/40d4851e550507ca14dc778b9b2266cc_MIT6_006S20_lec8.pdf).
For the exact Go interface used here, read the standard library's
[`container/heap` documentation](https://pkg.go.dev/container/heap).

## 7. State the structure choice plainly

Finish with two short explanations:

1. Why is the worst retained winner at the root when the product wants the
   highest scores?
2. What information does the bounded heap avoid computing, and what final
   sorting work does the response contract still require?
