# Exercise 20: The Brittle Match

**Application:** pipeline | **Difficulty:** Intermediate

## Symptoms

`Deduplicator.Ingest` is meant to absorb "this metric is a replay" errors
from its underlying `CacheStore` and return nil — a replayed metric is not
a failure, it is a no-op. The code does try to handle this, but it only
works for one of the two `CacheStore` implementations exercised by the
test. Against the other, a replayed metric produces an error that the
`Deduplicator` fails to recognise, and the caller sees a stray
`ErrDuplicate` it should never have seen.

## Reproduce

```bash
go test ./internal/ingest/ -run TestExercise20 -v
```

One subtest (`legacy`) passes. The other (`modern`) fails. Both subtests
use the same metric, the same `Deduplicator`, and both stores' `Put`
methods return a duplicate signal that wraps the **same** sentinel error.

## File to Investigate

`internal/ingest/dedup.go` — look at how `Ingest` decides whether an error
from the store represents a duplicate key.

`internal/ingest/dedup_test.go` — compare the two store implementations.
They differ in exactly one way: the prose of the error message surrounding
the sentinel.

## What You Will Learn

- Matching on `err.Error()` with `strings.Contains` couples your
  classification decision to another component's prose. When that
  component changes wording — a driver upgrade, a library rewrite, a
  second implementation — your classifier breaks silently.
- `errors.Is` walks the error chain and compares by identity. As long as
  the sentinel is wrapped with `%w` somewhere down the chain, the
  surrounding message does not matter.
- A string-match classifier tends to pass a happy-path test and fail
  only when the second store, the driver upgrade, or the library rewrite
  arrives — which is exactly when you least want to debug it.

## Related Exercises

- [Exercise 03: The Lost Alert](../03-lost-alert/README.md) — the other
  side of the same coin. Ex 03 teaches that `%v` breaks the chain so
  `errors.Is` cannot find the sentinel. This exercise teaches that even
  when the chain is intact, inspecting the error *text* instead of the
  *chain* fails as soon as the wrapping layer rewords its message.

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/ingest/ -run TestExercise20 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
