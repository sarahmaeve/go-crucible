# Exercise 19: The Graceless Shutdown

**Application:** pipeline | **Difficulty:** Advanced

## Symptoms

The pipeline daemon has three distinct bugs that interact:

1. Sending SIGINT or SIGTERM to the process has no effect — the daemon cannot be stopped via signals.
2. Calling `RunPipeline` a second time (e.g., in a test retry or restart) panics with "close of closed channel".
3. Even after the context is cancelled, an internal goroutine continues reading from sources — it outlives the shutdown because it ignores the context.

No single test surfaces all three at once, but the exercise covers all three root causes.

## Reproduce

```bash
go test ./cmd/pipeline/ -run TestExercise19 -v
```

## File to Investigate

`cmd/pipeline/main.go` — look at the `main` function and `RunPipeline`

Search for these three issues in order:
1. `signal.Notify` — is it actually called?
2. `doneCh` — what happens when it is closed twice?
3. The goroutine inside `RunPipeline` — which context does it pass to `src.Read`?

## What You Will Learn

- `signal.Notify` must be called to register a channel for OS signals; simply creating the channel is not enough
- Closing an already-closed channel panics; use a `sync.Once` or reset the channel between calls
- Every goroutine that does blocking I/O must receive and honour the caller's context — passing `context.Background()` creates an immortal goroutine
- Compound bugs are harder to diagnose because each individual symptom can look like a different root cause

## Fixing It

Apply your fixes (there are three), then run:

```bash
go test ./cmd/pipeline/ -run TestExercise19 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
