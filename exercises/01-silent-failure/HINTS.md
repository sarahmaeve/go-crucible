# Hints for Exercise 01: The Silent Failure

## Hint 1: Direction

The bug is in error handling. The function receives an error from an API call, does something with it, and then continues as if nothing happened. What should a well-behaved function do when it cannot retrieve the data it needs to do its job?

## Hint 2: Narrower

Open `internal/audit/pods.go` and read the `if err != nil` block after `c.ListPods`. There are two statements inside it. One of them is correct. The other one is missing — it should stop execution and propagate the error to the caller.

## Hint 3: Almost There

The block currently reads:

```go
if err != nil {
    slog.Error("AuditPodLimits: failed to list pods", "err", err)
    // fall through with empty pods slice
}
```

The fix is to add `return nil, err` after the log line (or replace the log with a return that carries the error). The function signature is `([]types.Finding, error)` so returning `nil, err` satisfies both return values.
