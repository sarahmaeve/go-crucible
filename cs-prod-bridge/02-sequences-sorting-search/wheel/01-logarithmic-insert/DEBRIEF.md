# W01 Debrief

Read this only after completing the handoff.

## Cause

Binary search finds an insertion position in `O(log n)` comparisons. Making
room at that position is a separate operation. The records share one contiguous
array, so `slices.Insert` must move every later record one position to the
right.

In this scenario, every incoming key belongs before the existing snapshot. Each
of the `m` insertions moves approximately `n` existing records:

```text
n + n + ... + n = Theta(m*n)
```

The growing batch can add another `Theta(m^2)` moves in the worst case. If `m`
grows with `n`, the total copying is quadratic. Fast position searches do not
change the cost of moving records.

The insertion-position experiment shows this behavior. Batches of the same size
take different amounts of time when their keys belong in different positions.
The movement counter explains the difference. The CPU profile confirms that
copying uses most of the time.

## Smallest repair

First, reduce duplicate keys in the incoming batch to the last record for each
key. Sort the remaining incoming records once. Then merge them with the current
snapshot into a new slice. When both inputs contain the same key, use the
incoming record.

This takes:

- `Theta(m log m)` to sort the incoming batch;
- `Theta(n+m)` to merge; and
- `Theta(n+m)` output space.

Apply the duplicate rule before sorting, or keep the original batch position as
a tie-breaker. Otherwise, sorting can change which duplicate counts as last.

Build the result in a new slice because readers can still use the current
snapshot. A map alone does not meet the contract because callers also require
iteration in key order.

## Check the repair

Run:

```bash
go test ./02-sequences-sorting-search/wheel/01-logarithmic-insert -v
go test -tags=csbridgewheel3 \
  ./02-sequences-sorting-search/wheel/01-logarithmic-insert \
  -count=1 -v
```

The ordinary tests check sorted output, duplicate handling, unchanged inputs,
and counter definitions. The tagged test counts writes instead of using a
machine-specific time limit. The original implementation writes millions of
records for the front-heavy batch. The repaired implementation writes each
final record once.

## What to measure in production

Record the current snapshot size, incoming batch size, inserted and replaced
keys, and records copied during refresh. Also record where incoming keys belong
relative to the current key range. Two regions can have equal snapshot sizes
but do very different amounts of copying.
