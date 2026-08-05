# W01 Debrief

Read this only after completing the handoff.

## Diagnosis

Finding an insertion point and making room in a slice are separate operations.
Binary search finds the position in `O(log n)` comparisons. The records live
in one contiguous backing array, however, so `slices.Insert` must copy every
record after that position one place to the right.

In this scenario, each incoming key sorts before the existing snapshot. Each
of the `m` insertions therefore moves roughly `n` existing records:

```text
n + n + ... + n = Theta(m*n)
```

The growing batch can add another `Theta(m^2)` moves in the worst case. If `m`
grows in proportion to `n`, the refresh performs quadratic copying; the cheap
position searches do not change that total.

The insertion-position experiment makes this visible: batches of the same size
take very different times depending on where their keys belong. The movement
counter explains that difference, while the CPU profile confirms that the
program spends most of the delay copying records.

## A repair with predictable cost

First collapse duplicate keys in the incoming batch, retaining the last
occurrence of each key. Sort the remaining incoming records once, then merge
them with the current snapshot into a new slice. When both inputs contain the
same key, copy the incoming record.

This takes:

- `Theta(m log m)` to sort the incoming batch;
- `Theta(n+m)` to merge; and
- `Theta(n+m)` output space.

The duplicate rule must be applied before sorting, or the original batch
position must be retained as a tie-breaker. Otherwise, sorting can change
which duplicate is considered last.

Construct the result in a new slice because readers may still be using the
current snapshot. A map alone is not a complete replacement: callers also
require iteration in key order.

## Verification

Run:

```bash
go test ./02-sequences-sorting-search/wheel/01-logarithmic-insert -v
go test -tags=csbridgewheel3 \
  ./02-sequences-sorting-search/wheel/01-logarithmic-insert \
  -count=1 -v
```

The ordinary tests check sorted output, duplicate handling, unchanged inputs,
and the counter definitions. The tagged test counts writes instead of imposing
a machine-specific time limit. The original implementation performs millions
of writes on the front-heavy batch; the repaired implementation writes each
final record once.

## Production follow-up

In production, record the current snapshot size, incoming batch size, number
of inserted and replaced keys, and number of records copied during refresh.
Also record where incoming keys fall relative to the current key range. Two
regions can have snapshots of the same size while doing very different amounts
of copying.
