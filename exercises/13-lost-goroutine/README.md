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
- This bug is intermittent and load-dependent, making it hard to catch without the race detector or repeated runs (`-count=N`)
- The fix is one line: move `wg.Add(1)` to just before `go func(...)`

## Fixing It

Apply your fix, then run:

```bash
go test -race ./internal/audit/ -run TestExercise13 -v -count=10
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
