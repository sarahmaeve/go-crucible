# Unit 03 Research and Design: Queues, Heaps, and Useful Retries

**Status:** foundations lesson, bounded top-k lab, and three Wheels implemented; dispatcher lab and case study pending

**Research reviewed:** 2026-08-12

**Target toolchain:** Go 1.27.x

## Decision

Build Unit 03 around one production claim:

> A scheduler is more than an ordering rule. Production behavior comes from
> which work is ready, which failed work must wait, which state change makes a
> retry useful, and what happens when arrivals exceed useful capacity.

The case-study spine will be the evolution of the Kubernetes scheduling queue.
It connects a binary priority heap to a public starvation report, a delayed
min-heap to retry backoff, and an unschedulable pool to event-specific
eligibility. It also has the evidence this track needs: reports from real
clusters, versioned Go source, design proposals, merged patches, rollout
history, and explicit limits.

The unit must not turn into a Kubernetes scheduler tour. Its organizing
question is:

> Where can the learner see each queue or heap concept doing consequential work
> in a real product?

Each core topic therefore needs a named production connection. A topic without
one is removed, shortened to prerequisite review, or left as an extension.

## Concept-to-production map

| CS concept | Public production connection | What the learner can inspect | Role |
|---|---|---|---|
| FIFO queue | Kubernetes `client-go` work queues | A slice-backed FIFO plus the `dirty` and `processing` sets that coalesce duplicate controller work | Prerequisite and Go implementation |
| Priority-queue interface | Kubernetes scheduler ActiveQ | The highest-priority pending Pod is selected first, with enqueue time used for equal priorities | Core positive implementation |
| Binary-heap invariant | Go `container/heap` and Kubernetes's heap wrapper | Array representation, root access, `Push`, `Pop`, `Remove`, indexed updates, and `Fix` | Core CS and Go bridge |
| Bounded top-k | Prometheus 3.13.1 PromQL `topk` | A heap of at most `k` samples; a better candidate replaces the root and calls `heap.Fix` | Small positive application |
| Delayed min-heap | Kubernetes `client-go` delaying queue and scheduler BackoffQ | The earliest `readyAt` or backoff expiry is at the root; an earlier duplicate deadline uses `heap.Fix` | Core implementation |
| Exponential retry backoff | Kubernetes issue #81214 and PR #81263 | A report from a cluster with 5,000 nodes and more than 100,000 Pods, plus the patch that made initial and maximum scheduler backoff configurable | Core production history |
| Terminal retry classification | GitHub Actions incident, August 6–7, 2026 | An official report separating the capacity failure that formed a backlog from invalid-job retries that prevented runners from taking valid work | Wheel production inspiration |
| Eligibility and useful requeueing | Kubernetes QueueingHint | Plugins decide whether a particular event could change a particular rejection instead of treating a broad event class as sufficient | Core case-study decision |
| Retry amplification | Scheduler-plugins issue #682 and PR #700 | Repeated sibling activation accompanied millions of log lines and a CPU spike; a merged patch marks the scheduling attempt that may activate the group | Wheel and patch study |
| Queue-owned state lifetime | Kubernetes issue #120622 and PRs #126962 and #127016 | In-flight event state could be retained when an item skipped its expected completion path or appeared twice | Case-study caution |
| Fairness and bounded overload | Kubernetes API Priority and Fairness | Bounded queues, priority levels, flow isolation, and explicit memory/latency/fairness trade-offs | Production boundary |
| When a heap is the wrong timer | Kafka request purgatory | A DelayQueue-based design retained completed requests; hierarchical timing wheels enabled immediate removal and changed the cost profile | Alternative-structure boundary |

This map keeps three uses of production evidence distinct:

- **Positive implementation:** a public product uses the structure for a
  concrete operation, such as Prometheus retaining the best `k` samples.
- **Reported failure and patch:** a production report exposes a violated model
  and upstream code changes address it, as in the Kubernetes scheduling cases.
- **Redesign boundary:** a team replaced the structure because a different
  workload operation dominated, as in Kafka's timer redesign.
- **Incident synthesis:** a report connects queue depth, usable capacity, and
  terminal retry decisions without establishing that the product used a heap,
  as in the GitHub Actions recovery story.

The binary heap is not presented as the hero of every story. It efficiently
maintains a partial order. It does not supply retry policy, admission control,
deduplication, fairness, or state ownership by itself.

## Why this scope

The provisional curriculum row combined FIFO queues, priority queues, heaps,
amortized operations, backpressure, fairness, and deadlines. Taught as a list,
that would become a survey with infrastructure nouns pasted onto textbook
examples. The Kubernetes scheduling queue supplies one coherent lifecycle:

```text
new or newly eligible work
        |
        v
active priority heap -----> attempt
                              |
                 +------------+-------------+
                 |                          |
              success                  cannot run
                                            |
                         +------------------+------------------+
                         |                                     |
                  retry after time                     wait for a change
                         |                                     |
                 delayed min-heap                    unschedulable pool
                         |                                     |
                         +---------- useful event -------------+
```

That lifecycle lets the learner ask a production question at every transition:

- What is the ordering key, and is the tie-break deterministic?
- Is the item eligible now, or merely important?
- What changes after a failed attempt?
- Is the next retry driven by elapsed time, a relevant event, or both?
- Can duplicate notifications multiply physical queue entries or attempts?
- Who removes per-item state on every exit path?
- Can runnable lower-priority work make progress?
- What queue depth and wait-time signals expose overload?

### In scope

