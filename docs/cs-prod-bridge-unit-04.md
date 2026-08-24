# Unit 04 Research and Design: Build Dependency Graphs

**Status:** initial implementation complete: foundations lesson, Build Graph
Explorer lab, and three Wheels

**Research reviewed:** 2026-08-12

**Implementation completed:** 2026-08-14

**Target toolchain:** Go 1.27.x

## Decision

Build Unit 04 around build dependency graphs, with Bazel as the primary
production example.

Bazel is a good fit even though Bazel itself is not implemented in Go. It is
widely used in build and delivery infrastructure, its documentation states the
dependency model precisely, and its query tools expose the questions that
graph algorithms answer in practice. The local implementation will remain Go.
Go's build system, BuildKit, Terraform, Pulumi, Prometheus, and Argo Workflows
provide inspectable Go code and public patches.

The unit's central claim should be:

> A graph algorithm can answer only the question represented by its nodes and
> edges. State that model first. Then choose traversal, cycle detection, or an
> ordering algorithm.

This replaces the earlier recovery-planner framing. A sentence such as “the
API needs the database” describes a real operational concern, but it does not
by itself define a stable DAG:

- The dependency may change by request path, failure mode, or fallback.
- Two services can depend on each other during normal operation.
- “Needs” does not say whether the graph represents calls, data flow, startup,
  recovery, deployment, or failure propagation.
- A recovery action can have different prerequisites from the service it
  repairs.

A build target declaration is a cleaner starting point. If target A declares
target B as a dependency, the relationship is explicit, inspectable, and
intended to be acyclic. It supports concrete questions:

- What does this target depend on?
- What targets depend on this library?
- Why is this dependency in the build?
- Is there a cycle?
- Which targets must finish before this target can be built?
- Which work can run in parallel?
- Does the declared graph include every dependency the build actually uses?

GitHub's report about circular dependencies during incident recovery remains
valuable, but as a boundary case near the end of the lesson. It shows why a
runtime service graph must not be treated as a build DAG without first deriving
the specific action constraints needed for recovery.

## Working title and lesson promise

**Working lesson title:** Build Graphs You Can Explain

**Lesson promise:** Given declared build dependencies, represent the graph in
Go, explain the meaning of its arrows, find dependency paths and reverse
dependents, report a useful cycle, and produce a dependency-first build order.

The lesson should not promise a miniature Bazel. It teaches the computer
science through a small build-graph explorer whose operations are recognizable
from Bazel's query language and from public Go implementations.

## The graph model

### Nodes

For the foundations example, each node is a build target:

~~~text
//app:server
//lib:http
//lib:config
//lib:logging
~~~

A target is a named unit in the declared build graph. It is not necessarily one
source file, one command, or one output.

### Edges

Use the same direction that Bazel uses in dependency queries:

~~~text
//app:server  ---->  //lib:http

source target depends on destination target
~~~

In short:

> A -> B means A directly depends on B.

This direction makes the most common question natural: starting at a target
and following outgoing edges discovers its dependencies.

Do not reverse the arrow merely to make a textbook ordering definition look
convenient. Reversing it forces the prose to translate a natural declaration
into a different relation before the learner has even met a graph.

### Ordering consequence

The edge direction creates an important distinction that the lesson should
state openly.

A mathematical topological order places the source of every edge before its
destination. With target-to-dependency edges, that produces a
**dependent-first order**:

~~~text
//app:server, //lib:http, //lib:logging
~~~

A build normally needs the opposite, **dependency-first order**:

~~~text
//lib:logging, //lib:http, //app:server
~~~

The build order is the reverse of a topological order of the stored graph, or
equivalently a topological order of the graph with every edge reversed.

Bazel's dependency-ordered query output follows its stored edge direction:
dependents appear before their dependencies. The lesson must not describe that
query order as execution order.

The learner-facing API should therefore say BuildOrder rather than expose a
vague TopologicalOrder method.

### Several graphs exist inside a build system

Avoid the phrase “the Bazel DAG” as though Bazel has one graph.

Bazel documentation distinguishes at least these models:

1. The **target graph** records declared dependencies among targets.
2. The **configured target graph** includes the effects of build
   configuration.
3. The **action graph** records actions, artifacts, and the relationships
   needed to produce requested outputs.
4. Bazel also uses an internal evaluation graph, commonly discussed as the
   Skyframe graph.

The foundations lesson uses a small target graph. The production section can
then explain that analysis turns requested targets into configured targets and
actions. A target can produce several actions, so target nodes and action nodes
must not be treated as interchangeable.

