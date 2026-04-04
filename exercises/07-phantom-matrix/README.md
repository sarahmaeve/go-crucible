# Exercise 07: The Phantom Matrix

**Application:** gh-forge | **Difficulty:** Intermediate

## Symptoms

`ExpandMatrix` is called with a two-dimensional strategy (e.g., OS × Go version). It returns a slice of combinations, but many of the combinations are identical — all pointing to the last value from the inner loop. Combinations that should be distinct end up sharing the same data. A 2×2 matrix that should produce 4 unique combinations instead produces 4 entries that all look like the last one.

## Reproduce

```bash
go test ./internal/parser/ -run TestExercise07 -v
```

## File to Investigate

`internal/parser/matrix.go` — look at the `ExpandMatrix` function

Trace the `base` variable inside the inner loop. Note what type `types.MatrixCombination` is (a `map[string]string`) and what `base := existing` actually copies.

## What You Will Learn

- Maps are reference types in Go: assigning a map to a new variable copies the pointer, not the underlying data
- `base := existing` shares the same backing map across all inner-loop iterations; mutations via `base[dim] = v` corrupt every previously appended entry
- The fix is to deep-copy the map before mutating it — create a new map and copy each key/value pair
- This class of bug ("shallow copy of a reference type") is one of the most common silent data-corruption bugs in Go

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/parser/ -run TestExercise07 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
