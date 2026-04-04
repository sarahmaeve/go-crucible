# Hints for Exercise 19: The Graceless Shutdown

## Hint 1: Direction

There are three independent bugs. Start by identifying which symptom each test assertion checks: (1) the signal channel never fires, (2) `RunPipeline` panics on a second call, (3) a goroutine outlives the context. Tackle them one at a time.

## Hint 2: Narrower

Open `cmd/pipeline/main.go`.

**Bug 1 — signal channel:** Find `sigCh := make(chan os.Signal, 1)`. The next line should call `signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)`. It does not — `signalNotify` is stored in a variable but never invoked. The goroutine below waits on `sigCh` forever because it is never fed.

**Bug 2 — panic on re-run:** Find `var doneCh = make(chan struct{})` at package level. It is closed inside a `defer` in `RunPipeline`. A second call closes an already-closed channel, which panics. It needs to be recreated each time, or the close needs to be protected with a `sync.Once` or condition.

**Bug 3 — immortal goroutine:** Find the goroutine inside `RunPipeline` that calls `src.Read(context.Background())`. `context.Background()` ignores the pipeline's `ctx` — the goroutine runs until the source is exhausted regardless of cancellation.

## Hint 3: Almost There

**Fix 1:** Add the signal registration call in `main`:
```go
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
```

**Fix 2:** Move `doneCh` inside `RunPipeline` as a local variable (or reset it at the start of each call) so closing it does not affect subsequent invocations.

**Fix 3:** Pass `ctx` instead of `context.Background()` to `src.Read` inside the goroutine in `RunPipeline`:
```go
m, err := src.Read(ctx)
```

All three fixes together produce a daemon that handles signals, can be called multiple times in tests, and cleans up all goroutines on shutdown.
