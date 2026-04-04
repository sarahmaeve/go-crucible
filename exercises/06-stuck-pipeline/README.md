# Exercise 06: The Stuck Pipeline

**Application:** pipeline | **Difficulty:** Intermediate

## Symptoms

The test cancels the context and then waits for `ReadMetrics` to clean up. It never does. The goroutine started by `ReadMetrics` is still alive, blocked trying to send a metric to the `out` channel — but the consumer on the other end has already stopped reading because the context was cancelled. The goroutine leaks and the test times out.

## Reproduce

```bash
go test ./internal/ingest/ -run TestExercise06 -v
```

## File to Investigate

`internal/ingest/reader.go` — look at the `ReadMetrics` function

Find the `out <- m` send statement inside the goroutine and consider what happens when no one is reading from `out`.

## What You Will Learn

- Sending to an unbuffered channel blocks forever if there is no receiver
- A goroutine that blocks on a channel send cannot be garbage-collected — it leaks
- The fix is a `select` with a `ctx.Done()` case so the goroutine can exit when the context is cancelled
- Channel blocking semantics: goroutine lifecycle must be tied to a cancellation signal

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/ingest/ -run TestExercise06 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
