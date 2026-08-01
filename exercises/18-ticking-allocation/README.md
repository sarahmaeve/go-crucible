# Exercise 18: The Ticking Allocation

**Application:** pipeline | **Difficulty:** Advanced

> **Pre-solved on `main`.** Reintroduce the exercise before reproducing it:
> `git apply -R solutions/18-ticking-allocation.patch`

## Symptoms

`TickerForwarder.Run` is used in a high-frequency polling path. An allocation
profile shows several new objects for every metric forwarded, with
`time.After` dominating the hot path. Live heap remains bounded on current Go
versions, but the avoidable timer allocation and garbage-collection work scale
linearly with throughput.

## Reproduce

```bash
go test ./internal/ingest/ -run TestExercise18 -v
```

## File to Investigate

`internal/ingest/ticker.go` — look at the `Run` method on `TickerForwarder`

Find the `time.After(interval)` call inside the `select` statement. Consider
whether recurring work needs a newly allocated one-shot timer on every
iteration.

## What You Will Learn

- `time.After(d)` creates a new one-shot timer on every call
- In this loop the timer fires before the next iteration, so this is allocation
  churn rather than a live-object leak
- Since Go 1.23, unreachable unstopped timers and tickers are garbage
  collectable; older “timer leak” advice must be interpreted in light of the
  module's Go version
- The fix: use `time.NewTicker(interval)` before the loop, and use the ticker's `C` channel in the `select`; remember to call `ticker.Stop()` via `defer`
- `time.NewTicker` panics for nonpositive durations, so validate a caller-supplied interval before constructing it
- A ticker follows a wall-clock cadence and may have a tick ready immediately
  after slow work; a reset timer expresses “wait this long after the previous
  iteration.” Choose according to the polling contract, not allocation count alone
- `testing.AllocsPerRun` runs the closure repeatedly and reports average
  allocations, making it suitable for detecting per-iteration growth without
  relying on noisy heap snapshots

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/ingest/ -run TestExercise18 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
