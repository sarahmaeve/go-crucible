# Exercise 19 — Maintainer Notes

## Solved in main as of 2026-04-16

`cmd/pipeline/main.go` on the default branch carries the canonical fix. To
reintroduce the buggy form for learners:

```bash
git apply -R solutions/19-graceless-shutdown.patch
```

Run `go test ./cmd/pipeline/ -run TestExercise19 -v` to confirm the exercise
now fails.

## Why the canonical fix uses `signal.NotifyContext`

The original (pre-2026-04-16) solution taught three bug fixes in the old
manual-channel form:

```go
sigCh := make(chan os.Signal, 1)
signalNotify(sigCh, syscall.SIGINT, syscall.SIGTERM)
go func() {
    <-sigCh
    cancel()
}()
```

The modernized fix collapses that into one call:

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
```

`signal.NotifyContext` has been in the standard library since Go 1.16. We
switched to it for three reasons:

1. **It's the idiom learners will see in real codebases.** Every mature Go
   daemon written in the last few years uses `NotifyContext`. Teaching the
   old form as canonical trains learners to write code that looks dated in
   code review.

2. **One of the original three sub-bugs becomes structurally inexpressible.**
   The "signal.Notify was never called" bug only exists because the old form
   separates channel creation from registration. With `NotifyContext` there
   is no separate registration step — if the call compiles, the signals are
   registered. We dropped the `signal_notify_called` subtest accordingly.

3. **`defer stop()` is a cleaner lifecycle than a hand-rolled handler
   goroutine.** The old form leaked a goroutine waiting on `sigCh` if the
   pipeline exited via any path other than a signal. `stop()` deregisters
   cleanly.

## What the exercise still teaches after modernization

Two of the three original bugs remain in the patch:

- **Bug 19-2 (double-close panic):** the package-level `doneCh` is closed
  without being reset between calls. Solution resets it at the start of
  `RunPipeline` and uses `defer close(doneCh)` directly instead of a
  wrapping deferred func. Lesson: closing an already-closed channel panics.

- **Bug 19-3 (goroutine ignores caller context):** a background goroutine
  inside `RunPipeline` calls `src.Read(context.Background())` instead of
  `src.Read(ctx)`. Lesson: every goroutine must honor the caller's context
  or you leak on shutdown. (This is the same lesson as Exercise 10, in a
  different surface.)

If you expand this exercise in the future, consider adding a fourth bug
that `signal.NotifyContext` *can* still express incorrectly: forgetting
`defer stop()` leaks the signal-handler goroutine until process exit. That
would be a good advanced variant.

## Test coverage

The exercise retains two subtests:

- `bug19-2_double_close_panic` — calls `RunPipeline` twice in sequence,
  panics on the second call if `doneCh` isn't reset.
- `goroutine_exits_on_context_cancellation` — uses `BlockingSource` and
  `runtime.NumGoroutine()` to detect leaks after cancellation.

The old third subtest (`signal_notify_called`) was removed because its
assertion became tautological under `NotifyContext` (see point 2 above).
