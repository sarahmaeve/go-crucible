# Exercise 19 — Maintainer Notes

## Solved in main

`cmd/pipeline/main.go` carries the canonical modern implementation. To
reintroduce the exercise form:

```bash
git apply -R solutions/19-graceless-shutdown.patch
```

The inverse patch deliberately stays within the current
`signal.NotifyContext` idiom. It no longer resurrects the older manual
`signal.Notify` channel implementation.

## Why the exercise was rewritten

The historical exercise said that failing to register SIGINT or SIGTERM made
the signals have no effect. That premise was inaccurate: by default, Go exits
on SIGINT and SIGTERM. The missing registration bypassed graceful cleanup; it
did not make the process unstoppable.

The historical form also used an otherwise-unused package-global `doneCh`
which panicked on a second `RunPipeline` call. Although double-close failures
are real, that particular state existed mainly to create a test-shaped bug.

The rewritten exercise keeps the advanced compound-shutdown goal while using
three failures that plausibly survive happy-path testing and reach production:

1. `shutdownContext` registers `os.Interrupt` but omits SIGTERM. Ctrl-C works
   locally, while the supervisor's SIGTERM takes the default immediate-exit
   path and skips cleanup.
2. An owned worker calls `src.Read(context.Background())`, severing
   cancellation propagation.
3. The worker `WaitGroup` is maintained but never waited, so `RunPipeline`
   does not uphold the ownership boundary implied by starting the goroutines.

## Test design

The signal subtest re-executes the test binary as a subprocess. The child
creates the real shutdown context and sends itself SIGTERM. With the exercise
form, the child is terminated by the default signal action; with the solution,
the context is cancelled and the child returns normally. This tests graceful
handling without risking the parent test process.

The context subtest uses a source that exits only when its received context is
cancelled. The join subtest uses a source whose current read is released by the
test, allowing it to prove that `RunPipeline` remains active until the owned
worker finishes.

Avoid returning to goroutine-count assertions. Explicit lifecycle signals are
deterministic and state the contract more precisely.

## Relationship to the review track

R07 remains a deliberate transfer exercise. Its `setupSignals` helper calls
`defer stop()` in the wrong scope, restoring default SIGTERM behavior before
the returned context can observe a signal. That is a different mechanism with
the same production consequence as this exercise's omitted SIGTERM.

R07's double-close finding is now a neighboring shutdown-ownership problem,
not a direct repetition of exercise 19's retired package-global channel.
