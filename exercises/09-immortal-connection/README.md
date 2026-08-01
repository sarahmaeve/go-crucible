# Exercise 09: The Immortal Connection

**Application:** pipeline | **Difficulty:** Beginner

## Symptoms

`FetchRules` downloads several remote rule documents. Every successful
`http.Client.Do` returns a response body, but the bodies are never closed. The
number of live responses grows with the batch; recurring reloads or large
source lists can consume transport connections and file descriptors and
prevent connection reuse.

## Reproduce

```bash
go test ./internal/alert/ -run TestExercise09 -v
```

## File to Investigate

`internal/alert/rules.go` — look at the `FetchRules` function

Find the call to `client.Do` and follow the lifetime of the returned response.

## What You Will Learn

- Successful HTTP responses own a body that callers must close
- Closing a consumed response body lets the transport release resources and reuse the connection
- The idiomatic Go pattern: acquire and defer cleanup in the same function
- Why a `defer` directly inside a long loop still accumulates resources until
  the outer function returns
- How a small per-iteration helper gives `defer` the correct lifetime
- An injected `http.Client` makes network code testable without production-only test hooks

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/alert/ -run TestExercise09 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
