# W03 Debrief

Read this only after completing the handoff.

## Cause

`NewGraph` kept all three records by `RecordID`, but it also built one
lookup entry per URN:

```go
byURN[resource.URN] = resource.RecordID
```

The later database record replaced the earlier entry. In the reported input
order, the service's `urn:database` declaration therefore linked to
`database-old`. The graph contained three nodes and one edge with a valid
shape, but that edge joined the wrong records.

The deletion algorithm used the correct direction. However, it saw no edge
between the two current records that it had been asked to delete. Its
dictionary-order tie-break put `database-current` first and exposed the unsafe
plan.

The changing results for different input orders were another symptom of the
same information loss. A map assignment chose whichever record with that URN
arrived last.

## Smallest repair

The constructor already checks and records the one current resource for each
URN in `currentByURN`. This index includes resource state. Use it when linking
the dependency URNs of current records:

```go
dependencyID, found := currentByURN[dependencyURN]
if !found {
    return nil, fmt.Errorf(
        "record %q depends on URN %q without a current resource",
        resource.RecordID,
        dependencyURN,
    )
}
```

Keep the URN set for the `LogicalURNs` count. Keep every resource in the
`RecordID` map. Do not combine the old and current records into one graph node.
A larger program might also link dependencies for old generations. It would
need a clear rule and enough generation or state information to choose the
correct records. One URN string would still not identify one specific record.

With the repaired edge:

```text
service-current -> database-current
```

the deletion walk places the service before the database. The old
database remains a separate record available to its cleanup operation.

## Check the repair

Run:

```bash
go test ./04-build-dependency-graphs/wheel/03-two-resources-one-urn \
  -count=1 -v
go test -tags=csbridgewheel10 \
  ./04-build-dependency-graphs/wheel/03-two-resources-one-urn \
  -count=1 -v
```

The ordinary tests preserve behavior when each URN is unique, dependent-first
deletion, allowed old/current pairs, and invalid-input checks. The tagged tests
check the specific record at the end of the edge, safe deletion order, edge
count, and equal results for different input orders.

## What the production source shows

[Pulumi PR #19179](https://github.com/pulumi/pulumi/pull/19179) documents that
snapshots can contain a current resource and any number of deleted copies with
the same URN. It shows that the old lookup could link B's dependency to the
deleted A. Later error handling could then allow deletion of the current A.
The merged change tracks current and deleted
resources separately.

The source supports the identity and deletion-safety lesson. It does not
support the local `RecordID`, `GraphStats`, heap, preview, or one-line repair;
those belong to this exercise.

## What to measure in production

Count individual records as well as distinct logical names. Check that the
record at each end of an edge has the state or generation that the operation
requires. Count current/old name collisions, references that cannot be linked,
and plans cancelled to protect a resource that still depends on another.
Tests for stable output should supply the same snapshot records in several
orders instead of testing only one order.
