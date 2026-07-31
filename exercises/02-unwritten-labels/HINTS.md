# Hints for Exercise 02: The Unwritten Labels

## Hint 1: Direction

The panic message is "assignment to entry in nil map". The helper first writes
its complete discovery result to a scratch map, then reads the map while
building findings. Compare how the two public entry points set up that state.

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
