# Exercise 02: The Unwritten Labels

**Application:** kube-patrol | **Difficulty:** Beginner

## Symptoms

Calling `AuditDeploymentLabels` against a namespace that contains a deployment missing a required label causes the program to panic at runtime with:

```
panic: assignment to entry in nil map
```

The panic occurs even though the code declares `missingLabels` as a variable. The companion struct method `DeploymentAuditor.Audit` works perfectly — only the standalone function panics.

## Reproduce

```bash
go test ./internal/audit/ -run TestExercise02 -v
```

## File to Investigate

`internal/audit/deployments.go` — look at the `AuditDeploymentLabels` function

Find where `missingLabels` is declared and compare it with how the same map is initialized inside `NewDeploymentAuditor`.

## What You Will Learn

- A `var m map[K]V` declaration creates a nil map — reads are safe but writes panic
- Maps must be initialized with `make(map[K]V)` before any key is written
- Why the struct method works while the standalone function panics: the struct's constructor uses `make`, the function does not

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/audit/ -run TestExercise02 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
