# Hints for Exercise 01: The Silent Failure

## Hint 1: Direction

The bug is in error handling. The function receives an error from an API call, does something with it, and then continues as if nothing happened. What should a well-behaved function do when it cannot retrieve the data it needs to do its job?

## Hint 2: Narrower

Open `internal/audit/pods.go` and read the `if err != nil` block after
`c.ListPods`. It records the error locally but then continues. This library
function should stop and return an error that retains the original cause.

## Hint 3: Almost There

The block currently logs and falls through. Replace that local handling with a
wrapped return, for example:

```go
if err != nil {
    return nil, fmt.Errorf("audit pod limits: listing pods: %w", err)
}
```

The `%w` keeps `errors.Is` and `errors.As` working. A caller at the application
boundary can log the returned error once with request or command context.
