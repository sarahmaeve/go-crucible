# Exercise 05: The Nil Check That Lies

**Application:** kube-patrol | **Difficulty:** Intermediate

## Symptoms

`NewAuditClient` is called with an invalid kubeconfig path. You guard the result with `if client != nil` before calling any method. The guard passes — `client` is not nil. The very next method call panics with a nil pointer dereference inside the `KubeClient` method body. The nil check provided no protection at all.

## Reproduce

```bash
go test ./internal/client/ -run TestExercise05 -v
```

## File to Investigate

`internal/client/client.go` — look at the `NewAuditClient` function

Focus on what is returned in the error paths: `(*KubeClient)(nil)` versus plain `nil`.

## What You Will Learn

- An interface value in Go is a pair of `(type, value)`. It is only nil when both parts are nil.
- `(*KubeClient)(nil)` is a typed nil: the type part is `*KubeClient`, the value part is nil. When stored in an `AuditClient` interface, the result is a non-nil interface.
- Callers checking `if client != nil` see `true` because the interface has a type — the nil-value pointer is hidden inside it.
- The fix: return `nil, err` (untyped nil) from error paths in functions that return an interface type.

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/client/ -run TestExercise05 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
