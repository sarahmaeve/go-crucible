# Hints for Exercise 02: The Unwritten Labels

## Hint 1: Direction

The panic message is "assignment to entry in nil map". Something is trying to write a key into a map that was never initialized. Find where the map is declared in `AuditDeploymentLabels` and compare it with how the same map is set up in `NewDeploymentAuditor`.

## Hint 2: Narrower

Open `internal/audit/deployments.go`. In `AuditDeploymentLabels`, find this line:

```go
var missingLabels map[string]bool
```

A `var` declaration for a map gives you a nil map. In `NewDeploymentAuditor`, the equivalent line uses `make`. That is the difference.

## Hint 3: Almost There

Change the declaration from:

```go
var missingLabels map[string]bool
```

to:

```go
missingLabels := make(map[string]bool)
```

This allocates the backing hash table so that subsequent writes (`missingLabels[key] = true`) succeed instead of panicking.
