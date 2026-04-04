# Hints for Exercise 03: The Lost Alert

## Hint 1: Direction

The error is returned correctly — the test can see that `err != nil`. The problem is that `errors.Is(err, types.ErrThresholdExceeded)` returns `false`. That means the error chain is broken. Look at how the error is constructed.

## Hint 2: Narrower

Open `internal/alert/evaluator.go` and find the `fmt.Errorf` call in the `Evaluate` function. It wraps `types.ErrThresholdExceeded`. Which format verb is used — `%v` or `%w`?

## Hint 3: Almost There

The current line is:

```go
return alerts, fmt.Errorf("evaluation failed: %v", types.ErrThresholdExceeded)
```

`%v` formats the error as a plain string — the resulting error has no wrapped error inside it, so `errors.Is` cannot find `ErrThresholdExceeded` in the chain.

Change `%v` to `%w`:

```go
return alerts, fmt.Errorf("evaluation failed: %w", types.ErrThresholdExceeded)
```

`%w` wraps the error so that `errors.Is` can unwrap the chain and find the sentinel.
