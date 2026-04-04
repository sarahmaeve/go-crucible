# Hints for Exercise 13: The Lost Goroutine

## Hint 1: Direction

`ParallelAudit` starts goroutines and then calls `wg.Wait()`. Sometimes it returns before any goroutine has contributed findings. There is a timing issue between when `wg.Add(1)` is called and when `wg.Wait()` is called. Look at the order of operations.

## Hint 2: Narrower

Open `internal/audit/report.go` and look at `ParallelAudit`. Find `wg.Add(1)`. It is inside the goroutine body — the first line of the `go func(...)` closure. This means the goroutine must be scheduled and start executing before `wg.Add(1)` runs. If `wg.Wait()` runs first (before the goroutine starts), it sees a counter of zero and returns immediately.

## Hint 3: Almost There

Move `wg.Add(1)` to before the `go` statement:

```go
for _, auditor := range auditors {
    wg.Add(1)          // must be here — before the goroutine starts
    go func(fn AuditFunc) {
        defer wg.Done()
        // ... rest of goroutine body
    }(auditor)
}
```

This guarantees `wg.Wait()` sees the correct count regardless of scheduling order.
