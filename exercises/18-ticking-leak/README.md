# Exercise 18: The Ticking Leak

**Application:** pipeline | **Difficulty:** Advanced

## Symptoms

`TickerForwarder.Run` is used in a long-running daemon with a short polling interval. Over time, heap memory grows steadily without bound. A pprof heap profile shows an accumulation of `*time.runtimeTimer` objects. The program is not obviously leaking goroutines, and it appears to function correctly — it just slowly consumes more and more memory.

## Reproduce

```bash
go test ./internal/ingest/ -run TestExercise18 -v
```

## File to Investigate

`internal/ingest/ticker.go` — look at the `Run` method on `TickerForwarder`

Find the `time.After(interval)` call inside the `select` statement. Consider what `time.After` allocates on each call and when that allocation is released.

## What You Will Learn

- `time.After(d)` creates a new `*time.Timer` on every call; the timer's channel is kept alive by the Go runtime until the timer fires naturally
- Inside a `select` loop, a new timer is created on every iteration; the previous timer's channel is abandoned but the timer object is not garbage-collected until it fires
- With a short interval and a long-running process, unreferenced timers accumulate
- The fix: use `time.NewTicker(interval)` before the loop, and use the ticker's `C` channel in the `select`; remember to call `ticker.Stop()` via `defer`

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/ingest/ -run TestExercise18 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
