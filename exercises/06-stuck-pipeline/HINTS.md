# Hints for Exercise 06: The Stuck Pipeline

## Hint 1: Direction

The goroutine inside `ReadMetrics` is still alive after the context is cancelled. It is blocked somewhere. Blocked goroutines are either waiting on a channel operation, a mutex, or a system call. Look at what blocking operations exist inside the goroutine.

## Hint 2: Narrower

Open `internal/ingest/reader.go` and read the goroutine in `ReadMetrics`. There is a send: `out <- m`. This blocks until a receiver is ready. When the context is cancelled, the caller stops reading from `out`. The goroutine is now permanently blocked on the send with no way to exit.

## Hint 3: Almost There

Replace the bare send with a `select` that also listens for context cancellation:

```go
select {
case out <- m:
case <-ctx.Done():
    return
}
```

This way, if the consumer stops reading, the goroutine unblocks via `ctx.Done()` and exits cleanly instead of leaking.
