+++
title = 'Build dependency graphs'
description = 'A build graph can answer dependency questions only when its nodes, arrows, identities, and ordering rules are clear.'
weight = 4
+++

**Unit 04 · Foundations**

# Build dependency graphs

{{< lead >}}A dependency graph lets a program answer questions about build
declarations such as `deps = ["//lib/http:http"]`. What does a target need?
Why does it need it? Which work can start now? The answers are useful only when
we say what each node and arrow means, and what the graph leaves out.{{< /lead >}}

{{< callout kind="production" title="Incoming reports" >}}
- [BuildKit PR #999](https://github.com/moby/buildkit/pull/999) documents a
  cycle between Dockerfile stages. `dockerd` followed the cycle until its
  goroutine stack overflowed. A later
  [patch](https://github.com/moby/buildkit/pull/4567) made the error name the
  Dockerfile instructions in the cycle.
- [Argo Workflows issue #16450](https://github.com/argoproj/argo-workflows/issues/16450)
  describes a retried workflow that remained Running after every Pod had
  finished. Retry code kept only one parent for a node that could have several.
  Go map order could change which parent it kept.
- [Pulumi PR #19179](https://github.com/pulumi/pulumi/pull/19179) fixes a
  deletion graph that confused two resources. One was current and one was an
  older copy awaiting deletion, but both had the same Pulumi resource name,
  called a URN.
{{< /callout >}}

These reports all involve graphs, but they have different causes. BuildKit did
not stop before following a cycle. Argo discarded all but one parent, then let
Go map order choose the parent it kept. Pulumi joined the wrong specific
resources. Saying “it uses a graph” does not explain any of these failures.
Ask instead:

- What does one node represent?
- What fact does one directed edge record?
- Can two distinct things receive the same identity?
- Is the relationship known to exist, known not to exist, or still unknown?
- Which direction does an operation need to follow?

This unit starts with Bazel target dependencies because Bazel defines that
relationship clearly. The code examples use Go.

By the end of the unit, you should be able to:

- read a Bazel target label and a direct dependency from an actual
  `BUILD.bazel` rule call;
- choose an exact graph from rule attributes that name dependencies;
- describe the nodes and arrows in a complete sentence about the real system;
- represent outgoing and incoming relationships with adjacency lists;
- distinguish a direct dependency from a reachable transitive dependency;
- use breadth-first search to return a path with the fewest edges;
- use depth-first search to distinguish a cycle from a harmless second visit;
- explain when a directed graph is a DAG;
- distinguish graph-theory topological order from dependency-first build order;
- calculate dependency-first order from remaining-dependency counts and explain
  how completed prerequisites release real work;
- state the time and space cost in terms of both nodes and edges;
- distinguish repeatable output from an answer that is mathematically unique;
- explain why a correct algorithm cannot fix missing, unclear, or wrongly
  identified edges; and
- distinguish this lesson's direct Go-library graph from Bazel's broader
  target, configured-target, and action graphs.

## Start with what Bazel actually reads

A Bazel build does not start as a list of arrows. Bazel evaluates each
[`BUILD.bazel` file as Starlark](https://bazel.build/versions/9.1.0/concepts/build-files),
a restricted programming language. A call to a rule function creates a target.
The rule defines what attributes such as `srcs` and `deps` mean and which value
types they accept.

The following teaching workspace is not taken from a production incident. It
uses the public rule interface for
[`rules_go` v0.60.0](https://github.com/bazel-contrib/rules_go/blob/v0.60.0/docs/go/core/rules.md).
The example is small enough to trace by hand. The arrows we draw are not Bazel
syntax.

Assume the repository makes that pinned version of `rules_go` available in
`MODULE.bazel`:

~~~starlark
bazel_dep(name = "rules_go", version = "0.60.0")
~~~

The pinned
[`rules_go` Bzlmod documentation](https://github.com/bazel-contrib/rules_go/blob/v0.60.0/docs/go/core/bzlmod.md)
explains the module setup and Go SDK choices. The line above makes the rule
definitions in the following BUILD examples available. The choice of Go SDK
is not part of our six-node graph.

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
- Calling `go_binary` creates a rule target. This file belongs to the `app`
  package. `name = "server"` names the target, so its full label is
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

The Go imports show why those dependencies exist. These excerpts omit the code
that uses the imports, so they are not complete source files:

~~~go
// app/main.go
import (
	"example.com/buildgraph/lib/config"
	apphttp "example.com/buildgraph/lib/http"
)

// lib/http/http.go
import "example.com/buildgraph/lib/logging"
~~~

For these `rules_go` rules, `deps` names the Go libraries that the target's
package imports directly. `main.go` imports config and HTTP, so the server lists
both. `http.go` imports logging, so HTTP lists logging. A package that the code
imports directly must not be available only through another library. Declaring
each direct dependency keeps the build correct when an intermediate library
changes its own dependencies.

{{< callout kind="contract" title="A direct-dependency list has two requirements" >}}
For the `deps` attributes in this example:

1. **List every direct dependency:** if the source imports a Go package outside
   the standard library, its library target must appear in `deps`.
2. **Do not copy indirect dependencies into the list:** a library does not
   belong in `deps` merely because another dependency imports it.

The server lists config and HTTP, but not logging. HTTP lists logging. Logging
is still an indirect, or **transitive**, dependency of the server. The server's
current source does not import it directly. Bazel's
[dependency guide](https://bazel.build/versions/9.1.0/concepts/dependencies)
says to declare every actual direct dependency and no extra ones. Attributes
such as `data`, `embed`, and `cdeps` have their own meanings. Do not treat them
as though they were all `deps`.
{{< /callout >}}

The code above gives us these three relationships:

~~~text
//app:server -> //lib/config:config
//app:server -> //lib/http:http
//lib/http:http -> //lib/logging:logging
~~~

These arrows come from the `deps` attributes. They do not appear in the
`BUILD.bazel` files. In this example, each arrow means:

> `A -> B` means the explicit `deps` attribute of `rules_go` target A directly
> names `rules_go` library target B.

The arrow starts at the target that contains `deps` and ends at the target
named in that list. This is the same target-to-prerequisite direction used by
[Bazel's query language](https://bazel.build/versions/9.1.0/query/language),
but Bazel's query graph includes more kinds of targets and dependencies than
our small graph.

### Select the graph before choosing the algorithm

`deps` does not mean “the whole Bazel graph.” Many rule attributes accept
labels and create dependencies. In the server declaration, `srcs` also links
the rule to `//app:main.go`. `data`, private rule attributes, build
configuration, and toolchain selection can add more relationships.

This lesson deliberately selects a smaller graph:

| Question | Bazel's broader graph family | Running teaching graph |
|---|---|---|
| What is a node? | Rule targets, source-file targets, generated-file targets, configured targets, actions, or artifacts, depending on the query | The six named `go_binary` and `go_library` rule targets |
| What creates an edge? | Attributes that contain labels and, in later phases, selected implicit and toolchain dependencies or action inputs and outputs | Only the explicit `deps` attributes shown in these `rules_go` calls |
| What does it leave out? | It depends on the interface: `query` has no configured targets or actions, `cquery` does not show the details inside actions, and `aquery` answers questions about actions | `srcs`, generated files, `data`, implicit dependencies, toolchains, configurations, actions, and artifacts |

This smaller graph is enough to learn traversal, cycle detection, and ordering.
It is not Bazel's complete internal graph.

{{< callout kind="warning" title="Do not parse BUILD files as if they were data tables" >}}
Bazel evaluates a `BUILD.bazel` file as Starlark. A macro can create a rule. A
`select()` can choose labels for one build configuration. A rule definition can
add dependencies that never appear in a literal `deps` list. A tool that needs
Bazel's graph should therefore use Bazel's supported `query` or `cquery`
interfaces and read structured output. `query --output=proto` is one supported
machine-readable format. The text from `query --output=build` looks like BUILD
syntax after Bazel expands macros and variables, but Bazel does not promise
that this text is a valid BUILD file. The Go model in this unit starts with
known targets and edges. It does not parse BUILD files.
{{< /callout >}}

A **node**, also called a vertex, is one thing in the graph. A **directed edge**
is a one-way relationship between two nodes. The drawing alone cannot tell us
what those things and relationships mean. The graph's **contract** states its
meaning and rules.

The edge does not mean that the server executes before the library, that
network traffic travels toward the library, or that failure must propagate
along the arrow. Those would be different relationships and therefore
different graphs.

Another system may store prerequisite-to-dependent edges instead:

~~~text
//lib/http:http  ---->  //app:server
~~~

That convention is also valid, but forward traversal and ordering now move in
the other direction. Problems begin when the text, diagrams, and code switch
between conventions without saying so.

{{< callout kind="contract" title="Complete the edge sentence" >}}
Do not document an API only as “Add an edge from A to B.” State the real fact
stored by the edge: “the explicit `deps` attribute of Go target A names Go
library B,” “action A must finish before action B,” or another precise rule.

The running direct-Go-dependency graph and the local Go API use
target-to-dependency edges. Later production cases state a different edge
meaning when they examine a different graph.
{{< /callout >}}

## Add the remaining targets

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

If we keep only the explicit `deps` relationships among these six targets, we
get the graph used throughout this lesson:

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

A **path** is a series of connected directed edges. The path

~~~text
//app:server -> //lib/http:http -> //lib/logging:logging
~~~

has length two because it contains two edges.

A node is **reachable** from a starting node if we can follow zero or more
arrows to it. A zero-edge path makes every node reachable from itself. A
target's **dependency closure** contains that target and every node reachable
from it. Bazel's `deps` operator uses the same rule on Bazel's larger target
graph, so it can return more nodes than this lesson does. The other nodes
reached after one or more edges are **transitive dependencies**.

The dependency closure of `//app:server`, shown here as a set, is:

~~~text
//app:server
//lib/config:config
//lib/http:http
//lib/logging:logging
~~~

It does not include `//tool/migrate:migrate` just because both targets depend
on `//lib/config:config`. No arrow from the server leads to the migration tool.
The disconnected lint target is not included either.

This unit uses finite, directed, unweighted graphs:

- **finite** means the graph has a fixed number of nodes and edges when the
  query starts;
- **directed** means `A -> B` and `B -> A` are different relationships; and
- **unweighted** means path length counts edges, not time, money, failure
  probability, or another cost.

Weighted shortest paths, undirected connectivity, and graph databases solve
different problems and are outside this unit.

## Store the next relationships, not every possible pair

An **adjacency list** stores the next nodes that each node points to. In this
graph, the forward list is named `dependsOn`. An outgoing edge starts at the
node whose list contains it. The same edge is incoming where its arrow ends.

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

These maps are two indexes for the same graph. One follows the selected `deps`
arrows forward; the other follows them backward. The input can also keep the
source location of each declaration. An error can then point to text that a
maintainer can change:

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

The target ID slices keep graph-walk data small. `sources` stores the source
information for each dependent/dependency pair. For adjacent nodes `A` and `B`
in a path, the edge key is `{Dependent: A, Dependency: B}`. A `CycleEdge` joins
that key with every known source location for the edge.

Source evidence has limits. An explicit label may point to an editable line in
a `BUILD.bazel` file. A dependency created by a macro or rule may not have one
literal `deps` entry. A Bazel-backed tool must report only the locations that
its query interface supplies. If Bazel reports an edge only at the rule or
macro level, the tool must not invent an exact source line.

The direction to follow depends on the question:

| Question | Index |
|---|---|
| What does the server need? | `dependsOn` |
| Why does the server need logging? | `dependsOn` |
| Which declared targets rely on config? | `requiredBy` |
| Which targets might be affected by changing HTTP? | `requiredBy` |

“Might be affected” is deliberate. Walking the `requiredBy` edges finds
declared dependents. This is called **reverse reachability**. It does not prove
that each target will run, rebuild, fail, or affect a user. This graph does not
include cache results, build configurations, runtime paths, or the contents of
the change.

The adjacency lists keep one entry for each node and edge. `V` nodes and `E`
edges therefore use `O(V + E)` space. If the graph also keeps `S` source
records, total space is `O(V + E + S)`. An adjacency matrix takes a different
approach: it reserves one cell for every possible ordered pair of nodes. That
uses `O(V²)` space before adding source evidence. A matrix can check one cell
to answer “does this exact edge exist?” Build graphs often contain far fewer
edges than the number that could exist. Such graphs are called **sparse**, and
usually should not reserve space for every absent edge.

### Keep nodes with no edges

If the builder discovers nodes only by reading edges, `//tool/lint:lint`
disappears. The graph can then omit a requested target simply because it has no
selected `deps` relationships.

Store the node set separately. An empty list means “this target is known and
has no outgoing edge in this graph.” A missing map key must not mean both
“known and empty” and “unknown.”

This lesson rejects an edge if either end names an unknown target. Another
system could create a temporary placeholder node instead. If it does, callers
must still be able to tell that placeholder from a fully loaded target.

### Store a repeated edge once

Two declarations for the same dependent/dependency pair must not make a target
wait twice. While loading, the builder uses a set to **deduplicate** the edge,
which means storing it only once. It produces slices for traversal afterward.

If the input supplies several source records for one edge, keep every relevant
location under one `DependencyEdge`. This does not mean Bazel allows one target
label to be declared independently in two BUILD files; it does not. The local
graph combines evidence records. It does not change Bazel's identity rules.

### Do not expose Go map order as graph order

The [Go specification](https://go.dev/ref/spec#For_statements) does not define
map iteration order. If public results, tests, or errors depend directly on
that order, the same valid input can produce different paths or build plans.

Sort public results, or store waiting nodes in a structure with a clear order.
Output that stays the same from run to run is **deterministic**. That behavior
is an API choice. A graph can allow several valid traversals or topological
orders; the graph itself does not select one.

## A graph walk follows selected arrows

A **graph traversal**, or graph walk, starts at one or more nodes and follows
selected edges. It records the nodes that it reaches. This avoids checking the
same shared branch again and again. It also prevents a cycle from causing an
endless walk.

Two common traversals choose the next discovered node differently:

- **breadth-first search**, or BFS, explores all nodes one edge away before
  nodes two edges away;
- **depth-first search**, or DFS, follows one path as far as it can before
  returning to try another branch.

Use BFS when the API promises a path with the fewest edges. DFS is useful when
the operation must follow one branch to its end. A cycle check, for example,
must know whether an edge returns to the path being explored now. Choose the
kind of walk from the answer that the operation must provide.

The explanation in
[MIT 6.006 Lecture 9](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/196a95604877d326c6586e60477b59d4_MIT6_006S20_lec9.pdf)
uses this sequence: define the nodes and directed edges, choose how to store
them, define paths and reachability, and then introduce BFS.
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

The first time BFS reaches logging, it has found a path with the fewest edges.
A shorter path would have ended in a layer that BFS already explored. A
`parent` map records how the search first reached each node. The code follows
those entries from logging back to the server, then reverses the list:

~~~text
//app:server -> //lib/http:http -> //lib/logging:logging
~~~

Assume the builder sorts each dependency list. If several shortest paths exist,
the code then makes the same choice each run. The following operation also
handles the zero-edge path directly:

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

The head index moves through the queue without copying the remaining items on
each removal. The code marks a dependency as seen when it enters the queue.
This prevents two parents from adding the same node before either copy is
processed. The first parent remains the one used to rebuild the path.

The example leaves out unknown-target errors so the traversal is easy to see.
A public API must distinguish an unknown target from a known target that cannot
reach the destination.

`FewestEdgePath(A, A)` returns `[A]` because a node is reachable from itself.
An operation named `WhyDepends` should not use that zero-edge path to claim
that A depends on itself. It can require two different target IDs. Callers that
need a non-empty path from A back to A should use cycle detection. This clear
rule prevents a self-edge or longer cycle from being hidden as “no path.”

The path is shortest only by number of edges. It may not be the fastest build
path, the riskiest path, or the path with the fewest source files. Those
questions need more data, such as duration or risk on each edge, and a rule for
using it.

{{< callout kind="contract" title="The local promise is stronger than Bazel somepath" >}}
Bazel's `somepath` query returns one dependency path. Its documentation does
not promise the shortest one. This Go API makes a stronger promise by using
BFS. That is the local API's contract, not a claim about Bazel.
{{< /callout >}}

## One “seen” set cannot detect directed cycles

A **directed cycle** is a path with at least one edge that returns to its
starting node.

A closed path that proves a cycle exists is called a **cycle witness**. It gives
a maintainer something to inspect instead of saying only that the graph is
invalid.

Consider a separate four-node teaching graph shaped like a diamond. It is not
another BUILD declaration from the running example:

~~~text
    +----> B ----+
    |            |
A --+            +----> D
    |            |
    +----> C ----+
~~~

Both branches reach D. Visiting D a second time does not form a cycle; it is
another path to a shared dependency.

Cycle detection needs to distinguish three states:

- **unseen:** the search has not started this node;
- **active:** the node is on the DFS path currently being explored;
- **finished:** the search has checked every outgoing edge from the node.

An edge to an active node closes a cycle. An edge to a finished node reaches a
branch that has already been checked; it does not prove a cycle.

For one DFS starting point, the state changes are:

~~~text
unseen --enter DFS--> active --all outgoing edges checked--> finished
~~~

The change from active to finished matters. If every seen node stayed active,
the second branch of a diamond would look like a return to the current path.

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

This declaration would mean that logging imports HTTP. HTTP already imports
logging, so the two packages now form an invalid cycle. Their `deps` entries
produce this closed path:

~~~text
//lib/http:http
    -> //lib/logging:logging
    -> //lib/http:http
~~~

The last edge returns to a target still on the current DFS path. Shared node D
in the diamond was already finished when the second branch reached it.

One way to express the state in Go is:

~~~go
type visitState uint8

const (
	unseen visitState = iota
	active
	finished
)
~~~

DFS marks a node active when it enters the node and adds it to the current path.
After checking every dependency without finding a cycle, it removes the node
from the path and marks it finished. If it finds an active dependency, it
copies the part of the path that starts at that dependency. It then adds the
dependency once more to close the cycle.

The main part of the check can be written directly in Go:

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

`closeCycle` performs that copy. The result is a closed node list such as
`[A, B, C, A]`.

The public operation must check every disconnected part of the graph. It also
needs a stable order for choosing where to start:

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

`sortedTargets` returns node IDs in stable order. Sorting only each dependency
list would not make the first reported cycle stable across disconnected parts.
`describeCycle` turns every adjacent pair in the closed path into a
`DependencyEdge`. It copies the matching locations from `sources` into each
`CycleEdge`. The result names both the cycle and the declarations that created
it.

`visit` returns as soon as it copies one cycle. The earlier calls in the chain
do not get a chance to mark their nodes finished. This is safe here because
`FindCycle` immediately describes the copied path and discards the search
state. Code that continues looking for more cycles must first let those calls
finish normally or start a new search.

Starting from only one target could miss a cycle elsewhere. “Cycle detected”
is enough for a yes-or-no check, but not for a maintainer who must change a
declaration.

A self-dependency is the smallest cycle:

~~~text
//lib/config:config -> //lib/config:config
~~~

It has one edge even though its first and last node are the same.

{{< callout kind="warning" title="A cycle check does not prevent every stack overflow" >}}
Stopping at an active node prevents a cycle from causing endless recursion. A
recursive function still uses stack space for the longest path even when the
graph has no cycle. If input can contain very long paths, use your own stack
data structure or set and document a maximum path length.
{{< /callout >}}

## BuildKit turned a cycle into an editable error

[BuildKit PR #999](https://github.com/moby/buildkit/pull/999) shows this problem
in production code. A cycle between Dockerfile stages could make `dockerd`
keep calling the same recursive function until the goroutine ran out of stack
space. The patch checked the completed stage-dependency graph. It tracked the
current path so that it could reject a cycle.

[BuildKit PR #4567](https://github.com/moby/buildkit/pull/4567) later improved
the same error. It kept a source location with each dependency edge, so the
error could show the Dockerfile instructions that created the cycle.

The two changes solve different parts of the problem:

1. stop the graph walk from following the same cycle forever; and
2. show which declarations created the bad edges.

The pull requests show a daemon stack overflow caused by a cycle between
stages. They do not show that every recursive graph walk in BuildKit had this
problem or that it caused a wider production outage.

## A DAG is a result of validation, not an assumption

A **directed acyclic graph**, or DAG, is a directed graph with no directed
cycle.

The dependency graph for one configured build must have no cycle. A target
cannot finish if it eventually needs that same unfinished target again. A bad
declaration or a bug that builds the graph can still create a cycle. The small
example has no configuration choices, so its check begins with a directed
graph:

~~~text
directed graph
    |
    +-- cycle found ------> invalid selected graph; report the path
    |
    +-- no cycle ---------> DAG; an ordering exists
~~~

Do not call the input a DAG before checking it. The absence of a cycle is what
makes ordering possible. Bazel's post-loading `query` graph is a special case.
It combines every possible result of `select()`, so it can show a cycle that
does not exist in any one configured build. A later section explains this
difference.

A directed graph has a topological order if and only if it has no cycle. Cycle
detection and ordering are related, but they answer different questions. DFS
can return a path that shows a cycle. An ordering algorithm can notice that it
processed too few nodes, but it may report only that some cycle exists unless
we add more work.

## Topological order follows the stored arrows

A **topological order** places the source of every directed edge before its
destination.

Our stored edge is:

~~~text
//app:server -> //lib/http:http
~~~

A topological order of the stored graph therefore places the server before the
HTTP library. For the server and all the dependencies it can reach, one valid
order is:

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

This target-level build order reverses a topological order of the stored
graph. Another way to say this is that it is a topological order after every
edge has been reversed.

This list orders the six selected rule targets. It is not the schedule that
Bazel will run. During analysis, one Bazel target can produce several actions.
Bazel schedules those actions, not the targets themselves. A later section
explains the difference.

{{< callout kind="warning" title="Bazel query order is not execution order" >}}
With `--order_output=deps`, Bazel query prints a topological order of its
target-to-prerequisite graph: dependents first and dependencies afterward.
That output order describes the query graph. It is not the order in which
build actions execute.
{{< /callout >}}

For this API, `BuildOrder` is clearer than `TopologicalOrder`. The name states
the direction we need. Callers do not have to remember which way the stored
arrows point.

## Build dependency-first order from remaining counts

Start with the requested roots: the targets named in the build request. Add
only the dependencies that those targets can reach. This keeps unrelated users
of a shared dependency out of the plan. For example, a request for the server
includes `//lib/config:config`, but it must not also pull in
`//tool/migrate:migrate`.

For `//app:server`, begin with:

| Target | Direct dependencies not yet placed |
|---|---:|
| `//lib/config:config` | 0 |
| `//lib/logging:logging` | 0 |
| `//lib/http:http` | 1 |
| `//app:server` | 2 |

The **ready set** contains config and logging. Neither one needs another target
to appear before it in this build order.

To compute an order without executing anything, repeat:

1. remove one target from the ready set and append it to the build order;
2. follow its `requiredBy` edges to targets in this request that have not yet
   been added;
3. decrement each such target's remaining-dependency count; and
4. add a target to the ready set when its count reaches zero.

When several targets are ready, compare their label strings as a dictionary
would. This is called **lexicographic order**. It gives this result:

| Step | Append to order | Newly ready | Ready afterward |
|---:|---|---|---|
| 1 | `//lib/config:config` | none | `//lib/logging:logging` |
| 2 | `//lib/logging:logging` | `//lib/http:http` | `//lib/http:http` |
| 3 | `//lib/http:http` | `//app:server` | `//app:server` |
| 4 | `//app:server` | none | empty |

The result is dependency-first. If several targets are ready, choosing a
different one can produce another valid order. A fixed tie-breaking rule makes
the result the same on every run. It does not make that result the only
correct order.

These steps are Kahn's algorithm applied after reversing the stored edges. The
counts show which dependencies still need to appear earlier in the order. The
`requiredBy` index tells us exactly which counts to change when we add a
dependency. This code is only calculating a list. Removing and adding targets
does not start or finish any build action.

If the process stops before it has handled every node and no target is ready,
the remaining graph contains a cycle. A separate DFS check can return the
closed path and the declarations that created it.

### The same counts can release work during execution

A scheduler can use similar counts, but now it changes a count only when real
work finishes. After checking the graph, give each action a count of its
unfinished prerequisites. When a prerequisite finishes successfully, lower
the counts of the actions that need it. A cached result can count as finished
too. Merely starting a prerequisite must not lower any count.

Every action in the scheduler's ready set is allowed to run by the dependency
rules. That does not mean every ready action will run at once. A real build
system must also handle:

- a finite number of workers;
- CPU, memory, network, and remote-execution limits;
- cached actions that need no execution;
- actions with different costs;
- failures and cancellation; and
- scheduling policy among equally ready actions.

It is therefore better to say, “These actions may run at the same time,” than
to say that they will do so.

## Go's command builder uses the same ready-work idea

The Go command's builder shows how a completed prerequisite updates only the
actions that are waiting for it:

- [`cmd/go/internal/work/action.go` at Go 1.26.5](https://cs.opensource.google/go/go/+/refs/tags/go1.26.5:src/cmd/go/internal/work/action.go)
- [`cmd/go/internal/work/exec.go` at Go 1.26.5](https://cs.opensource.google/go/go/+/refs/tags/go1.26.5:src/cmd/go/internal/work/exec.go)

An action records the actions it needs first. The builder also records the
relationship in the other direction, in a field named `triggers`. For each
action, `triggers` lists the actions that are waiting for it. The builder
counts unfinished prerequisites and makes an action ready when its count
reaches zero. This reverse list lets a completed action update its known users
instead of checking every action in the build.

The foundations graph is not a copy of `cmd/go`. The Go tool also handles
caches, failures, resource limits, and many kinds of actions. Our small model
focuses on one mechanism:

~~~text
unfinished prerequisite count reaches zero -> action may become ready
~~~

It also shows where the small target graph stops matching a full build system.
A build target is not always one action. One target can create several compile,
link, copy, or metadata actions.

## Bazel contains several related graphs

“The Bazel DAG” is not a precise name. A Bazel label can identify different
kinds of nodes during different phases. Also, `deps` is only one of several
places where an edge can come from.

| Graph or view | Nodes | What joins them | How to inspect it |
|---|---|---|---|
| Post-loading target graph | Rule targets and file targets | Rule inputs from attributes such as `srcs`, `deps`, and `data`, plus dependencies added by rules | `query` |
| Configured target graph | Configured targets: a target label together with a build configuration | Dependencies after Bazel chooses configurations and toolchains | `cquery` |
| Action graph | Actions and artifacts | An action reads input artifacts and creates output artifacts | `aquery` |
| Skyframe graph | Bazel's internal calculation keys and results | One calculation needs another; Bazel uses this graph to know what to repeat after an input changes | Bazel internals |

The difference between a target and a configured target matters. Bazel may
analyze the same label in more than one configuration. Each combination is a
separate configured target. If an operation needs to tell them apart, a map
keyed only by label would incorrectly combine them.

[Bazel's extension concepts](https://bazel.build/versions/9.1.0/extending/concepts)
describe three broad phases:

1. **Loading** reads and evaluates the needed BUILD and extension files. It
   creates rule targets.
2. **Analysis** applies a build configuration and runs rule implementations.
   It creates actions from the configured targets.
3. **Execution** runs the required actions.

The
[Bazel glossary](https://bazel.build/versions/9.1.0/reference/glossary)
defines an action as a command with declared input and output artifacts.
Artifacts are files or file-like build objects that actions read or create.
Bazel creates the action graph during analysis and uses it during execution.

Traditional `bazel query` examines the target graph after loading but before
Bazel applies one build configuration. This graph contains more than our
example. Source files named by `srcs` are targets, and Bazel includes implicit
dependencies by default. `--noimplicit_deps` hides implicit dependencies. It
does not hide explicit `srcs`, `data`, or other rule inputs.

A `select()` can name different dependencies for different platforms or build
options. Traditional query has not selected one configuration, so it includes
every possible choice. Its result can contain more dependencies than any one
build uses. `cquery` examines configured targets after Bazel has made those
choices and selected toolchains. `aquery` shows the actions and artifacts that
analysis created:

- [Bazel query reference](https://bazel.build/versions/9.1.0/query/language)
- [Bazel action graph query](https://bazel.build/versions/9.1.0/query/aquery)

In the traditional target graph, Bazel's query operations answer several
specific questions:

- `labels(deps, //app:server)` reads the server rule's `deps` attribute. In
  this example, it selects config and HTTP directly;
- `deps(//app:server)` returns the server and every target it can reach across
  all rule-input edges in the query graph, not only `deps` edges;
- `rdeps(//..., //lib/http:http)` follows edges in reverse, but only within the
  set selected by its first argument, `//...`. Here that set contains the
  main repository's rules and everything they depend on;
- `somepath(//app:server, //lib/logging:logging)` returns one path from the
  server to logging. It does not promise the shortest path; and
- `allpaths(//app:server, //lib/logging:logging)` returns the graph formed by
  targets on dependency paths from the server to logging.

The first argument to `rdeps` sets the search boundary, which Bazel calls the
**universe**. The result means “reverse dependencies inside this set,” not
“every possible user in every repository.”

The distinction changes what a result means:

~~~text
bazel query 'deps(//app:server)'
~~~

asks for the server and every target it can reach in the broader traditional
query graph. The result can include `//app:main.go`, rules from external
repositories, and implicit dependencies. It is not the six-node graph in this
lesson.

~~~text
bazel aquery '//app:server'
~~~

asks about actions created for the configured build. It can show commands,
inputs, outputs, and action mnemonics. A mnemonic is Bazel's short name for a
kind of work. The two commands walk different kinds of nodes.

Traditional query can continue when the graph before configuration contains a
cycle. It does not treat that cycle as a build error. Bazel explains that
combining several possible configurations can create an apparent cycle that
disappears after Bazel chooses one configuration. `cquery` and `aquery` do
report cycles in the configured target graph. Always ask which graph an
operation uses before interpreting “cycle” or “DAG.”

{{< callout kind="note" title="Why Bazel belongs in a Go production course" >}}
Bazel is not written in Go. It belongs in this unit because infrastructure
engineers use build graphs to build and deliver software. Bazel also documents
its target dependencies and graph queries clearly. The lab code remains Go.
BuildKit, the Go command, Pulumi, Prometheus, and Argo provide additional Go
source and patches.
{{< /callout >}}

## Correct algorithms still need an accurate graph

The algorithms work only with the facts that graph construction kept. The next
cases show four ways to lose an important fact:

- a direct dependency is missing;
- an analysis did not produce an answer;
- two different nodes received the same identity; or
- a relationship with several parents was reduced to one parent.

BFS, DFS, and topological ordering cannot recover information that never
entered the graph.

### Declared edges can leave out actual dependencies

[Bazel's dependency documentation](https://bazel.build/versions/9.1.0/concepts/dependencies)
distinguishes two graphs:

- the **declared dependency graph** comes from build metadata; and
- the **actual dependency graph** contains what targets really need to
  build or execute correctly.

For a correct build, each direct dependency that the code needs must also be a
direct declared dependency. In mathematical terms, the actual graph must be a
subgraph of the declared graph.

In the running BUILD declarations, logging is already transitively reachable:

~~~text
declared:
//app:server -> //lib/http:http -> //lib/logging:logging
~~~

Now suppose `main.go` starts importing
`example.com/buildgraph/lib/logging` directly, but `app/BUILD.bazel` still
lists only config and HTTP. The source code now has this direct requirement:

~~~text
actual but undeclared:
//app:server -----------------> //lib/logging:logging
~~~

The existing path to logging does not repair the missing declaration. The
server now uses logging directly, so its own `deps` list must include logging:

~~~starlark
deps = [
    "//lib/config:config",
    "//lib/http:http",
    "//lib/logging:logging",
]
~~~

`rules_go` checks this rule while compiling. Its
[import check](https://github.com/bazel-contrib/rules_go/blob/v0.60.0/go/tools/builders/importcfg.go)
compares imports in the Go source with the packages supplied by direct `deps`.
It reports a missing strict dependency when they do not match. Other rule
implementations may check this boundary differently or may not check it fully.
For that reason, Bazel's general documentation still separates the declared
graph from the graph that the code really needs.

No graph walk can infer the missing direct edge. BFS, DFS, reverse reachability,
and build ordering will all give correct answers for the incomplete graph that
they received.

Extra declarations cause a different problem. Bazel warns that unneeded
declared dependencies can make builds slower and binaries larger. Listing
every possible dependency “to be safe” has a cost. It is not a substitute for
maintaining the correct direct dependencies.

{{< callout kind="production" title="The production connection" >}}
A build tool repeatedly asks what a target needs, what depends on it, why one
target reaches another, whether the declarations contain a cycle, and which
work has no unfinished prerequisite. Adjacency lists make those questions
efficient. They do not prove that every needed declaration is present.
Building and checking the graph are part of correctness, not only setup for a
graph walk.
{{< /callout >}}

### Missing, empty, and undetermined are different states

A missing target is not the same as a known target with no dependencies. There
is also a third case: the target exists, but the dependency check did not
finish. The program does not yet know whether that target has dependencies.

[Prometheus PR #15560](https://github.com/prometheus/prometheus/pull/15560)
fixes this distinction in its rule dependency controller. A Prometheus rule
group regularly evaluates recording or alerting rules. One recording rule can
produce a time series that another group reads. For this case, make each rule
group a node. `A -> B` means “group A depends on results from group B.” The
controller must schedule B before A.

The controller had treated “the check did not finish” as “there are no
dependencies.” It could then run rule groups at the same time without knowing
that this was safe.

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

`TargetMissing` means that the requested node does not exist in the graph.
`DependenciesUndetermined` means that the target exists but the check could
not answer. `DependenciesKnown` with an empty `IDs` slice means that the check
finished and found no dependencies under its stated rules. A Go API could
return an error for a missing target instead. The important rule is that one
empty slice must not stand for all three cases.

How to handle an undetermined result depends on the operation. A scheduler
might keep the work in sequence or block it and show an error. It must not
silently turn “could not determine” into “safe to run.”

### Node identity must distinguish specific nodes

An adjacency list assumes that each key names exactly one node. If two
different resources get the same key, the program can create edges between
the wrong resources even though those edges look valid.

#### Pulumi: current and pending-deletion resources

[Pulumi PR #19179](https://github.com/pulumi/pulumi/pull/19179) describes a
snapshot that can contain:

~~~text
current A
B depends on A
old A, still awaiting deletion
~~~

In Pulumi's deletion graph, each node should represent one specific resource
record. The edge `B -> A` means that B depends on A, so A must remain while B
still exists. The edge is correct only if it points to the specific A that B
uses.

Pulumi calls its logical resource identifier a **URN**, or uniform resource
name. The current and old resources can share that URN. A lookup with one entry
for each URN connected B to the old A instead of the current A. Later code
could then allow deletion of the current A. This would leave B pointing to a
resource that no longer existed, which is called a **dangling reference**. The
patch tracked current resources and resources waiting for deletion separately.

The lesson is not simply “use a longer string.” Define identity from the facts
that the operation needs to tell apart. If current and pending-deletion copies
can exist at the same time, the key must include enough information to
distinguish them.

“Dependency order” can mean two different things during deletion. If B depends
on A, creation usually needs A before B. Deletion usually needs B before A.
Name the operation and direction instead of saying only “sort the
dependencies.”

#### Argo: a hash collision manufactured a cycle

[Argo Workflows issue #16376](https://github.com/argoproj/argo-workflows/issues/16376)
concerns Argo's records for parts of a workflow, such as Pods and grouping
nodes. In its recursive completion check, `A -> B` means that record A names B
as a child. B's completion contributes to A's result.

Two different node names produced the same node ID under a 32-bit FNV-1a hash.
A hash converts a name into a fixed-size number. A **collision** happens when
two different names produce the same number. A lookup then returned an
unrelated record. The controller recorded an ancestor as one of its own
descendants. The completion check followed that false child link around the
cycle until the process ran out of stack space and restarted.

[Argo Workflows PR #16625](https://github.com/argoproj/argo-workflows/pull/16625),
still open on August 12, 2026, prevents two recursive walks from descending
into a child that is already on the current chain of calls. When the recursive
call returns, the function removes the child from that set. A bad link can no
longer make the function call itself forever.

The patch's tests include a diamond whose two branches share a child. The walk
may visit that child once through each branch. By the time the second branch
reaches it, the child is no longer on the first chain. This does not conflict
with the three-state DFS described earlier. A function that only checks for
cycles can mark a node finished and skip its outgoing edges on a later visit.
It has already proved that the graph below that node has no cycle. Argo's
functions do completion work along each route, so skipping every later visit
would change their result. A global cache of completed nodes would need proof
that the result does not depend on the route used to reach a node.

This change stops a bad graph from crashing the controller. It does not remove
the source of the bad edge. Three separate protections are needed:

- give different node names different identities;
- detect a collision while building the graph; and
- stop a later recursive walk safely.

### Keeping one parent discards the others

A workflow node can appear under several parents. Argo's retry code needed to
walk from a selected node back to its ancestors. It changed the
parent-to-children relationship into `map[child]parent`. This map can store
only one parent for each child. Storing another parent replaces the first, so
the other relationship is lost.

[Argo Workflows issue #16450](https://github.com/argoproj/argo-workflows/issues/16450)
documents this change while the code looped over a Go map. For a node with
several parents, the last parent visited replaced the previous entry. Go does
not define map iteration order, so the same retry could choose a different
parent chain on another run. A **TaskGroup** is Argo's node for tasks created
by a loop. One parent choice reset a TaskGroup to Running but did not reset the
work that could complete it. The group stayed Running after all Pods had
finished.

Merged
[PR #16451](https://github.com/argoproj/argo-workflows/pull/16451)
sorts node IDs before building the one-parent map. The chosen chain is now the
same on every run. The patch clearly states that it does not keep every parent.

This supports two separate conclusions:

- output that depends on Go map order can change between runs; and
- choosing the same parent every time does not restore the other parents.

Sometimes keeping one parent is correct. For example, an operation may ask for
one explanation path rather than every path. Its contract must say how it
chooses that parent and that it leaves out the other valid parents on purpose.

## Runtime service dependencies are not automatically a DAG

The clean build example does not mean that every production dependency must
use the same model.

[GitHub's deployment-safety report](https://github.blog/engineering/infrastructure/how-github-uses-ebpf-to-improve-deployment-safety/)
describes cycles in deployment tools. A script needed to repair an outage
might itself call github.com, an internal service, or a tool that contacted a
system that was down. The dependency might remain hidden until it delays
recovery from an incident.

The report starts with a small Go experiment. It then describes a production
system that uses Linux's eBPF feature to observe and sometimes block network
calls from deployment scripts. It does not put a general service graph into
topological order.

That is the important boundary:

- services can call one another in a cycle without every call being a design
  error;
- the path by which one service failure affects another may differ from the
  steps needed to recover the services;
- services, deployment scripts, data stores, and repair actions are different
  possible node types; and
- recovery may require a prebuilt tool or package, or a control route that
  still works when the affected services do not.

For recovery planning, ask:

> Which actions must succeed for this specific recovery operation, and what
> does each action require?

If those action requirements form a cycle, the recovery operation needs a
design change or an independent route that breaks the cycle. A cycle between
running services does not by itself prove that the services are designed
incorrectly.

## Account for both nodes and edges

The production cases above show what the input graph must get right. We can
now return to the adjacency-list algorithms and count the work that they do on
a valid requested part of the graph.

Let `V` be the number of nodes reached in the requested subgraph, `E` the
number of edges among them, `d` the number of direct dependencies of one
target, and `S_cycle` the number of source-location records attached to a
returned cycle.

| Operation | Time | Additional working space |
|---|---:|---:|
| Store a repeated edge only once | expected `O(1)` | `O(1)`, excluding evidence text |
| List direct dependencies | `O(d)` | `O(d)` for a copied result |
| Dependency closure with BFS or DFS | `O(V + E)` | `O(V)` |
| Fewest-edge path with BFS | `O(V + E)` | `O(V)` |
| One cycle witness with DFS | `O(V + E + S_cycle)` | `O(V + S_cycle)` |
| Dependency-first order with an unordered ready set | `O(V + E)` | `O(V)` |
| Dependency-first order with a min-heap tie-break | `O(E + V log V)` | `O(V)` |

The `O(V + E)` limit comes from processing each reached node once and checking
each outgoing edge from those nodes once. Saying only “linear time” is unclear
because a graph can have many more edges than nodes.

For a cycle report, DFS still checks at most `O(V + E)` graph data. Copying
source locations into the returned error adds `O(S_cycle)` time and space.

These limits apply to one graph walk after the adjacency lists exist. Loading,
checking, removing duplicate declarations, and sorting have their own costs.
If callers repeat the same query, another index or a cached result may help.
The program must update or discard that cache when the graph changes.

The expected constant-time edge insertion assumes a hash-set index and target
IDs with a limited length. Under that assumption, computing one hash counts as
constant work. This is the usual average result for a hash table, not a promise
for every possible set of keys. If the program instead searches a slice for a
duplicate, the time grows with the target's existing direct dependencies.

{{< callout kind="note" title="Include the cost of fixed tie-breaking" >}}
Kahn's basic algorithm takes `O(V + E)` time when the ready set may return any
available node. Always choosing the first target in dictionary order adds
work. A **min-heap** is a priority queue that returns the smallest target name.
It adds and removes each node at most once, for a total of
`O(E + V log V)` time. A small teaching program may sort a slice instead, but
it should state the cost of the code it actually uses.
{{< /callout >}}

## Check correctness before measuring speed

Graph tests should make the relationship and unusual cases easy to see.

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
- two different records that would receive the same incomplete identity.

Small graphs are easy to draw. You can check their expected paths and orders
by looking at them.

### Check every returned order

For every stored `target -> dependency` edge among the requested targets and
their dependencies, check:

~~~text
position(dependency) < position(target)
~~~

Also check that the order contains each requested or reached node exactly
once. It must not contain a disconnected node unless the request named that
node.

Do not require one exact order when several orders are valid. Require it only
if the API promises a fixed tie-breaking rule.

### Check cycle witnesses

A returned cycle must:

- contain at least one edge;
- form a chain in which, for each adjacent pair, the first edge's dependency
  is the second edge's dependent;
- have a last dependency equal to the first dependent;
- use only edges present in the graph;
- keep edge source locations; and
- report a self-edge as a one-edge cycle.

A diamond must not be reported as a cycle.

### Check path promises

If `FewestEdgePath` promises a path with the fewest edges, check that every
adjacent pair in the result is a declared edge. For a tiny generated graph, a
separate test can list every path that does not repeat a node. It can then
compare the returned length with the shortest one. This complete, or
**exhaustive**, search is too expensive for production. On a deliberately
small test graph, however, it provides an independent check of BFS. Also check
that a known but unreachable target returns “no path,” while an unknown target
gets the API's separate missing-target result.

### Check promised output stability

Build the same graph from declarations in different input orders. Call the
public operations several times. Direct-neighbor lists, equal-length paths,
cycle reports, and build plans should stay the same if the API promises stable
output.

### Measure after correctness

Count reached nodes and checked edges before comparing elapsed time. Then
measure sparse chains, graphs where one node has many outgoing edges, diamonds
with shared dependencies, and invalid cycles separately. A measurement of only
one graph shape does not show the cost for every graph.

## Read each source for one role

This lesson uses course notes, product documentation, source code, issue
reports, and patches. Each kind of source answers a different question:

| Source | What it tells us |
|---|---|
| [MIT 6.006 Lecture 9](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/196a95604877d326c6586e60477b59d4_MIT6_006S20_lec9.pdf) | Directed graph representation, reachability, BFS, fewest-edge distance, and `O(V + E)` analysis |
| [MIT 6.006 Lecture 10](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/f3e349e0eb3288592289d2c81e0c4f4d_MIT6_006S20_lec10.pdf) | DFS, directed cycles, DAGs, and topological order |
| [Open Data Structures: Graphs](https://opendatastructures.org/ods-go/12_Graphs.html) | A second, Go-oriented treatment of adjacency lists and traversal |
| [Bazel BUILD files](https://bazel.build/versions/9.1.0/concepts/build-files) and [labels](https://bazel.build/versions/9.1.0/concepts/labels) | BUILD-file evaluation, rule-target creation, packages, target names, and label structure |
| [Bazel rule concepts](https://bazel.build/versions/9.1.0/extending/rules) | How label-bearing attributes, private attributes, toolchains, rule analysis, and actions create different relationships |
| [Bazel dependencies](https://bazel.build/versions/9.1.0/concepts/dependencies) | Bazel's target-dependency contract and declared-versus-actual distinction |
| [Bazel query reference](https://bazel.build/versions/9.1.0/query/language) | Query graph direction, `deps`, `rdeps`, paths, cycles, configurations, and output order |
| [`rules_go` v0.60.0 rule reference](https://github.com/bazel-contrib/rules_go/blob/v0.60.0/docs/go/core/rules.md) | The actual `go_binary` and `go_library` attributes used by the example and the direct-import meaning of `deps` |
| [`rules_go` v0.60.0 import check](https://github.com/bazel-contrib/rules_go/blob/v0.60.0/go/tools/builders/importcfg.go) | The pinned implementation that checks source imports against direct dependencies |
| Pinned Go source | What one named Go release implements |
| Public issue report | What a reporter observed and how they reproduced it |
| Merged pull request | What maintainers changed; it does not prove that one patch explains every similar symptom |
| Open pull request | A proposed change that reviewers can inspect; it is not released behavior |

The
[MIT BFS recitation](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/resources/mit6_006s20_r09/)
and
[DFS recitation](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/resources/mit6_006s20_r10/)
provide more worked problems.

## Explain the design to a colleague

A short answer to “How would you represent and schedule these build
dependencies?” might be:

> I would first define the graph. In this example, each node is one of the six
> `rules_go` rule targets. `A -> B` means that A's explicit `deps` attribute
> names B. This graph leaves out source files, implicit dependencies,
> configured targets, toolchains, actions, and artifacts. Production code
> should get its graph through Bazel's supported query tools. It should not try
> to extract the graph by reading BUILD files as plain text.
>
> I would store adjacency lists in both directions. Forward BFS can return an
> explanation path with the fewest edges. A walk in the other direction can
> find declared users of a target. DFS can return a closed cycle. When the
> input has a reliable source location for an edge, the error can keep it. An
> implicit edge or an edge created by a macro may have only a less exact
> location.
>
> For the requested targets and their dependencies, I would count how many
> dependencies each target still needs. I would add a target to the
> dependency-first order only after adding its dependencies. The graph walks
> take `O(V + E)` time. A min-heap that always selects the smallest ready
> target adds `O(V log V)` work. Before a scheduler uses the result, I would
> check node identity. I would also keep declared, actual, missing, empty, and
> undetermined relationships separate. Finally, putting a target in a
> calculated list is not the same as completing an action. Running work can
> release a dependent only after its prerequisite finishes successfully or a
> cache supplies the result.

That answer states the graph, operations, direction, costs, checks, and limits.
It does not claim that a graph algorithm is a complete build system.

## Put the model to work

The [Build Graph Explorer lab](lab/) turns the example into a small Go API. The
API finds direct dependencies, reverse dependencies, shortest explanation
paths by edge count, cycles, ready work, and dependency-first order. It
receives target and edge records whose identities are already known. Reading
and evaluating BUILD files is a separate Bazel integration task.

The [Unit 04 Wheels of Misfortune](wheel/) begin with reports inspired by the
BuildKit, Prometheus, and Pulumi changes in this lesson. You will distinguish a
real cycle from a harmless second visit, an unfinished dependency check from a
known empty result, and a logical resource name from the identity of one
specific resource record. The reports and evidence do not reveal the repairs.
Each hidden debrief explains one approach after the exercise.

## Reflection

For each answer, name the graph and say what one edge means:

1. How do `app/BUILD.bazel`, package `app`, and `name = "server"` combine to
   identify `//app:server`?
2. Why does `srcs = ["main.go"]` create a relationship in Bazel's target graph
   even though the running teaching graph leaves it out?
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
9. Why does adding a target to a calculated order not mean that its work has
   finished? What does a scheduler's ready set tell you about work that may
   run at the same time?
10. Why can `bazel query`, `cquery`, and `aquery` return information about
   different graphs for the same target label?
11. What went wrong when Pulumi used a shared URN as the complete identity of
   both current and pending-deletion resources?
12. Which problem did Argo fix by choosing the same parent on every run? Which
    information did the one-parent map still discard?
13. Why does GitHub's circular deployment dependency call for an independent
    recovery route rather than a blanket rule that runtime service graphs must
    be DAGs?

The durable production habit is:

> Build the graph from real declarations. State what it leaves out. Check it
> before trusting an algorithm's answer.
