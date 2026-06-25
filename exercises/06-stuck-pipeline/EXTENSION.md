# Extension 06: Catch the Leak Deterministically with `testing/synctest`

> Prerequisite: finish the main exercise first. This extension is about *how you
> test* the bug, not how you fix it. See [docs/synctest.md](../../docs/synctest.md)
> for the full background on the `testing/synctest` package.

## The problem with the canonical test

Open `internal/ingest/reader_test.go` and read `TestExercise06_StuckPipeline`.
To decide whether the goroutine leaked, it:

1. samples `runtime.NumGoroutine()` before starting,
2. cancels the context,
3. **sleeps 200 ms** to "give the goroutine time to exit," then
4. samples `runtime.NumGoroutine()` again and compares.

Every step is a compromise. The sleep is wall-clock time the suite pays on every
run. The goroutine count is global and noisy — the test comment even hedges
"allow +1 for transient runtime goroutines." And when it fails, the message is a
count that drifted (`baseline 2, after cancel 3`), which tells you *that*
something leaked but not *what* or *where*.

## The synctest version

`internal/ingest/reader_synctest_test.go` runs the same scenario inside a
synctest **bubble**. After `cancel()`, `synctest.Wait()` blocks until every other
goroutine in the bubble is durably blocked or has exited. There is no sleep and
no goroutine counting. The file is gated behind a build tag so it never runs in
the canonical suite — run it on purpose:

```bash
# On the buggy tree (main): the reader goroutine is stuck on the send forever,
# so the bubble deadlocks and the test fails — pointing at the exact line.
go test -tags synctest ./internal/ingest/ -run TestExercise06_Synctest -v
```

You should see synctest report the still-blocked goroutine by location:

```
goroutine N [chan send (durable), synctest bubble 1]:
    .../internal/ingest/reader.go:18      <-- the bare `out <- m`
```

Now apply your fix to `reader.go` and run it again — the goroutine observes
`ctx.Done()`, the bubble drains, and the test passes.

## What changed, and why it matters

| | Canonical test | synctest extension |
|---|---|---|
| Wait mechanism | `time.Sleep(200ms)` | `synctest.Wait()` (no real time) |
| Leak signal | global goroutine count, ±1 fudge | durable-block / deadlock detection |
| Failure message | "count drifted" | exact file:line of the blocked send |
| Flake surface | transient runtime goroutines | none — execution is deterministic |

The lesson: a goroutine leak is a *blocking* fact, and synctest can observe
blocking directly instead of inferring it from timing and counts.

## When this technique applies (and when it doesn't)

synctest detects goroutines that are **durably blocked** (channel ops, mutexes,
`time.Sleep`, etc.). It does **not** detect a goroutine spinning in a busy loop —
that goroutine stays runnable, never blocks, and synctest would hang rather than
report it. That's exactly why exercise 14 (a busy-spin on a closed channel) is a
poor fit for synctest. See [docs/synctest.md](../../docs/synctest.md#fit-map) for
the full fit map.
