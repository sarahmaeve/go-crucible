+++
title = 'Queues, heaps, and scheduling'
description = 'Priority queues make the next choice cheap; production schedulers must also model readiness, retries, fairness, and state ownership.'
weight = 3
+++

**Unit 03 · Foundations**

# Queues, heaps, and scheduling

{{< lead >}}A heap can efficiently choose the next item according to one
priority rule. It cannot tell you whether that item is ready, whether a failed
attempt should be retried, whether a new event changes anything, or whether
strict priority is starving other work. Those are scheduling policies built
around the data structure.{{< /lead >}}

{{< callout kind="production" title="Incoming reports" >}}
- [Kubernetes issue #81214](https://github.com/kubernetes/kubernetes/issues/81214)
  reports a cluster with 5,000 nodes and more than 100,000 Pods where
  lower-priority Pods wait while unschedulable higher-priority Pods keep
  returning to the active queue.
- [Scheduler-plugins issue #682](https://github.com/kubernetes-sigs/scheduler-plugins/issues/682)
  reports millions of log lines, much higher CPU, and lost progress on
  unrelated runnable Pods in a production machine-learning workload.
- [Kafka's delayed-request account](https://www.confluent.io/blog/apache-kafka-purgatory-hierarchical-timing-wheels/)
  and [KAFKA-2147](https://issues.apache.org/jira/browse/KAFKA-2147) connect
  retained delayed requests to memory growth and expensive cleanup.
{{< /callout >}}

Each report involves queued work, but none is explained by the running time of
`heap.Pop` alone. The first is about priority, backoff, and broad
reactivation. The second is about repeated group-wide work. The third is about
the cost of removing completed timers, not merely finding the next timer.

By the end of this unit, you should be able to:

- distinguish FIFO queues, priority queues, delayed queues, and schedulers;
- state the ordering rule and invariant of a binary heap;
- derive the cost of heap construction, insertion, removal, and key changes;
- implement a heap correctly with Go's `container/heap` contract;
- derive the cost of bounded-heap top-k selection and explain when sorting or
  another batch-selection method is a better fit;
- model readiness, identity, retries, and in-flight ownership separately from
  priority;
- explain how broad wakeups and group activation amplify work;
- identify starvation, head-of-line blocking, and overload even when every
  individual queue operation is fast; and
- test queue behavior with invariants and state transitions before timing it.

## A production queue answers more than “what is next?”

A queue answers “which stored item is next?” A production scheduler must also
answer when an item is eligible, whether duplicates represent one logical
item, what happens under overload, and who owns the item.

| Question | How the system answers it | Failure when the answer is missing |
|---|---|---|
| Which ready item is next? | Arrival order or a stated priority rule | Important work waits, or ties behave unpredictably |
| When may this item run? | A ready time, retry delay, or yes-or-no readiness check | Work that cannot succeed is retried immediately |
| Is this item already represented? | A stable ID and a lookup of queued or running IDs | Duplicate events multiply queued work |
| What happens when the system is full? | A size limit and a decision to reject, wait, or replace | Memory grows without limit |
| Who owns an item right now? | An explicit queued, delayed, running, or completed state | Work is lost, duplicated, or retained forever |

These dimensions should not be compressed into one word such as “priority.”
An item may have the highest business priority and still be ineligible until a
dependency changes. A duplicate may refer to work already in flight rather
than an item currently stored in the heap.

We will use these workload variables:

- `n`: items currently stored in one queue or heap
- `a`: new logical work items that arrive during an interval
- `d`: items waiting for their ready time in a delayed queue
- `u`: items currently classified as unschedulable
- `r`: candidates examined by one top-k selection
- `k`: winners that selection must retain
- `h`: later candidates that enter the top `k` and replace the current cutoff,
  where `0 <= h <= r-k`
- `e`: external events that can trigger reconsideration
- `g`: members of one related work group
- `f`: consecutive failures for one item
- `c`: useful work items completed during that interval

| Operation | Modeled time | Condition behind the claim |
|---|---:|---|
| Enqueue or dequeue in a suitable FIFO | amortized `O(1)` | The implementation does not shift a slice on every dequeue |
| Inspect a heap root | `O(1)` | The heap is non-empty and its invariant holds |
| Insert or remove a heap root | `O(log n)` | At most one path through the tree needs swaps |
| Restore order after a priority change | `O(log n)` | The item's current heap index is known |
| Build a heap bottom-up | `O(n)` | Heap order is restored from the leaves upward |
| Fully sort `r` candidates | `O(r log r)` | Comparisons are constant-cost |
| Select the best `k` of `r` candidates with a bounded heap | `O(r + h log k)` time, `O(k)` working storage | `1 <= k <= r`; the first `k` entries are built into a heap bottom-up |
| Sort the `k` retained winners | `O(k log k)` | The result contract requires the winners in order, not merely the winning set |
| Wake every unschedulable item for every event | up to `O(eu)` | No check identifies which items the event might help |
| Every group member activates every other member in one round | `Θ(g²)` attempts | Each directed pair can generate work |

The logarithmic heap bounds matter, but the last two rows often dominate a
production incident. Count how many operations the surrounding policy creates,
not only how fast the container performs one operation.

As in Unit 01, these formulas count modeled operations before predicting wall
time. They treat one call to the ordering function as constant-size work. In
the lab, comparing scores has a fixed cost, but a tied score leads to a Go
string comparison of service names. That string comparison can examine bytes
up to the first difference or the end of one name. The formulas therefore
assume service names have a bounded length. If comparison keys can grow, their
comparison cost must be included as another factor.

## FIFO preserves arrival order

A first-in, first-out queue returns ready items in the order they entered:

~~~text
enqueue A, B, C
dequeue A, then B, then C
~~~

With a linked queue, circular buffer, or slice plus a head index, enqueue and
dequeue can take constant time on average. Repeatedly deleting index zero from
a Go slice by shifting every remaining element is linear per dequeue and can
make draining `n` items quadratic.

Among ready items, FIFO prevents later arrivals from jumping ahead. It does
not by itself provide:

- duplicate suppression;
- a bound on memory;
- retry delay;
- cancellation;
- synchronization between producers and consumers; or
- protection from head-of-line blocking when the first item cannot proceed.

The versioned
[Kubernetes client-go work queue](https://github.com/kubernetes/client-go/blob/v0.32.0/util/workqueue/queue.go)
is a useful production bridge. Alongside its FIFO storage, it keeps a
`dirty` set for work that needs processing and a
`processing` set for work currently owned by a worker. Adding the
same key repeatedly does not create one stored entry per call. If the key is
added while it is being processed, `Done` can enqueue it again so the
new information is not lost.

{{< callout kind="warning" title="What queue deduplication cannot prevent" >}}
Keeping at most one stored entry for each key prevents repeated submissions of
that key from inflating the queue. It does not place an overall bound on the
queue: distinct keys can still accumulate without limit. Deduplication also
does not undo the CPU, allocations, locks, logs, or API calls spent submitting
the same key again. Identify where repeated work is created as well as where
the queue coalesces repeated submissions.
{{< /callout >}}

## A priority queue exposes the most important item

A priority queue supports a smaller interface than a sorted collection:

~~~text
insert(item)
peek most important
remove most important
~~~

It need not make the second, third, or hundredth item directly accessible in
sorted order. That narrower promise permits a useful compromise:

| Representation | Insert | Inspect best | Remove best |
|---|---:|---:|---:|
| Unsorted slice | amortized `O(1)` | `O(n)` | `O(n)` |
| Sorted slice | `O(n)` movement | `O(1)` | `O(1)` from the end |
| Binary heap | `O(log n)` | `O(1)` | `O(log n)` |

[MIT 6.006 Lecture 8: Binary Heaps](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/40d4851e550507ca14dc778b9b2266cc_MIT6_006S20_lec8.pdf)
develops this interface, derives the array representation, proves the heap
property, and analyzes insertion, deletion, and linear-time construction. Its
router, operating-system scheduler, and discrete-event simulation examples
are good supplementary applications. Here we will connect the same model to
public product reports and patches.

## A binary heap is an implicit complete tree

A binary heap stores a complete binary tree densely in an array. For a
zero-based index `i`:

~~~text
parent(i) = floor((i - 1) / 2), for i > 0
left(i)   = 2*i + 1
right(i)  = 2*i + 2
~~~

There are no child pointers. A complete tree fills every level except perhaps
the last, which fills from left to right. Its height is logarithmic in the
number of items, so moving one item between a leaf and the root takes
`O(log n)` steps.

A min-heap obeys this local invariant:

~~~text
key(parent(i)) <= key(i)
~~~

A max-heap reverses the comparison. In either case, transitivity makes the root
the best item according to the comparison.

For example, this is a valid min-heap:

~~~text
index:  0  1  2  3  4  5  6
value: [2, 5, 4, 9, 7, 8, 6]
~~~

The array is not sorted: `5` appears before `4`, and
`9` before `7`. The heap promises that every parent is no
greater than its children, not that siblings or subtrees are ordered relative
to one another.

{{< callout kind="contract" title="Decide what happens when priorities tie" >}}
“Highest priority first” is incomplete when equal priorities are common. A
scheduler might compare `(priority DESC, sequence ASC, ID ASC)`:
business priority first, older enqueue sequence next, and immutable identity as
a deterministic final tie-breaker. The same comparison must govern every heap
operation.
{{< /callout >}}

## Push restores heap order upward; pop restores it downward

Before either operation, every parent-child pair is in the required order.
Adding or removing one item can break that order, but only along one path
through the tree. The rest of the heap remains valid.

To insert an item:

1. append it at the next leaf position;
2. compare it with its parent;
3. if it should come before its parent, swap them;
4. repeat with the next parent until the order is correct.

The new item starts at the bottom, so it can only move upward. At most it
travels from a leaf to the root, which takes `O(log n)` steps.

To remove the root:

1. exchange the root and the last item;
2. remove the last item, which is the former root;
3. compare the replacement root with its children;
4. if a child should come first, swap with the better child;
5. repeat until the order is correct.

The replacement starts at the top, so it can only move downward. At most it
travels from the root to a leaf, again taking `O(log n)` steps.

This is why heap implementations often describe these operations as
**restoring** or **repairing** heap order. They do not rebuild or sort the
whole array. One mutation creates one possible break, and the algorithm fixes
the affected path.

Building a heap by inserting `n` items one at a time costs
`O(n log n)` in the worst case. A faster build starts with the full
array and restores heap order at each internal node, working from the last
parent back to the root.

The amount of work depends on each node's height, not on the height of the
whole heap. About half the nodes are leaves and move zero levels. About one
quarter are one level above the leaves and can move at most one level; about
one eighth can move at most two levels; and so on. An upper bound has the form:

~~~text
(n/4) * 1 + (n/8) * 2 + (n/16) * 3 + ...
~~~

The fractions shrink faster than the possible distance grows, so this series
is bounded by a constant multiple of `n`. Bottom-up heap construction
therefore costs `O(n)`, even though restoring order at the root can take
`O(log n)` steps.

If an existing item's priority changes, it may need to move in either
direction. A heap implementation can restore the order in
`O(log n)` if it knows the item's current index.

{{< callout kind="warning" title="Changing a priority requires another heap operation" >}}
Changing a stored item's priority does not automatically move it. Until heap
order is restored, the next removal may return the wrong item. Keep each
item's index current during every swap, then call `heap.Fix` or the
equivalent operation supplied by the implementation.
{{< /callout >}}

## Go separates heap mechanics from your ordering rule

Go's [`container/heap` documentation](https://pkg.go.dev/container/heap)
defines the standard-library contract. A type supplies
`Len`, `Less`, and `Swap` from
`sort.Interface`, plus methods that append and remove the final
element. The package supplies the upward and downward moves that maintain the
heap.

This max-priority heap makes older sequence numbers win ties:

~~~go
package work

import "container/heap"

type Item struct {
	ID       string
	Priority int
	Sequence uint64
	index    int
}

type ItemHeap []*Item

func (h ItemHeap) Len() int { return len(h) }

func (h ItemHeap) Less(i, j int) bool {
	if h[i].Priority != h[j].Priority {
		return h[i].Priority > h[j].Priority
	}
	if h[i].Sequence != h[j].Sequence {
		return h[i].Sequence < h[j].Sequence
	}
	return h[i].ID < h[j].ID
}

func (h ItemHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *ItemHeap) Push(x any) {
	item := x.(*Item)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *ItemHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[:n-1]
	return item
}

func (h *ItemHeap) ChangePriority(item *Item, priority int) {
	item.Priority = priority
	heap.Fix(h, item.index)
}
~~~

`ChangePriority` assumes that `item` is currently stored in this heap and that
its `index` still identifies its position. After `Pop` sets the index to `-1`,
that call would violate `heap.Fix`'s precondition. A production API should use
its identity lookup to establish that the item is present rather than accept an
arbitrary `*Item` without checking it.

Callers use the package functions, not the methods directly:

~~~go
ready := ItemHeap{
	{ID: "rebuild-search", Priority: 20, Sequence: 2},
	{ID: "rotate-key", Priority: 50, Sequence: 1},
}
for i, item := range ready {
	item.index = i
}
heap.Init(&ready)

heap.Push(&ready, &Item{
	ID:       "renew-certificate",
	Priority: 100,
	Sequence: 3,
})

next := heap.Pop(&ready).(*Item)
~~~

The names can be surprising. `heap.Pop` swaps the root with the item at index
`Len()-1`, restores order in the remaining heap, and then calls the
implementation's `Pop` method to shorten the slice and return that final item.
Calling `ready.Pop()` directly would merely remove the current final element
without restoring order or consulting the heap.

The documented complexity is:

- `heap.Init`: `O(n)`;
- `heap.Push` and `heap.Pop`:
  `O(log n)`;
- `heap.Remove` and `heap.Fix`:
  `O(log n)`.

{{< callout kind="contract" title="What container/heap does not provide" >}}
The package maintains one ordering invariant. It does not provide concurrent
access, stable identity, duplicate suppression, bounded capacity, readiness,
backoff, cancellation, persistence, fairness, or metrics. A production queue
must define those policies and protect the heap with the appropriate ownership
or synchronization.
{{< /callout >}}

## Prometheus `topk` needs the winners, not a ranking of everything

Suppose a query examines `r` samples but returns only the `k`
highest values, where `1 <= k <= r`. When `k < r`, an exact
answer must inspect all `r` candidates: an unexamined candidate could be
larger than every candidate seen so far and change the winning set. Selection
therefore has a lower bound of `Ω(r)` for that unsorted input. When
`k = r`, membership is trivial because every candidate wins, but an
ordered result still has to read and order them all. The choice of algorithm
determines what additional work and storage the inspection requires.

Sorting all `r` candidates takes `O(r log r)` comparisons and
determines the exact rank of every candidate. The query does not need that
information about the `r-k` candidates it will discard.

### Put the weakest current winner at the root

To find the largest `k` values with a bounded heap, order the heap so
its root is the **worst candidate currently included**. When score alone is
the key, this is a min-heap: the smallest retained score is at the root.
The root is useful because it is the cutoff a new candidate must cross:

1. copy the first `k` candidates and build a heap from them bottom-up;
2. compare each later candidate with the root;
3. discard the candidate if it is no better than that root;
4. otherwise replace the root and restore heap order along one path.

A max-heap of the same `k` winners would expose the best winner. That
is the wrong end for this decision: knowing the best retained value does not
tell us whether a new candidate should replace the weakest one. For the
smallest `k` values, reverse the design and use a max-heap so the largest
retained value becomes the cutoff.

The heap comparison must include the whole result rule, including ties. In the
lab, a lower score is worse; for equal scores, the alphabetically later service
name is worse. The root is therefore the worst retained candidate under that
complete rule, not merely a candidate with the smallest numeric score.

### Derive the bound from the operations

Let `h` be the number of later candidates that cross the cutoff and
replace the root. It can range from zero to `r-k`. The selection work
has three parts:

- building one heap from the first `k` candidates costs `O(k)`;
- comparing each remaining candidate with the cutoff costs
  `Θ(r-k)`; and
- restoring heap order after `h` root replacements costs
  `O(h log k)`.

Because `k <= r`, the first two parts are linear in the input size.
The selection phase is therefore:

~~~text
O(r + h log k) working time
O(k)             working storage
~~~

In the worst case, every later candidate enters the result, so
`h = r-k` and the familiar upper bound is `O(r log k)`.
That shorter bound is correct for `k >= 2`, but it hides useful
information. Every later candidate receives a cutoff check, while only
candidates that enter the result cause a logarithmic heap update. If no later
candidate enters, selection is still `Θ(r)` because the algorithm must reject
them one by one.

If the product contract requires the winners in ranked order, the heap has not
finished the operation. A heap identifies the winning set but does not store
that set in full order. Sorting the `k` survivors adds
`O(k log k)`, for a complete bound of:

~~~text
O(r + h log k + k log k)
~~~

These boundary cases make the formula concrete:

- For fixed `k`, both `log k` and the final winner sort are bounded
  constants, so the work grows as `Θ(r)`.
- For `k = 1`, the single retained candidate needs no heap movement;
  the algorithm is a linear maximum scan.
- When `k` is close to `r`, the final `k log k` sort approaches
  the cost of sorting the input, so the bounded heap offers little time
  advantage.
- When `k = r`, nothing is discarded. Building a heap and then sorting
  all winners adds work compared with sorting directly.
- The lab accepts `k = 0`, but its contract still checks every score for
  `NaN` before returning an empty result. That implementation is
  `Θ(r)` for this boundary. An API that promises immediate return for
  `k = 0` could make a different choice.

### Compare complete methods, not slogans

The bounded heap is one option, not a general replacement for every selection
algorithm. The following table assumes an exact, ordered top-k result and the
lab's promise not to change the caller's input. Its storage column excludes the
input and returned output but includes a required working copy.

| Method | Time to return an ordered top `k` | Working candidate storage | Useful distinction |
|---|---:|---:|---|
| Sort all candidates | `O(r log r)` | `O(r)` | Direct and often fast; computes ranks the result does not use |
| Build a max-heap of all candidates, then remove `k` roots | `O(r + k log r)` | `O(r)` | Bottom-up construction is linear and removals arrive in result order, but every candidate remains stored |
| Keep a min-heap of `k` current winners | `O(r + h log k + k log k)` | `O(k)` | Can select in one pass with bounded state; its worst-case heap-update cost depends on how often the cutoff changes |
| Partition with quickselect, then sort the winners | expected `O(r + k log k)` | `O(r)` here; a mutable input can be partitioned in place | Suits an in-memory batch; ordinary pivot choices can have `O(r^2)` worst cases and do not maintain an answer as a stream arrives |
| Keep a sorted slice of `k` winners | `O(rk)` worst case | `O(k)` | Linear insertion is simple and can be competitive for very small `k`, but scales poorly as `k` grows |

The full max-heap and quickselect rows show why “top-k improves sorting from
`O(r log r)` to `O(r log k)`” is incomplete. That statement
describes one useful comparison with full sorting. It does not prove that a
bounded heap has the best time bound for every in-memory batch. Its strongest
advantages are one-pass processing and working storage that depends on
`k`, especially when `k` is much smaller than `r`.

The lab receives an already allocated `[]Candidate`, so it demonstrates
`O(k)` **additional working storage**, not an end-to-end `O(k)`
memory process. The same bounded-heap operation can consume candidates from an
iterator, channel, file, or network stream without first retaining all
`r` candidates, provided the surrounding API also supports streaming.

If the request is for the `k` most frequent values in raw events, the
heap does not count those events. First build frequencies—often with the hash
table model from Unit 01—then run top-k selection over the distinct values and
their counts. Counting and selection are separate costs and may use different
input sizes.

The pinned
[Prometheus v3.13.1 `topk` implementation](https://github.com/prometheus/prometheus/blob/v3.13.1/promql/engine.go#L3951-L4085)
uses this pattern in its query engine. For each aggregation group, it keeps at
most `k` samples in a min-heap. When a better candidate replaces the root and
`k > 1`, it calls `heap.Fix`; a one-element heap needs no rearrangement.
Before returning an instant-query result, Prometheus sorts the retained samples
in descending order.

{{< callout kind="production" title="What the heap saves" >}}
A `topk` query still examines all `r` samples, but it does not need to
rank the `r-k` samples that will be discarded. The bounded heap stores the
current `k` winners and exposes the weakest one as the cutoff for accepting a
new candidate. When `k` is much smaller than `r`, it replaces a full
sort with linear cutoff checks and logarithmic work only when a new candidate
enters. In the lab's
non-mutating API, it also reduces working candidate storage from `O(r)` to
`O(k)`. If the response must be ordered, the retained `k` still need a
final sort. These are the exact savings; the heap does not make input inspection
disappear or guarantee lower elapsed time for every `r` and `k`.
{{< /callout >}}

The array representation and the operations that restore heap order after a
change are the same in selection and scheduling, but the root does not have one
universal meaning. The bounded top-k heap puts the **worst retained winner** at
the root so a new candidate can be rejected cheaply. A ready-work heap usually
puts the **next item to serve** at the root so dispatch is cheap. The comparison
supplied to the heap determines which meaning applies.

The next sections leave batch selection and return to scheduling. They add one
missing responsibility at a time—future deadlines, reasons to retry, identity,
cleanup, sharing, and capacity—and use production reports to show what goes
wrong when that responsibility is missing.

## Scheduling needs at least two orders

Scheduling needs two independent orders. Active priority chooses among ready
items; time order chooses which delayed item becomes eligible next:

- **active order:** among ready items, which should be attempted next?
- **time order:** among delayed items, which becomes ready next?

Those are different comparisons. A common design uses a max-heap for active
business priority and a min-heap for `readyAt`:

~~~text
                    success ──────────────> completed
                   /
active ──> attempt
                   \
                    retryable failure ────> delayed
                                             |
                                             | readyAt <= now
                                             v
                                           active

non-time dependency failure ───────────────> blocked
                                             |
                                             | relevant state change
                                             v
                                           active
~~~

Putting future and ready items in one heap with one fixed ordering rule does
not answer both questions correctly. If the heap orders first by business
priority, a high-priority item whose deadline is still in the future can sit at
the root and hide ready work below it. If it orders first by `readyAt`, it can
expose the next deadline, but it no longer selects the highest-priority item
among everything already ready. A comparison based on `readyAt <= now` would
itself change as time passes, without a heap operation to restore the new
order. Separate heaps—or equivalent separate indexes—let the scheduler answer
the two questions independently. Move an item from the time-ordered structure
to the active priority heap when its deadline arrives.

The pinned
[Kubernetes v1.32 scheduling queue](https://github.com/kubernetes/kubernetes/blob/v1.32.0/pkg/scheduler/backend/queue/scheduling_queue.go)
is a concrete system to inspect. Its queueing design separates an active
priority queue, a backoff queue, and Pods that are currently unschedulable.
The scheduler's versioned
[heap wrapper](https://github.com/kubernetes/kubernetes/blob/v1.32.0/pkg/scheduler/backend/heap/heap.go)
combines heap ordering with a key-to-item map so an existing item can be found,
updated, or removed without scanning the whole heap.

That map illustrates a recurring production extension:

~~~text
heap array       -> choose by priority and restore heap order
identity index   -> find, update, deduplicate, or remove by key
state machine    -> decide which structure currently owns the item
~~~

The index is useful only if every swap, insert, removal, and transition keeps
it consistent with the heap.

## Backoff controls when a retry may happen

After a retryable failure, immediate requeue can create a tight loop. A common
exponential policy is:

~~~text
delay(f) = min(maxDelay, initialDelay * 2^(f-1))
~~~

where `f >= 1` is the number of consecutive failures. The cap prevents
the delay from growing forever. Code should check whether another doubling
would reach the cap before doing the multiplication; otherwise a large duration
can overflow before `min` sees it. If the policy adds a small random
variation, called **jitter**, state whether the final result is clamped again.
Jitter reduces the chance that many workers with the same retry count wake at
the same instant; it does not guarantee that their wakeups will differ.

The pinned
[client-go v0.32 delaying queue](https://github.com/kubernetes/client-go/blob/v0.32.0/util/workqueue/delaying_queue.go)
keeps delayed entries ordered by their target time. When an existing entry
receives an earlier deadline, the queue changes that entry's `readyAt` value—the
field used by its ordering rule—and calls `heap.Fix` at the entry's stored
index. The stable item key does not change. This is the changed-priority
operation from the abstract heap, now applied to a retry deadline.

Use an injected or fake clock when testing this policy. Wall-clock sleeps make
boundary tests slow and nondeterministic; a controlled clock can advance
directly to just before, at, and just after a deadline.

{{< callout kind="contract" title="Backoff answers when, not whether" >}}
Backoff spaces attempts. It does not prove that another attempt can succeed.
If no relevant dependency changed, a slower futile retry is still futile.
Eligibility and backoff are separate decisions.
{{< /callout >}}

## Production report: Kubernetes kept retrying high-priority Pods that could not run

[Kubernetes issue #81214](https://github.com/kubernetes/kubernetes/issues/81214)
is a user report from a cluster with 5,000 nodes and more than 100,000 Pods. The
reporter observed lower-priority Pods waiting while unschedulable higher-priority
Pods were repeatedly moved from backoff into the active queue. The report also
calls out PVC and Service events that moved all unschedulable Pods back to active
consideration.

Events unrelated to a particular Pod could still return it to the active heap.
The heap then selected high-priority Pods that still could not run ahead of
feasible lower-priority Pods, just as its ordering rule required. The
components can each behave as specified:

1. an event makes many high-priority Pods active;
2. the priority heap correctly returns them first;
3. their unchanged constraints make them fail again;
4. backoff delays them, but later events reactivate them;
5. lower-priority feasible work repeatedly loses the comparison.

The report therefore connects three distinct concepts:

- strict priority can starve lower-priority work;
- a retry delay can be too short for the workload; and
- events that wake every blocked item can defeat the protection of backoff.

The merged
[Kubernetes PR #81263](https://github.com/kubernetes/kubernetes/pull/81263)
introduced scheduler options and configuration for initial and maximum Pod
backoff durations. That patch made an important policy tunable. It did not
make every external event relevant to every blocked Pod, so backoff tuning and
retry eligibility remain different layers of the design.

{{< callout kind="production" title="Read the report before the patch" >}}
Start with issue #81214 and write down the proposed causal chain. Then inspect
PR #81263 and name exactly which edge of that chain it changes. A merged patch
is strong evidence about the implementation response, but it need not solve
every mechanism described by the original report.
{{< /callout >}}

## Eligibility asks whether anything relevant changed

An unschedulable item is waiting on a predicate—a yes-or-no condition—such as
sufficient resources, a compatible node, a dependency, or a quota. An event
should ideally reactivate only the items for which it might change that
predicate.

Filtering and indexing affect different parts of the work. For one event, let
`s` be the number of blocked items that the event selects for another attempt:

~~~text
broad requeue:      up to u candidate checks
                    + up to u activation requests

filtered requeue:   cost of finding or testing candidates
                    + s activation requests, where 0 <= s <= u
~~~

Across `e` events, a broad requeue can therefore perform up to `eu`
candidate checks and `eu` activation requests. A relevance filter reduces the
number of requests only when it selects fewer than `u` items. Without an index,
that filter may still test all `u` blocked items to discover the `s` it selects.
An index may reduce that search cost by identifying likely candidates directly,
but filtering and indexing are separate improvements.

Kubernetes's official
[QueueingHint design account](https://kubernetes.io/blog/2024/12/12/scheduler-queueinghint/)
describes the change. Previously, a scheduler plugin named broad categories of
cluster events that might solve one of its failures. Any event in a named
category could cause another attempt, even when that particular event could
not help that particular Pod. With QueueingHint, the plugin can examine the
specific event and say whether it might make the Pod schedulable.

That check can also be wrong. If it says an important event is irrelevant, a
Pod that could now run may remain blocked. Periodically retrying all blocked
Pods can catch those mistakes, but the retry period creates a trade-off:

- more frequent scans reduce the time a missed item remains stuck;
- less frequent scans reduce background amplification;
- better event indexes and predicates reduce reliance on the scan.

{{< callout kind="production" title="Correct heap order cannot prevent useless retries" >}}
If the active heap selects the first item according to its comparison rule, it
is working correctly. The problem may be that the queue has been filled with
items that still cannot run. In that case, change which events return items to
the queue; changing the heap will not help.
{{< /callout >}}

## Deduplicating the queue does not eliminate duplicate work

Consider one pass over a group of `g` related items.
If every member asks to activate the other `g-1` members, that pass
generates exactly:

~~~text
g * (g - 1) = Θ(g²)
~~~

The queue may still store each of the `g` keys only once. But before
the queue declines to add another stored entry, callers have already made
`g(g-1)` activation requests. Each request can acquire a lock, check
a map, and write a log message. Queue depth can therefore look safe while CPU
use and log volume grow rapidly.

[Scheduler-plugins issue #682](https://github.com/kubernetes-sigs/scheduler-plugins/issues/682)
is a public report from a production machine-learning workload using the
Kubernetes coscheduling plugin. When quota and resource restrictions left a
Pod group pending, the reporter saw millions of log lines, a large CPU
increase, and loss of scheduling progress for unrelated Pods whose resources
were available.

The author of the merged patch identified repeated attempts by group members to
activate their siblings as unnecessary work. The
[scheduler-plugins PR #700](https://github.com/kubernetes-sigs/scheduler-plugins/pull/700)
stores an `Activate` flag in state that Kubernetes carries for one Pod's
scheduling attempt. The coscheduling plugin reaches its `Permit` step after
earlier group and resource checks have passed. At this step, it either lets the
Pod continue or makes it wait for enough group members. The patch sets
`Activate` only when the number of assigned group members is zero.
`ActivateSiblings` returns without doing group-wide work unless that flag is
present.

For one scheduling attempt that reaches that path and performs sibling
activation once, the plugin generates at most `g-1` sibling requests:

~~~text
one scheduling attempt + at most g - 1 sibling activation requests = O(g)
~~~

That is a bound on this operation during one scheduling attempt, not on the
whole workload. Total work still depends on how many attempts reach the
path and perform the activation.

{{< callout kind="production" title="Measure requests as well as queue entries" >}}
Count activation requests, distinct items activated, heap insertions, and
useful completions separately. Comparing those counts shows how much repeated
work occurs before the queue coalesces requests for duplicate keys. Queue
length alone cannot show it.
{{< /callout >}}

## Remembered events must be cleaned up

To decide whether a retry might now succeed, a scheduler sometimes needs to
remember cluster events that happened while an item was being evaluated. If
the attempt fails, those events help decide where the item should go next.
The event records occupy memory and must eventually be removed.

A useful set of state changes is:

~~~text
queued -> in-flight -> queued again
                    -> completed
                    -> rejected
                    -> cancelled
~~~

Every way out of the in-flight state must remove the records owned by that
attempt. Cleaning up after an ordinary success is not enough. Skipped items,
cancellation, errors, duplicate entries, and shutdown also need defined
cleanup.

[Kubernetes issue #120622](https://github.com/kubernetes/kubernetes/issues/120622)
describes the memory cost of accumulating cluster events that occur while Pods
are being scheduled. The issue also links several follow-up changes rather
than claiming one universal correction. One of them,
[PR #126962](https://github.com/kubernetes/kubernetes/pull/126962), fixes a
specific path where a skipped Pod did not pass through the expected requeue or
bind cleanup and could leave in-flight state behind.

{{< callout kind="warning" title="New bookkeeping needs its own cleanup" >}}
Avoiding useless retries can require indexes, rejection reasons, and saved
event records. For each one, define which part of the program owns it, when it
is deleted, and which metric reports how many entries remain. If those records
keep growing, garbage collection cannot decide that they are obsolete.
{{< /callout >}}

For a fixed batch of submissions, while the model retains one terminal record
for every submitted generation, a count-based conservation check is:

~~~text
number of submitted generations =
    active count
  + delayed count
  + blocked count
  + in-flight count
  + completed count
  + rejected count
  + cancelled count
~~~

This equality applies only while the model retains those terminal records. The
same generation must not appear in two states. An item in two states has two
owners; an item in no state has been lost. A generation means one submission
instance: if a logical ID can be reused, pair it with a generation or
submission ID.

## A correct priority queue can still leave work waiting forever

A priority heap can return exactly the item its comparison rule asks for while
lower-priority work never runs. If high-priority work keeps arriving as fast as
workers can complete it, the heap always has another high-priority item at its
root. Lower-priority items remain ready but are never selected. This is
**starvation**.

FIFO prevents later arrivals from jumping ahead of earlier ready work, but it
creates a different risk. If the item at the front cannot proceed and the
system insists on handling it first, work behind it also waits. This is
**head-of-line blocking**.

“Fair” does not describe one universal behavior. State the measurable service
promise the system actually needs:

- once an item is ready, later arrivals cannot postpone it beyond a stated
  number of dispatches;
- each continuously backlogged tenant receives a turn within a stated number
  of dispatches or amount of time; or
- continuously backlogged tenants receive stated shares of the available
  worker time over a stated interval.

Different policies implement different guarantees. A round-robin scheduler
visits each non-empty tenant queue in turn. A weighted version gives some
queues more turns than others. An aging policy gradually raises the priority
of work that has waited a long time. None of these behaviors comes from the
heap itself.

Each promise has conditions. A dispatch-count bound assumes that the service
continues selecting work and that the set of queues covered by the promise is
finite. A wall-clock bound also needs a limit on how long dispatched work can
occupy capacity. A capacity-share promise must name both the measurement
interval and what happens when some tenants do not use their shares.

The Kubernetes API server's
[Priority and Fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/)
feature is one production example. It assigns requests to priority levels,
limits how many requests in each level may execute at once, and separates
request streams into queues so one busy source is less able to crowd out the
others. Its complete algorithm is outside this unit. The relevant lesson is
that a priority label alone does not decide how capacity is shared.

No queue policy can guarantee another turn or a maximum wait after workers stop
finishing work and releasing capacity.

## A fast queue cannot create service capacity

Track logical backlog and scheduling attempts separately:

~~~text
change in logical backlog = new submissions - terminal exits
attempt demand             = first attempts + retry attempts
~~~

For a workload in which completion is the only terminal exit, let `a` be new
logical arrivals and `c` useful completions during the same interval. If
`a > c` for long enough, the number of live identities grows regardless of
whether queue operations take constant or logarithmic time. If rejection or
cancellation also removes live work, include those outcomes on the exit side.

A retry does not create a new logical identity, so it is not another arrival
in the backlog equation. It does create another scheduling attempt and consume
worker capacity. Retry traffic can therefore saturate the service even while
the number of distinct queued identities remains stable.

At a capacity boundary, choose explicitly among:

- reject new work and return a retryable signal;
- block or slow producers;
- coalesce superseded updates by stable identity;
- shed a documented low-value class;
- persist overflow elsewhere; or
- allow bounded growth for a measured burst.

Under sustained overload, an unbounded in-memory queue permits memory use and
wait time to grow until the process or host reaches a limit. A bounded queue
makes overload visible, but it must also define which item is rejected and how
the caller learns about it.

Operational signals should describe policy as well as container size:

- active, delayed, blocked, and in-flight counts;
- oldest and percentile wait time by priority or tenant;
- arrivals, attempts, useful completions, and permanent failures;
- attempts per completion and futile-attempt rate;
- activation requests versus distinct identities activated;
- rejection, coalescing, and cancellation counts;
- heap comparisons, swaps, pushes, pops, and fixes; and
- retained event records or other queue-owned metadata.

These measures distinguish “the heap is expensive” from “the system creates
too many heap operations” and from “there is not enough completion capacity.”

## Kafka: finding the next timeout was not the only job

A heap-backed timer queue makes one question cheap: which request times out
next? Kafka also had to handle a second, very common event: a request completing
before its timeout.

For example, a produce request may wait for replicas to acknowledge a write,
and a fetch request may wait for new data. Either request can finish in two
ways:

1. the condition it is waiting for becomes true; or
2. its timeout expires.

Kafka calls the requests still waiting for one of those outcomes **request
purgatory**. The older design described in
[Apache Kafka, Purgatory, and Hierarchical Timing Wheels](https://www.confluent.io/blog/apache-kafka-purgatory-hierarchical-timing-wheels/)
stored each request in two kinds of structures: a Java `DelayQueue`
ordered by timeout and lists used to find requests when relevant broker state
changed.

The `DelayQueue` could efficiently expose the next timeout. The
problem appeared when a request completed because its condition became true.
The old design did not immediately remove that request from the timer queue or
the other lists. A cleanup thread periodically scanned those structures and
removed completed requests.

That left an unpleasant choice:

- scan infrequently, and completed requests continue occupying memory; or
- scan frequently, and spend substantial CPU repeatedly searching the lists.

The public incident
[KAFKA-2147](https://issues.apache.org/jira/browse/KAFKA-2147) reports this
failure at production scale. The number of requests retained in purgatory and
the Java process's managed memory grew together. In this report, **JVM heap**
means the memory managed by the Java runtime, not the binary-heap data
structure. Draining a very large backlog then caused latency and leadership
disruption.

### How deadline buckets and direct removal changed the work

A **timing wheel** groups nearby timeout times into buckets, like marks around
a clock. As time advances, the system processes the bucket for the current
time range. Timers farther in the future go into additional wheels whose
buckets cover progressively larger ranges. Those several levels are what make
the structure **hierarchical**.

Kafka's new design also gave each request a direct link to its position in a
bucket. When the request completed before its timeout, Kafka could unlink it
immediately with a constant number of list changes. It no longer had to leave
the completed request in the timer queue for a later scan.

Kafka still used a `DelayQueue`, but it stored time buckets rather
than individual requests there. The number of possible queued buckets had a
fixed upper limit and was usually much smaller than the number of waiting
requests. This reduced both the number of heap operations and the number of
completed request entries retained for cleanup.

The article includes before-and-after benchmarks, but they measure Kafka's
Java implementation under its chosen test workload. They show why the change
was useful to Kafka; they do not prove that a timing wheel is always faster
than a heap.

{{< callout kind="production" title="Why Kafka changed structures" >}}
The old design made the next timeout easy to find, but many requests completed
for another reason before their timeout. Removing those completed requests was
therefore just as important as finding the next deadline. Kafka combined two
mechanisms. The timing wheel grouped requests with nearby deadlines, so the
`DelayQueue` tracked buckets rather than every request. Each request also had a
direct link to its linked-list entry inside a bucket, so completion could
unlink it immediately. Bucketing reduced timer-queue work; direct links removed
completed requests without a full scan.

The general question is not “heap or timing wheel?” First ask how work usually
leaves the system. If items commonly disappear before reaching the front of a
heap, the cost of finding and removing those items may matter more than cheap
access to the root.
{{< /callout >}}

## Work the exercises

The [bounded top-k lab](lab/) keeps the heap exercise deliberately smaller
than a scheduler. It compares sorting all candidates with retaining only the
`k` candidates that can still appear in the response. The tests separate
cutoff checks from candidates that actually replace a winner, and distinguish
bottom-up heap construction from the final sort required by the response. The
benchmarks vary `r`, `k`, and arrival order; the written exercise also
compares the bounded heap with a full heap, quickselect, and a sorted bounded
slice.

Then investigate [The Lost Assignment Loop](wheel/01-lost-assignment-loop/)
from its incoming report. This Wheel is inspired by GitHub's
[August 6–7, 2026 Actions incident](https://www.githubstatus.com/incidents/qcvjkzcs7j74):
after an initial capacity failure, runners repeatedly attempted jobs that were
no longer valid and could not pick up valid work. The local exercise asks a
narrow scheduling question—when can another attempt by the same worker still
succeed?—without claiming that GitHub used its Go code or a heap for runner
assignment.

[The Event That Woke Everything](wheel/02-event-that-woke-everything/) uses a
max-priority heap in its local dispatcher. Inventory events repeatedly return a
high-priority repair that still cannot run, while routine repairs wait. Start
with the report and decide which evidence would distinguish an ordering defect
from a problem elsewhere in the repair's path through the dispatcher.

[The Sibling Stampede](wheel/03-sibling-stampede/) examines work performed before
a queue discovers that it already contains an ID. It uses the
scheduler-plugins #682 report and merged #700 patch as production inspiration,
then counts requests, stored IDs, repeated logs, and progress for unrelated
work in a smaller synthetic model.

Use the [investigation worksheet](wheel/worksheet/) to keep the incident
trigger, the condition that prevents recovery, and the proposed state
transition separate.

## Check behavior before measuring speed

Start tests with observable rules that do not depend on timing noise.

### Check the heap invariant

After every push, pop, removal, and priority change, inspect each parent-child
pair. For every child `i`, assert
`!Less(i, parent(i))`: no child may outrank its parent. Also verify
that each item's stored index points back to that item.

For small randomized inputs, compare removals with a simple reference model
that sorts all current items by the complete comparison rule after each
mutation. The reference can be slow because its purpose is to make correctness
obvious.

### Check state conservation

At every transition, assert that each submitted generation appears in exactly
one of:

~~~text
active | delayed | blocked | in-flight | completed | rejected | cancelled
~~~

Completion and permanent failure must release live queue-owned retry state,
indexes, and in-flight event records. Retained history or metrics should live
separately. Duplicate notification must not create two physical owners for the
same logical item.

### Check readiness and service guarantees

With a fake clock, verify that:

- no delayed item runs before `readyAt`;
- an item becomes eligible at the exact boundary;
- backoff reaches but never exceeds its cap;
- an irrelevant event activates no blocked item;
- a relevant event activates the affected items; and
- if the policy limits how many dispatches may pass before a continuously ready
  item runs, lower-priority work advances within that stated limit.

Tests of a waiting-time or dispatch-count promise need a finite adversarial
scenario and an explicit limit. They should not depend on a goroutine
eventually winning a race on a developer's machine.

### Count work before benchmarking it

Count comparisons, swaps, activation requests, distinct activations, futile
attempts, and completions. Then benchmark across increasing `n`,
`u`, and `g`. Profiles and allocation measurements explain
constant factors, while the counters reveal whether growth is logarithmic,
linear, or quadratic for the modeled operation.

This order matters. A faster implementation of the wrong transition policy
can make a retry storm faster without making the system useful.

## Read each source for one role

This lesson uses several kinds of evidence. Do not flatten them into equal
claims:

| Source | What it establishes |
|---|---|
| [MIT 6.006 Lecture 8](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/40d4851e550507ca14dc778b9b2266cc_MIT6_006S20_lec8.pdf) | The priority-queue interface, heap invariant, operations, proofs, and asymptotic model |
| [Go `container/heap` docs](https://pkg.go.dev/container/heap) | The public Go API contract and documented complexity |
| Pinned product source | What one named release implemented, not what every release must implement |
| Public issue report | What an operator observed and hypothesized in a particular production setting |
| Merged pull request | What maintainers changed; not proof that every reported symptom had one cause |
| Official design account | The intended model, rollout history, trade-offs, and documented boundaries |

When you paraphrase a patch, name the behavior it changes and link the whole
patch. When you generalize from an incident, label the generalization as a
model to test rather than a fact established for every scheduler.

## Translate the model in a design conversation

A compact answer to “how would you design this dispatcher?” might be:

> I would separate readiness from priority. A max-heap can select the most
> important of `n` active items with constant-time root access and
> logarithmic updates, while a min-heap can expose the earliest of
> `d` retry deadlines. Failed work should return only after backoff
> or a state change relevant to its rejection. I would index logical IDs for
> updates and deduplication, but also count activation requests because work
> can amplify before the queue coalesces it. Correctness tests would cover both
> heap invariants, one owner for each submission, readiness, and a testable
> limit on how long ready work may wait or a stated rule for sharing capacity.
> Finally, I would bound the queue and define what happens when new submissions
> exceed terminal exits or retry attempts consume the available worker
> capacity.

That answer gives the operations, structures, complexity, correctness
conditions, and overload policy. It does not claim that a heap supplies the
entire scheduler.

## Reflection

For each answer, point to the model, implementation, or production evidence
that supports it:

1. Why is a valid heap not a sorted array?
2. Why is bottom-up heap construction linear even though restoring order at
   one node may take logarithmic time?
3. What breaks if `Swap` does not update stored indexes?
4. When does bounded top-k improve on sorting every candidate, and what final
   work may still remain?
5. Which part of issue #81214 concerns ordering, which concerns time, and which
   concerns eligibility?
6. Why could scheduler-plugins issue #682 consume quadratic work while its
   destination queue deduplicated keys?
7. What state did selective requeueing need to retain, and which cleanup paths
   deserve tests?
8. What concrete service guarantee—maximum wait, turn frequency, or capacity
   share—would you promise for your own workload?
9. How do new logical arrivals and retry attempts place different kinds of
   pressure on the service?
10. Why did requests completing before their deadlines make Kafka's previous
    timer organization a poor fit?

The durable production habit is to ask two questions together:

> Which data structure makes the required operation cheap, and which policy
> determines whether the system should perform that operation at all?
