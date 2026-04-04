# Hints for Exercise 07: The Phantom Matrix

## Hint 1: Direction

The function builds combinations by iterating over existing results and extending them with a new dimension value. The final slice looks the right length, but many entries are duplicates of the last value. Something is being shared between combinations that should be independent.

## Hint 2: Narrower

Open `internal/parser/matrix.go` and look at the inner loop in `ExpandMatrix`. The line `base := existing` copies the loop variable — but `types.MatrixCombination` is a map (`map[string]string`). Assigning a map copies the map header (a pointer), not the underlying key-value data. All combinations derived from the same `existing` entry share the same backing map.

## Hint 3: Almost There

Before mutating `base`, deep-copy the map:

```go
base := make(types.MatrixCombination, len(existing))
for k, v := range existing {
    base[k] = v
}
// Now base[dim] = v only affects this copy.
```

Replace the `base := existing` line with this copy, then proceed with `base[dim] = v` and the append. Each combination will then have its own independent map.
