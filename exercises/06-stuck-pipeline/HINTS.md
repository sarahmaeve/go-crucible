# Hints for Exercise 06: The Stuck Pipeline

## Hint 1: Direction

The caller's goroutine running `ReadMetrics` is still alive after the context is
cancelled. Look at the blocking operations inside the function.

## Hint 2: Narrower

Open `internal/ingest/reader.go` and find the send `out <- m`. It blocks until a
receiver is ready. When the context is cancelled, the consumer stops reading
from `out`, leaving the caller-owned goroutine with no way to return.

## Hint 3: Almost There

Replace the bare send with a `select` that also listens for context cancellation:

```go
select {
case out <- m:
case <-ctx.Done():
    return ctx.Err()
}
```

This way, if the consumer stops reading, the function reports cancellation and
the goroutine that the caller chose to start exits cleanly.