### A DAG is a validity condition

A valid build dependency graph must be acyclic. That does not mean every graph
loaded from a BUILD file is already a DAG. Erroneous declarations can contain a
cycle, and the tool must detect and explain it before scheduling.

The teaching sequence is therefore:

~~~text
directed graph
    |
    +-- cycle found ------> invalid build graph; report the path
    |
    +-- no cycle ---------> DAG; a dependency-first build order exists
~~~

Calling the input a DAG before checking for cycles would hide the reason the
cycle-detection step exists.

## Terms the lesson must define

- A **node**, also called a vertex, is one thing represented by the graph.
- A **directed edge** is a one-way relation between two nodes.
- A **direct dependency** is one edge away.
- A **transitive dependency** is reachable through one or more dependency
  edges.
- An **adjacency list** stores the outgoing neighbors of each node.
- A node is **reachable** from a start node if following zero or more directed
  edges can arrive there.
- A **path** is a sequence of connected edges. This unit counts edges; it does
  not assign latency, cost, or risk weights.
- A **cycle** is a non-empty path that returns to its starting node. A
  self-dependency is a cycle of length one.
- A **directed acyclic graph**, or DAG, is a directed graph with no cycle.
- A **topological order** places each edge's source before its destination.
- A **dependency-first build order** places every dependency before targets
  that require it. Under this unit's edge convention, it reverses the graph's
  topological direction.
- A **ready target** has no unfinished dependencies within the requested build.

“Dependency” is not a complete edge definition. The lesson should always state
what the nodes are and complete the sentence “A -> B means ...”.

## Foundations lesson sequence

### 1. Begin with one build declaration

Start with:

~~~text
//app:server depends on //lib:http
~~~

Represent it without translation:

~~~text
//app:server  ---->  //lib:http
~~~

Ask two questions:

- Following the arrow, what does the server need?
- Following the reverse index, which declared targets might be affected if the
  HTTP library changed?

This establishes that direction is part of the model, not a drawing detail.

### 2. Expand to a small graph

Use one graph through the introduction:

~~~text
//app:server ----> //lib:http ----> //lib:logging
       |
       +---------> //lib:config

//tool:migrate ---> //lib:config

//tool:lint
~~~

This graph supplies:

- a direct dependency;
- a transitive dependency;
- a dependency shared by two targets;
- two branches that can be built independently;
- an isolated target.

The isolated target matters. Traversal from the server should not silently
include every known target. A build plan includes the requested roots and
their dependency closure.

### 3. Store both useful directions

The local graph can maintain:

~~~go
dependsOn[target] = append(dependsOn[target], dependency)
requiredBy[dependency] = append(requiredBy[dependency], target)
~~~

The two maps represent the same declared edges. They make different questions
direct:

| Question | Index to follow |
| --- | --- |
| What does this target need? | dependsOn |
| What could this library affect? | requiredBy |
| Why does server depend on logging? | dependsOn |
| What should be reconsidered after config changes? | requiredBy |

Repeated declarations should form one logical edge. If several files or
analysis sources support that edge, retain those sources as evidence rather
than inflating the neighbor count.

### 4. Introduce traversal as controlled exploration

Breadth-first and depth-first search are ways to explore reachable nodes, not
competing answers to every graph problem.

Use breadth-first search when the lab promises a path with the fewest edges:

~~~text
Why does //app:server depend on //lib:logging?
~~~

The returned path is an explanation:

~~~text
//app:server -> //lib:http -> //lib:logging
~~~

Bazel's somepath query promises an arbitrary path, not the shortest path. The
local lab can intentionally offer the stronger fewest-edges promise, but the
text must present that as the lab's contract rather than as Bazel behavior.

Use depth-first search when the active recursive path matters, particularly for
cycle detection. A global “visited” set alone is not enough: revisiting a node
that has already been completely explored is not a cycle. A cycle exists when
an edge returns to a node still on the current path.

### 5. Turn a cycle into an explanation

Add one invalid declaration:

~~~text
//lib:logging -> //app:server
~~~

The tool should return the closed path:

~~~text
//app:server -> //lib:http -> //lib:logging -> //app:server
~~~

“Cycle detected” is technically true but operationally weak. A maintainer
needs the targets, edges, and source locations that can be edited.

BuildKit's public fixes make this production connection concrete. One patch
added cycle validation so a cyclic Dockerfile stage graph no longer recursed
until dockerd overflowed its stack. A later patch attached Dockerfile source
locations to the cycle error.

