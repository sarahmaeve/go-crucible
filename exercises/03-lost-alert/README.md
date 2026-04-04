# Exercise 03: The Lost Alert

**Application:** pipeline | **Difficulty:** Beginner

## Symptoms

`AlertEvaluator.Evaluate` correctly detects a threshold violation and returns a non-nil error. However, when the caller checks `errors.Is(err, types.ErrThresholdExceeded)` the result is always `false` — the error type information is lost even though the error message contains the sentinel text. Alert routing logic that depends on `errors.Is` never fires.

## Reproduce

```bash
go test ./internal/alert/ -run TestExercise03 -v
```

## File to Investigate

`internal/alert/evaluator.go` — look at the `Evaluate` function

Find the `fmt.Errorf` call that wraps `types.ErrThresholdExceeded` and examine which verb is used.

## What You Will Learn

- `fmt.Errorf("...: %v", err)` formats the error as a string — the chain is broken and `errors.Is` cannot traverse it
- `fmt.Errorf("...: %w", err)` wraps the error — the chain is preserved and `errors.Is` works correctly
- The `%w` verb was introduced in Go 1.13 specifically to support error wrapping

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/alert/ -run TestExercise03 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
