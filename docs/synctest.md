# Testing Concurrent Code with `testing/synctest`

> Canonical sources:
> [blog: Testing concurrent code](https://go.dev/blog/synctest) ·
> [blog: Testing Time](https://go.dev/blog/testing-time) ·
> [pkg.go.dev/testing/synctest](https://pkg.go.dev/testing/synctest) ·
> [Go 1.25 release notes](https://go.dev/doc/go1.25)

`testing/synctest` is a standard-library package for writing **fast and
deterministic** tests of concurrent, time-dependent code. This page explains the
model, then shows how it sharpens two Crucible exercises and where it does *not*
help.

---

## Availability

| Go version | Status |
|---|---|
| 1.24 | Experimental, hidden behind `GOEXPERIMENT=synctest`; API was `synctest.Run` |
| **1.25** | **Graduated to the standard library.** No flag. API is `synctest.Test` + `synctest.Wait` |
| 1.26 | The old `GOEXPERIMENT` API is removed; only `Test`/`Wait` remain |

This repo builds with `go 1.25` (see `go.mod`), so the package is available with
no build flags or experiments. The extension tests use an ordinary `synctest`
**build tag** purely to keep themselves out of the canonical suite — that is not
related to the old `GOEXPERIMENT`.

---

## The model

`synctest.Test` runs your test function inside a **bubble**: an isolated group of
goroutines with two special properties.

**1. A fake clock.** Inside the bubble, `time.Now`, `time.Sleep`, `time.After`,
`time.NewTimer`, `time.NewTicker`, and `context` deadlines all read a fake clock
that starts at a fixed instant. The clock does not advance on its own — it jumps
forward only when every goroutine in the bubble is blocked, and then it advances
to the next scheduled timer. A `time.Sleep(10 * time.Second)` returns in zero real
time, but still takes ten *fake* seconds, which `time.Since` will report.

**2. Durable-block awareness.** A goroutine is *durably blocked* when it can only
be unblocked by another goroutine in the same bubble or by the fake clock — for
example, parked on a channel send/receive, a mutex, a `select`, or `time.Sleep`.

Two functions exploit this:

- **`synctest.Test(t, f)`** runs `f` in a fresh bubble and waits for all bubble
  goroutines to exit before returning. If the bubble reaches a state where every
  goroutine is durably blocked and the clock can't advance, that's a deadlock and
  the test **fails** — with a goroutine dump that names the blocked line.
- **`synctest.Wait()`** blocks the calling goroutine until every *other* goroutine
  in the bubble is durably blocked (or has exited). It is the deterministic
  replacement for "sleep a bit and hope the background work caught up."

The payoff: timing becomes exact and reproducible, leaks become hard failures
with locations, and wall-clock time drops to near zero.

---

## How it improves Crucible exercises

### Exercise 06 — goroutine leak → deadlock with a location

The canonical `TestExercise06_StuckPipeline` infers a leak from
`runtime.NumGoroutine()` after a `time.Sleep(200ms)`, and reports it as a drifted
count. The synctest version (`internal/ingest/reader_synctest_test.go`) cancels
the context, calls `synctest.Wait()`, and lets the bubble end. On the buggy code
the reader goroutine is stuck on `out <- m`, so synctest fails the test and points
straight at `reader.go:18` — no sleep, no counting, no flake. Walkthrough:
[exercises/06-stuck-pipeline/EXTENSION.md](../exercises/06-stuck-pipeline/EXTENSION.md).

### Exercise 10 — deadline proxy → exact assertion in zero time

The canonical `TestExercise10_HangingHealthCheck` races a real 500 ms deadline
against a real 1 s timeout and can only assert "returned within a second" — a
proxy that a fast-but-wrong `Check` would pass, costing ~0.5–1 s per run. The
synctest version (`internal/health/checker_synctest_test.go`) runs in fake time
and asserts the real property: `Check` returned at fake `t=500ms` with
`context.DeadlineExceeded`. It finishes in ~0.00 s and catches the bug exactly.
Walkthrough:
[exercises/10-hanging-health-check/EXTENSION.md](../exercises/10-hanging-health-check/EXTENSION.md).

---

## Fit map

synctest is a precise tool, not a universal one. The same property that makes it
powerful — it reasons about *durable blocking* and a *fake clock* — also bounds
where it helps.

**Good fits**

- **Deadlines, timeouts, retries, backoff, tickers, rate limiters** — anything
  whose correctness is a timing contract (exercise 10).
- **Goroutine-leak / lifecycle tests** where the leak manifests as a goroutine
  *blocked* with no exit path (exercise 06).

**Partial fit**

- **Exercise 19 (graceful shutdown)** — the leg where a background goroutine
  ignores `ctx` and outlives shutdown is the same blocked-leak shape as 06 and
  fits well. But synctest cannot deliver OS signals into a bubble (the
  `signal.Notify` leg) and does nothing for the double-close panic. One of three
  legs benefits.

**Not a fit**

- **Busy-spins / live-locks (exercise 14).** A goroutine spinning on a closed
  channel stays *runnable*, never durably blocks, so synctest can't flag it — it
  would hang instead. The existing explicit 2 s timeout is the better signal.
- **Allocation / memory growth (exercise 18).** synctest has a fake clock but
  exposes no way to count live timers, and it doesn't measure allocations. Use
  `testing.AllocsPerRun` or `runtime.MemStats` (as the canonical test does).
- **Data races (exercises 08, 12).** That's the `-race` detector's job. synctest
  serializes execution within a bubble and can *mask* a race rather than expose
  it.

The rule of thumb: if the bug is "something blocked, or something happened at the
wrong time," synctest helps. If the bug is "something raced, spun, or allocated,"
reach for `-race`, a real timeout, or an allocation assertion instead.

---

## Running the extensions

The extension tests are gated behind the `synctest` build tag so they stay out of
`go test ./...`, `make status`, and `make verify`. Run them deliberately:

```bash
go test -tags synctest ./internal/ingest/  -run TestExercise06_Synctest -v
go test -tags synctest ./internal/health/  -run TestExercise10_Synctest -v
```
