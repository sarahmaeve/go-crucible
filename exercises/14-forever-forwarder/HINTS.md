# Hints for Exercise 14: The Forever Forwarder

## Hint 1: Direction

There are two places this forwarder can get stuck: repeatedly receiving from a
closed input, and sending to an output whose consumer has stopped. Both exit
paths are part of the function's context-aware contract.

## Hint 2: Narrower

Open `internal/ingest/reader.go` and look at `ForwardMetrics`. When `!ok`, the
code hits `continue`, immediately reads the closed channel again, and repeats
forever. The later `out <- m` is also a bare blocking send with no cancellation
case.

## Hint 3: Almost There

When `ok` is `false`, the channel is closed and there is nothing more to read. Return from the function:

```go
case <-ctx.Done():
    return ctx.Err()
case m, ok := <-in:
    if !ok {
        return nil
    }
    select {
    case out <- m:
    case <-ctx.Done():
        return ctx.Err()
    }
```

The inner `select` is necessary: once the outer input case has been selected,
the outer context case can no longer interrupt a blocked output send.