- FIFO semantics and controller-style duplicate coalescing
- The priority-queue abstract interface
- Array-backed binary min-heaps and max-heaps
- The heap-order invariant and index arithmetic
- `O(1)` root inspection; `O(log n)` push, pop, remove, and fix; and
  `O(n)` bottom-up heap construction
- Explicit tie policy and deterministic compound priority keys
- Bounded top-k selection as a compact non-scheduling heap application
- Ready, delayed, blocked, in-flight, and completed work states
- Capped exponential backoff and ready-time ordering
- Event-specific requeue eligibility
- Duplicate suppression and retry amplification
- Starvation and fairness as observed policy properties, not heap guarantees
- Queue depth, futile attempts, wait time, and retained state as operational
  signals
- Pure simulation, fake time, and deterministic operation counts

### Deliberately out of scope

- CPU scheduling, preemption algorithms, and operating-system internals
- Kubernetes node filtering, scoring, binding, or the full plugin framework
- Implementing gang scheduling or resource allocation
- A full treatment of fair queuing, shuffle sharding, or queuing theory
- Distributed queue delivery guarantees, consensus, and exactly-once claims
- Kafka's hierarchical timing-wheel implementation
- General graph scheduling and workflow DAGs
- Learned ranking or recommendation objectives
- Treating goroutine concurrency as a prerequisite for understanding the data
  structures

Kubernetes API Priority and Fairness may show why strict priority and unbounded
queues are insufficient under overload, but the guide should not implement its
algorithm. Kafka may show why random cancellation and timer cardinality can
favor a timing wheel, but the guide should not implement one. Those boundaries
teach structure selection without doubling the unit's scope.

## Learning outcomes

After the unit, a learner should be able to:

- Separate a queue's item order from eligibility, retry timing, capacity, and
  ownership policy.
- Explain the priority-queue operations a client needs before choosing a
  representation.
- Derive the binary-heap parent and child indexes and state the heap-order
  invariant.
- Explain why a heap exposes only the next item in constant time rather than
  maintaining a fully sorted sequence.
- Use Go's `container/heap` contract correctly, including calling
  `heap.Fix` after changing an indexed item's priority.
- Compare sorting all `r` candidates with retaining a bounded heap of size
  `k`.
- Model a delayed queue as a min-heap ordered by readiness time.
- Explain why retry backoff limits repeated work but does not decide whether a
  retry is useful.
- Explain how a broad cluster event can reactivate impossible high-priority
  work and starve runnable lower-priority work.
- Detect retry amplification by counting activation or attempt operations
  rather than relying on elapsed-time tests.
- State the lifecycle invariant that each logical item occupies exactly one
  queue-owned state and that every terminal path releases associated state.
- Recognize when cancellation, removal frequency, deadline resolution, or
  timer cardinality makes a heap a poor fit.
- Attribute reported production symptoms, upstream implementation details, and
  local synthetic measurements separately.

## Cost model

The guide should name workload variables:

- `a`: items in the active queue
- `d`: items waiting for a deadline or backoff expiry
- `u`: items in the unschedulable or event-blocked pool
- `r`: candidates considered by a top-k query
- `k`: results retained by top-k
- `h`: later top-k candidates that replace the current cutoff, where
  `0 <= h <= r-k`
- `e`: cluster or eligibility events
- `g`: members of one related work group
- `x`: scheduling or reconciliation attempts
- `f`: failed attempts for one item
- `c`: useful processing capacity per interval

Representative modeled costs:

| Operation or policy | Modeled work | Important condition |
|---|---:|---|
| FIFO append and removal from a suitable queue | amortized `O(1)` | Representation does not repeatedly shift the remaining slice |
| Inspect heap root | `O(1)` | Heap invariant is valid |
| Push or pop one heap item | `O(log n)` | Comparator is consistent |
| Change an indexed priority and call `Fix` | `O(log n)` | The stored index is updated on every swap |
| Build a heap bottom-up | `O(n)` | Use initialization, not `n` independent pushes |
| Sort all `r` candidates | `O(r log r)` | Produces a total order that may be unnecessary |
| Select `k` best candidates with a bounded heap | `O(r + h log k)` time, `O(k)` working storage | `1 <= k <= r`; initialize the first `k` bottom-up, compare all later candidates with the cutoff, and restore order only after replacements |
| Put the `k` retained winners in result order | `O(k log k)` | The contract needs an ordered result rather than only the winning set |
| Pop the next delayed item | `O(log d)` after `O(1)` peek | Root is ordered by earliest readiness |
| Scan all delayed items on every tick | proportional to ticks times `d` | A deliberately poor comparison path |
| Broadly reconsider every blocked item for every event | up to `O(eu)` eligibility checks, plus resulting attempts | Events are not filtered by rejection reason |
| Every one of `g` members activates `g-1` siblings | `Theta(g^2)` activation requests | Logical queue deduplication does not erase request-generation cost |
| Capped exponential delay | `min(base * 2^(f-1), cap)` | Overflow and attempt limits are handled explicitly |

The unit should be precise about amortization. A binary heap's path length gives
the logarithmic bound for reordering. Dynamic-array capacity changes are a
separate amortized cost inherited from its storage. Go's documented
`container/heap` operations state their own bounds; the guide should not
invent a stronger language guarantee for slice growth.

The top-k model should likewise expose the operations hidden by its usual
worst-case `O(r log k)` summary. When `k < r`, exact selection
from unsorted input must inspect all `r` candidates; when `k = r`,
ordered output still reads and sorts them all. Only `h` candidates cause a
logarithmic heap update, and ordered output adds a separate
`O(k log k)` sort. For fixed `k`, selection grows linearly with
`r`; when `k` approaches
`r`, the final winner sort approaches a full sort. Comparison counts also
assume bounded-size keys. A tied comparison of arbitrary-length Go strings is
not constant-size work.

