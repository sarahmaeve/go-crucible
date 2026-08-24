# Lab: Build Graph Explorer

This lab starts after Bazel has evaluated the build declarations. It receives
a known set of targets and dependency records. It does not read or evaluate
`BUILD.bazel` files.

For every local edge:

```text
A -> B means target A directly depends on target B.
```

Keep that sentence beside you. Follow an arrow to find dependencies. Follow
the reverse index to find targets that directly need a dependency. A
dependency-first build order must put `B` before `A`. That is the opposite of
the stored arrow.

## 1. Draw the example before running it

The tests use six known targets:

```text
//app:server ----> //lib/http:http ----> //lib/logging:logging
       |
       +---------> //lib/config:config

//tool/migrate:migrate ---> //lib/config:config

//tool/lint:lint
```

Write down your answers before opening `graph_test.go`. Questions 4–6 plan a
build whose only requested root is `//app:server`:

1. Which targets can the server reach by following dependency edges? This set,
   including the server, is its dependency closure.
2. Which targets directly require `//lib/config:config`?
3. Why does the server require logging? Give the path and count its edges.
4. Which targets are ready before any work completes?
5. After config and logging both complete, which target becomes ready?
6. Which targets must not appear when only the server is requested?

The lint target has no selected edges on purpose. It tests whether the graph
keeps a known node that has no edges. The migration target shares config with
the server, but the server does not depend on the migration target.

## 2. Inspect how the graph is stored

Open `graph.go` and find these four collections:

```go
targets       map[TargetID]struct{}
dependsOn     map[TargetID][]TargetID
requiredBy    map[TargetID][]TargetID
sources       map[DependencyEdge][]SourceLocation
```

`targets` separates an unknown target from a known target with no dependencies.
`dependsOn` and `requiredBy` store the same edges in opposite directions.
`sources` keeps the source locations used in errors separate from the short
lists of neighboring nodes.

`NewGraph` also makes three choices:

- both targets named by an edge must be in the known target set;
- repeated declarations produce one edge; and
- public results use a fixed dictionary order instead of Go map order.

Add the server-to-HTTP declaration a second time with a different source
location. The constructor contract above is enough to predict the result:
state whether the direct-dependency list, closure edge count, and build order
change. Section 5 adds a cycle so that
`TestCycleWitnessIsClosedAndPreservesSources` can also check the source records
kept for that one edge.

## 3. Follow the question in the correct direction

Run the lab from the `cs-prod-bridge` directory:

```bash
go test ./04-build-dependency-graphs/lab -count=1 -v
```

`Dependencies(server)` reads `dependsOn[server]`. It returns only the two
direct dependencies. It does not return logging merely because logging is
reachable.

`ReverseDependencies(config)` reads `requiredBy[config]`. It returns the
server and migration tool. This is a list of targets that directly declare the
dependency. It does not prove that both targets will rebuild, run, fail, or
affect users after every config change.

`DependencyClosure` follows dependency edges from the requested roots. Its
`TraversalStats` reports:

- `NodesReached`: distinct targets added to the search; and
- `EdgesExamined`: outgoing dependency edges inspected.

For the part of the graph that the search can reach, let `V` be the number of
targets and `E` the number of selected edges. The walk uses `O(V + E)` time and
`O(V)` extra search space. The graph indexes use `O(V + E)` space, plus the
source records they keep.

## 4. Use BFS for the path this API promises

`FewestEdgePath` uses breadth-first search. It explores paths one edge at a
time, so the first time it removes the destination from the queue, the saved
parent links describe a path with the fewest edges.

The promise has clear limits:

- it counts edges, not build duration, cost, or risk;
- it returns one path when several shortest paths exist; and
- its dictionary ordering of neighbors makes that choice the same on every
  run.

A target is reachable from itself by a path with zero edges. This does not mean
that the target directly depends on itself. Use the cycle check to look for a
path with at least one edge back to the same target.

Change the equal-length-path test so that its declarations arrive in a
different order. The returned path should not change. Then rename target `B`
to `Z` everywhere in that test. Because `C` now sorts before `Z`, predict which
equal-length path the API will select.

## 5. Distinguish a shared dependency from a cycle

A single “seen before” record cannot tell these cases apart:

```text
diamond: A -> B -> D       cycle: A -> B -> C -> A
          \> C -> D
```

In the diamond, the second route reaches a node whose outgoing edges have
already been checked. In the cycle, an edge returns to a node that is still on
the current depth-first path.

`FindCycle` records three states: unseen, active, and finished. When it reaches
an active target, it copies the matching part of the current path and closes
the cycle. The result turns each neighboring pair into a `CycleEdge` and
includes every saved source location for that declaration.

