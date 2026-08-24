# W02 Debrief

Read this only after completing the handoff.

## Cause

`Analysis` already contained the fact that the planner needed. When
`Determined` was false, `PlanReady` counted the result but did not put it in
`dependenciesByTarget`. A later map lookup for that known target returned a
nil slice. The ready check treated the nil slice exactly like a complete empty
slice. It found no unfinished dependency and put the target in the ready list.

The nil slice was not the problem by itself. The problem was that the program
reduced three states to two:

```text
known empty --------+
                     +--> no stored dependency IDs --> ready
undetermined -------+
known one or more ----------------------------> check completion
```

An empty adjacency list shows independence only when the builder finished
checking the relationship that this operation uses.

## Smallest repair

Keep undetermined results during planning and handle them before the ordinary
dependency check. One small change adds a map beside the dependency index:

```go
undetermined := make(map[TargetID]string)

for _, analysis := range analyses {
    // Existing target validation stays here.
    known[analysis.Target] = struct{}{}
    if !analysis.Determined {
        plan.Stats.UndeterminedAnalyses++
        undetermined[analysis.Target] = analysis.Detail
        continue
    }
    dependenciesByTarget[analysis.Target] = deduplicate(analysis.Dependencies)
}
```

Then handle that state before checking stored edges:

```go
for _, target := range targets {
    if _, done := completedSet[target]; done {
        continue
    }
    plan.Stats.TargetsConsidered++

    if detail, unknown := undetermined[target]; unknown {
        plan.Blocked = append(plan.Blocked, target)
        plan.Diagnostics = append(plan.Diagnostics, Diagnostic{
            Target:  target,
            Message: "dependency analysis undetermined: " + detail,
        })
        continue
    }

    // Existing completion checks follow.
}
```

This repair does not invent a dependency. It keeps the target out of the ready
list because required evidence is missing, and it explains that decision in
the output. A complete empty analysis still reaches the ordinary loop and
remains ready.

## Check the repair

Run:

```bash
go test ./04-build-dependency-graphs/wheel/02-unknown-is-not-independent \
  -count=1 -v
go test -tags=csbridgewheel9 \
  ./04-build-dependency-graphs/wheel/02-unknown-is-not-independent \
  -count=1 -v
```

The ordinary tests preserve real independence, known prerequisites, fixed
ordering, and storage of each repeated edge only once. The tagged tests make
known-empty and undetermined results appear differently in the returned plan.

## What the production source shows

[Prometheus PR #15560](https://github.com/prometheus/prometheus/pull/15560)
states that an empty dependency map made rules appear to have no dependencies
or dependents. Before the fix, its tests ran affected rules at the same time.
The pull request was merged in December 2024.

The source supports the unknown-versus-empty lesson. It does not support the
local data types, target names, choice to block when uncertain, counters, or
exact repair shown here; those belong to this exercise.

## What to measure in production

Separately count completed checks that found none, completed checks that found
one or more, and checks that did not reach a conclusion. Also count targets
kept out of the ready list because knowledge was incomplete. Publish reasons
from a small fixed set. Compare how much work runs at the same time with
whether evaluation results stay correct. One “dependency count” cannot show
whether zero means independent or unknown.
