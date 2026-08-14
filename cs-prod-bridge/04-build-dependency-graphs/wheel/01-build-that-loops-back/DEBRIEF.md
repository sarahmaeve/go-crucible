# W01 Debrief

Read this only after completing the handoff.

## Cause

`Validate` used one global `seen` bit for two different facts. A second visit
could mean either:

- the stage was completely checked through another branch; or
- the stage is still on the current chain of calls.

The function returned immediately in both cases. In the affected graph, it
therefore accepted `base -> package -> test -> base`. Recursive conversion did
not have that shortcut and followed the loop until the worker's stack was
exhausted.

The shared-base graph shows why “reject every second visit” is also wrong. Its
second visit reaches a stage whose earlier check has already returned. No path
with at least one edge leads from that stage back to itself.

## Smallest repair

Keep separate states for a stage that is being checked and a stage whose
outgoing edges are finished. Keep the current path too. The code can then turn
an edge back to an active stage into `CycleEdge` values with source locations.

One repair keeps the public API unchanged:

```go
func (g *Graph) Validate() (ValidationStats, error) {
    const (
        unseen uint8 = iota
        active
        finished
    )

    stats := ValidationStats{}
    state := make(map[StageID]uint8, len(g.stages))
    path := make([]StageID, 0, len(g.stages))

    var visit func(StageID) error
    visit = func(stage StageID) error {
        state[stage] = active
        path = append(path, stage)
        stats.StagesEntered++

        for _, dependency := range g.dependencies[stage] {
            stats.EdgesExamined++
            switch state[dependency] {
            case unseen:
                if err := visit(dependency); err != nil {
                    return err
                }
            case active:
                first := slices.Index(path, dependency)
                nodes := append(slices.Clone(path[first:]), dependency)
                edges := make([]CycleEdge, 0, len(nodes)-1)
                for i := 0; i+1 < len(nodes); i++ {
                    key := edgeKey{stage: nodes[i], needs: nodes[i+1]}
                    edges = append(edges, CycleEdge{
                        Stage:   nodes[i],
                        Needs:   nodes[i+1],
                        Sources: slices.Clone(g.sources[key]),
                    })
                }
                return &CycleError{Edges: edges}
            case finished:
                // This branch was already checked and is no longer active.
            }
        }

        path = path[:len(path)-1]
        state[stage] = finished
        return nil
    }

    for _, stage := range g.stages {
        if state[stage] == unseen {
            if err := visit(stage); err != nil {
                return stats, err
            }
        }
    }
    return stats, nil
}
```

Sorting the stages and dependency lists makes the first reported cycle the same
on every run. Each neighboring pair in the closed node list supplies an edge.
The existing source index supplies the declarations that a maintainer can
edit. A self-dependency finds the active stage at once and produces one edge.

## Check the repair

Run:

```bash
go test ./04-build-dependency-graphs/wheel/01-build-that-loops-back -count=1 -v
go test -tags=csbridgewheel8 \
  ./04-build-dependency-graphs/wheel/01-build-that-loops-back \
  -count=1 -v
```

The ordinary diamond test protects shared prerequisites. The tagged tests
protect the closed path, every source location, and the one-edge self-cycle.

## What the production sources show

[BuildKit PR #999](https://github.com/moby/buildkit/pull/999) records that a
Dockerfile stage cycle could make `dockerd` call itself until its goroutine ran
out of stack space. The change added a check for circular stage dependencies.
[PR #4567](https://github.com/moby/buildkit/pull/4567) later kept source
locations and added the relevant Dockerfile commands to the error.

The sources show that failure and those upstream changes. They do not support
the local type names, graph-walk counters, examples, or exact repair shown
here; those belong to this exercise.

## What to measure in production

Count validation failures by kind, call depth or explicit-stack depth, nodes
reached, edges checked, and errors that include source locations. A general
“cycle” counter can show how often the problem occurs. The closed path and
declaration locations help a maintainer find the needed edit faster.
