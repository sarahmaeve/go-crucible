# Exercise 01: The Silent Failure

**Application:** kube-patrol | **Difficulty:** Beginner

## Symptoms

You call `AuditPodLimits` against a namespace that does not exist (or whose API call fails). The function returns zero findings and a `nil` error — as if everything is fine. No findings are raised, no alert fires, and the operator has no idea the audit did not actually run. The failure evaporates silently into a log line.

## Reproduce

```bash
go test ./internal/audit/ -run TestExercise01 -v
```

## File to Investigate

`internal/audit/pods.go` — look at the `AuditPodLimits` function

The bug is in the error handling block immediately after `c.ListPods` is called. When `ListPods` returns an error, the function logs it but does not return — it falls through and processes an empty pod list, returning `(nil findings, nil error)` to the caller.

## What You Will Learn

- The difference between "log and continue" and "log and return" in Go error handling
- Why swallowed errors make programs silently incorrect rather than explicitly broken
- How callers cannot distinguish a successful empty result from a failed audit when errors are not propagated

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/audit/ -run TestExercise01 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
