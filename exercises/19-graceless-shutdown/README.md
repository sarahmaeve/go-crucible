# Exercise 19: The Graceless Shutdown

**Application:** pipeline | **Difficulty:** Advanced

> **Pre-solved on `main`.** Reintroduce the exercise before reproducing it:
> `git apply -R solutions/19-graceless-shutdown.patch`

## Symptoms

The exercise form has three lifecycle defects that appear during production
shutdown rather than ordinary metric processing:

1. An interactive interrupt begins graceful shutdown, but the SIGTERM sent by
   a process supervisor takes Go's default path and terminates the daemon
   immediately. Cleanup and the final shutdown log never run.
2. Cancelling `RunPipeline` does not reach a source blocked in `Read`; its
   worker remains alive after the pipeline has stopped.
3. `RunPipeline` returns as soon as its context is cancelled even when an owned
   source worker is still completing its current operation.

All three failures can pass happy-path tests: they require a supervisor signal,
a blocked source, or cancellation during active work.

## Reproduce

```bash
go test ./cmd/pipeline/ -run TestExercise19 -v
```

## File to Investigate

`cmd/pipeline/main.go` — inspect `shutdownContext` and `RunPipeline`.

Trace the shutdown path as three ownership questions:

1. Which operating-system signals cancel the root context?
2. Which context does each source worker receive?
3. What proves every worker has finished before `RunPipeline` returns?

## What You Will Learn

- `signal.NotifyContext` changes behavior only for the signals passed to it.
  If SIGTERM is omitted, a supervisor's termination signal follows Go's
  default behavior and exits instead of initiating application cleanup.
- Cancellation is cooperative: every blocking operation must receive and
  honor the caller's context.
- Starting a goroutine creates an ownership obligation. Graceful shutdown must
  join owned workers, not merely broadcast cancellation and return.
- Shutdown tests should assert cleanup effects and worker completion, not only
  that a process eventually exits.
- Independent lifecycle defects can mask one another: abrupt termination can
  hide both failed cancellation propagation and missing joins.

## Fixing It

Apply all three repairs, then run:

```bash
go test ./cmd/pipeline/ -run TestExercise19 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
