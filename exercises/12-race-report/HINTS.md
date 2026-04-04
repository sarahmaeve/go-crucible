# Hints for Exercise 12: The Race Report

## Hint 1: Direction

The race detector output points to the `findings` slice inside `ConcurrentAudit`. Multiple goroutines are appending to it simultaneously. Slice append reads and writes the slice header (pointer, length, capacity) — concurrent access without a lock is a data race.

## Hint 2: Narrower

Open `internal/audit/report.go` and look at `ConcurrentAudit`. There is already an `errMu sync.Mutex` protecting `firstErr`. The `findings` slice has no equivalent protection. Compare the error-update pattern with what needs to happen for findings.

## Hint 3: Almost There

The function already declares `errMu`. Either reuse it for findings or add a dedicated `findingsMu sync.Mutex`. Then wrap the append:

```go
errMu.Lock()
findings = append(findings, result...)
errMu.Unlock()
```

Alternatively, add `var findingsMu sync.Mutex` to the var block and use that. Either way, the key point is that every access to the shared slice must be under the lock.
