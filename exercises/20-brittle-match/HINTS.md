# Hints for Exercise 20: The Brittle Match

## Hint 1: Direction

The two subtests (`legacy` and `modern`) use the same metric and both call
`Deduplicator.Ingest` twice. On the second call, both underlying stores
report the write was a duplicate. Both stores signal that by returning an
error that wraps the **same** sentinel with `%w`. One subtest passes, one
fails. The only thing that differs between the stores is the wording of
the error message. Ask yourself: why does `Deduplicator.Ingest` care about
wording?

## Hint 2: Narrower

Open `internal/ingest/dedup.go`. The `Ingest` method asks whether an error
means "this was a duplicate" and, on a yes, returns nil. It is answering
that question by inspecting the *textual* form of the error — the bytes
that would be printed. Both stores wrap `types.ErrDuplicate`, but only one
of them writes a message that contains the specific phrase the check is
looking for.

Now look at `internal/ingest/dedup_test.go`. Compare the error messages
produced by `legacyStore.Put` and `modernStore.Put` when a key is already
present. What does each one literally say? Which one would match the
substring the buggy check is looking for?

## Hint 3: Almost There

The current check is:

```go
if strings.Contains(err.Error(), "already recorded") {
    return nil
}
```

This asks "does the error message contain the phrase the legacy store
happens to use?" The question you actually want to ask is "does the
error chain contain `types.ErrDuplicate`?" The Go standard library has
`errors.Is` for exactly that — it walks each `%w`-wrapped layer and
compares by identity, so the wording that any wrapping layer chose does
not matter.

Replace the substring check with an identity check:

```go
if errors.Is(err, types.ErrDuplicate) {
    return nil
}
```

Remove the unused `strings` import and add `errors` in its place.