Add this declaration to the example:

```go
Dependency{
    Target: logging,
    Needs:  server,
    Source: SourceLocation{File: "lib/logging/BUILD.bazel", Line: 14},
}
```

Before running the test, write the exact closed path you expect. Also try a
self-dependency. It is a cycle with one edge.

## 6. Calculate build order from ready targets

`BuildOrder` starts with the requested roots and the dependencies that they can
reach. It counts each target's dependencies in that group. A target with a
zero count enters a min-heap, which returns names in dictionary order. Each
removal adds one target to the calculated order. The code then follows
`requiredBy` edges and lowers the counts of waiting targets.

The result is dependency-first. For every included edge
`dependent -> dependency`, a correct result satisfies:

```text
position(dependency) < position(dependent)
```

The heap makes output the same on every run, but its tie-breaking adds work.
This implementation takes `O(E + V log V)` time for the requested part of the
graph. An unordered ready set would take `O(V + E)`. `BuildStats` records the
target and edge counts, additions to the ready set, and targets placed in the
order.

`Ready` uses the same dependency rule for work that can run. It returns
unfinished targets whose direct dependencies in this build are complete.
These targets may run at the same time under the dependency rules. The number
of workers, available resources, failures, and scheduling policy can still
limit what actually runs.

Trace the example after each completion:

```text
completed = []
completed = [config]
completed = [config, logging]
completed = [config, logging, http]
```

Compare your predicted ready sets with
`TestReadyChangesOnlyAfterDependenciesComplete`.

## 7. Read counts before timings

`TestTraversalCountsGrowWithNodesAndEdges` builds chains of 10, 100, and 1,000
targets. Each chain adds one node and one edge at a time. The test checks those
counts directly, so changes in timing cannot hide how the work grows.

Then run the benchmarks:

```bash
go test ./04-build-dependency-graphs/lab \
  -run '^$' -bench BenchmarkGraphQueries -benchmem -count=5
```

The benchmark separates two graph shapes:

- a chain has a long path and only one ready target at a time; and
- a fan-out has one root with many leaf dependencies, so the ready heap holds
  many targets at once.

At a given size, both shapes have nearly the same number of nodes and edges.
However, they use the queue, recursive calls, and ready set differently.
Compare memory allocations as well as elapsed time. These measurements
describe this Go program on this machine. They do not change the stated
complexity limits.

The ordinary test command also runs the seed corpus for
`FuzzGraphOperationsAgreeWithReference`. Run a generated-input campaign with:

```bash
go test ./04-build-dependency-graphs/lab \
  -run '^$' -fuzz '^FuzzGraphOperationsAgreeWithReference$' -fuzztime=10s
```

The fuzz property builds small arbitrary directed graphs. It compares direct
and reverse neighbors, closure, fewest-edge paths, cycles, ready work, and
build order with an independent reference model. It also rebuilds each graph
with reversed input order and checks that public answers do not change.

## 8. Check the boundary of the model

The lab models only the target records supplied to `NewGraph`. Production code
that connects to Bazel should use supported query tools. It must say whether it
reads the target graph before configuration, configured targets, or the action
graph. Reading Starlark as plain text would miss Bazel's evaluation, macros,
configuration choices, implicit dependencies, and toolchains.

Use the sources for separate purposes:

- [MIT 6.006 Lecture 9](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/196a95604877d326c6586e60477b59d4_MIT6_006S20_lec9.pdf)
  explains adjacency-list traversal, BFS, and paths with the fewest edges.
- [MIT 6.006 Lecture 10](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/f3e349e0eb3288592289d2c81e0c4f4d_MIT6_006S20_lec10.pdf)
  explains DFS, directed cycles, DAGs, and topological order.
- [The Go specification](https://go.dev/ref/spec#For_statements) states that
  map iteration order is not specified.
- [Bazel's dependency guide](https://bazel.build/versions/9.1.0/concepts/dependencies)
  defines its declared target dependency contract.
- [Bazel's query reference](https://bazel.build/versions/9.1.0/query/language)
  documents target-to-prerequisite edge direction and graph queries.
- [BuildKit PR #999](https://github.com/moby/buildkit/pull/999) and
  [PR #4567](https://github.com/moby/buildkit/pull/4567) show cycle validation
  and errors that identify source instructions in production Go code.

Finish with a short design handoff. State the node type and complete the edge
sentence. Name the requested roots. Give the cost in both nodes and edges.
Explain the dictionary-order tie-break. Name at least two relationships that
this graph leaves out.