### Ordering is not eligibility

If priority is descending, an impossible high-priority item can remain more
important than every runnable item. Removing it, failing it, and immediately
reinserting it simply causes the heap to select it again. The heap is behaving
correctly.

Backoff changes when the item can return. An event-specific hint changes which
state transition is justified. Fairness changes which classes are allowed to
make progress. These are separate policy layers and should have separate local
types, counters, and tests.

### Logical deduplication is not free

A queue may prevent the same logical key from occupying several physical queue
positions while callers still generate a quadratic number of activation
requests, lock acquisitions, map lookups, logs, or callbacks. Scheduler-plugins
issue #682 is valuable because it makes that distinction operationally visible:
coalescing at the queue boundary did not make aggressive sibling activation
cheap.

## Production-source selection

Candidates were judged on:

1. A real workload or product operation
2. A symptom or design consequence tied to the proposed CS concept
3. A first-party report, official source, or public issue
4. A versioned implementation or patch that can be inspected
5. A local exercise that can preserve the mechanism without claiming to
   reproduce the product

| Candidate | Production value | Limitation | Unit role |
|---|---|---|---|
| [Kubernetes scheduler starvation report #81214](https://github.com/kubernetes/kubernetes/issues/81214) | Real cluster with 5,000 nodes and more than 100,000 Pods; high-priority unschedulable work repeatedly precluded lower-priority work | The first related patch made backoff configurable but did not by itself solve selective requeueing | **Case-study opening** |
| [Kubernetes backoff options PR #81263](https://github.com/kubernetes/kubernetes/pull/81263) | Small historical patch connecting the report to configurable initial and maximum backoff | Configuration relieves pressure; it is not the later QueueingHint policy | **Patch step one** |
| [Kubernetes QueueingHint history](https://kubernetes.io/blog/2024/12/12/scheduler-queueinghint/) and [KEP-4247](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/4247-queueinghint/README.md) | Official queue model, overly broad retries, per-plugin event relevance, rollout reversal, and documented risks | The complete implementation spans many plugins and must be reduced to one callback model | **Selected design step** |
| [Kubernetes in-flight state issue #120622](https://github.com/kubernetes/kubernetes/issues/120622) and [memory fix #126962](https://github.com/kubernetes/kubernetes/pull/126962) | The new event-tracking design exposed retained-state failure paths; the PR explains one concrete cleanup omission | Several patches addressed the area, so one PR must not be presented as the sole universal fix | **Lifecycle caution** |
| [GitHub Actions incident, August 6–7, 2026](https://www.githubstatus.com/incidents/qcvjkzcs7j74) | Official two-stage account: capacity loss formed a backlog, then runners repeatedly tried invalid jobs and could not take valid work | It names behavior and mitigations, not GitHub's private queue structure or the exact incident patch | **Wheel 01 production inspiration** |
| [Scheduler-plugins report #682](https://github.com/kubernetes-sigs/scheduler-plugins/issues/682) and [fix #700](https://github.com/kubernetes-sigs/scheduler-plugins/pull/700) | Production ML workload, millions of logs, CPU spike, unrelated runnable work blocked, and a compact merged Go patch | Gang scheduling itself is outside scope | **Wheel 03 and patch deep read** |
| [Kubernetes v1.32 scheduler queue](https://github.com/kubernetes/kubernetes/blob/v1.32.0/pkg/scheduler/backend/queue/scheduling_queue.go) | Pinned Go source containing ActiveQ, BackoffQ, unschedulable state, and QueueingHint decisions | Large file; only a named path should be read | **Implementation companion** |
| [Kubernetes client-go v0.32.0 workqueue](https://github.com/kubernetes/client-go/tree/v0.32.0/util/workqueue) | Small reusable Go implementation of FIFO, delayed, and rate-limited work | It is implementation evidence, not an incident narrative | **Go companion** |
| [Prometheus 3.13.1 `topk`](https://github.com/prometheus/prometheus/blob/v3.13.1/promql/engine.go#L3951-L4085) | Familiar SRE query, bounded heap, root replacement, and `heap.Fix` in public Go source | No incident or redesign narrative | **Positive application** |
| [Kafka request-purgatory redesign](https://www.confluent.io/blog/apache-kafka-purgatory-hierarchical-timing-wheels/) and [KAFKA-2147](https://issues.apache.org/jira/browse/KAFKA-2147) | Real retained-request memory failure, a different timer structure, benchmark data, public issue, and attached patches | Java implementation and timing-wheel mechanics are too large for the core lab | **Alternative-structure boundary** |
| [Kubernetes API Priority and Fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/) | A real overload-control design connecting bounded queues, priority, fairness, memory, and latency | Full shuffle sharding would introduce another algorithms unit | **Backpressure boundary** |
| [LinkedIn feed architecture](https://engineering.linkedin.com/teams/data/artificial-intelligence/feed) | Real multi-stage top-k vocabulary carried forward from Unit 02 | It does not expose the heap or a queue failure closely enough | Context only |

The Kubernetes scheduler is selected because it supplies the most complete
concept-to-production chain. Prometheus prevents “heap” from becoming a synonym
for “scheduler.” Kafka prevents “heap” from becoming the default answer to
every deadline problem.

## Required source spine

### CS model

- [MIT 6.006 Lecture 8: Binary Heaps](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/40d4851e550507ca14dc778b9b2266cc_MIT6_006S20_lec8.pdf)
  defines the priority-queue interface, array-backed complete tree, heap-order
  invariant, logarithmic insert/delete, and linear bottom-up construction.
- [Open Data Structures: Heaps](https://opendatastructures.org/newhtml/ods/latex2/heaps.html)
  supplies an accessible second derivation and makes array-resize amortization
  explicit.

The guide needs only a short FIFO recap from Unit 02's sequence material. It
should not assign another generic queue chapter before reaching the production
work queue.

### Go contract

- [Go `container/heap` documentation](https://pkg.go.dev/container/heap)
- [Go 1.27 `container/heap` source](https://go.dev/src/container/heap/heap.go)

Claims to make precisely:

- `heap.Interface` extends `sort.Interface` with package-called `Push` and
  `Pop` methods.
- The item selected at index zero follows `Less`; a max-heap reverses that
  comparison.
- `Init` is `O(n)`.
- `Push`, `Pop`, `Remove`, and `Fix` are `O(log n)`.
- Mutating an item's priority does not repair the heap automatically.
- The caller owns item identity, indexes used for updates, concurrency control,
  deduplication, and tie policy.

### Pinned Go implementation paths

Read only these small paths:

1. [Kubernetes v1.32.0 priority comparator](https://github.com/kubernetes/kubernetes/blob/v1.32.0/pkg/scheduler/framework/plugins/queuesort/priority_sort.go)
   for priority followed by queue timestamp.
2. [Kubernetes v1.32.0 heap wrapper](https://github.com/kubernetes/kubernetes/blob/v1.32.0/pkg/scheduler/backend/heap/heap.go)
   for the slice of keys, key-to-item/index map, `AddOrUpdate`, `Remove`, and
   `Fix`.
3. [Kubernetes v1.32.0 scheduling queue](https://github.com/kubernetes/kubernetes/blob/v1.32.0/pkg/scheduler/backend/queue/scheduling_queue.go)
   for ActiveQ, the backoff-expiry heap, unschedulable Pods, and the
   `queueSkip` / `queueAfterBackoff` / `queueImmediately` decision.
4. [client-go v0.32.0 FIFO queue](https://github.com/kubernetes/client-go/blob/v0.32.0/util/workqueue/queue.go)
   for FIFO storage and dirty/processing state.
5. [client-go v0.32.0 delaying queue](https://github.com/kubernetes/client-go/blob/v0.32.0/util/workqueue/delaying_queue.go)
   for the earliest-ready min-heap, fake-clock injection, duplicate deadline
   handling, and `heap.Fix`.
6. [client-go v0.32.0 rate-limiting queue](https://github.com/kubernetes/client-go/blob/v0.32.0/util/workqueue/rate_limiting_queue.go)
   for the boundary between “when may this retry?” and queue storage.
7. [Prometheus 3.13.1 top-k implementation](https://github.com/prometheus/prometheus/blob/v3.13.1/promql/engine.go#L3951-L4085)
   for a second product using the same Go heap contract for a different
   operation.

Version 1.32 is deliberate: it matches the release in which QueueingHint was
enabled by default again. These source links are implementation evidence, not
promises about current or future Kubernetes releases.

### Production evidence

Required:

- [Kubernetes issue #81214](https://github.com/kubernetes/kubernetes/issues/81214)
- [Kubernetes PR #81263](https://github.com/kubernetes/kubernetes/pull/81263)
- [Kubernetes v1.32 QueueingHint article](https://kubernetes.io/blog/2024/12/12/scheduler-queueinghint/)
- [KEP-4247: per-plugin callbacks for efficient requeueing](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/4247-queueinghint/README.md)
- [Scheduler-plugins issue #682](https://github.com/kubernetes-sigs/scheduler-plugins/issues/682)
- [Scheduler-plugins PR #700](https://github.com/kubernetes-sigs/scheduler-plugins/pull/700)
- [Kubernetes in-flight state issue #120622](https://github.com/kubernetes/kubernetes/issues/120622)
- [Kubernetes QueueingHint memory fix #126962](https://github.com/kubernetes/kubernetes/pull/126962)
- [GitHub Actions incident, August 6–7, 2026](https://www.githubstatus.com/incidents/qcvjkzcs7j74)

Related implementation proposal, not part of the required source spine:

- [actions/runner PR #4618](https://github.com/actions/runner/pull/4618),
  an open community pull request as of August 12, 2026. It provides an
  inspectable distinction between terminal lost-assignment responses for an
  ephemeral runner and responses that retain the existing retry behavior. It
  is not evidence of GitHub's official incident fix unless GitHub says so.

Required alternative-structure reading:

- [Apache Kafka, Purgatory, and Hierarchical Timing Wheels](https://www.confluent.io/blog/apache-kafka-purgatory-hierarchical-timing-wheels/)
- [KAFKA-2147: extreme purgatory growth](https://issues.apache.org/jira/browse/KAFKA-2147)

Optional production boundary:

- [Kubernetes API Priority and Fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/)

## Case-study design

### Working title

**When priority keeps choosing impossible work**

### Production opening

Begin with Kubernetes issue #81214, not with a heap diagram. The report came
from a cluster with 5,000 nodes and more than 100,000 Pods. Lower-priority Pods
could wait a long time because unschedulable higher-priority Pods repeatedly
returned from backoff to the active priority queue after broad cluster events.
The higher-priority work was selected again before the runnable lower-priority
work got a chance.

Ask the learner to predict which of these changes would alter the symptom:

1. Make heap push and pop faster.
2. Increase the maximum backoff.
3. Requeue only Pods whose failed constraint may have changed.
4. Periodically ignore priority.
5. Bound the number of attempts per interval.

Several can change the outcome, but they act on different layers. Faster heap
operations do not repair the policy.

### Narrative sequence

#### 1. Priority and backoff expose the problem

The active queue's priority ordering is useful: important runnable work should
not sit behind an arbitrary arrival order. The backoff queue is also useful:
known failed work should not consume every serial scheduling cycle.

Issue #81214 shows their interaction. Broad events moved unschedulable work
back toward ActiveQ. A maximum backoff of ten seconds was too short to keep
repeated high-priority failures from dominating the reported workload.

PR #81263 refactored the scheduling queue options and added configurable
initial and maximum Pod backoff durations. Present this as a pressure-relief
and operability change, not as the final selective-requeue solution.

#### 2. Eligibility becomes specific to the rejection

The official QueueingHint account describes three queue-owned states:

- ActiveQ for new or retry-ready Pods
- BackoffQ for retryable Pods whose delay has not elapsed
- an unschedulable pool for Pods whose failed condition has not relevantly
  changed

Before QueueingHint, plugins registered broad event classes. A Node update, for
example, often could not change the NodeAffinity result for a particular Pod.
QueueingHint lets the plugin that rejected a Pod inspect the particular event
and return whether that Pod is worth queueing.

The local model should reduce this to:

```go
type Hint func(item Item, event Event) RequeueDecision
```

The guide should paraphrase the interface and decision, not copy Kubernetes
plugin code or imply that the local dispatcher implements its framework.

#### 3. The optimization creates new retained state

Events can occur while a Pod is being scheduled against a snapshot. To avoid
missing an event that would make the Pod eligible, QueueingHint tracks events
that occur while work is in flight. That improves correctness but creates a
new state-lifetime obligation.

The feature was enabled by default experimentally in Kubernetes 1.28, disabled
in a patch release after memory reports, and enabled by default again in 1.32
after further implementation and fixes.

Issue #120622 records why the state can be expensive: all relevant cluster
events during scheduling were accumulated for later queue decisions. PR
#126962 describes one leak path in which a skipped Pod did not traverse the
normal requeue or bind cleanup, leaving an in-flight marker that retained later
events. PR #127016 added protection against duplicate in-flight instances.

This is an important production conclusion: reducing retries may require more
state, and every new state needs explicit ownership, cardinality signals, and
cleanup on exceptional paths.

#### 4. Aggressive activation amplifies work before deduplication

Scheduler-plugins issue #682 came from production ML training workloads. When
quota or resources kept a Pod group pending, the scheduler sometimes stopped
making useful progress on unrelated schedulable Pods while emitting millions
of log lines and consuming much more CPU.

The public investigation found that each group member could try to reactivate
its siblings after the group became eligible. For a group of `g` members,
that produces a `Theta(g^2)` activation-request shape even if the downstream
queue coalesces duplicate keys.

PR #700 records a small, teachable repair: it marks the state for a single Pod's
scheduling attempt only when that attempt sees zero assigned group members, and
the later sibling-activation step requires that mark. W03 models the resulting
one-activation bound in a maintenance-coordination domain, but its Boolean is
shared for one synthetic pass rather than owned by one scheduling attempt.

### Source attribution

The case and Wheels must distinguish:

- **Reported by a Kubernetes user:** a workload of more than 100,000 Pods on
  5,000 nodes, repeated high-priority reactivation, and lower-priority waiting
  in #81214.
- **Implemented in PR #81263:** configurable initial and maximum backoff.
- **Documented by Kubernetes:** the three queue states, broad-event problem,
  QueueingHint decision, 1.28 disablement, and 1.32 re-enablement.
- **Described by KEP-4247:** the per-plugin callback, risk of missed events,
  periodic safety flush, in-flight event tracking, and memory risk.
- **Reported in scheduler-plugins #682:** production ML use, pending groups,
  millions of logs, CPU increase, and loss of useful scheduling progress.
- **Implemented in scheduler-plugins #700:** an `Activate` flag on the Pod
  attempt that sees zero assigned group members; sibling activation requires
  that attempt-local flag.
- **Implemented in Kubernetes memory patches:** specific cleanup and duplicate
  in-flight protections; neither should be described as proof that no other
  leak was possible.
- **Reported by GitHub:** the deployment and capacity trigger, accumulated
  queued work, repeated attempts to acquire invalid jobs, blocked valid work,
  and the incident mitigation that stopped the repetitions.
- **Proposed in actions/runner #4618:** public control flow for retiring an
  ephemeral runner after selected lost-assignment responses. The open pull
  request is related implementation evidence, not an official root-cause
  patch identified by the incident report.
- **Modeled locally:** work-item states, event kinds, exact operation counts,
  fairness assertion, and all synthetic timings or fixtures.

The local lab does not reproduce Kubernetes scale, plugin behavior, or
throughput. Its purpose is to preserve the queue transitions and let the
learner falsify explanations deterministically.

## Exploration-lab design

### Working title

**Dispatching repair work without retry storms**

Use a fleet-repair dispatcher rather than Pods. This forces transfer of the CS
model and avoids producing a toy Kubernetes clone.

### Part A: one heap, one bounded query

Start with a short top-k exercise over synthetic service-risk observations:

```go
type Candidate struct {
    Service string
    Score   float64
}
```

Compare sorting all `r` candidates with maintaining a min-heap of at most
`k`. Build the initial size-`k` heap bottom-up. Then inspect the
current worst retained candidate at the root, replace it when a better sample
arrives, and restore the invariant. Count cutoff comparisons, replacements,
heap-construction work, heap-restoration work, retained candidates, and the
final sort of the winners as separate phases. Tie the operation to Prometheus's
public `topk` implementation.

This section exists to teach the heap without scheduler policy. It should be
small enough that the learner can draw the array and count comparisons and
swaps. It should also cover `k = 1`, fixed small `k`, `k`
near `r`, best-first versus worst-first arrival, and the difference between
`O(k)` additional working storage and end-to-end streaming memory. Compare
the bounded heap with full sorting, a full bottom-up max-heap, quickselect plus
winner sorting, and a sorted size-`k` slice without requiring all of those
alternatives to be implemented.

### Part B: two orders and an eligibility decision

Use repair work with explicit production-shaped fields:

```go
type WorkItem struct {
    ID         string
    Tenant     string
    Priority   int
    Sequence   uint64
    Attempts   int
    EnqueuedAt time.Time
}

type FailureReason struct {
    Kind string
    Key  string
}

type Event struct {
    Kind string
    Key  string
}
```

Keep readiness time in the delayed-queue entry, not as an overloaded meaning
of priority. Use these deterministic orders:

- Active max-heap: `(priority DESC, sequence ASC, ID ASC)`
- Delayed min-heap: `(readyAt ASC, sequence ASC, ID ASC)`

The unique ID tie-break makes local output repeatable even when timestamps or
priorities tie. It is a local contract, not a claim about Kubernetes's exact
comparator.

### Variants

1. `FIFO` is a correctness reference for arrival order.
2. `ActiveHeap` selects the highest-priority eligible repair.
3. `DelayedHeap` exposes only repairs whose capped backoff has elapsed.
4. `BlockedPool` records the failure reason and waits for events.
5. `BroadRequeue` reactivates every blocked item for a matching event class.
6. `HintedRequeue` reactivates only items whose reason key can be changed by
   the event.
7. A bounded-admission extension rejects or defers arrivals explicitly after a
   configured limit.

Implement the simulator as a single-threaded state machine with an injected
or explicitly advanced time. Goroutines and wall-clock sleeps would obscure
the structure and make the evidence less deterministic.

### State invariant

Every submitted logical item is in exactly one state:

```text
active | delayed | blocked | in-flight | completed | rejected
```

Maintain one identity index so moving an item is explicit. A state transition
must remove the previous representation before adding the next. Completion,
rejection, and cancellation must remove all queue-owned metadata.

The lab should include a retained-state experiment based on the QueueingHint
history: skip one completion path in a deliberately defective variant and show
that an in-flight marker pins later event records. The repair is a lifecycle
operation, not a garbage-collector tuning exercise.

### Workload scenarios

1. One permanently blocked high-priority repair, many runnable lower-priority
   repairs, and frequent irrelevant events
2. A temporarily blocked high-priority repair followed by the one event that
   makes it eligible
3. Many failures sharing the same reason key
4. Equal priorities and equal readiness times
5. Duplicate submissions while an item is queued and while it is in flight
6. Arrival rate above useful completion capacity
7. Group activation sizes `g = 4, 16, 64, 256`

### Deterministic counters

Counters are primary evidence:

- heap comparisons and swaps
- pushes, pops, removes, and fixes
- scheduling attempts
- futile attempts
- activation requests and distinct items activated
- items skipped by a hint
- maximum active, delayed, blocked, and in-flight cardinality
- retained event records
- completed item IDs and completion order
- per-priority and per-tenant wait intervals
- rejected or deferred arrivals

Benchmarks and allocation profiles support the counters. They are not the only
pass/fail signal.

### Correctness properties

- No delayed item is dispatched before its `readyAt`.
- The active and delayed heap invariants hold after every mutation.
- Changing an indexed key uses `Fix` or an equivalent remove-and-push.
- A logical ID appears in exactly one state.
- Duplicate notification does not create duplicate physical work.
- A successful or permanently failed item clears retry and in-flight state.
- Capped backoff never overflows and never exceeds the configured cap.
- An irrelevant event activates no blocked item in the hinted variant.
- A relevant event activates exactly the affected blocked items.
- Under the stated policy, runnable lower-priority work progresses while
  impossible higher-priority work is delayed or blocked.
- A bounded queue reports rejection or deferral explicitly rather than growing
  silently.

### Predictions required before execution

- Which top-k operations depend on `r`, which depend on `k`, and which
  depend on the number of candidates that cross the current cutoff?
- Why is the root of the delayed heap the next useful deadline but not
  necessarily the most important item?
- What changes if an item priority is mutated without restoring heap order?
- How many activation requests does broad group activation produce as `g`
  doubles?
- Why can a deduplicating queue still suffer CPU and logging amplification?
- Which event distributions make QueueingHint valuable?
- What retained state is added to avoid missing an event during processing?
- At what arrival rate does queue depth grow even when every queue operation is
  fast?

## Wheel of Misfortune designs

### W01: The Lost Assignment Loop

**Upstream basis:** GitHub's official August 6–7, 2026 Actions incident
report. The open actions/runner PR #4618 is a related code-reading option, not
an official incident fix.

**Incoming report:** A deployment temporarily reduces assignment-service
capacity and a valid-work backlog forms. Capacity is restored and new arrivals
are throttled, but connected ephemeral workers continue retrying assignment
references that the service reports as missing or superseded. Newly started
workers complete valid jobs.

**Hidden first local cause:** The worker maps every unsuccessful acquire result
to `RetryAssignment`. A retry keeps the same assignment. Missing and
superseded assignments therefore occupy every worker slot without a possible
successful transition.

**Evidence packets:**

1. Capacity, arrival rate, connected workers, and backlog across two recovery
   stages
2. Acquire attempts, completions, session retirements, and lost-assignment
   responses
3. A trace showing one worker repeatedly requesting the same absent assignment
4. The response contract separating transient from terminal outcomes

**Repair:** Retain retries when the assignment may become obtainable. Retire
an ephemeral session when the named assignment is missing or superseded so its
supervisor can start a worker capable of accepting valid queued work.

**Deterministic verification:**

- Every initially occupied worker slot reaches a terminal exit for a missing
  or superseded assignment.
- Valid work behind those assignments completes within a stated attempt
  budget.
- Service-unavailable and rejected-request results retain their retry path.
- No assertion depends on wall-clock timing.

This Wheel is intentionally not a heap exercise. It uses the unit's broader
scheduling model to show that queue depth and process count do not equal usable
capacity. Neither the official report nor the local exercise claims that
GitHub's runner-assignment path used a heap.

### W02: The Event That Woke Everything

**Upstream basis:** Kubernetes issue #81214, the QueueingHint article, and
KEP-4247.

**Incoming report:** A fleet-repair dispatcher has plenty of worker capacity
for routine repairs, but ordinary work waits for minutes. CPU climbs whenever
the inventory watcher delivers a burst of metadata changes. The queue's
highest-priority item repeatedly fails because its required hardware class
does not exist.

**Hidden first local cause:** Every inventory event with the same broad kind as
a blocked condition moves that repair back to the active priority heap. The
impossible high-priority repair is selected, fails, and returns again. The heap
and priority comparator are correct; the event rule is too broad.

**Evidence packets:**

1. Queue depth, pending age by priority, and attempts per completion
2. Event kinds alongside activation and futile-attempt counts
3. Active, blocked, and completed state snapshots for one repair
4. Active ordering, the event check, and the readiness check

**Repair:** Record which constraint rejected each repair and call a
reason-specific hint for the concrete event. Irrelevant metadata events leave
the repair blocked; the matching capacity event reactivates it.

**Deterministic verification:**

- `m` irrelevant events cause zero activations and zero new futile attempts
  for the blocked repair.
- The one matching event activates that repair exactly once.
- Runnable lower-priority repairs complete before the matching event.
- No claim depends on wall-clock timing.

The debrief should explicitly map the local condition key to QueueingHint's
idea while stating that a matching key permits another attempt but does not
prove readiness. The exercise is synthetic and much smaller.

### W03: The Sibling Stampede

**Upstream basis:** Scheduler-plugins issue #682 and merged PR #700.

**Incoming report:** A coordinated maintenance group waits until capacity is
available for all members. When an update makes the group ready to try again,
the dispatcher emits many “already active” messages, CPU spikes, and unrelated
single-item repairs stop making useful progress.

**Hidden first local cause:** Once the group reaches its activation threshold,
each of its `g` members requests activation of the other `g-1` members.
Queue deduplication limits physical duplicates but does not prevent
`Theta(g^2)` calls, lookups, locks, and logs.

**Evidence packets:**

1. Request count, duplicate logs, queue depth, and unrelated progress
2. Activation requests versus distinct activated IDs as group size grows
3. A call-count trace for four group members
4. Where the repeated sibling loop begins

**Local repair:** Store one Boolean in synthetic state shared for one pass. The
first member examined after the group becomes ready activates its siblings;
later members observe the recorded action and do not repeat it.

**Deterministic verification:**

- A group of `g` items produces at most `g-1` sibling activation requests
  for the transition, rather than `g(g-1)`.
- All intended siblings become eligible.
- An unrelated item is dispatched within a stated operation bound.
- The patch changes activation policy, not the heap implementation.

The debrief must not present the shared local `RoundState` as Kubernetes's
mechanism. Kubernetes stores the flag for one Pod's scheduling attempt and
selects that attempt using the assigned-member count. The local model shares a
Boolean across an entire pass only to isolate the request-count lesson.

## Alternative-structure boundary: Kafka request purgatory

Kafka's delayed produce and fetch requests complete either when another
cluster action satisfies them or when a timeout expires. The first-party
request-purgatory account describes tens of thousands of in-flight requests.
Its older DelayQueue-and-watcher design did not immediately remove completed
requests. Cleanup scans could lag, retain heap objects, and lead to JVM
out-of-memory failure; scanning more often traded memory for CPU.

Kafka moved timer management to hierarchical timing wheels with linked bucket
entries. The reported design made completed-request removal constant time and
kept only a bounded number of buckets in a DelayQueue. Its published local
benchmark saturated the old design around 40,000 requests per second in one
short-timeout case and 25,000 in a longer-timeout case, while the new design
reached roughly 105,000 requests per second in the latter benchmark.

Teach the result as workload-specific evidence:

- A heap is attractive when exact next-deadline access dominates.
- A timing wheel is attractive when timer cardinality is large, deadline
  resolution is bounded, and cancellation/removal is extremely frequent.
- Kafka's benchmark is not a Go result and not a universal timing-wheel
  guarantee.
- The local unit does not implement a timing wheel.

KAFKA-2147 supplies an accompanying public incident and attached patches:
purgatory grew by thousands of requests per second, heap memory followed it,
and draining millions of requests caused latency and leadership disruption.

## Backpressure and fairness boundary

The Kubernetes API server's Priority and Fairness documentation is sufficient
for one closing section. It demonstrates that protecting a real overloaded
service requires more than a fast priority queue:

- concurrency is finite;
- queues are bounded;
- requests are classified into priority levels and flows;
- queue configuration trades memory, burst tolerance, latency, and fairness;
- isolation prevents one noisy or buggy flow from starving unrelated work.

Do not implement shuffle sharding here. Ask instead which local dispatcher
metrics would show that arrival rate exceeds useful capacity and what must
happen at a full boundary: reject, defer upstream, shed, or reserve capacity.

## Guide shape

The published guide should use this order:

1. The 5,000-node starvation report and a prediction prompt
2. Queue order versus eligibility, timing, capacity, and ownership
3. FIFO controller work and duplicate coalescing
4. Priority-queue interface before implementation
5. Binary heap invariant, operations, and Go contract
6. Prometheus top-k as a compact positive application
7. Active and delayed heaps in pinned Kubernetes Go source
8. Backoff and the cost of futile attempts
9. QueueingHint and event-specific eligibility
10. Queue-owned in-flight state and the memory-leak rollout lesson
11. The scheduler-plugins sibling-activation incident
12. Exploration lab and Wheels
13. Kafka's timing wheel as the structure-selection boundary
14. Bounded overload and fairness boundary
15. Interview translation and reflection

The first heap diagram should appear only after the learner has met the
production choice it supports.

## Interview translation

End with an answer shaped like:

> I would separate ordering from readiness. A max-heap can select the most
> important of `a` active items with constant-time root access and logarithmic
> updates, while a second min-heap can expose the earliest of `d` retry
> deadlines. Failed work should not return merely because an event occurred; I
> would retain the rejection reason and requeue only when the event can change
> it, with capped backoff as a safety limit. I would also deduplicate logical
> keys and count activation requests, because callers can still generate
> quadratic work before deduplication. Correctness checks would cover the heap
> invariant, one-state-per-item ownership, no dispatch before readiness, and
> progress for runnable work. Operationally I would watch queue depth, pending
> age, attempts per completion, and retained in-flight state, then define what
> happens when arrivals exceed capacity.

## Implementation layout and remaining plan

```text
cs-prod-bridge/
  content/03-queues-heaps-scheduling/
    _index.md
    case-study.md
    alternative-structures.md
    lab/_index.md
    wheel/
      _index.md
      worksheet.md
      01-lost-assignment-loop/
        _index.md
        candidate.md
        debrief.md
        evidence/
      02-event-that-woke-everything/
        _index.md
        candidate.md
        debrief.md
        evidence/
      03-sibling-stampede/
        _index.md
        candidate.md
        debrief.md
        evidence/
  03-queues-heaps-scheduling/
    README.md
    CASE-STUDY.md
    lab/
      README.md
      topk.go
      topk_test.go
      topk_bench_test.go
      dispatcher.go
      dispatcher_test.go
      dispatcher_bench_test.go
    wheel/
      README.md
      01-lost-assignment-loop/
        REPORT.md
        CANDIDATE.md
        DEBRIEF.md
        runner.go
        runner_test.go
        scale_test.go
        evidence/
      02-event-that-woke-everything/
        REPORT.md
        CANDIDATE.md
        DEBRIEF.md
        dispatcher.go
        dispatcher_test.go
        scale_test.go
        evidence/
      03-sibling-stampede/
        REPORT.md
        CANDIDATE.md
        DEBRIEF.md
        group.go
        group_test.go
        scale_test.go
        evidence/
```

The bounded top-k lab and all three Wheel paths now exist. The dispatcher lab,
case study, and separate alternative-structures page remain planned. Reuse the
existing report-first Wheel flow, hidden debrief/evidence navigation, and
opt-in symptom tags. Do not add shared queue abstractions to Units 01 or 02.

## Implementation checklist

The eventual implementation should include:

- A visible concept-to-production mapping for every core topic
- A clear boundary between the priority-queue interface and binary-heap
  implementation
- Tests of the heap invariant after push, pop, remove, and priority change
- A bounded top-k comparison tied to pinned Prometheus source
- Separate active and delayed comparators
- Explicit blocked and in-flight ownership rather than sentinel priorities
- A fake-time or pure-time simulation with no sleep-based correctness checks
- Operation counts for futile attempts and activation amplification
- A one-state-per-logical-item invariant
- The #81214, QueueingHint, #120622, and #682 narratives with source roles and
  provenance limits
- A bounded upstream decision or paraphrasable patch path for each Wheel
- A Kafka boundary that explains why the alternative won without implementing
  it
- A bounded-overload discussion that does not claim a heap supplies fairness
- Ordinary nested-module tests unaffected by opt-in Wheel symptoms
- Generated Unit 03 site output comes from Hugo Markdown and is never
  hand-authored

## Research conclusions to carry forward

- **Teach operations through products, not products through a topic list.**
  Prometheus makes bounded top-k concrete; Kubernetes makes active and delayed
  heaps concrete; Kafka makes removal-heavy timer selection concrete.
- **The scheduler queue is the right spine, but Kubernetes is not the subject.**
  Read only the comparator, heap wrapper, queue states, and relevant patches.
- **A correct heap can participate in an incorrect production policy.**
  Starvation and retry storms live in transitions around the heap.
- **Backoff and QueueingHint solve different problems.** Backoff limits retry
  frequency; the hint estimates whether changed state makes a retry useful.
- **Deduplication does not cancel upstream work amplification.** Count calls,
  transitions, logs, and lock acquisitions, not only final queue length.
- **Optimization state has a space cost.** Remembering in-flight events avoids
  missed eligibility changes but creates ownership and cleanup obligations.
- **Heaps are not universal deadline structures.** Kafka's workload made
  prompt arbitrary removal more important than a per-request exact ordering.
- **Fairness is a policy outcome.** It needs a stated progress rule and capacity
  model; it is not implied by FIFO, priority, or logarithmic operations.
