# Exercise 12: The Race Report

**Application:** kube-patrol | **Difficulty:** Intermediate

## Symptoms

`ConcurrentAudit` runs multiple auditors in parallel and collects their findings into a single report. Under the race detector it reports a data race on the shared `findings` slice. Without the race detector some findings are silently dropped or the program panics with an index-out-of-range error. The function has a mutex protecting errors but no mutex protecting findings.

## Reproduce

```bash
go test -race ./internal/audit/ -run TestExercise12 -v
```

## File to Investigate

`internal/audit/report.go` — look at the `ConcurrentAudit` function

Find the `findings = append(findings, result...)` line inside the goroutine and compare it with how `firstErr` is protected by `errMu`.

## What You Will Learn

- Slice `append` is not atomic — concurrent appends to the same slice cause data races
- The race detector is the most reliable way to find these bugs; always run tests with `-race`
- The fix mirrors how `firstErr` is already protected: acquire a mutex before appending, release it after
- Compare `ConcurrentAudit` with `ParallelAudit` in the same file — one protects findings, the other does not

## Fixing It

Apply your fix, then run:

```bash
go test -race ./internal/audit/ -run TestExercise12 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
