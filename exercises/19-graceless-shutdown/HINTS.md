# Hints for Exercise 19: The Graceless Shutdown

## Hint 1: Direction

Treat graceful shutdown as three separate obligations: receive the production
signal, deliver cancellation to every owned worker, and wait for those workers
to finish. The tests isolate those obligations so one failure does not conceal
the others.

## Hint 2: Narrower

Open `cmd/pipeline/main.go`.

1. Compare the signals passed to `shutdownContext` with the signal a Unix
   process supervisor or container runtime normally sends.
2. Inside the source goroutine, compare the context accepted by `RunPipeline`
   with the context passed to `src.Read`.
3. The `sync.WaitGroup` records worker starts and completions. Find where the
   shutdown path waits on that accounting before returning.

## Hint 3: Almost There

1. Register both interactive interruption and `syscall.SIGTERM` with
   `signal.NotifyContext`.
2. Pass `ctx`, not `context.Background()`, to `src.Read`.
3. After cancellation, call `workers.Wait()` before `RunPipeline` returns.

The ordering matters: cancellation unblocks context-aware sources, and the
wait proves their goroutines have actually exited.