### 6. Derive ready work and build order

After cycle validation, restrict the graph to the requested roots and their
dependencies.

For each target in that closure:

1. Count its unfinished direct dependencies.
2. Put targets with count zero in the ready set.
3. Remove one ready target and append it to the build order.
4. Follow requiredBy edges and decrement the waiting targets.
5. Add a waiting target when its count reaches zero.

This is Kahn's algorithm on the graph with the stored edges reversed. The
dependsOn counts say what remains unfinished; the requiredBy index carries each
completion toward waiting targets. The ready set also explains possible
parallelism: every target in it may run concurrently if resource limits and
the build system permit it.

Use a stable tie-break, such as lexical target name, so examples and tests do
not depend on Go map iteration order. Make clear that the tie-break chooses one
valid order; it does not make that order mathematically unique.

### 7. Ask whether the graph tells the truth

Bazel distinguishes:

- **declared dependencies**, written in build metadata; and
- **actual dependencies**, required by the source or action.

For a correct build, actual dependencies must be contained in the declared
graph. A missing declaration can be masked when another target happens to make
the dependency available transitively. That makes a build appear healthy until
the graph, cache, or build environment changes.

Extra declarations are also not free. They can increase analysis and build
work and may enlarge outputs.

This is the unit's bridge from graph mechanics to production judgment:

> Correct traversal over an inaccurate graph produces an accurate answer to
> the wrong model.

## Computer-science source spine

Use these as the formal sources:

- [MIT 6.006 Lecture 9: Breadth-First Search](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/196a95604877d326c6586e60477b59d4_MIT6_006S20_lec9.pdf)
  for vertices, directed edges, adjacency lists, reachability, BFS, and
  O(V + E) traversal.
- [MIT 6.006 Lecture 10: Depth-First Search](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/f3e349e0eb3288592289d2c81e0c4f4d_MIT6_006S20_lec10.pdf)
  for DFS, finishing order, cycle detection, DAGs, and topological order.
- [MIT 6.006 Recitation 9](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/resources/mit6_006s20_r09/)
  for worked BFS and graph-representation practice.
- [MIT 6.006 Recitation 10](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/resources/mit6_006s20_r10/)
  for worked DFS, cycles, and ordering practice.
