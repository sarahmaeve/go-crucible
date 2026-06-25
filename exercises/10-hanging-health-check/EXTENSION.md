# Extension 10: Assert the Deadline Exactly with `testing/synctest`

> Prerequisite: finish the main exercise first. This extension is about *how you
> test* the bug, not how you fix it. See [docs/synctest.md](../../docs/synctest.md)
> for the full background on the `testing/synctest` package.

## The problem with the canonical test

Open `internal/health/checker_test.go` and read
`TestExercise10_HangingHealthCheck`. The dependency check waits on a real
`time.After(10 * time.Second)`; the caller's context gets a real 500 ms
deadline; the test spawns a goroutine and races `<-done` against a real
`time.After(1 * time.Second)`. Two compromises fall out of this:

- It spends **0.5–1 s of real wall-clock time** every run.
- It can only assert a *proxy*: "Check returned within a second." It never checks
  *why* it returned. A `Check` that ignored the deadline but happened to be fast
  would pass; the assertion is blind to the actual property the exercise teaches.

## The synctest version

`internal/health/checker_synctest_test.go` runs `Check` inside a synctest
**bubble**, where the clock is fake. The 500 ms deadline and the 10 s dependency
timer both resolve in *zero real time*, and `time.Since` measures fake-elapsed
duration — so the test can assert the exact thing that matters: the call returned
*because of the caller's deadline*, with `context.DeadlineExceeded`.

Exercise 10 is pre-solved on `main`, so the extension passes as-is:

```bash
# Fixed code (main): returns at fake t=500ms with DeadlineExceeded — in ~0.00s.
go test -tags synctest ./internal/health/ -run TestExercise10_Synctest -v
```

To watch it fail, reintroduce the bug and run again:

```bash
git apply -R solutions/10-hanging-health-check.patch
go test -tags synctest ./internal/health/ -run TestExercise10_Synctest -v
# FAIL: Check ignored the caller deadline: returned after 10s (fake) with err=<nil>
git apply solutions/10-hanging-health-check.patch   # restore
```

On the buggy code the dependency never sees the 500 ms deadline (it got
`context.Background()`), so it returns `nil` only when its own 10 s timer fires —
at fake `t=10s`. The assertion catches both facts (wrong duration, wrong error)
instantly.

## What changed, and why it matters

| | Canonical test | synctest extension |
|---|---|---|
| Real time spent | ~0.5–1 s per run | ~0 s (fake clock) |
| Assertion | "returned within 1 s" (proxy) | exact `DeadlineExceeded` at fake t=500ms |
| Plumbing | goroutine + `done` channel + outer timeout | a plain synchronous call |
| Catches a fast-but-wrong `Check`? | no | yes |

The lesson: deadline and timeout behaviour is a *timing contract*. With a fake
clock you can test the contract directly and exactly, instead of approximating it
with real sleeps and a generous margin.

## Where this technique shines

Any code whose correctness depends on time — deadlines, timeouts, retries with
backoff, tickers, rate limiters — is a candidate. The fake clock makes those
tests both instant and exact. See [docs/synctest.md](../../docs/synctest.md) for
the full picture and the cases where synctest is the *wrong* tool.
