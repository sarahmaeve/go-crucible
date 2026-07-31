# Exercise 09: The Immortal Connection

**Application:** kube-patrol | **Difficulty:** Beginner

## Symptoms

`AuditSecretExpiry` processes a batch of secrets. Each secret that carries an
expiry annotation opens an `io.ReadCloser`. Readers are never closed, so the
number of live resources grows with the batch. In production this would
manifest as file-descriptor exhaustion proportional to the number of secrets
audited.

## Reproduce

```bash
go test ./internal/audit/ -run TestExercise09 -v
```

## File to Investigate

`internal/audit/secrets.go` — look at the `AuditSecretExpiry` function

Find the call to `newSecretReader` and look for the matching `Close()` call. There isn't one.

## What You Will Learn

- Any type implementing `io.ReadCloser` (HTTP response bodies, file handles, database cursors, gRPC streams) must be explicitly closed
- The idiomatic Go pattern: acquire and defer cleanup in the same function
- Why a `defer` directly inside a long loop still accumulates resources until
  the outer function returns
- How a small per-iteration helper gives `defer` the correct lifetime
- This exercise intentionally uses a simple in-process closer; the same pattern applies to network connections and OS resources

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/audit/ -run TestExercise09 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
