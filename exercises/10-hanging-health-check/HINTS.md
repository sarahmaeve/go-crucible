# Hints for Exercise 10: The Hanging Health Check

## Hint 1: Direction

The test cancels a context and expects `Check` to respect the cancellation and return promptly. It does not. The deadline from the caller's context is being ignored somewhere inside `Check`. Look at what context is handed to the slow dependency check.

## Hint 2: Narrower

Open `internal/health/checker.go` and look at the `Check` method. Find the line where it calls `c.Fn(...)`. The argument passed to the function is not `ctx` — it is `context.Background()`. `context.Background()` is never cancelled and has no deadline, so the dependency check runs to completion regardless of what the caller's context says.

## Hint 3: Almost There

Change:

```go
if err := c.Fn(context.Background()); err != nil {
```

to:

```go
if err := c.Fn(ctx); err != nil {
```

That is the entire fix. The caller's context — with its deadline and cancellation — is now propagated into each dependency check, so they all respect the request's lifecycle.
