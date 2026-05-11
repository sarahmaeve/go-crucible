# Exercise 22: The Hollow Recovery

**Application:** pipeline | **Difficulty:** Advanced

## Symptoms

`worker.Pool` is documented as panic-safe: its contract says that a
processor function which panics on a malformed input must be caught,
logged, and surfaced as an error on that input's `Result`. The rest of
the batch is supposed to continue processing.

In practice, a panicking processor takes down the caller. The deferred
recovery handler runs — but the panic propagates past it anyway. The
batch never completes; the goroutine running `Pool.Process` unwinds with
the panic, and whatever ran it has to recover, restart, or die.

## Reproduce

```bash
go test ./internal/worker/ -run TestExercise22 -v
```

The exercise test passes a three-element batch through the pool. The
middle element triggers a `panic` inside the supplied processor. The
test asserts that all three `Result`s come back, with `Err` populated on
the middle one and `nil` on the other two.

The test wraps the call to `Pool.Process` in an inner closure with its
own `recover`, so when the bug fires you get a clean assertion failure
rather than a stack-trace crash of the test process. Read the failure
message — it tells you which side of the contract is broken.

Two companion tests (`TestPoolHappyPath`, `TestPoolProcessorErrorPath`)
exercise the non-panic paths. They must remain green both before and
after the fix.

## File to Investigate

`internal/worker/pool.go` — look at `Pool.processOne`. The function
defers a recovery handler before calling the user-supplied
`ProcessFunc`. Walk through what happens, step by step, when the
processor panics:

1. The processor's `panic` begins unwinding the stack.
2. Deferred functions in `processOne` run.
3. Inside the deferred work, `recoverPanic` is called.
4. Inside `recoverPanic`, `recover()` is called.

Then ask: does `recover()` actually return the panic value here, or does
it return `nil`? The Go spec is very specific about *where* `recover()`
has to be called to take effect.

## What You Will Learn

- `recover()` is only effective when called **directly** by a function
  that was itself deferred. If a deferred function calls a helper which
  calls `recover()`, the helper is one frame too deep — `recover()`
  returns `nil` and the panic continues to propagate.
- Per the Go specification:
  > The return value of recover is nil if [...] recover was not called
  > directly by a deferred function.
- This is one of the few places in Go where extracting a function for
  cleanliness silently breaks correctness. `defer recoverer()` works;
  `defer func() { recoverer() }()` does not.
- The bug has no compile-time warning, no `go vet` flag, and no runtime
  symptom until a panic actually fires — at which point the recovery
  handler that "exists" turns out to do nothing. Hollow recovery is
  worse than no recovery, because the code reads as if it were safe.
- Recovery is also incomplete without a structured signal to the
  caller. A correct recoverer does three things in one place: catch the
  panic with `recover()`, record it (log with stack), and convert it
  into a return value the caller can act on. Don't split these across
  frames.

## Related Exercises

- [Exercise 09: The Immortal Connection](../09-immortal-connection/README.md)
  — also a `defer` exercise, but the bug is omission. Here the `defer`
  is present and the bug is structural.
- [Exercise 16: The Leaking Linter](../16-leaking-linter/README.md)
  — another "defer is there but doing the wrong thing" trap, this time
  because `defer` is function-scoped rather than loop-scoped. Together
  with this exercise it forms a pair on subtle defer misuse.
- [Exercise 19: The Graceless Shutdown](../19-graceless-shutdown/README.md)
  — one of its three bugs is a panic on closing a closed channel.
  Reasoning about who recovers from process-level panics overlaps with
  the worker pattern here.

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/worker/ -run TestExercise22 -v
go test ./internal/worker/ -v
```

All three tests in the package must pass. See [HINTS.md](./HINTS.md)
for progressive hints if you get stuck.