- [Open Data Structures: Graphs](https://opendatastructures.org/ods-go/12_Graphs.html)
  for a Go-oriented adjacency-list treatment and traversal costs.

The lesson should translate the formal topological-order definition into the
chosen edge convention rather than changing either definition silently.

## Bazel production source spine

### Dependency contract

[Bazel 9.1 dependency documentation](https://bazel.build/versions/9.1.0/concepts/dependencies)
defines target A as depending on target B when B is needed for A at build or
execution time. It says this relation induces a DAG and distinguishes actual
from declared dependencies.

Use it for:

- the natural target-to-dependency arrow;
- the DAG validity contract;
- declared versus actual edges;
- missing direct dependencies that work accidentally through transitive
  availability;
- the cost of unnecessary declared dependencies.

### Graph phases

[Bazel's build-system concepts](https://bazel.build/versions/9.1.0/concepts/build-ref)
define repositories, packages, rules, files, and targets.
[Bazel's extension concepts](https://bazel.build/versions/9.1.0/extending/concepts)
then separate loading, analysis, and execution. Loading evaluates the BUILD
files and instantiates rules in a graph; analysis turns that graph into
actions; execution runs the actions needed for requested outputs.

Use this distinction to prevent three common teaching errors:

- treating a target as one command;
- treating the declared target graph as the runtime action graph;
- claiming that graph construction and execution are the same phase.

The
[Bazel glossary](https://bazel.build/versions/9.1.0/reference/glossary)
provides stable definitions for target graph, configured target, action,
artifact, action graph, and Skyframe.

### Queries

[Bazel's query language](https://bazel.build/versions/9.1.0/query/language)
provides production-shaped graph questions:

- deps for a dependency closure;
- rdeps for reverse dependencies within a universe;
- somepath for one path;
- allpaths for every node on dependency paths;
- dependency-ordered output.

The documentation also states that edges point from targets to their
prerequisites. Preserve its warning that somepath returns an arbitrary path.

There is another useful qualification. Traditional query operates on an
unconfigured post-loading graph, which can contain cycles that disappear after
configuration. Bazel reports cycles in the configured graph through cquery and
aquery. This reinforces the question “which graph are we querying?” and keeps
the lesson from claiming that every intermediate Bazel graph is already a DAG.

[Bazel aquery](https://bazel.build/versions/9.1.0/query/aquery)
operates after analysis and exposes actions, artifacts, and their
relationships. Use it only after learners understand why a target graph and an
action graph answer different questions.

### Production scale

[Bazel's FAQ](https://bazel.build/versions/9.1.0/about/faq)
documents its relationship to Google's internal build tool and gives scale
examples. Use those facts only to establish that this is production
infrastructure, not as proof of a particular graph algorithm or data
structure.

## Go implementation and patch spine

Bazel supplies the clean production model. These Go sources show how related
graph problems appear in code and incident fixes.

### Go command builder: ready work

Pin the standard-library reading to the Go 1.27.0 tree:

- [cmd/go/internal/work/action.go](https://cs.opensource.google/go/go/+/refs/tags/go1.27.0:src/cmd/go/internal/work/action.go)
- [cmd/go/internal/work/exec.go](https://cs.opensource.google/go/go/+/refs/tags/go1.27.0:src/cmd/go/internal/work/exec.go)

The builder creates actions with prerequisite actions, computes reverse
triggers, tracks pending prerequisites, and releases work when the pending
count reaches zero.

Teaching use:

- dependency counts connect a DAG to a scheduler;
- reverse edges avoid rescanning the whole graph whenever work completes;
- the ready set explains legal parallel work;
- graph order and resource-constrained execution are different concerns.

Do not imply that the small lab reproduces the Go tool's entire scheduler.

### BuildKit: cycle validation and useful errors

- [moby/buildkit PR #999](https://github.com/moby/buildkit/pull/999)
- [moby/buildkit PR #4567](https://github.com/moby/buildkit/pull/4567)
- [BuildKit Dockerfile stage graph](https://github.com/moby/buildkit/blob/v0.32.2/frontend/dockerfile/dockerfile2llb/convert.go)

PR #999 is the strongest cycle-detection patch study. A cyclic Dockerfile
stage graph could recurse until dockerd hit a stack overflow. The fix tracked
the current traversal path and rejected the cycle.

PR #4567 improved that validation by attaching source locations to dependency
edges and reporting the Dockerfile commands involved in the cycle.

Teaching use:

- an active-path set is different from a completed-node set;
- validate before recursive conversion;
- report the closed cycle, not only a boolean;
- edge provenance turns a graph result into an editable explanation.

Keep the claim narrow: the source documents a daemon stack overflow caused by
cyclic stage dependencies. Do not inflate it into an unreported broader
incident.

### Terraform: dependency and reverse-dependency indexes

- [Terraform DAG internals](https://developer.hashicorp.com/terraform/internals/graph)
- [dag/dag.go at v1.15.4](https://github.com/hashicorp/terraform/blob/v1.15.4/internal/dag/dag.go)
- [dag/walk.go at v1.15.4](https://github.com/hashicorp/terraform/blob/v1.15.4/internal/dag/walk.go)

Terraform stores edges from a dependent to its dependency. Its descendants
query therefore means dependencies; ancestors means reverse dependents. Its
walker schedules a node after its dependencies complete and allows independent
nodes to run in parallel.

Teaching use:

- edge direction is a contract, not a universal drawing convention;
- forward and reverse indexes serve different queries;
- prerequisite-first execution can be opposite the stored edge direction;
- a walker needs both graph correctness and execution controls.

### Pulumi: delete order needs the right resource identity

- [Pulumi pkg/resource/graph at v3.256.0](https://github.com/pulumi/pulumi/tree/v3.256.0/pkg/resource/graph)
- [Pulumi PR #19179](https://github.com/pulumi/pulumi/pull/19179)

Pulumi's resource graph exposes dependency and dependent traversal. Its code
also makes a useful ordering boundary visible: deleting resources generally
starts with dependents and works toward their dependencies.

PR #19179 documents a more subtle graph-construction bug. A snapshot can
contain both the current resource and an older copy still awaiting deletion,
and those records can share a URN. A lookup keyed only by URN associated a
dependent with the old copy instead of the current one. Later deletion logic
could then allow the current dependency to be removed, leaving a dangling
reference. The patch tracks current and pending-deletion resources separately.

Teaching use:

- name an operation's order instead of saying “dependency order”;
- distinguish graph-node identity from a human-facing resource name;
- preserve current and pending-deletion states when both exist;
- reversing an order is valid only when the underlying constraint reverses.

### Prometheus: unknown is not empty

- [Prometheus PR #15560](https://github.com/prometheus/prometheus/pull/15560)
- [Rule dependency code at v3.13.1](https://github.com/prometheus/prometheus/tree/v3.13.1/rules)

The rule scheduler had to distinguish “no dependencies” from “dependency
analysis could not determine a safe relation.” Treating unknown as empty
allowed rule groups to run concurrently when ordering was still needed.

Teaching use:

- absence of a recorded edge can mean independent, missing, or unknown;
- graph construction is part of correctness;
- conservative behavior can be safer when dependency evidence is incomplete.

This case belongs after the basic graph API. Introducing partial knowledge in
the opening would obscure the simpler directed-graph model.

### Argo Workflows: multi-parent retries and corrupt node graphs

- [Argo Workflows issue #16450](https://github.com/argoproj/argo-workflows/issues/16450)
- [Argo Workflows PR #16451](https://github.com/argoproj/argo-workflows/pull/16451)
- [Argo Workflows issue #16376](https://github.com/argoproj/argo-workflows/issues/16376)
- [Argo Workflows PR #16625](https://github.com/argoproj/argo-workflows/pull/16625)

Issue #16450 describes retry code that reduced a node's several parents to one
parent while ranging over a Go map. Retrying the same workflow could therefore
choose different parent chains. In the worst case it reset a TaskGroup to
Running without resetting work that could finish the group, leaving the
workflow Running after every pod had completed. Merged PR #16451 sorts node IDs
so the selected parent is stable. The patch deliberately makes the projection
deterministic; it does not change the one-parent representation into a
multi-parent one.

Issue #16376 reports a different failure. Two distinct node names produced the
same FNV-32a node ID. A lookup then returned the wrong node, an ancestor was
recorded as a child, and recursive completion checks followed the resulting
cycle until the workflow controller overflowed its stack and restarted. Open
PR #16625, as of the research date, adds current-path checks to contain that
corrupt graph. Its diamond test also demonstrates why a global visited set
would be the wrong substitute: reaching a shared child twice is valid.

Teaching use:

- preserve every parent unless a one-parent projection has an explicit
  contract;
- deterministic output does not restore information already discarded;
- node IDs must distinguish nodes, not merely make collisions unlikely;
- use the current search path to detect cycles while allowing shared children.

Do not merge these into one incident. One is a merged retry-determinism fix;
the other is a reported identity collision with a containment patch still open
at the research date.

## Operational boundary: GitHub recovery dependencies

[How GitHub uses eBPF to improve deployment safety](https://github.blog/engineering/infrastructure/how-github-uses-ebpf-to-improve-deployment-safety/)
describes a deployment-safety problem: a deployment used for remediation can
itself call github.com or an internal service made unavailable by the outage.
GitHub notes that many such dependencies are not discovered until an incident,
when they can delay recovery.

This is useful after the build example because it asks the learner to identify
the graph before reaching for a DAG algorithm:

- A runtime call graph may contain valid cycles.
- A failure-propagation graph is not automatically a recovery-order graph.
- Recovery actions, services, data stores, and deployment mechanisms are
  different possible node types.
- An operator may need an out-of-band path rather than a topological sort of
  the service graph.

GitHub's response monitors and can block network calls made by deployment
scripts, using eBPF and a Go proof of concept. It does not topologically sort a
service graph. That difference is part of the lesson.

The safe conclusion is not “all service dependencies should be DAGs.” It is:

> Model the actions and prerequisites relevant to the operation. If those
> constraints contain a cycle, the operation needs a design change or an
> escape path.

The report should inspire discussion and transfer, not serve as the
foundations graph.

## Concept-to-production map

| CS concept | Foundations example | Production evidence |
| --- | --- | --- |
| Directed edge | target -> dependency | Bazel query edge convention |
| Adjacency list | dependsOn map | Build and infrastructure graph implementations |
| Reverse adjacency | requiredBy map | Bazel rdeps, Go action triggers, Terraform ancestors |
| BFS | fewest-edge dependency explanation | Local contract; Bazel somepath is only arbitrary |
| DFS active path | return to an unfinished ancestor | BuildKit cycle validation |
| Cycle witness | closed target path plus source locations | BuildKit PRs #999 and #4567 |
| DAG validity | cycle-free target graph | Bazel dependency contract |
| Dependency-first order | dependencies before requested roots | Go builder and Terraform walker |
| Ready set | targets with no unfinished dependencies | Go build actions |
| Parallel branches | several ready targets | Go and Terraform walkers |
| Declared versus actual edges | compare model with observed needs | Bazel strict dependency guidance |
| Unknown relation | not the same as no edge | Prometheus PR #15560 |
| Stable tie-break | lexical choice among ready targets | Argo retry issue #16450 and PR #16451 |
| Multi-parent node | retain every incoming relation | Argo retry issue #16450 and PR #16451 |
| Node identity | distinct nodes require distinct identities | Argo issue #16376 and PR #16625 |
| Resource version | current and pending-deletion records may share a URN | Pulumi PR #19179 |
| Model boundary | target DAG is not service recovery graph | GitHub deployment-safety report |

## Scope

### In scope

- finite directed graphs;
- target-to-dependency edge semantics;
- adjacency and reverse-adjacency lists;
- direct and transitive dependencies;
- BFS for a fewest-edge explanation path;
- DFS with active-path state for cycle detection;
- dependency-first build order;
- ready sets and legal parallel work;
- deterministic tie-breaking;
- declared, actual, missing, extra, and unknown dependencies;
- edge provenance and actionable cycle reports;
- O(V + E) time and space for the core traversals.

### Deferred

- weighted shortest paths;
- critical-path scheduling;
- distributed schedulers and worker leases;
- resource-capacity optimization;
- incremental graph algorithms;
- strongly connected component algorithms;
- graph databases;
- dynamic dependency discovery during execution;
- Bazel configuration transitions and aspects in depth;
- a faithful implementation of Skyframe;
- a full BUILD-file parser or Bazel integration.

Strongly connected components are valuable but not required for the first
graph unit. A single cycle witness is enough to reject the local build plan and
connect to the BuildKit patch.

## Learning outcomes

After the foundations lesson and lab, the learner should be able to:

1. State exactly what a node and a directed edge represent.
2. Explain why reversing an edge changes the questions the graph answers.
3. Build adjacency and reverse-adjacency indexes in Go.
4. Use BFS to return a fewest-edge dependency path.
5. Use DFS state to distinguish an active-path cycle from a harmless revisit.
6. Return a closed cycle with evidence a maintainer can act on.
7. Produce a deterministic dependency-first order for requested targets.
8. Explain why more than one valid build order can exist.
9. Derive the ready set and identify independent work.
10. Analyze traversal cost in terms of both nodes and edges.
11. Distinguish a correct algorithm from a complete and truthful graph.
12. Decide when a production relationship is not a DAG and should not be
    forced into one.

## Complexity contract

Let V be the number of targets in the requested subgraph and E be the number of
declared dependency edges among them.

| Operation | Expected time | Additional space |
| --- | ---: | ---: |
| Add a deduplicated edge | expected O(1) | O(1), excluding evidence text |
| Direct dependencies | O(out-degree) | O(out-degree) for a copied result |
| Reverse dependents | O(reverse out-degree) | O(reverse out-degree) |
| Dependency closure | O(V + E) | O(V) |
| Fewest-edge explanation path | O(V + E) | O(V) |
| One cycle witness | O(V + E) | O(V) |
| Dependency-first build order | O(V + E), plus tie-break cost | O(V) |

If the implementation repeatedly sorts every ready set, document the real
cost. For the small lab, a simple deterministic structure may be clearer than
a priority queue. The lesson should not advertise O(V + E) while hiding a
sorting term.

The adjacency maps consume O(V + E) space. Maintaining both directions roughly
duplicates edge references, not the evidence records themselves.

## Implemented Go lab: Build Graph Explorer

### Domain

Learners receive target declarations from a small repository-like fixture.
They implement or repair a package that can answer build questions and explain
invalid graphs.

The lab is Bazel-inspired but uses its own small text or Go fixture. It should
not require Bazel to be installed.

### Implemented types

~~~go
type TargetID string

type SourceLocation struct {
    File string
    Line int
}

type Dependency struct {
    Target TargetID
    Needs  TargetID
    Source SourceLocation
}

type Graph struct {
    // unexported indexes
}
~~~

`DependencyEdge` and `CycleEdge` expose one logical target-to-dependency edge
and its retained source locations. `TraversalStats` and `BuildStats` make node,
edge, and ready-set work observable without using elapsed time. The field names
encode direction: Target depends on Needs. The public input does not use a
generic From/To pair.

### Implemented operations

~~~go
func NewGraph(targets []TargetID, deps []Dependency) (*Graph, error)

func (g *Graph) Dependencies(target TargetID) ([]TargetID, error)
func (g *Graph) ReverseDependencies(target TargetID) ([]TargetID, error)
func (g *Graph) DependencyClosure(roots []TargetID) ([]TargetID, TraversalStats, error)
func (g *Graph) FewestEdgePath(start, want TargetID) ([]TargetID, bool, TraversalStats, error)
func (g *Graph) FindCycle() []CycleEdge
func (g *Graph) BuildOrder(roots []TargetID) ([]TargetID, BuildStats, error)
func (g *Graph) Ready(roots, completed []TargetID) ([]TargetID, error)
~~~

The implementation preserves these contracts:

- Unknown target is different from a known target with no dependencies.
- Dependencies returns direct dependencies, not the full closure.
- FewestEdgePath returns the fewest-edge path promised by the lab.
- FindCycle returns enough edge evidence to locate the declarations.
- BuildOrder includes only requested roots and their dependency closure.
- BuildOrder places dependencies before dependents.
- Ready is deterministic and contains only targets in the requested closure.
- Duplicate edge declarations do not change neighbor or readiness counts, but
  distinct source records for the same logical edge remain available in a
  cycle report.

### Error behavior

Prefer concrete errors:

- unknown target;
- unknown dependency named by a declaration;
- dependency cycle, with the closed path and source locations;
- duplicate source entry if the fixture format forbids exact duplicates.

A self-dependency may be accepted during graph construction and returned by
Cycle, or rejected immediately. Pick one contract and test it consistently.
For teaching cycle mechanics, returning it as a one-edge cycle is likely more
coherent.

### Determinism

Sort public result slices or use a deterministic ready structure. Go map
iteration must not decide:

- which valid order appears in examples;
- which equal-length explanation path is returned;
- which cycle is reported when several exist.

The documentation should say that deterministic output is an API choice, not a
graph-theory guarantee.

### Candidate fixture

~~~text
//app:server -> //lib:http        app/BUILD:12
//app:server -> //lib:config      app/BUILD:13
//lib:http  -> //lib:logging      lib/BUILD:8
//tool:migrate -> //lib:config    tool/BUILD:5
//tool:lint                        tool/BUILD:9
~~~

An invalid variant adds:

~~~text
//lib:logging -> //app:server     lib/BUILD:14
~~~

The fixture is intentionally small enough to draw by hand. Larger production
cases belong in the Wheels.

## Implemented Wheels

The Wheels reuse the foundations vocabulary but introduce a different failure
each time. Each production report or patch is inspiration, not a claim that
the exercise reproduces the entire product. Their symptom tests use build tags
`csbridgewheel8`, `csbridgewheel9`, and `csbridgewheel10`.

### W01: The Build That Loops Back

**Primary inspiration:** BuildKit PRs #999 and #4567, with Bazel's acyclic
target contract as context.

**Starting failure:** A graph validator records only a global visited set, or
recursively converts targets before validating the graph. A cyclic declaration
causes either a false diagnosis or unbounded recursion. The returned error says
only “cycle detected.”

**Learner work:**

- separate unseen, active, and finished DFS states;
- reconstruct a closed cycle;
- attach source references to the reported edges;
- handle a self-dependency;
- produce stable output when several cycles exist.

**Production question:** What information does a maintainer need to edit the
bad declaration?

**Guardrail:** Do not expose the full solution in the exercise prompt. Tests
may require the behavior without naming the exact implementation.

### W02: Unknown Is Not Independent

**Primary inspiration:** Prometheus PR #15560.

**Starting failure:** A dependency analyzer returns an empty slice both when a
target has no dependencies and when analysis lacks enough information. The
scheduler treats both as ready and runs work concurrently.

**Learner work:**

- represent known-empty, known-nonempty, and unknown states explicitly;
- choose conservative readiness behavior for unknown relations;
- preserve clear error or diagnostic output;
- test that genuinely independent work remains parallel.

**Production question:** What does a missing edge mean, and what evidence
supports that interpretation?

**Guardrail:** The prompt should not contain a hint or solution section.

### W03: Two Resources, One URN

**Primary inspiration:** Pulumi PR #19179.

**Starting failure:** A snapshot contains a current resource and an older copy
still awaiting deletion. Both have the same URN. A lookup that stores one
resource per URN overwrites one record with the other, so the dependency graph
associates an edge with the wrong concrete resource.

**Learner work:**

- represent current and pending-deletion records as distinct nodes;
- build separate indexes for the two states;
- distinguish create order from delete order;
- delete dependents before the dependency they still reference;
- test a snapshot containing more than one generation of a logical resource.

**Production question:** When two records share a logical name, which concrete
resource does a dependency edge refer to?

### Later candidates

Keep these in reserve rather than overloading the first three:

- multi-parent retry semantics inspired by Argo issue #16450 and merged PR
  #16451;
- node-ID collision and defensive traversal inspired by Argo issue #16376 and
  open PR #16625;
- declared-versus-actual dependency checking inspired by Bazel strict
  dependencies;
- recovery escape-path design inspired by GitHub's deployment-safety report.

## Writing guidance

The prose should favor transparency over shorthand.

Prefer:

> Starting from the server target, follow dependency edges one level at a time.
> The first time BFS reaches logging, the path uses the fewest dependency edges.

Avoid:

> BFS finds the closest dependency.

“Closest” hides the unit of distance.

Prefer:

> A target becomes ready when every direct dependency in this build has
> completed.

Avoid:

> The frontier unlocks downstream vertices.

Prefer:

> DFS reports a cycle when an edge returns to a target still on the current
> search path.

Avoid:

> A back edge violates DAG invariants.

The precise term may follow the explanation, but should not replace it.

Other wording rules:

- Complete every use of “depends on” with named node types.
- Say whether a path is arbitrary, fewest-edge, or weighted.
- Say whether an order is dependent-first, dependency-first, creation, or
  deletion order.
- Name the requested subgraph when discussing V and E.
- Distinguish “can run in parallel” from “will run in parallel.”
- Distinguish graph validity from graph completeness.
- Do not call a runtime service graph a DAG unless acyclicity is established
  for the exact modeled relation.
- Do not use “blast radius” without defining the concrete reverse-reachability
  question and its limits.

## Source and attribution rules

- Link the exact report, issue, pull request, or versioned source file used for
  each factual production claim.
- Pin source-code links to a release tag or commit when practical.
- Label inference as inference.
- Do not claim that a product uses BFS, DFS, or Kahn's algorithm unless its
  source or documentation establishes that fact.
- Do not claim Bazel somepath returns a shortest path.
- Do not equate Bazel's dependency-ordered query output with build execution
  order.
- Do not collapse Bazel target, configured-target, action, and Skyframe graphs.
- Do not describe every invalid directed graph as a DAG.
- Do not generalize one issue report into a product-wide architecture claim.
- Paraphrase patches and reports; quote only when exact wording is necessary.

## Implementation decisions

1. `FindCycle` is a separate whole-graph query. `BuildOrder` returns a
   structured `CycleError` when the requested closure is cyclic.
2. Sorted adjacency lists make equal-length BFS choices and the first cycle
   witness deterministic. The contract does not claim that the chosen path or
   cycle is mathematically unique.
3. One logical edge can retain several distinct source locations. Duplicate
   declarations do not inflate neighbor or dependency counts.
4. A lexical min-heap selects ready targets. The documented ordering cost is
   therefore `O(E + V log V)` for the requested closure.
5. The fixture uses the six `rules_go` labels developed in the foundations
   lesson. Bazel remains a linked production model; the Go lab does not require
   Bazel or parse BUILD files.
6. Declared-versus-actual checking remains a discussion and possible later
   Wheel. The initial implementation contains the three selected scenarios.

## Validation and maintenance

- Ordinary lab and Wheel tests must pass without build tags.
- Each of tags `csbridgewheel8`, `csbridgewheel9`, and `csbridgewheel10` must
  fail on the starting tree for its documented reason and pass after the
  debrief repair.
- Hugo must render the foundations, lab, Wheel overview, reports, candidate
  guides, evidence packets, and hidden debriefs without unresolved source-tree
  Markdown links.
- Source and status claims should be reviewed when pinned versions or upstream
  records change.
- The plain-language review was completed on 2026-08-14. Later wording changes
  must not change graph direction, ordering rules, source boundaries, or
  exercise answers.

## Current implementation

Bazel remains Unit 04's foundations example because a declared build target
dependency is explicit and supports traversal, cycle detection, readiness, and
ordering. The Go lab makes those operations inspectable. The BuildKit,
Prometheus, and Pulumi Wheels then show that a correct graph algorithm still
depends on validation, knowledge state, and concrete node identity. The lesson
ends with runtime service and recovery relationships as a modeling boundary:
not every dependency-shaped problem is already a DAG.
