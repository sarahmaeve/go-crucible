# Exercise 13: The Lost Goroutine

**Application:** kube-patrol | **Difficulty:** Intermediate

## Symptoms

`ParallelAudit` runs auditors concurrently and waits for them to finish. Occasionally — especially under the race detector or with `-count=10` — the function returns a report with fewer findings than expected, or an empty report. No error is returned. The auditors ran, but their results were not collected because `wg.Wait()` returned before the goroutines had a chance to call `wg.Add(1)`.

## Reproduce

```bash
go test -race ./internal/audit/ -run TestExercise13 -v -count=10
```

## File to Investigate

`internal/audit/report.go` — look at the `ParallelAudit` function

Find where `wg.Add(1)` is called relative to the `go func(...)` statement.

## What You Will Learn

- `sync.WaitGroup.Add` must be called before the `go` statement, not inside the goroutine body
- If `Add` is inside the goroutine, the scheduler may run `wg.Wait()` before any goroutine starts — `Wait` sees a counter of zero and returns immediately
- Go 1.25's `go vet` includes a `waitgroup` analyzer that reports this exact misuse; repeated test runs still demonstrate the runtime consequence
- The fix is one line: move `wg.Add(1)` to just before `go func(...)`
- In Go 1.25+, `WaitGroup.Go` combines task registration, goroutine launch, and `Done`; use it when its contract fits, including its requirement that the function must not panic

## Fixing It

Apply your fix, then run:

```bash
go test -race ./internal/audit/ -run TestExercise13 -v -count=10
go vet ./internal/audit/
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
