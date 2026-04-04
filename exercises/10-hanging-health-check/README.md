# Exercise 10: The Hanging Health Check

**Application:** pipeline | **Difficulty:** Intermediate

## Symptoms

A health-check request arrives with a context that has a short deadline. The slow dependency check takes longer than that deadline. The request context times out — but the health checker does not notice and continues waiting for the slow check to complete. The handler hangs until the slow check finishes on its own, ignoring the caller's deadline entirely.

## Reproduce

```bash
go test ./internal/health/ -run TestExercise10 -v
```

## File to Investigate

`internal/health/checker.go` — look at the `Check` method on `HealthChecker`

Find the argument passed to each check function (`c.Fn(...)`) and ask: is it the caller's context or a fresh one?

## What You Will Learn

- `context.Background()` creates a context that is never cancelled and has no deadline
- Passing `context.Background()` to a downstream call disconnects it from the caller's cancellation chain
- The fix is trivial — pass `ctx` (the method's parameter) instead of `context.Background()`
- Context propagation is a discipline: every blocking call should receive the caller's context so that cancellation and deadlines flow end-to-end

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/health/ -run TestExercise10 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
