# Exercise 09: The Immortal Connection

**Application:** kube-patrol | **Difficulty:** Intermediate

## Symptoms

`AuditSecretExpiry` processes a batch of secrets. Each secret that carries an expiry annotation opens an `io.ReadCloser`. The test verifies that each opened reader is eventually closed. It never is. In production this would manifest as file descriptor exhaustion or goroutine leaks proportional to the number of secrets audited.

## Reproduce

```bash
go test ./internal/audit/ -run TestExercise09 -v
```

## File to Investigate

`internal/audit/secrets.go` — look at the `AuditSecretExpiry` function

Find the call to `newSecretReader` and look for the matching `Close()` call. There isn't one.

## What You Will Learn

- Any type implementing `io.ReadCloser` (HTTP response bodies, file handles, database cursors, gRPC streams) must be explicitly closed
- The idiomatic Go pattern: `defer reader.Close()` immediately after the resource is opened
- Why a `defer` inside a loop iteration still accumulates — and when a helper function is the right fix for that
- This exercise intentionally uses a simple in-process closer; the same pattern applies to network connections and OS resources

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/audit/ -run TestExercise09 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
