# Exercise 06: The Stuck Pipeline

**Application:** pipeline | **Difficulty:** Intermediate

## Symptoms

The caller runs `ReadMetrics` in a goroutine, cancels the context, and waits for
the function to return. It never does. `ReadMetrics` is blocked trying to send
a metric to `out`, but the consumer has stopped reading because the operation
was cancelled. The caller-owned goroutine leaks and no terminal error is
reported.

## Reproduce

```bash
go test ./internal/ingest/ -run TestExercise06 -v
```

## File to Investigate

`internal/ingest/reader.go` — look at the `ReadMetrics` function

Find the `out <- m` send statement and consider what happens when no one is
reading from `out`.

## What You Will Learn

- Sending to an unbuffered channel blocks forever if there is no receiver
- A goroutine that blocks on a channel send cannot be garbage-collected — it leaks
- The fix is a `select` with a `ctx.Done()` case so the function can return when the context is cancelled
- A synchronous streaming function preserves source and cancellation errors; callers choose whether to run it in a goroutine
- Channel blocking semantics: operation lifecycle must be tied to a cancellation signal

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/ingest/ -run TestExercise06 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.

## Extension

Once you have fixed the bug, see [EXTENSION.md](./EXTENSION.md) for a
`testing/synctest` rewrite of this exercise's test that catches the leak
deterministically — no `time.Sleep`, no goroutine counting, and a failure
message that names the blocked line.
