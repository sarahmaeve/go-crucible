# Hints for Exercise 09: The Immortal Connection

## Hint 1: Direction

The test installs a hook that counts how many times `Close()` is called on the readers opened inside `AuditSecretExpiry`. It expects one close per opened reader. The count stays at zero. Something is opening a resource and never closing it.

## Hint 2: Narrower

Open `internal/audit/secrets.go` and find the `reader := newSecretReader(...)` line inside the loop in `AuditSecretExpiry`. Immediately below it there is a comment pointing out the missing `Close`. No `defer reader.Close()` is present, and there is no explicit `reader.Close()` call anywhere in the loop body.

## Hint 3: Almost There

Add a deferred close immediately after opening the reader:

```go
reader := newSecretReader(secret.Data["value"])
defer reader.Close()
```

Because this `defer` is inside a loop it will accumulate — but for the purposes of this exercise that is acceptable (and the test does not check for FD exhaustion). If you want zero-accumulation, extract the body of the loop into a helper function so each `defer` runs at the end of the helper's scope.
