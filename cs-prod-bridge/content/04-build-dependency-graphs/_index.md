+++
title = 'Build dependency graphs'
description = 'Directed graphs make build dependencies traversable, but only after the nodes, arrows, identities, and ordering contract are stated precisely.'
weight = 4
+++

**Unit 04 · Foundations**

# Build dependency graphs

{{< lead >}}A dependency graph turns concrete build declarations such as
`deps = ["//lib/http:http"]` into questions a program can answer: what must be
present, why is it needed, what else relies on it, does the graph contain a
cycle, and which work can begin now? The answers are trustworthy only when we
state how the build declaration becomes an edge and which parts of the build
we chose to represent.{{< /lead >}}

{{< callout kind="production" title="Incoming reports" >}}
- [BuildKit PR #999](https://github.com/moby/buildkit/pull/999) documents a
  cyclic Dockerfile stage dependency that made `dockerd` recurse until its
  goroutine stack overflowed. A later
  [patch](https://github.com/moby/buildkit/pull/4567) made the error identify
  the Dockerfile instructions that formed the cycle.
- [Argo Workflows issue #16450](https://github.com/argoproj/argo-workflows/issues/16450)
  describes a retried workflow that remained Running after every Pod had
  finished. Retry code kept one parent for a node that could have several, and
  Go map iteration could change which parent it kept.
- [Pulumi PR #19179](https://github.com/pulumi/pulumi/pull/19179) fixes a
  deletion graph that confused a current resource with an older copy awaiting
  deletion because both records had the same resource identifier, called a
  URN by Pulumi.
{{< /callout >}}

These failures happened in different systems and have different causes.
BuildKit needed to recognize a cycle before recursive conversion continued.
Argo discarded part of a multi-parent relationship and then made the remaining
choice nondeterministically. Pulumi built edges between the wrong concrete
resources. “It uses a graph” does not explain any of them. The useful questions
are:

- What does one node represent?
- What does one directed edge assert?
- Can two distinct things receive the same identity?
- Is the relationship known, absent, or undetermined?
- Which direction does an operation need to follow?

This unit starts with Bazel target dependencies because that relationship has a
clear production contract. The implementation examples remain in Go.

By the end of the unit, you should be able to:

- read a Bazel target label and a direct dependency from an actual
  `BUILD.bazel` rule call;
- derive a precisely scoped graph from dependency-bearing rule attributes;
- define nodes and directed edges in a complete sentence about the represented
  system;
- represent outgoing and incoming relationships with adjacency lists;
- distinguish a direct dependency from a reachable transitive dependency;
- use breadth-first search to return a path with the fewest edges;
- use depth-first search to distinguish a cycle from a harmless second visit;
- explain when a directed graph is a DAG;
- distinguish graph-theory topological order from dependency-first build order;
- derive dependency-first order from remaining-dependency counts and explain
  how completed prerequisites release real work;
- state the time and space cost in terms of both nodes and edges;
- distinguish deterministic output from a unique mathematical answer;
- explain why a correct algorithm cannot compensate for missing, ambiguous, or
  incorrectly identified edges; and
- distinguish this lesson's direct Go-library graph from Bazel's broader
  target, configured-target, and action graphs.

## Start with what Bazel actually reads

A Bazel build does not begin as a list of arrows. Bazel evaluates each
[`BUILD.bazel` file as Starlark](https://bazel.build/versions/9.1.0/concepts/build-files),
a restricted programming language. Calling a rule function in that file
creates a target. The rule defines the types and meanings of attributes such
as `srcs` and `deps`.

The following is a constructed teaching workspace, not an excerpt from a
production incident. It uses the public rule interface documented for
[`rules_go` v0.60.0](https://github.com/bazel-contrib/rules_go/blob/v0.60.0/docs/go/core/rules.md).
The small example lets us trace every declaration without pretending that the
arrow notation is Bazel syntax.

Assume the repository makes that pinned version of `rules_go` available in
`MODULE.bazel`:

~~~starlark
bazel_dep(name = "rules_go", version = "0.60.0")
~~~

The pinned
[`rules_go` Bzlmod documentation](https://github.com/bazel-contrib/rules_go/blob/v0.60.0/docs/go/core/bzlmod.md)
explains the module setup and Go SDK choices. The single line above is the part
that makes the rule definitions used in the BUILD snippets available; SDK
selection is not part of the six-node graph.

The server target is declared in `app/BUILD.bazel`:

~~~starlark
load("@rules_go//go:def.bzl", "go_binary")

go_binary(
    name = "server",
    srcs = ["main.go"],
    deps = [
        "//lib/config:config",
        "//lib/http:http",
    ],
)
~~~

Read the declaration from the outside in:

- `load` makes the `go_binary` rule function available in this file. It does
  not create the server target.
- Calling `go_binary` creates a rule target. The file belongs to the `app`
  package and `name = "server"` names the target, so its full label is
  `//app:server`.
- `srcs = ["main.go"]` names a source-file target in the same package. Its full
  label is `//app:main.go`.
- Each string in `deps` is a label for another rule target. A label of the form
  `//lib/http:http` identifies package `lib/http`, then target `http` within
  that package.

The HTTP library has its own declaration in `lib/http/BUILD.bazel`:

~~~starlark
load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "http",
    srcs = ["http.go"],
    importpath = "example.com/buildgraph/lib/http",
    deps = ["//lib/logging:logging"],
    visibility = ["//visibility:public"],
)
~~~

The relevant Go import declarations are concrete too. These excerpts omit the
package bodies that use the imports; they are not presented as complete source
files:

~~~go
// app/main.go
import (
	"example.com/buildgraph/lib/config"
	apphttp "example.com/buildgraph/lib/http"
)

// lib/http/http.go
import "example.com/buildgraph/lib/logging"
~~~

For these `rules_go` rules, `deps` names the Go libraries imported directly by
the target's Go package. Because `main.go` imports both the config and HTTP
packages, both labels belong in the server's `deps` list. Because `http.go`
imports logging, the HTTP target declares logging directly. It is not enough
for a directly imported package to happen to be reachable through some other
library. That principle—declare a direct dependency directly—keeps the build
correct when an intermediate library later changes its own dependencies.

{{< callout kind="contract" title="A direct-dependency list has two requirements" >}}
For the `deps` attributes in this fixture:

1. **No missing direct dependencies:** every non-standard-library Go package
   imported directly by the target's source must have its library target in
   `deps`.
2. **No transitive dependencies copied in as extras:** a library that the
   source does not import directly does not belong in `deps` merely because
   another dependency imports it.

The server therefore lists config and HTTP, but not logging. HTTP lists
logging. Logging still belongs to the server's transitive dependency closure;
it is simply not a direct dependency of the server's current source. Bazel's
[dependency guide](https://bazel.build/versions/9.1.0/concepts/dependencies)
states the general rule as declaring all actual direct dependencies, and no
more. Rule-specific attributes such as `data`, `embed`, or `cdeps` have their
own meanings and should not be forced into this `deps` rule.
{{< /callout >}}

The code above establishes these three relationships:

~~~text
//app:server -> //lib/config:config
//app:server -> //lib/http:http
//lib/http:http -> //lib/logging:logging
~~~

The arrows are a representation derived from specific `deps` attributes. They
are not text found in a `BUILD.bazel` file. For the running example, the edge
contract is:

> `A -> B` means the explicit `deps` attribute of `rules_go` target A directly
> names `rules_go` library target B.

The source of the edge is the target containing the `deps` attribute. The
destination is the target named by one label in that attribute. This is the
same target-to-prerequisite direction used by
[Bazel's query language](https://bazel.build/versions/9.1.0/query/language),
although Bazel's query graph contains more kinds of targets and dependencies
than this lesson's selected view.

### Select the graph before choosing the algorithm

`deps` is not synonymous with “the Bazel graph.” Rule attributes are typed,
and many attributes that accept labels introduce dependencies. In the
server declaration, `srcs` creates a relationship to `//app:main.go` as well.
Attributes such as `data`, private attributes supplied by a rule definition,
configuration choices, and toolchain resolution can introduce still more
relationships.

This lesson deliberately selects a smaller graph:

| Question | Bazel's broader graph family | Running teaching graph |
|---|---|---|
| What is a node? | Rule targets, source-file targets, generated-file targets, configured targets, actions, or artifacts, depending on the query | The six named `go_binary` and `go_library` rule targets |
| What creates an edge? | Label-bearing attributes and, in later phases, resolved implicit and toolchain dependencies or action inputs and outputs | Only the explicit `deps` attributes shown in these `rules_go` calls |
| What is omitted? | It depends on the interface: `query` has no configured targets or actions, `cquery` does not expose action internals, and `aquery` answers action-level questions | `srcs`, generated files, `data`, implicit dependencies, toolchains, configurations, actions, and artifacts |

That narrower graph is useful for learning traversal, cycle detection, and
ordering. It must not be presented as Bazel's complete internal graph.

{{< callout kind="warning" title="Do not parse BUILD files as if they were data tables" >}}
A `BUILD.bazel` file is evaluated Starlark. A macro can create a rule, a
`select()` can choose different labels under different configurations, and a
rule definition can add dependencies that never appear as a literal `deps`
list in the BUILD file. A tool that needs Bazel's graph should use Bazel's
supported `query` or `cquery` interfaces and consume structured output. The
`query --output=proto` format is one supported machine-readable option. Even
`query --output=build`, whose text resembles BUILD syntax after macros and
variables have been expanded, is not guaranteed by Bazel to be a valid BUILD
file. The local Go model in this unit begins with already identified targets
and edges; it is not a BUILD-file parser.
{{< /callout >}}

A **node**, also called a vertex, is one thing represented by the graph. A
**directed edge** is a one-way relationship between two nodes. The meaning of
the node and edge is part of the data structure's contract; neither can be
recovered from the shape of the drawing alone.

The edge does not mean that the server executes before the library, that
network traffic travels toward the library, or that failure must propagate
along the arrow. Those would be different relationships and therefore
different graphs.

Another system may store prerequisite-to-dependent edges instead:

~~~text
//lib/http:http  ---->  //app:server
~~~

That convention is not mathematically wrong. It answers forward-traversal and
ordering questions in the other direction. Problems begin when prose,
diagrams, and code silently move between the two conventions.

{{< callout kind="contract" title="Complete the edge sentence" >}}
Do not document an API as “Add an edge from A to B” and leave its meaning in the
real system implicit. Name the selected nodes and the relationship: “the
explicit `deps` attribute of Go target A names Go library B,” “action A must
finish before action B,” or whichever fact the graph actually stores.

The running direct-Go-dependency graph and the local Go API use
target-to-dependency edges. Later production cases state a different edge
meaning when they examine a different graph.
{{< /callout >}}

## Complete the running declaration

The remaining four targets also come from rule calls.

`lib/config/BUILD.bazel`:

~~~starlark
load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "config",
    srcs = ["config.go"],
    importpath = "example.com/buildgraph/lib/config",
    visibility = ["//visibility:public"],
)
~~~

`lib/logging/BUILD.bazel`:

~~~starlark
load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "logging",
    srcs = ["logging.go"],
    importpath = "example.com/buildgraph/lib/logging",
    visibility = ["//visibility:public"],
)
~~~

`tool/migrate/BUILD.bazel`:

~~~starlark
load("@rules_go//go:def.bzl", "go_binary")

go_binary(
    name = "migrate",
    srcs = ["main.go"],
    deps = ["//lib/config:config"],
)
~~~

`tool/lint/BUILD.bazel`:

~~~starlark
load("@rules_go//go:def.bzl", "go_binary")

go_binary(
    name = "lint",
    srcs = ["main.go"],
)
~~~

Selecting only explicit `deps` relationships among these six rule targets
produces the graph carried through the foundations:

~~~text
//app:server ----> //lib/http:http ----> //lib/logging:logging
       |
       +---------> //lib/config:config

//tool/migrate:migrate ---> //lib/config:config

//tool/lint:lint
~~~

It contains six rule-target nodes and four selected edges:

- `//app:server` directly depends on `//lib/http:http` and
  `//lib/config:config`.
- `//lib/http:http` directly depends on `//lib/logging:logging`.
- `//app:server` transitively depends on `//lib/logging:logging` through
  `//lib/http:http`.
- `//lib/config:config` is shared by the server and migration tool.
- `//tool/lint:lint` is known to this selected graph but has no `deps` edges.
  In Bazel's broader target graph, it still has a `srcs` edge to its source
  file.

A **path** is a sequence of connected directed edges. The path

~~~text
//app:server -> //lib/http:http -> //lib/logging:logging
~~~

has length two because it contains two edges.

A node is **reachable** from a starting node if following zero or more edges
can arrive there. Allowing a zero-edge path means a node is reachable from
itself. In this lesson, a target's **dependency closure** includes the target
itself and everything reachable from it. Bazel's `deps` operator uses the same
reflexive-closure definition, but it applies it to Bazel's broader target
graph, so it need not return the same node set as this selected view. We use
**transitive dependencies** for the other nodes reached after following at
least one edge.

The dependency closure of `//app:server`, shown here as a set, is:

~~~text
//app:server
//lib/config:config
//lib/http:http
//lib/logging:logging
~~~

It does not include `//tool/migrate:migrate` merely because both targets depend
on `//lib/config:config`. The arrows from the server do not lead to the
migration tool. It also does not include the disconnected lint target.

This unit uses finite, directed, unweighted graphs:

- **finite** means a build contains a bounded number of represented nodes and
  edges at the time of the query;
- **directed** means `A -> B` and `B -> A` are different relationships; and
- **unweighted** means path length counts edges rather than latency, money,
  failure probability, or another numeric cost.

Weighted shortest paths, undirected connectivity, and graph databases solve
different problems and are outside this unit.

## Store the next relationships, not every possible pair

An **adjacency list** stores the outgoing neighbors for each node. In this
graph, the forward list is naturally named `dependsOn`:

An outgoing edge starts at the node whose list contains it. The same edge is
incoming at the node where the arrow ends.

~~~text
dependsOn["//app:server"] = ["//lib/config:config", "//lib/http:http"]
dependsOn["//lib/http:http"] = ["//lib/logging:logging"]
dependsOn["//lib/config:config"] = []
dependsOn["//lib/logging:logging"] = []
dependsOn["//tool/migrate:migrate"] = ["//lib/config:config"]
dependsOn["//tool/lint:lint"] = []
~~~

A second index stores the same edges in reverse:

~~~text
requiredBy["//app:server"] = []
requiredBy["//lib/http:http"] = ["//app:server"]
requiredBy["//lib/config:config"] = ["//app:server", "//tool/migrate:migrate"]
requiredBy["//lib/logging:logging"] = ["//lib/http:http"]
requiredBy["//tool/migrate:migrate"] = []
requiredBy["//tool/lint:lint"] = []
~~~

These maps do not describe two different graphs. They are two indexes over the
same selected `deps` relationships. The local input can also retain the source
location of each explicit declaration, so a diagnostic can point back to the
text that needs to change:

~~~go
type TargetID string

type DependencyEdge struct {
	Dependent  TargetID
	Dependency TargetID
}

type SourceLocation struct {
	File string
	Line int
}

type CycleEdge struct {
	Edge    DependencyEdge
	Sources []SourceLocation
}

type Graph struct {
	targets    map[TargetID]struct{}
	dependsOn  map[TargetID][]TargetID
	requiredBy map[TargetID][]TargetID
	sources    map[DependencyEdge][]SourceLocation
}
~~~

The slices of target IDs keep traversal compact. `sources` preserves the
diagnostic evidence supplied for the same dependent/dependency pairs. Given
consecutive nodes `A` and `B` in a path, the corresponding edge key is
`{Dependent: A, Dependency: B}`. A `CycleEdge` combines that key with every
source location that declared it.

This evidence also has a scope. An explicit label may have an editable
location in a `BUILD.bazel` file, but a macro-generated or implicit dependency
may not correspond to one literal `deps` entry. A Bazel-backed implementation
must report the locations its supported query interface actually supplies; it
must not invent a precise source line for an edge whose origin Bazel reports
only at rule or macro level.

The direction to follow depends on the question:

| Question | Index |
|---|---|
| What does the server need? | `dependsOn` |
| Why does the server need logging? | `dependsOn` |
| Which declared targets rely on config? | `requiredBy` |
| Which targets might be affected by changing HTTP? | `requiredBy` |

“Might be affected” is deliberate. Reverse reachability describes declared
dependents. It does not prove that every dependent will execute, rebuild, fail,
or experience a user-visible impact. Caching, configurations, runtime paths,
and the content of the change remain outside this graph.

The adjacency lists store one entry per node and edge, so `V` nodes and `E`
edges use `O(V + E)` space. If `S` source-location records are retained across
those edges, the complete structure uses `O(V + E + S)` space. An adjacency
matrix instead reserves one cell for every possible ordered pair of nodes,
using `O(V²)` space before source evidence is added. A matrix can answer “is
this exact edge present?” by checking one cell, but sparse build graphs usually
do not need to reserve space for every edge that does not exist.

### Preserve nodes with no edges

If graph construction discovers nodes only while reading edges,
`//tool/lint:lint` disappears. That changes the answer to “which selected
targets exist?” and can omit a requested target with no `deps` relationships
from a build plan.

Store the node set separately. An empty adjacency list means “this known target
has no outgoing edge in the selected graph.” A missing map key must not
ambiguously mean both “known and empty” and “unknown target.”

This lesson's graph builder rejects an edge if either endpoint is not in the
known node set. Another system could create placeholder nodes instead, but that
would be a different contract and must remain distinguishable from a fully
loaded target.

### Deduplicate repeated declarations

Two declarations with the same dependent and dependency should not make a
target wait twice for one dependency. A graph builder can use a set while
loading and produce slices for traversal afterward.

If an input source supplies more than one evidence record for the same edge,
retain every relevant location under one `DependencyEdge`. Do not imply that
Bazel permits one target label to be independently declared in two BUILD
files; it does not. The local graph is deduplicating input evidence, not
changing Bazel's target-identity rules.

### Do not expose Go map order as graph order

The
[Go specification](https://go.dev/ref/spec#For_statements)
does not define map iteration order. If public results, tests, or error
messages range directly over maps, equal valid inputs can produce different
paths or build plans.

Sort public results or keep waiting nodes in a data structure with a stated
order. Deterministic output is an API choice. A graph can permit several valid
traversals or topological orders, and the data structure does not choose among
them by itself.

## Traversal means controlled exploration

A graph traversal begins at one or more nodes and follows selected edges. It
records which nodes it has reached so a shared node does not cause repeated
exploration and a cycle does not cause an endless walk.

Two standard traversals differ mainly in which discovered node they explore
next:

- **breadth-first search**, or BFS, explores all nodes one edge away before
  nodes two edges away;
- **depth-first search**, or DFS, follows one path as far as it can before
  returning to try another branch.

BFS is the direct choice when the operation promises a path with the fewest
edges. DFS is a natural choice when the operation needs to follow one complete
branch, such as checking whether an edge returns to the path currently being
explored. The required answer, rather than a general preference for one
traversal, determines the choice.

The formal sequence in
[MIT 6.006 Lecture 9](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/196a95604877d326c6586e60477b59d4_MIT6_006S20_lec9.pdf)
is useful here: define the vertices and directed edges, choose a
representation, define paths and reachability, and then introduce BFS.
[Lecture 10](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/f3e349e0eb3288592289d2c81e0c4f4d_MIT6_006S20_lec10.pdf)
continues from DFS to directed cycles and topological order.

## BFS can return a path with the fewest edges

Suppose the question is:

> Why does `//app:server` depend on `//lib/logging:logging`?

BFS explores the graph in layers:

| Distance from server | Newly reached targets |
|---:|---|
| 0 edges | `//app:server` |
| 1 edge | `//lib/config:config`, `//lib/http:http` |
| 2 edges | `//lib/logging:logging` |

The first time BFS reaches logging, it has found a path using the fewest
dependency edges. Any path with fewer edges would have ended in an earlier
layer, which BFS has already explored. A `parent` map records how each node was
first reached. Starting at logging, the code follows parent entries back to
the server and then reverses that list to reconstruct:

~~~text
//app:server -> //lib/http:http -> //lib/logging:logging
~~~

Assume the graph builder has sorted each dependency slice so that equal-length
choices are deterministic. The graph operation below implements reachability
and its zero-edge case directly:

~~~go
func (g *Graph) FewestEdgePath(start, want TargetID) ([]TargetID, bool) {
	if start == want {
		return []TargetID{start}, true
	}

	queue := []TargetID{start}
	seen := map[TargetID]bool{start: true}
	parent := make(map[TargetID]TargetID)

	for head := 0; head < len(queue); head++ {
		current := queue[head]
		if current == want {
			return rebuildPath(parent, start, want), true
		}

		for _, dependency := range g.dependsOn[current] {
			if seen[dependency] {
				continue
			}
			seen[dependency] = true
			parent[dependency] = current
			queue = append(queue, dependency)
		}
	}
	return nil, false
}
~~~

Using a head index avoids a queue implementation that copies all remaining
elements on every removal. Marking a dependency seen when it enters the queue
prevents two parents from queuing the same node before either queued copy is
processed. The first parent therefore remains the path used for reconstruction.

The example omits unknown-target errors so the traversal remains visible. A
public API must first distinguish an unknown target from a known target that
has no path to the destination.

`FewestEdgePath(A, A)` returns the one-node path `[A]`, which is consistent with
the definition of reachability. A product operation named `WhyDepends` should
not present that zero-edge path as evidence that `A` depends on itself. It can
require two distinct target IDs and direct callers to cycle detection when they
need to find a non-empty path from a target back to itself. Making that policy
explicit avoids hiding either a self-edge or a longer cycle behind “no path.”

The returned path is shortest only under the stated measure: number of edges.
It is not necessarily the fastest build path, highest-risk path, or smallest
set of source files. Those questions require more data—for example, duration
or risk attached to an edge—and a rule that uses that data.

{{< callout kind="contract" title="The local promise is stronger than Bazel somepath" >}}
Bazel's `somepath` query returns one dependency path. Its documentation does
not promise the shortest path. A local Go API may deliberately promise a
fewest-edge path by using BFS, but it must present that as its own contract,
not as a claim about Bazel's implementation.
{{< /callout >}}

## A global visited set alone does not detect directed cycles

A **directed cycle** is a path of one or more edges that returns to its
starting node.

A **cycle witness** is the concrete closed path that demonstrates the cycle.
It gives a maintainer something to inspect rather than only reporting that the
graph is invalid.

Consider a separate four-node teaching graph shaped like a diamond. It is not
another BUILD declaration from the running fixture:

~~~text
    +----> B ----+
    |            |
A --+            +----> D
    |            |
    +----> C ----+
~~~

Both branches reach D. The second visit to D is not a cycle. It is another path
to a shared dependency.

Cycle detection needs to distinguish three states:

- **unseen:** traversal has not started this node;
- **active:** the node is on the DFS path currently being explored;
- **finished:** every outgoing edge from the node has been explored.

An edge to an active node closes a cycle. An edge to a finished node reaches
work that has already been checked and does not imply a cycle.

For one connected search, the state changes are:

~~~text
unseen --enter DFS--> active --all outgoing edges checked--> finished
~~~

The transition to finished matters. If every previously seen node remained
active, the second route through a diamond would look indistinguishable from a
return to the current path.

Now suppose `lib/logging/BUILD.bazel` is changed so the logging library names
the HTTP library directly:

~~~starlark
go_library(
    name = "logging",
    srcs = ["logging.go"],
    importpath = "example.com/buildgraph/lib/logging",
    deps = ["//lib/http:http"],
    visibility = ["//visibility:public"],
)
~~~

This declaration would correspond to logging source importing HTTP. Together
with HTTP's existing import of logging, it is an invalid Go package cycle. The
two `deps` entries produce the closed path:

~~~text
//lib/http:http
    -> //lib/logging:logging
    -> //lib/http:http
~~~

DFS sees the last edge return to a target still on the current path. That is
the fact that distinguishes this cycle from shared node D in the diamond.

One way to express the state in Go is:

~~~go
type visitState uint8

const (
	unseen visitState = iota
	active
	finished
)
~~~

The DFS marks a node active when it enters and appends it to the current path.
When a call examines every dependency without finding a cycle, it removes the
node from that path and marks it finished. When it finds an active dependency,
it locates that node in the path, copies from that position to the current node,
and appends the active dependency again to close the cycle.

The central control flow can be written directly in Go:

~~~go
func (s *cycleSearch) visit(target TargetID) bool {
	s.state[target] = active
	s.path = append(s.path, target)

	for _, dependency := range s.graph.dependsOn[target] {
		switch s.state[dependency] {
		case unseen:
			if s.visit(dependency) {
				return true
			}
		case active:
			s.cycle = closeCycle(s.path, dependency)
			return true
		case finished:
			// This dependency was already checked on another path.
		}
	}

	s.path = s.path[:len(s.path)-1]
	s.state[target] = finished
	return false
}
~~~

`closeCycle` copies the suffix of `path` that begins with `dependency`, then
appends `dependency` once more. The internal witness is therefore a closed
node sequence such as `[A, B, C, A]`.

The public operation must also start from every disconnected component and
choose that starting order deterministically:

~~~go
func (g *Graph) FindCycle() []CycleEdge {
	search := newCycleSearch(g)

	for _, target := range g.sortedTargets() {
		if search.state[target] != unseen {
			continue
		}
		if search.visit(target) {
			return g.describeCycle(search.cycle)
		}
	}
	return nil
}
~~~

`sortedTargets` returns the node IDs in stable order; sorting only the
dependency slices would not make the first reported cycle deterministic.
`describeCycle` takes each adjacent pair in the closed node sequence, builds
its `DependencyEdge`, and copies the locations from `sources` into a
`CycleEdge`. The result identifies both the cycle and the declarations that
created it.

`visit` returns as soon as it copies one witness, so the successful call stack
does not finish changing its active states to finished. `FindCycle` does not
reuse that search object: it immediately describes the copied witness and
returns. Code that wanted to continue searching for additional cycles would
need to unwind or create a new search instead of reusing these states.

Starting from only one target would leave disconnected components unchecked.
Returning only “cycle detected” is enough for a boolean validity check but not
for a maintainer who must remove or change a declaration.

A self-dependency is the smallest cycle:

~~~text
//lib/config:config -> //lib/config:config
~~~

It has one edge even though its first and last node are the same.

{{< callout kind="warning" title="Cycle checks do not bound legitimate depth" >}}
Stopping when DFS returns to an active node prevents a cycle from causing
unbounded recursion. A recursive implementation still uses stack space
proportional to the longest acyclic path. If graph depth is untrusted or can be
very large, use an explicit stack or enforce a documented bound.
{{< /callout >}}

## BuildKit turned a cycle into an editable error

[BuildKit PR #999](https://github.com/moby/buildkit/pull/999) provides a direct
production example. A cyclic relationship between Dockerfile stages could
make `dockerd` recurse until the goroutine stack exceeded its limit. The patch
validated the completed stage-dependency graph and tracked the current path so
it could reject a circular dependency.

[BuildKit PR #4567](https://github.com/moby/buildkit/pull/4567) improved the
same failure path. Dependency edges retained source locations, and the error
showed the Dockerfile instructions involved in the cycle.

The two changes solve different parts of diagnosis:

1. stop traversal from following the same active cycle indefinitely; and
2. identify the declarations that created the bad edges.

The source establishes a daemon stack overflow caused by cyclic stage
dependencies. It does not establish that every recursive graph walk in
BuildKit had the same failure or that the event caused a wider production
outage.

## A DAG is a result of validation, not an assumption

A **directed acyclic graph**, or DAG, is a directed graph with no directed
cycle.

The dependency graph used to perform one configured build must be acyclic: a
target cannot be completed if satisfying it eventually requires the same
unfinished configured target again. Erroneous declarations and
graph-construction bugs can still produce cycles. The small fixture has no
configuration choices, so its validator begins with a directed graph:

~~~text
directed graph
    |
    +-- cycle found ------> invalid selected graph; report the path
    |
    +-- no cycle ---------> DAG; an ordering exists
~~~

Calling the input a DAG before checking it hides the condition that makes the
ordering possible. Bazel's post-loading `query` graph needs a separate caveat:
because it combines all possible `select()` results, it may contain an apparent
cycle that no single configured build contains. The configured-target
distinction is developed below.

A directed graph has a topological order exactly when it is acyclic. Cycle
detection and ordering are therefore related, but they answer different
questions for the caller. DFS can return an explanatory cycle path. An
ordering algorithm can detect that it processed too few nodes, but without
additional work it may report only that some cycle exists.

## Topological order follows the stored arrows

A **topological order** places the source of every directed edge before its
destination.

Our stored edge is:

~~~text
//app:server -> //lib/http:http
~~~

Therefore a topological order of the stored graph places the server before the
HTTP library. For the server's dependency closure, one valid topological order
is:

~~~text
//app:server
//lib/http:http
//lib/logging:logging
//lib/config:config
~~~

This is **dependent-first** order. Every target appears before the dependencies
it points to.

If each target were one indivisible unit of work, a dependency-respecting build
plan would need the opposite:

~~~text
//lib/config:config
//lib/logging:logging
//lib/http:http
//app:server
~~~

This is **dependency-first build order**. Every dependency appears before a
target that needs it.

This target-level build order is the reverse of a topological order of the
stored graph. It is equivalently a topological order of the graph with every
edge reversed.

The list is an order for the six selected rule targets, not a literal Bazel
execution schedule. During analysis, one Bazel target can produce several
actions, and those actions are the units execution schedules. The distinction
is developed below.

{{< callout kind="warning" title="Bazel query order is not execution order" >}}
With `--order_output=deps`, Bazel query prints a topological order of its
target-to-prerequisite graph: dependents first and dependencies afterward.
That output order describes the query graph. It is not the order in which
build actions execute.
{{< /callout >}}

The name `BuildOrder` is clearer for a learner-facing API than
`TopologicalOrder`. It states the operation's required direction instead of
requiring every caller to remember which way the stored arrows point.

## Derive dependency-first order from remaining counts

Restrict the selected graph to the requested roots—the targets named in the
build request—and their dependency closure. This keeps a shared dependency's
unrelated dependents out of the plan. For example, placing
`//lib/config:config` in the server's order must not pull
`//tool/migrate:migrate` into a request that named only the server.

For `//app:server`, begin with:

| Target | Direct dependencies not yet placed |
|---|---:|
| `//lib/config:config` | 0 |
| `//lib/logging:logging` | 0 |
| `//lib/http:http` | 1 |
| `//app:server` | 2 |

The **ready set** contains config and logging because neither needs another
target to appear earlier in this build order.

To compute an order without executing anything, repeat:

1. remove one target from the ready set and append it to the build order;
2. follow its `requiredBy` edges to targets in this requested closure that are
   still in the calculation;
3. decrement each such target's remaining-dependency count; and
4. add a target to the ready set when its count reaches zero.

Using lexicographic target-name order—the ordinary dictionary-like comparison
of the label strings—to break ties gives this trace:

| Step | Append to order | Newly ready | Ready afterward |
|---:|---|---|---|
| 1 | `//lib/config:config` | none | `//lib/logging:logging` |
| 2 | `//lib/logging:logging` | `//lib/http:http` | `//lib/http:http` |
| 3 | `//lib/http:http` | `//app:server` | `//app:server` |
| 4 | `//app:server` | none | empty |

The result is dependency-first. If the ready set contains several targets,
another choice can produce another valid order. A deterministic tie-break
makes runs reproducible; it does not make one order the only correct order.

This is Kahn's algorithm applied to the graph with the stored edges reversed.
The counts record which dependencies have not yet been placed earlier in the
order. The `requiredBy` index identifies exactly which counts change when a
dependency is appended. In this calculation, “remove” and “append” are
bookkeeping operations: no build action has started or completed.

If the process stops with nodes unprocessed and no ready target, the remaining
subgraph contains a cycle. A separate DFS cycle witness can identify the
closed path and its source declarations.

### The same counts can release work during execution

A scheduler can use similar counts, but the event that changes a count is now
real completion rather than removal from an abstract graph. After validating
the graph, initialize each action's count to its unfinished prerequisites. A
prerequisite that completes successfully—or whose result is satisfied from a
cache—can decrement the counts of actions that depend on it. Merely starting
the prerequisite must not decrement those counts.

Every action in the scheduler's ready set is allowed to proceed according to
dependency constraints. It does not follow that every ready action will run
immediately. A real build system also has:

- a finite number of workers;
- CPU, memory, network, and remote-execution limits;
- cached actions that need no execution;
- actions with different costs;
- failures and cancellation; and
- scheduling policy among equally ready actions.

“These actions are eligible to run concurrently” is therefore more accurate
than “these actions run at the same time.”

## Go's command builder uses the same ready-work idea

The Go command's builder is a useful implementation source because it shows
how a completed prerequisite reaches only the actions waiting for it:

- [`cmd/go/internal/work/action.go` at Go 1.26.5](https://cs.opensource.google/go/go/+/refs/tags/go1.26.5:src/cmd/go/internal/work/action.go)
- [`cmd/go/internal/work/exec.go` at Go 1.26.5](https://cs.opensource.google/go/go/+/refs/tags/go1.26.5:src/cmd/go/internal/work/exec.go)

An action records its prerequisite actions. The builder also constructs the
reverse relation, stored in a field named `triggers`: for each action, this is
the list of actions waiting for it. The builder counts pending prerequisites
and makes an action ready when that count reaches zero. The reverse list lets a
completed action update only its known dependents instead of scanning every
action in the build.

This production connection does not mean the foundations graph is a copy of
`cmd/go`. The Go tool includes caching, failure propagation, resource limits,
and many kinds of actions. The small model isolates one mechanism:

~~~text
unfinished prerequisite count reaches zero -> action may become ready
~~~

It also shows where the small target graph stops matching a full build system.
A build target is not necessarily one action. One target can generate several
compile, link, copy, or metadata actions.

## Bazel contains several related graphs

“The Bazel DAG” is too vague. A Bazel label does not identify the same kind of
node in every phase, and a `deps` attribute is not the only source of an edge.

| Graph or view | Nodes | Relationships | Public interface |
|---|---|---|---|
| Post-loading target graph | Rule targets and file targets | Rule inputs, including labels from `srcs`, `deps`, and `data`, plus implicit dependencies | `query` |
| Configured target graph | Configured targets: a target label together with a build configuration | Dependencies after configuration choices, transitions, and toolchain resolution | `cquery` |
| Action graph | Actions and artifacts | Which artifacts an action consumes and produces | `aquery` |
| Skyframe graph | Internal evaluation keys and values | Which computations must be reevaluated when an input changes | Bazel's incremental-evaluation internals |

The configured-target distinction is substantial: the same label may be
analyzed in more than one configuration and therefore represent more than one
configured target. A map keyed only by label would collapse those nodes if the
operation needed to distinguish them.

[Bazel's extension concepts](https://bazel.build/versions/9.1.0/extending/concepts)
describe three broad phases:

1. **Loading** evaluates needed BUILD and extension files and instantiates
   rule targets.
2. **Analysis** applies a build configuration, evaluates rule implementations,
   and creates actions from the configured targets.
3. **Execution** runs required actions.

The
[Bazel glossary](https://bazel.build/versions/9.1.0/reference/glossary)
defines an action as a command with declared input and output artifacts—the
files or file-like build objects that actions consume or produce. Its action
graph is produced during analysis and used during execution.

Traditional `bazel query` examines the post-loading target graph before Bazel
has applied one build configuration. Its graph is already broader than the
running example: source files named by `srcs` are targets, and implicit
dependencies are included by default. `--noimplicit_deps` suppresses implicit
dependencies, but it does not remove explicit `srcs`, `data`, or other rule
inputs.

A `select()` can name different dependencies for different platforms or build
options. Traditional query returns every possible resolution because it has
not selected one configuration, so its result can be a conservative
overestimate of any one build. `cquery` examines configured targets after
those choices and toolchains have been resolved. `aquery` exposes the actions
and artifacts created during analysis:

- [Bazel query reference](https://bazel.build/versions/9.1.0/query/language)
- [Bazel action graph query](https://bazel.build/versions/9.1.0/query/aquery)

Within the traditional target graph, Bazel's query operations make several
production questions concrete:

- `labels(deps, //app:server)` evaluates the server rule's named `deps`
  attribute; for the fixture above, it selects config and HTTP directly;
- `deps(//app:server)` returns the server target and its closure across all
  rule-input edges present in the query graph, not only `deps` attributes;
- `rdeps(//..., //lib/http:http)` follows edges in reverse, but only within the
  transitive dependency closure rooted at the main-repository rules matched by
  its first argument, `//...`;
- `somepath(//app:server, //lib/logging:logging)` returns one path from the
  server to logging, with no shortest-path guarantee; and
- `allpaths(//app:server, //lib/logging:logging)` returns the graph formed by
  targets on dependency paths from the server to logging.

The universe in `rdeps` is part of the question. A result means “reverse
dependencies found inside this stated set,” not “every possible consumer in
every repository.”

The distinction changes what a result means:

~~~text
bazel query 'deps(//app:server)'
~~~

asks about the broader target dependency closure visible to traditional query.
Its result can include `//app:main.go`, rules from external repositories, and
implicit dependencies. It is not the six-node graph drawn in this lesson.

~~~text
bazel aquery '//app:server'
~~~

asks about actions generated for the configured build. It can show commands,
inputs, outputs, and action mnemonics—the short names Bazel gives categories of
work. The two commands do not traverse interchangeable node types.

Traditional query deliberately continues when its pre-configuration graph
contains a cycle rather than reporting that cycle as a build error. Bazel
documents that a cycle formed by combining several possible configurations can
disappear after one configuration is selected. `cquery` and `aquery` do report
cycles in the configured target graph. This is another reason to ask which
graph an operation is querying before interpreting “cycle” or “DAG.”

{{< callout kind="note" title="Why Bazel belongs in a Go production course" >}}
Bazel is not implemented in Go. It belongs here because build graphs sit
inside tools that infrastructure engineers use to build and deliver software,
and its public documentation makes target dependencies and graph queries
unusually explicit. The local implementation remains Go, while BuildKit, the
Go command, Pulumi, Prometheus, and Argo provide Go source and patches.
{{< /callout >}}

## Correct algorithms still need an accurate graph

The algorithms so far assume that graph construction preserved the facts the
operation needs. The next cases break that assumption in four different ways:
a direct dependency is omitted, an analysis result is undetermined, two
concrete nodes receive one identity, or a multi-parent relationship is reduced
to one parent. BFS, DFS, and topological ordering cannot reconstruct
information that never reached the graph.

### Declared edges can omit actual dependencies

[Bazel's dependency documentation](https://bazel.build/versions/9.1.0/concepts/dependencies)
distinguishes two graphs:

- the **declared dependency graph** comes from build metadata; and
- the **actual dependency graph** contains what targets genuinely need to
  build or execute correctly.

For a correct build, every direct actual dependency must also be a direct
declared dependency. In set language, the actual graph must be a subgraph of
the declared graph.

In the running BUILD declarations, logging is already transitively reachable:

~~~text
declared:
//app:server -> //lib/http:http -> //lib/logging:logging
~~~

Now suppose `main.go` begins importing
`example.com/buildgraph/lib/logging` directly, but `app/BUILD.bazel` still lists
only config and HTTP. The source has introduced this direct requirement:

~~~text
actual but undeclared:
//app:server -----------------> //lib/logging:logging
~~~

Reachability does not repair the missing declaration. The server now needs
logging directly even though another path also reaches it. The correct BUILD
change is to add the logging target to the server's own `deps` list:

~~~starlark
deps = [
    "//lib/config:config",
    "//lib/http:http",
    "//lib/logging:logging",
]
~~~

`rules_go` enforces this principle while compiling: its
[import check](https://github.com/bazel-contrib/rules_go/blob/v0.60.0/go/tools/builders/importcfg.go)
compares Go source imports with the packages supplied by direct `deps` and
reports a missing strict dependency when they do not match. Other rule
implementations may enforce the boundary differently or incompletely, which
is why Bazel's general documentation still distinguishes the declared graph
from the graph the code actually needs.

No traversal algorithm can infer the omitted direct edge from the declared
graph. BFS, DFS, reverse reachability, and build ordering will all compute
correct answers for the incomplete input they received.

Extra declarations cause a different problem. Bazel's documentation warns that
redundant declared dependencies can make builds slower and binaries larger.
Declaring every possible dependency “to be safe” therefore has a cost and
cannot replace maintaining accurate direct dependencies.

{{< callout kind="production" title="The production connection" >}}
A build tool repeatedly needs to answer questions about declared
relationships: what a target needs, what depends on it, why one target reaches
another, whether declarations form a cycle, and which work has no unfinished
prerequisite. Adjacency lists support those repeated questions. They do not
prove that every necessary declaration is present. Constructing and checking
the graph are therefore part of correctness, not merely preparation for
traversal.
{{< /callout >}}

### Missing, empty, and undetermined are different states

A missing target is not the same as a known target with no dependencies. A
known target whose dependency analysis did not finish is a third case: the
program has a node, but it does not know whether that node is independent.

[Prometheus PR #15560](https://github.com/prometheus/prometheus/pull/15560)
fixes this distinction in its rule dependency controller. A Prometheus rule
group periodically evaluates recording or alerting rules; a recording rule can
produce a time series that another group reads. For this case, treat each rule
group as a node and `A -> B` as “group A depends on results produced by group
B.” That edge requires the controller to respect B before A when it schedules
their evaluation.

The controller had treated an undetermined relation like an empty relation. It
could therefore run rule groups concurrently even though dependency analysis
had not established that concurrent evaluation was safe.

One possible result type makes the distinction visible:

~~~go
type DependencyLookupState uint8

const (
	TargetMissing DependencyLookupState = iota
	DependenciesUndetermined
	DependenciesKnown
)

type DependencyLookup struct {
	State DependencyLookupState
	IDs   []TargetID
}
~~~

`TargetMissing` means the requested node does not exist in the graph.
`DependenciesUndetermined` means the target exists but the analysis could not
answer. `DependenciesKnown` with an empty `IDs` slice means the analysis
completed and found no dependencies under its stated rules. A Go API could
return an error for the first case instead; the important requirement is not
to encode all three cases as an empty slice.

What to do with an undetermined result depends on the operation. A scheduler
may keep the work sequential or blocked and expose a diagnostic. It should not
silently convert “could not determine” into “safe to run.”

### Node identity must distinguish concrete nodes

An adjacency list assumes each key names exactly one node. If two concrete
resources collapse to one key, the program can construct valid-looking edges
between the wrong things.

#### Pulumi: current and pending-deletion resources

[Pulumi PR #19179](https://github.com/pulumi/pulumi/pull/19179) describes a
snapshot that can contain:

~~~text
current A
B depends on A
old A, still awaiting deletion
~~~

In Pulumi's deletion graph, each node should represent one concrete resource
record. The edge `B -> A` means B depends on A, so A must remain while B still
exists. That relationship can be correct only if the endpoint identifies the
particular A that B uses.

Pulumi calls its logical resource identifier a **URN**, or uniform resource
name. The current and old resources may share that URN. A lookup with one entry
per URN associated B with the old A instead of the current A. Later deletion
logic could then permit the current A to be deleted, leaving B with a dangling
reference—that is, B would refer to a resource that no longer existed. The
patch tracked current and pending-deletion resources separately.

The graph lesson is not simply “use a longer string.” Define identity from the
facts the operation must distinguish. If current and pending-deletion copies
can coexist, the key must retain enough state to tell them apart.

Deletion also gives “dependency order” two possible meanings. If B depends on
A, creation ordinarily needs A before B, while deletion ordinarily needs B
before A. Name the operation and direction rather than saying only “sort the
dependencies.”

#### Argo: a hash collision manufactured a cycle

[Argo Workflows issue #16376](https://github.com/argoproj/argo-workflows/issues/16376)
concerns Argo's records for pieces of a workflow, such as Pods and grouping
nodes. In the recursive completion check, `A -> B` means record A names record
B as a child whose completion contributes to A's result.

Two distinct node names produced the same node ID under a 32-bit FNV-1a hash. A
hash converts a name into a fixed-size numeric value; a **collision** occurs
when two different names produce the same value. A lookup returned an unrelated
record, so the controller recorded an ancestor as one of its own descendants.
The completion check then followed that false child link back around the cycle
until the controller process overflowed its stack and restarted.

[Argo Workflows PR #16625](https://github.com/argoproj/argo-workflows/pull/16625),
still open on August 12, 2026, prevents two recursive walks from descending
into a child already on the current chain of calls. The function removes that
child from the set when the recursive call returns. A malformed link can no
longer make the function recurse forever.

The patch's tests include a diamond whose two branches share a child. That
child may be visited once through each branch because it is no longer on the
first chain when the second branch reaches it. This does not contradict the
three-state DFS used earlier. A pure cycle validator may mark a node finished
and skip its outgoing edges on a later visit because it has already proved
that subgraph acyclic. Argo's functions perform completion work along each
route; simply suppressing every later visit would change that operation. A
global completed-node cache would require a separate proof that the computed
result does not depend on the route.

This change prevents a malformed graph from crashing the controller. It does
not remove the source of the bad edge. Giving distinct node names distinct
identities, detecting a collision during graph construction, and stopping a
later recursive walk safely are three separate responsibilities.

### Keeping one parent discards the others

A workflow node can appear beneath several parents. Argo's retry code needed
to walk from a selected node back toward its ancestors, so it transformed the
parent-to-children relation into `map[child]parent`. That map has room for only
one parent per child; assigning another parent replaces the first. Reducing a
richer relation in this way is sometimes called a **projection**.

[Argo Workflows issue #16450](https://github.com/argoproj/argo-workflows/issues/16450)
documents that transformation while the code ranged over a Go map. For a node
with several parents, whichever parent was visited last replaced the previous
entry. Because Go does not define map iteration order, repeating the same retry
could choose a different parent chain. A **TaskGroup** is Argo's node for tasks
created by a loop. One resulting parent choice reset a TaskGroup to Running
without resetting the work that could complete it. The group therefore
remained Running after all Pods had finished.

Merged
[PR #16451](https://github.com/argoproj/argo-workflows/pull/16451)
sorts node IDs before building that one-parent map. The selected chain is now
stable across runs. The patch explicitly does not preserve every parent.

This supports two separate conclusions:

- an output that depends on Go map order is not deterministic; and
- deterministic selection of one parent does not restore the other parents.

Sometimes keeping one parent is appropriate, such as when an operation asks
for one explanation path rather than every path. Its contract must state how
that parent is selected and that the other valid parents are intentionally
omitted.

## Runtime service dependencies are not automatically a DAG

The clean build example does not justify forcing every dependency-shaped
production problem into the same model.

[GitHub's deployment-safety report](https://github.blog/engineering/infrastructure/how-github-uses-ebpf-to-improve-deployment-safety/)
describes circular dependencies in deployment tooling. A script needed to
repair an outage could itself call github.com, an internal service, or a tool
that contacted an unavailable system. Such a dependency may remain hidden
until an incident delays recovery.

The account begins with a Go proof of concept and describes a production
system that uses Linux's eBPF facility to observe and optionally block network
calls from deployment scripts. It does not topologically sort a general
service graph.

That is the important boundary:

- services can call one another in a cycle without every call being a design
  error;
- the path by which one service failure affects another is not necessarily the
  set of steps required to recover them;
- services, deployment scripts, data stores, and repair actions are different
  possible node types; and
- recovery may require a prebuilt tool or package, or a control route that
  still works when the affected services do not.

For recovery planning, ask:

> Which actions must succeed for this specific recovery operation, and what
> does each action require?

If those action constraints form a cycle, the operation needs a design change
or an independent recovery route that breaks the cycle. The existence of a
cyclic service relationship alone does not prove that the services are
incorrectly designed.

## Account for both nodes and edges

The production cases above establish what must be true of the input graph.
With those limits stated, we can return to the adjacency-list algorithms and
account for the work they perform after a valid requested subgraph exists.

Let `V` be the number of nodes reached in the requested subgraph, `E` the
number of edges among them, `d` the number of direct dependencies of one
target, and `S_cycle` the number of source-location records attached to a
returned cycle.

| Operation | Time | Additional working space |
|---|---:|---:|
| Add one deduplicated edge | expected `O(1)` | `O(1)`, excluding evidence text |
| List direct dependencies | `O(d)` | `O(d)` for a copied result |
| Dependency closure with BFS or DFS | `O(V + E)` | `O(V)` |
| Fewest-edge path with BFS | `O(V + E)` | `O(V)` |
| One cycle witness with DFS | `O(V + E + S_cycle)` | `O(V + S_cycle)` |
| Dependency-first order with an unordered ready set | `O(V + E)` | `O(V)` |
| Dependency-first order with a min-heap tie-break | `O(E + V log V)` | `O(V)` |

The `O(V + E)` traversal bound comes from processing each reached node once and
examining each outgoing edge of those nodes once. Saying only “linear time” is
ambiguous because a graph can have far more edges than nodes.

For a cycle report, DFS still examines at most `O(V + E)` graph data. Copying
the source evidence into the returned diagnostic adds `O(S_cycle)` time and
space.

These bounds apply to one traversal after the adjacency lists exist. Loading,
validating, deduplicating, and sorting declarations have their own costs.
Repeatedly asking the same query can also justify another index or cached
result, but cache invalidation must follow graph updates correctly.

The expected constant-time edge insertion assumes a hash-set index and target
IDs whose length is bounded, so computing one hash is treated as constant
work. It is the usual average-case claim for a hash table, not a worst-case
guarantee for every possible set of keys. Searching a slice to detect a
duplicate would instead take time proportional to that target's existing
direct dependencies.

{{< callout kind="note" title="Be honest about deterministic tie-breaking" >}}
Basic Kahn traversal is linear when the ready set may return any available
node. Requiring the lexicographically smallest ready target can add work. A
min-heap—a priority queue that returns the smallest target name—pushes and
removes each node at most once, producing
`O(E + V log V)` time. A small teaching implementation may instead sort a
slice, but it should state the cost it actually pays.
{{< /callout >}}

## Test the graph contract before benchmarking it

Graph tests should make the relationship and its edge cases visible.

### Use small named examples

Cover at least:

- a chain: `A -> B -> C`;
- a diamond with a shared dependency;
- two disconnected components;
- an isolated requested target;
- a self-dependency;
- a cycle of several nodes;
- duplicate declarations with the same dependent and dependency;
- an edge that names an unknown target; and
- two concrete records that would collide under an incomplete identity.

Small graphs are easy to draw, and the expected paths and orders can be checked
by inspection.

### Check every returned order

For every stored `target -> dependency` edge in the requested closure, assert:

~~~text
position(dependency) < position(target)
~~~

Also assert that the order contains each node in the requested closure exactly
once and no disconnected node unless it was explicitly requested.

Do not assert one entire order when several are valid unless deterministic
tie-breaking is part of the API contract.

### Check cycle witnesses

A returned cycle must:

- contain at least one edge;
- form a chain in which, for each adjacent pair, the first edge's dependency
  is the second edge's dependent;
- have a last dependency equal to the first dependent;
- use only edges present in the graph;
- preserve edge source locations; and
- report a self-edge as a one-edge cycle.

A diamond must not be reported as a cycle.

### Check path promises

If `FewestEdgePath` promises a fewest-edge path, verify every adjacent pair in
the result is a declared edge. On tiny generated graphs, a separate test can
enumerate all paths that do not repeat a node and compare the returned length
with the smallest one. Exhaustive enumeration is too expensive for production,
but on deliberately small test graphs it provides an independent check of the
BFS result. Verify also that an unreachable known target returns “no path,”
while an unknown target receives the API's distinct missing-target result.

### Check determinism deliberately

Build equivalent graphs from declarations in different input orders and call
the public operations repeatedly. The returned direct-neighbor lists, equal
length paths, cycle witnesses, and build plans should remain stable if the API
promises stable output.

### Measure after correctness

Count reached nodes and examined edges before comparing elapsed time. Then
benchmark sparse chains, graphs where one node has many outgoing edges,
diamonds with shared dependencies, and invalid cycles separately. A benchmark
of only one shape does not establish a general graph cost.

## Read each source for one role

This lesson uses formal sources, product documentation, source code, issue
reports, and patches. They establish different things:

| Source | What it establishes |
|---|---|
| [MIT 6.006 Lecture 9](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/196a95604877d326c6586e60477b59d4_MIT6_006S20_lec9.pdf) | Directed graph representation, reachability, BFS, fewest-edge distance, and `O(V + E)` analysis |
| [MIT 6.006 Lecture 10](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/f3e349e0eb3288592289d2c81e0c4f4d_MIT6_006S20_lec10.pdf) | DFS, directed cycles, DAGs, and topological order |
| [Open Data Structures: Graphs](https://opendatastructures.org/ods-go/12_Graphs.html) | A second, Go-oriented treatment of adjacency lists and traversal |
| [Bazel BUILD files](https://bazel.build/versions/9.1.0/concepts/build-files) and [labels](https://bazel.build/versions/9.1.0/concepts/labels) | BUILD-file evaluation, rule-target creation, packages, target names, and label structure |
| [Bazel rule concepts](https://bazel.build/versions/9.1.0/extending/rules) | How label-bearing attributes, private attributes, toolchains, rule analysis, and actions create different relationships |
| [Bazel dependencies](https://bazel.build/versions/9.1.0/concepts/dependencies) | Bazel's target-dependency contract and declared-versus-actual distinction |
| [Bazel query reference](https://bazel.build/versions/9.1.0/query/language) | Query graph direction, `deps`, `rdeps`, paths, cycles, configurations, and output order |
| [`rules_go` v0.60.0 rule reference](https://github.com/bazel-contrib/rules_go/blob/v0.60.0/docs/go/core/rules.md) | The actual `go_binary` and `go_library` attributes used by the fixture and the direct-import meaning of `deps` |
| [`rules_go` v0.60.0 import check](https://github.com/bazel-contrib/rules_go/blob/v0.60.0/go/tools/builders/importcfg.go) | The pinned implementation that checks source imports against direct dependencies |
| Pinned Go source | What one named Go release implements |
| Public issue report | What a reporter observed and how they reproduced it |
| Merged pull request | What maintainers changed, not proof that one patch explains every similar symptom |
| Open pull request | A proposed and reviewable change, not released behavior |

The
[MIT BFS recitation](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/resources/mit6_006s20_r09/)
and
[DFS recitation](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/resources/mit6_006s20_r10/)
provide supplementary worked problems.

## Translate the model in a design conversation

A compact answer to “how would you model and schedule these build
dependencies?” might be:

> I would first choose the graph. For this example, a node is one of the six
> `rules_go` rule targets, and `A -> B` means A's explicit `deps` attribute
> names B. That excludes source-file, implicit, configured-target, toolchain,
> action, and artifact relationships. A production integration should obtain
> its chosen graph through Bazel's supported query interfaces, not by scraping
> BUILD text.
>
> I would retain forward and reverse adjacency lists. Forward BFS can return a
> fewest-edge explanation path, while reverse traversal can find declared
> dependents. DFS can return a closed cycle. If the input provides trustworthy
> source evidence for an edge, the diagnostic can preserve it; implicit or
> macro-generated edges may have only a coarser location.
>
> For a valid requested subgraph, I would count each target's dependencies and
> append it to a dependency-first order only after those dependencies have
> appeared. Those traversals take `O(V + E)` time; a min-heap used to select
> the smallest ready target adds `O(V log V)`. Before a scheduler uses the
> result, I would validate node identity and distinguish declared, actual,
> missing, empty, and undetermined relationships. I would also distinguish
> placing a target in a calculated order from completing an action: running
> work releases a dependent only after its prerequisite has completed
> successfully or a cache has supplied its result.

That answer states the model, operations, direction, costs, correctness checks,
and limits. It does not claim that the graph algorithm is a complete build
system.

## What comes next

The planned Build Graph Explorer lab will turn the running graph into a small
Go API for direct dependencies, reverse dependencies, explanation paths, cycle
witnesses, ready work, and dependency-first order. Its input will be explicit,
already identified target and edge records based on the fixture above. Parsing
or evaluating BUILD files is a separate Bazel-integration problem and is not
silently included in that lab. The planned Wheel scenarios will use the
BuildKit, Prometheus, and Pulumi failures without placing their solutions in
the incoming reports.

Those exercises are not published yet. The foundations lesson stands on its
own and links only to the production and supplementary sources already
available.

## Reflection

For each answer, name the graph and edge meaning you are using:

1. How do `app/BUILD.bazel`, package `app`, and `name = "server"` combine to
   identify `//app:server`?
2. Why does `srcs = ["main.go"]` create a relationship in Bazel's target graph
   even though the running teaching graph omits it?
3. If `main.go` imports logging directly, why must the server declare logging
   directly instead of relying on the path through HTTP?
4. Why can `bazel query 'deps(//app:server)'` return more targets than the
   direct-Go-dependency graph drawn here?
5. Under this unit's convention, why does following `dependsOn` answer a
   different question from following `requiredBy`?
6. Why does a second visit to a shared child in a diamond not prove a cycle?
7. What additional state lets DFS distinguish that shared child from an edge
   back to the current path?
8. Why is a dependency-first build order the reverse of a topological order of
   the stored explicit-`deps` edges?
9. Why does appending a target to a calculated order not mean its work has
   completed? What does a scheduler's ready set prove about concurrent work?
10. Why can `bazel query`, `cquery`, and `aquery` return information about
   different graphs for the same target label?
11. What went wrong when Pulumi used a shared URN as the complete identity of
   both current and pending-deletion resources?
12. Which problem did Argo's deterministic parent selection fix, and which
   information did the one-parent map still discard?
13. Why does GitHub's circular deployment dependency call for an independent
    recovery route rather than a blanket rule that runtime service graphs must
    be DAGs?

The durable production habit is:

> Derive the graph from real declarations, state what the graph omits, and
> verify it before trusting an algorithm's answer.
