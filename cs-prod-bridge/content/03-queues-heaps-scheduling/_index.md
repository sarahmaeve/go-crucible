+++
title = 'Queues, heaps, and scheduling'
description = 'A heap can choose the next item quickly. A scheduler must also decide what is ready, when to retry, how to share capacity, and who owns each item.'
weight = 3
+++

**Unit 03 · Foundations**

# Queues, heaps, and scheduling

{{< lead >}}A heap can quickly choose the next item under one priority rule. It
cannot decide whether that item is ready, whether a retry could succeed, or
whether high-priority work is preventing other work from running. A scheduler
must make those decisions around the heap.{{< /lead >}}

{{< callout kind="production" title="Incoming reports" >}}
- [Kubernetes issue #81214](https://github.com/kubernetes/kubernetes/issues/81214)
  reports a cluster with 5,000 nodes and more than 100,000 Pods. Lower-priority
  Pods wait while higher-priority Pods that cannot run keep returning to the
  active queue.
- [Scheduler-plugins issue #682](https://github.com/kubernetes-sigs/scheduler-plugins/issues/682)
  reports millions of log lines, much higher CPU, and lost progress on
  unrelated runnable Pods in a production machine-learning workload.
- [Kafka's delayed-request account](https://www.confluent.io/blog/apache-kafka-purgatory-hierarchical-timing-wheels/)
  and [KAFKA-2147](https://issues.apache.org/jira/browse/KAFKA-2147) describe
  completed requests that remained in timer structures, used memory, and made
  cleanup expensive.
{{< /callout >}}

Each report involves queued work, but the speed of `heap.Pop` does not explain
any of them. The first concerns priority, retry delay, and events that return
too much work to the active queue. The second concerns repeated work across a
group. The third concerns removing completed timers, not finding the next one.

By the end of this unit, you should be able to:

- explain the different jobs of FIFO queues, priority queues, delayed queues,
  and schedulers;
- state a binary heap's ordering rule and the property it must preserve;
- calculate the cost of building and changing a heap;
- implement a heap correctly with Go's `container/heap` contract;
- calculate the cost of bounded-heap top-k selection and explain when sorting or
  another batch-selection method is a better fit;
- keep readiness, identity, retries, and worker ownership separate from
  priority;
- explain how broad wakeups and group activation multiply work;
- identify starvation, head-of-line blocking, and overload even when every
  individual queue operation is fast; and
- test ordering rules and state changes before measuring speed.

## A scheduler answers more than “what is next?”

A queue answers “which stored item is next?” A scheduler must also decide when
an item may run, whether repeated messages refer to the same work, what happens
when the system is full, and which component owns the item now.

| Question | How the system answers it | Failure when the answer is missing |
|---|---|---|
| Which ready item is next? | Arrival order or a stated priority rule | Important work waits, or ties behave unpredictably |
| When may this item run? | A ready time, retry delay, or yes-or-no readiness check | Work that cannot succeed is retried immediately |
| Is this item already present? | An ID that does not change and a record of queued or running IDs | Repeated events multiply queued work |
| What happens when the system is full? | A size limit and a decision to reject, wait, or replace | Memory grows without limit |
| Who owns an item right now? | A clear queued, delayed, running, or completed state | Work is lost, duplicated, or retained forever |

Do not combine all these decisions under the word “priority.” An item can have
the highest business priority and still be unable to run until a dependency
changes. A repeated message can refer to work already **in flight**—currently
owned by a worker—rather than an item stored in the heap.

We will use these workload variables:

- `n`: items currently stored in one queue or heap
- `a`: new work items that arrive during a time interval
- `d`: items waiting for their ready time in a delayed queue
- `u`: items that currently cannot run
- `r`: candidates examined by one top-k selection
- `k`: winners that selection must retain
- `h`: later candidates that enter the top `k` and replace the current cutoff,
  where `0 <= h <= r-k`
- `e`: external events that can cause the scheduler to reconsider work
- `g`: members of one related work group
- `f`: consecutive failures for one item
- `c`: useful work items completed during the interval

| Operation | Modeled time | Condition behind the claim |
|---|---:|---|
| Enqueue or dequeue in a suitable FIFO | usually `O(1)` over a series of operations | The implementation does not shift a slice on every dequeue |
| Inspect a heap root | `O(1)` | The heap is non-empty and its ordering rule holds |
| Insert or remove a heap root | `O(log n)` | At most one path through the tree needs swaps |
| Restore order after a priority change | `O(log n)` | The item's current heap index is known |
| Build a heap bottom-up | `O(n)` | Heap order is restored from the leaves upward |
| Fully sort `r` candidates | `O(r log r)` | Comparisons are constant-cost |
| Select the best `k` of `r` candidates with a bounded heap | `O(r + h log k)` time, `O(k)` working storage | `1 <= k <= r`; the first `k` entries are built into a heap bottom-up |
| Sort the `k` retained winners | `O(k log k)` | The result contract requires the winners in order, not merely the winning set |
| Wake every blocked item for every event | up to `O(eu)` | No check identifies which items the event might help |
| Every group member activates every other member in one round | `Θ(g²)` attempts | Each directed pair can generate work |

The heap bounds matter, but the last two rows can create much more work during
an incident. Count how many operations the scheduling rules create, not only
how fast one heap operation runs.

As in Unit 01, these formulas count operations before predicting running time.
They treat one comparison as constant work. In the lab, comparing scores has a
fixed cost. When scores tie, Go also compares the service names and may examine
each byte until it finds a difference. The formulas therefore assume a fixed
maximum name length. If comparison keys can grow, include that cost too.

## FIFO preserves arrival order

A first-in, first-out queue returns ready items in the order they entered:

~~~text
enqueue A, B, C
dequeue A, then B, then C
~~~

With a linked queue, circular buffer, or slice with a head index, adding and
removing items can take constant time on average. Deleting index zero from a Go
slice shifts every remaining element. Repeating that operation can make
draining `n` items quadratic.

Among ready items, FIFO prevents later arrivals from jumping ahead. It does
not by itself provide:

- storing each key only once;
- a memory limit;
- retry delay;
- cancellation;
- safe coordination between producers and consumers; or
- a way around a first item that cannot proceed.

The versioned
[Kubernetes client-go work queue](https://github.com/kubernetes/client-go/blob/v0.32.0/util/workqueue/queue.go)
shows this distinction in production. Alongside its FIFO storage, it keeps a
`dirty` set for keys that need work and a `processing` set for keys owned by a
worker. Adding the same key repeatedly does not create one stored entry per
call. If the key is added during processing, `Done` can enqueue it again so the
new information is not lost.

{{< callout kind="warning" title="What storing each key once cannot prevent" >}}
Keeping at most one stored entry for each key prevents repeats of that key from
increasing queue length. It does not limit the whole queue because distinct
keys can still accumulate. It also does not undo CPU work, allocations, locks,
logs, or API calls already spent on repeated submissions. Find where the
system creates repeated work and where the queue combines repeated keys.
{{< /callout >}}

## A priority queue exposes the most important ready item

A priority queue supports a smaller interface than a sorted collection:

~~~text
insert(item)
peek most important
remove most important
~~~

It does not need to expose every item in sorted order. This smaller promise
allows a useful compromise:

| Representation | Insert | Inspect best | Remove best |
|---|---:|---:|---:|
| Unsorted slice | usually `O(1)` over many inserts | `O(n)` | `O(n)` |
| Sorted slice | `O(n)` movement | `O(1)` | `O(1)` from the end |
| Binary heap | `O(log n)` | `O(1)` | `O(log n)` |

[MIT 6.006 Lecture 8: Binary Heaps](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/40d4851e550507ca14dc778b9b2266cc_MIT6_006S20_lec8.pdf)
explains this interface, the array representation, the heap property, and the
cost of each operation. It also gives router, operating-system scheduler, and
event-simulation examples. This unit connects the same model to public product
reports and patches.

## Store a complete tree in an array

A binary heap stores a complete binary tree in an array without child pointers.
For a zero-based index `i`:

~~~text
parent(i) = floor((i - 1) / 2), for i > 0
left(i)   = 2*i + 1
right(i)  = 2*i + 2
~~~

A complete tree fills every level except perhaps the last, which fills from
left to right. Its height is logarithmic in the
number of items, so moving one item between a leaf and the root takes
`O(log n)` steps.

A min-heap obeys this local rule, called the **heap invariant**:

~~~text
key(parent(i)) <= key(i)
~~~

A max-heap reverses the comparison. If the comparison is consistent, the root
is the best item under that rule.

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
“Highest priority first” is incomplete when priorities can tie. A scheduler
might compare `(priority DESC, sequence ASC, ID ASC)`: business priority first,
the older queue entry next, and an ID that does not change last. The ID makes
ties resolve the same way every time. Every heap operation must use this rule.
{{< /callout >}}

## Restore heap order along one path

Before a push or pop, every parent-child pair must follow the heap rule. Adding
or removing one item can break that rule along only one path through the tree.
The rest of the heap remains valid.

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

These operations **restore** or **repair** heap order. They do not rebuild or
sort the whole array. One change creates one possible break, and the algorithm
fixes that path.

Inserting `n` items one at a time can cost `O(n log n)`. A bottom-up build is
faster: start with the full array and restore order at each parent, working
back toward the root.

The work depends on how far each node can move. About half the nodes are leaves
and cannot move down. About one quarter can move down at most one level, one
eighth can move at most two levels, and so on. An upper bound has this form:

~~~text
(n/4) * 1 + (n/8) * 2 + (n/16) * 3 + ...
~~~

The number of nodes shrinks faster than their possible movement grows. The
total is therefore at most a constant times `n`. Bottom-up construction costs
`O(n)`, even though restoring order at the root can take `O(log n)` steps.

If an existing item's priority changes, it may need to move in either
direction. A heap implementation can restore the order in
`O(log n)` if it knows the item's current index.

{{< callout kind="warning" title="Changing a priority requires another heap operation" >}}
Changing a stored item's priority does not move it automatically. Until the
program restores heap order, the next removal can return the wrong item. Update
each item's saved index during every swap. Then call `heap.Fix` or the
equivalent operation.
{{< /callout >}}

## In Go, your type supplies the ordering rule

Go's [`container/heap` documentation](https://pkg.go.dev/container/heap)
defines the standard-library contract. Your type supplies `Len`, `Less`, and
`Swap`, plus methods that append and remove the final element. The package
moves items up or down to maintain heap order.

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

`ChangePriority` requires `item` to be in this heap and `index` to name its
current position. After `Pop` sets the index to `-1`, calling `heap.Fix` for
that item is invalid. A production API should look up the item by ID and check
that it is present before changing it.

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

The names can be surprising. `heap.Pop` moves the root to index `Len()-1`,
restores order in the remaining heap, and then calls your type's `Pop` method
to shorten the slice and return that final item. Calling `ready.Pop()` directly
would only remove the last element. It would not remove the heap root correctly.

The documented complexity is:

- `heap.Init`: `O(n)`;
- `heap.Push` and `heap.Pop`:
  `O(log n)`;
- `heap.Remove` and `heap.Fix`:
  `O(log n)`.

{{< callout kind="contract" title="What container/heap does not provide" >}}
The package preserves one rule: no child can outrank its parent. It does not
provide safe concurrent access, IDs, duplicate handling, capacity limits,
readiness, retry delay, cancellation, storage, fairness, or metrics. A
production queue must provide those policies and control concurrent access.
{{< /callout >}}

## Top-k needs the winners, not every rank

Suppose a query examines `r` samples and returns the `k` highest values, where
`1 <= k <= r`. An exact answer must inspect every candidate. A skipped
candidate might have the highest value. Selection from unsorted input therefore
requires at least `Ω(r)` work. When `k = r`, every candidate wins, but an
ordered result must still read and sort them all. The algorithm determines the
extra work and storage beyond that inspection.

Sorting all `r` candidates takes `O(r log r)` comparisons and gives every
candidate an exact rank. The query does not need the ranks of the `r-k`
candidates it will discard.

### Put the weakest current winner at the root

To find the largest `k` values with a heap that stores only `k` candidates,
put the **worst current winner** at the root. If score is the only key, use a
min-heap so the smallest retained score is at the root. That root is the cutoff
for each new candidate:

1. copy the first `k` candidates and build a heap from them bottom-up;
2. compare each later candidate with the root;
3. discard the candidate if it is no better than that root;
4. otherwise replace the root and restore heap order along one path.

A max-heap would expose the best winner. That value cannot tell us whether a
new candidate should replace the weakest winner. To find the smallest `k`
values, reverse the design: use a max-heap so the largest retained value is the
cutoff.

The heap comparison must include the tie rule. In the lab, a lower score is
worse. If scores tie, the alphabetically later service name is worse. The root
is the worst candidate under both fields, not only the lowest score.

### Calculate the work from the operations

Let `h` be the number of later candidates that replace the root. It can range
from zero to `r-k`. Selection has three parts:

- building one heap from the first `k` candidates costs `O(k)`;
- comparing each remaining candidate with the cutoff costs
  `Θ(r-k)`; and
- restoring heap order after `h` root replacements costs
  `O(h log k)`.

Because `k <= r`, the first two parts are linear in the input size. Selection
therefore takes:

~~~text
O(r + h log k) working time
O(k)             working storage
~~~

In the worst case, every later candidate enters the result. Then `h = r-k`,
which gives the familiar `O(r log k)` upper bound for `k >= 2`. That shorter
bound hides an important distinction: every later candidate gets one cutoff
check, but only a new winner causes a logarithmic heap update. Even when no
later candidate wins, selection is still `Θ(r)` because each candidate must be
checked.

If the API requires the winners in order, the heap has not finished. It finds
the winning set but does not fully order it. Sorting the `k` winners adds
`O(k log k)`, for a total of:

~~~text
O(r + h log k + k log k)
~~~

The boundary cases clarify the formula:

- For a fixed `k`, both `log k` and the final winner sort are constants, so the
  work grows as `Θ(r)`.
- For `k = 1`, the single retained candidate needs no heap movement;
  the algorithm is a linear maximum scan.
- When `k` is close to `r`, the final sort approaches the cost of sorting the
  complete input, so the bounded heap saves little time.
- When `k = r`, nothing is discarded. Building a heap and then sorting
  all winners adds work compared with sorting directly.
- The lab accepts `k = 0`, but still checks every score for `NaN` before
  returning an empty result. This boundary is `Θ(r)`. Another API could choose
  to return immediately instead.

### Compare complete methods

A bounded heap is one option, not the answer to every selection problem. This
table assumes an exact, ordered top-k result and promises not to change the
caller's input. Working storage excludes the input and returned result but
includes any copy needed during selection.

| Method | Time to return an ordered top `k` | Working candidate storage | Useful distinction |
|---|---:|---:|---|
| Sort all candidates | `O(r log r)` | `O(r)` | Direct and often fast; computes ranks the result does not use |
| Build a max-heap of all candidates, then remove `k` roots | `O(r + k log r)` | `O(r)` | Bottom-up construction is linear and removals arrive in result order, but every candidate remains stored |
| Keep a min-heap of `k` current winners | `O(r + h log k + k log k)` | `O(k)` | Can select in one pass with bounded state; its worst-case heap-update cost depends on how often the cutoff changes |
| Partition with quickselect, then sort the winners | expected `O(r + k log k)` | `O(r)` here; a mutable input can be partitioned in place | Suits an in-memory batch; ordinary pivot choices can have `O(r^2)` worst cases and do not maintain an answer as a stream arrives |
| Keep a sorted slice of `k` winners | `O(rk)` worst case | `O(k)` | Linear insertion is simple and can be competitive for very small `k`, but scales poorly as `k` grows |

The full max-heap and quickselect rows show why “top-k changes sorting from
`O(r log r)` to `O(r log k)`” is incomplete. That comparison does not prove
that a bounded heap is best for every in-memory batch. Its main advantages are
one-pass processing and working storage based on `k`, especially when `k` is
much smaller than `r`.

The lab receives an existing `[]Candidate`. It demonstrates `O(k)`
**additional working storage**, not total `O(k)` memory use. The same heap
operation can read an iterator, channel, file, or network stream without
storing all `r` candidates first, but only if the surrounding API also streams.

If the request asks for the `k` most frequent values in raw events, the heap
does not count them. First count each value, often with a hash table, and then
select from the distinct values and their counts. Counting and selection have
separate costs and can have different input sizes.

The pinned
[Prometheus v3.13.1 `topk` implementation](https://github.com/prometheus/prometheus/blob/v3.13.1/promql/engine.go#L3951-L4085)
uses this pattern. For each group, it keeps at most `k` samples in a min-heap.
When a better candidate replaces the root and `k > 1`, it calls `heap.Fix`. A
one-element heap needs no repair. Prometheus sorts the retained samples before
returning an instant-query result.

{{< callout kind="production" title="What the heap saves" >}}
A `topk` query still examines all `r` samples. It avoids ranking the `r-k`
samples that it discards. When `k` is much smaller than `r`, the bounded heap
uses a simple cutoff check for each candidate and logarithmic work only for a
new winner. In the lab, it also reduces working candidate storage from `O(r)`
to `O(k)`. An ordered response still requires a final sort of the `k` winners.
The heap does not remove the input scan or guarantee a faster run for every
value of `r` and `k`.
{{< /callout >}}

Selection and scheduling use the same heap operations, but the root means
different things. A top-k heap puts the **worst retained winner** at the root
so it can reject a new candidate cheaply. A ready-work heap usually puts the
**next item to serve** at the root. The comparison rule creates that meaning.

The next sections return to scheduling. They add future deadlines, reasons to
retry, identity, cleanup, fair sharing, and capacity one at a time. Production
reports show what happens when one of those responsibilities is missing.

## Keep ready work and delayed work in separate orders

A scheduler often needs two independent orders. Priority chooses among ready
items. Time chooses which delayed item becomes ready next:

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

One fixed heap rule cannot answer both questions. If business priority comes
first, a high-priority future item can hide ready work below it. If `readyAt`
comes first, the root shows the next deadline but not the highest-priority item
that is ready now. A rule based on `readyAt <= now` changes as time passes,
without a heap operation to restore order. Separate heaps, or equivalent
indexes, answer the questions independently. When a deadline arrives, move the
item from the time-ordered structure to the active priority heap.

The pinned
[Kubernetes v1.32 scheduling queue](https://github.com/kubernetes/kubernetes/blob/v1.32.0/pkg/scheduler/backend/queue/scheduling_queue.go)
is a concrete system to inspect. It separates an active priority queue, a
retry-delay queue, and Pods that currently cannot be scheduled.
The scheduler's versioned
[heap wrapper](https://github.com/kubernetes/kubernetes/blob/v1.32.0/pkg/scheduler/backend/heap/heap.go)
combines heap ordering with a key-to-item map so an existing item can be found,
updated, or removed without scanning the whole heap.

The map adds a responsibility that a heap alone does not provide:

~~~text
heap array     -> choose by priority and restore heap order
ID lookup      -> find, update, combine, or remove an item by key
state rules    -> decide which structure currently owns the item
~~~

The lookup is correct only if every swap, insertion, removal, and state change
keeps it consistent with the heap.

## Backoff delays retries

After a temporary failure, an immediate retry can create a tight loop.
**Backoff** adds a delay. A common exponential rule is:

~~~text
delay(f) = min(maxDelay, initialDelay * 2^(f-1))
~~~

Here `f >= 1` is the number of consecutive failures. `maxDelay` prevents the
delay from growing forever. Check whether the next doubling reaches the cap
before multiplying; otherwise a large duration can overflow first. A policy
can also add a small random change called **jitter**. Jitter makes simultaneous
wakeups less likely, but does not guarantee different wakeup times. State
whether the policy applies the maximum again after adding it.

The pinned
[client-go v0.32 delaying queue](https://github.com/kubernetes/client-go/blob/v0.32.0/util/workqueue/delaying_queue.go)
keeps delayed entries ordered by target time. If an existing entry receives an
earlier deadline, the queue changes its `readyAt` value and calls `heap.Fix` at
the entry's saved index. The item ID does not change. This is a priority change
applied to a retry deadline.

Use a fake clock to test this policy. Real sleeps make boundary tests slow and
unpredictable. A controlled clock can move directly to just before, at, and
just after a deadline.

{{< callout kind="contract" title="Backoff answers when, not whether" >}}
Backoff spaces attempts. It does not show that another attempt can succeed. If
no relevant dependency changed, a slower useless retry is still useless.
Readiness and backoff are separate decisions.
{{< /callout >}}

## Production report: high-priority Pods kept returning without becoming runnable

[Kubernetes issue #81214](https://github.com/kubernetes/kubernetes/issues/81214)
is a user report from a cluster with 5,000 nodes and more than 100,000 Pods. The
reporter observed lower-priority Pods waiting while higher-priority Pods that
could not run repeatedly moved from backoff to the active queue. The report
also names PVC and Service events that returned all blocked Pods to active
consideration.

An event unrelated to a Pod could still return it to the active heap. The heap
then correctly selected that high-priority Pod ahead of lower-priority Pods
that could run. Each part could follow its own rule while the system made no
useful progress:

1. an event makes many high-priority Pods active;
2. the priority heap correctly returns them first;
3. their unchanged constraints make them fail again;
4. backoff delays them, but later events return them to the active queue;
5. lower-priority work that can run repeatedly loses the comparison.

The report connects three separate problems:

- strict priority can starve lower-priority work;
- a retry delay can be too short for the workload; and
- events that wake every blocked item can cancel the benefit of backoff.

The merged
[Kubernetes PR #81263](https://github.com/kubernetes/kubernetes/pull/81263)
added settings for the initial and maximum Pod backoff. That patch made retry
timing configurable. It did not make every event relevant to every blocked
Pod. Retry timing and the decision to retry remain separate.

{{< callout kind="production" title="Read the report before the patch" >}}
Start with issue #81214 and write the proposed chain of causes. Then inspect PR
#81263 and name exactly which step it changes. A merged patch shows what the
implementation changed, but it does not prove that one change solved every
part of the report.
{{< /callout >}}

## Retry only when a relevant condition may have changed

A blocked item is waiting for a yes-or-no condition, such as enough resources,
a compatible node, a dependency, or available quota. An event should return an
item to active work only when it might change that condition. This return to
active work is called **reactivation**.

Filtering and indexing reduce different work. For one event, let `s` be the
number of blocked items selected for another attempt:

~~~text
return everything:  up to u candidate checks
                    + up to u activation requests

select by relevance: cost of finding or testing candidates
                     + s activation requests, where 0 <= s <= u
~~~

Across `e` events, returning everything can perform up to `eu` checks and `eu`
activation requests. A relevance rule reduces requests when it selects fewer
than `u` items. Without an index, the rule may still inspect all `u` blocked
items to find those `s` items. An index can identify likely items directly.
Choosing fewer items and finding them faster are separate improvements.

Kubernetes's official
[QueueingHint design account](https://kubernetes.io/blog/2024/12/12/scheduler-queueinghint/)
describes the change. Previously, a scheduler plugin named broad event
categories that might resolve one of its failures. Any event in a named
category could cause another attempt, even when it could not help that Pod.
With QueueingHint, the plugin can inspect the specific event and decide whether
it might make the Pod schedulable.

That check can also be wrong. If it says an important event is irrelevant, a
Pod that could now run may remain blocked. Periodically retrying all blocked
Pods can catch those mistakes, but the retry period creates a trade-off:

- more frequent scans reduce the time a missed item remains stuck;
- less frequent scans reduce repeated background work;
- better event indexes and relevance rules reduce reliance on the scan.

{{< callout kind="production" title="Correct heap order cannot prevent useless retries" >}}
The active heap can follow its comparison rule perfectly while the queue is
full of items that still cannot run. In that case, change which events return
items to the queue. Changing the heap order will not help.
{{< /callout >}}

## One stored key can still receive many requests

Consider one pass over `g` related items. If every member asks to activate the
other `g-1` members, the pass generates:

~~~text
g * (g - 1) = Θ(g²)
~~~

The queue can still store each key only once. Before it rejects a repeated key,
however, callers have already made `g(g-1)` requests. Each request can take a
lock, check a map, and write a log. Queue depth can look safe while CPU use and
log volume grow quickly.

[Scheduler-plugins issue #682](https://github.com/kubernetes-sigs/scheduler-plugins/issues/682)
reports a production machine-learning workload using the Kubernetes
coscheduling plugin. Quota and resource limits left a Pod group waiting. The
reporter saw millions of log lines, much higher CPU use, and no scheduling
progress for some unrelated Pods that could have run.

The merged patch addresses repeated requests by group members to activate their
siblings. The
[scheduler-plugins PR #700](https://github.com/kubernetes-sigs/scheduler-plugins/pull/700)
stores an `Activate` flag for one Pod's scheduling attempt. The coscheduling
plugin reaches `Permit` after earlier group and resource checks. It either lets
the Pod continue or makes it wait for enough group members. The patch sets
`Activate` only when no group member is assigned. `ActivateSiblings` does no
group-wide work unless that flag is present.

For one scheduling attempt that reaches that path and performs sibling
activation once, the plugin generates at most `g-1` sibling requests:

~~~text
one scheduling attempt + at most g - 1 sibling activation requests = O(g)
~~~

This bound covers one scheduling attempt, not the whole workload. Total work
still depends on how many attempts reach this path.

{{< callout kind="production" title="Measure requests as well as queue entries" >}}
Count activation requests, distinct items activated, heap insertions, and
useful completions separately. Their differences show how much work happens
before the queue combines repeated keys. Queue length alone cannot show it.
{{< /callout >}}

## Delete saved state when its owner finishes

A scheduler may save events that happen while it evaluates an item. If the
attempt fails, those events can help decide where the item goes next. The saved
records use memory and need a clear deletion rule.

A useful set of state changes is:

~~~text
queued -> in-flight -> queued again
                    -> completed
                    -> rejected
                    -> cancelled
~~~

Every exit from the in-flight state must remove records owned by that attempt.
Cleaning up after success is not enough. Skipped items, cancellation, errors,
duplicate entries, and shutdown also need cleanup.

[Kubernetes issue #120622](https://github.com/kubernetes/kubernetes/issues/120622)
describes memory used by cluster events that accumulated while Pods were being
scheduled. The issue links several follow-up changes rather than claiming one
universal fix. One of them,
[PR #126962](https://github.com/kubernetes/kubernetes/pull/126962), fixes one
path where a skipped Pod missed the expected requeue or bind cleanup and
could leave in-flight state behind.

{{< callout kind="warning" title="New saved state needs its own cleanup" >}}
Avoiding useless retries can require indexes, rejection reasons, and saved
event records. For each structure, name its owner, deletion points, and count
metric. Garbage collection cannot remove records that the program still
references.
{{< /callout >}}

For a fixed batch, while the model keeps one final record for every submission,
this accounting check should hold:

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

This equality applies only while the model keeps the final records. One
submission must not appear in two states. An item in two states has two owners;
an item in no state has been lost. If a logical ID can be reused, pair it with
a generation or submission ID so each submission remains distinct.

## Strict priority can make lower-priority work wait forever

A priority heap can follow its rule exactly while lower-priority work never
runs. If high-priority work arrives as fast as workers finish it, another
high-priority item is always at the root. Ready lower-priority items are never
selected. This is **starvation**.

FIFO prevents later arrivals from jumping ahead. It creates a different risk:
if the first item cannot proceed and must remain first, everything behind it
also waits. This is **head-of-line blocking**.

“Fair” is not one behavior. State a promise that the system can measure:

- once an item is ready, later arrivals cannot postpone it beyond a stated
  number of dispatches;
- each continuously backlogged tenant receives a turn within a stated number
  of dispatches or amount of time; or
- continuously backlogged tenants receive stated shares of the available
  worker time over a stated interval.

Different policies provide different promises. Round robin visits each
non-empty tenant queue in turn. A weighted version gives some queues more turns.
An aging policy raises the priority of work that has waited a long time. The
heap does not provide any of these rules by itself.

Each promise has conditions. A dispatch-count limit assumes that workers keep
selecting work and that the number of covered queues is finite. A time limit
also needs a bound on how long one task can hold a worker. A capacity-share
promise must name its measurement period and explain what happens to unused
shares.

The Kubernetes API server's
[Priority and Fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/)
feature is one production example. It assigns requests to priority levels,
limits how many requests in each level run at once, and uses separate queues so
one busy source is less able to crowd out others. Its complete algorithm is
outside this unit. The relevant lesson is that a priority label does not decide
how to share capacity.

No queue policy can guarantee another turn or a maximum wait after workers stop
finishing work and releasing capacity.

## A faster queue cannot create worker capacity

Track distinct work and scheduling attempts separately:

~~~text
change in logical backlog = new submissions - final exits
attempt demand             = first attempts + retry attempts
~~~

If completion is the only final outcome, let `a` be new arrivals and `c` useful
completions in the same period. If `a > c` for long enough, live work grows no
matter how fast the queue is. If rejection or cancellation also removes work,
include those outcomes as exits.

A retry does not create a new work item, so it is not another arrival in the
backlog equation. It does create another attempt and use worker capacity. Retry
traffic can overload the service while the number of distinct queued IDs stays
flat.

When the system is full, choose explicitly among:

- reject new work and return a retryable signal;
- block or slow producers;
- combine superseded updates that share an ID;
- shed a documented low-value class;
- persist overflow elsewhere; or
- allow limited growth for a measured burst.

During sustained overload, an unbounded in-memory queue lets memory use and
wait time grow until the process or host reaches a limit. A bounded queue makes
overload visible, but must define what it rejects and how the caller learns.

Operational signals should describe policy as well as container size:

- active, delayed, blocked, and in-flight counts;
- oldest wait and the distribution of wait times by priority or tenant;
- arrivals, attempts, useful completions, and permanent failures;
- attempts per completion and attempts made before a required condition changed;
- activation requests versus distinct IDs activated;
- rejection, combined-update, and cancellation counts;
- heap comparisons, swaps, pushes, pops, and fixes; and
- saved event records or other queue-owned data.

These measures distinguish an expensive heap operation from too many heap
operations and from too little worker capacity.

## Kafka also needed to remove requests before their timeout

A heap-backed timer queue quickly finds the next request to time out. Kafka also
needed to handle a common second outcome: a request completing before its
timeout.

For example, a produce request may wait for replicas to acknowledge a write,
and a fetch request may wait for new data. Either request can finish in two
ways:

1. the condition it is waiting for becomes true; or
2. its timeout expires.

Kafka calls requests waiting for either outcome **request purgatory**. The older
design described in
[Apache Kafka, Purgatory, and Hierarchical Timing Wheels](https://www.confluent.io/blog/apache-kafka-purgatory-hierarchical-timing-wheels/)
stored each request in a Java `DelayQueue` ordered by timeout and in lists used
to find requests when relevant broker state changed.

The `DelayQueue` quickly exposed the next timeout. When a request completed for
another reason, however, the old design did not immediately remove it from the
timer queue or other lists. A cleanup thread periodically scanned those
structures for completed requests.

That left an unpleasant choice:

- scan infrequently, and completed requests continue occupying memory; or
- scan frequently, and spend substantial CPU repeatedly searching the lists.

The public incident
[KAFKA-2147](https://issues.apache.org/jira/browse/KAFKA-2147) reports this
failure at production scale. Retained requests and Java-managed memory grew
together. Here **JVM heap** means memory managed by the Java runtime, not the
binary-heap data structure. Draining the large backlog then caused latency and
leadership disruption.

### Group deadlines and remove completed requests directly

A **timing wheel** groups nearby deadlines into buckets, like marks around a
clock. As time advances, the system processes the current bucket. Later timers
go into additional wheels whose buckets cover larger ranges. These levels make
the structure **hierarchical**.

The new design gave each request a direct link to its bucket entry. If a request
completed early, Kafka could unlink it with a constant number of list changes.
It no longer remained in the timer queue until a later scan.

Kafka still used a `DelayQueue`, but stored time buckets instead of individual
requests. The number of possible buckets had a fixed upper limit and was
usually much smaller than the number of requests. This reduced heap operations
and completed entries waiting for cleanup.

The article includes before-and-after benchmarks for Kafka's Java code and test
workload. They show why this change helped Kafka. They do not prove that a
timing wheel is always faster than a heap.

{{< callout kind="production" title="Why Kafka changed structures" >}}
The old design found the next timeout quickly, but many requests completed for
another reason. Kafka combined two mechanisms. The timing wheel grouped nearby
deadlines, so the `DelayQueue` tracked buckets rather than every request. Each
request linked directly to its entry inside a bucket, so completion could
remove it immediately. Buckets reduced timer-queue work. Direct links avoided a
full cleanup scan.

Do not begin with “heap or timing wheel?” First ask how work usually leaves the
system. If items often finish before reaching the heap root, finding and
removing those items may matter more than fast root access.
{{< /callout >}}

## Use the lab and Wheels

The [bounded top-k lab](lab/) isolates the heap operation from the larger
scheduler. It compares sorting all candidates with keeping only the `k` current
winners. Tests count cutoff checks, changed winners, heap construction, and the
final result sort separately. Benchmarks vary `r`, `k`, and input order. The
written exercise also compares a full heap, quickselect, and a sorted slice.

Then investigate [The Lost Assignment Loop](wheel/01-lost-assignment-loop/)
from its incoming report. This Wheel draws on GitHub's
[August 6–7, 2026 Actions incident](https://www.githubstatus.com/incidents/qcvjkzcs7j74):
after an initial capacity failure, runners repeatedly attempted jobs that were
no longer valid and could not take valid work. The local exercise asks when
another attempt by the same worker can still succeed. It does not claim that
GitHub used this Go code or a heap for runner assignment.

[The Event That Woke Everything](wheel/02-event-that-woke-everything/) uses a
max-priority heap in its local dispatcher. Inventory events repeatedly return a
high-priority repair that still cannot run while routine repairs wait. Use the
evidence to decide whether the ordering rule or another state change causes the
problem.

[The Sibling Stampede](wheel/03-sibling-stampede/) examines work done before a
queue discovers that it already contains an ID. It uses the
scheduler-plugins #682 report and merged #700 patch as production inspiration,
then counts requests, stored IDs, repeated logs, and unrelated progress in a
smaller invented model.

Use the [investigation worksheet](wheel/worksheet/) to separate the event that
started the incident, the condition that stopped recovery, and the state change
you propose.

## Check behavior before measuring speed

Begin with rules that tests can observe without relying on timing.

### Check the heap rule

After every push, pop, removal, and priority change, inspect each parent-child
pair. For every child `i`, assert `!Less(i, parent(i))`: no child may outrank
its parent. This property is the **heap invariant**. Also check that every saved
index points to the correct item.

For small random inputs, compare removals with a simple reference that sorts all
current items after each change. The reference can be slow; its purpose is to
make the expected answer obvious.

### Account for every submitted item

After every state change, check that each submission appears in exactly one of:

~~~text
active | delayed | blocked | in-flight | completed | rejected | cancelled
~~~

Completion and permanent failure must release retry state, indexes, and saved
events owned by the queue. Keep history and metrics separately. A repeated
notification must not give the same item two owners.

### Check readiness and service promises

With a fake clock, verify that:

- no delayed item runs before `readyAt`;
- an item becomes eligible at the exact boundary;
- backoff reaches but never exceeds its cap;
- an irrelevant event activates no blocked item;
- a relevant event activates the affected items; and
- if the policy limits how many dispatches may pass before a continuously ready
  item runs, lower-priority work advances within that stated limit.

Tests of a wait-time or dispatch-count promise need a finite, deliberately
difficult scenario and a clear limit. Do not depend on a goroutine eventually
winning a race on one developer's machine.

### Count work before benchmarking it

Count comparisons, swaps, activation requests, distinct activations, useless
attempts, and completions. Then increase `n`, `u`, and `g`. Profiles and
allocation measurements explain machine-specific costs. Counters show whether
the modeled work grows logarithmically, linearly, or quadratically.

This order matters. Faster code with the wrong state rule can create retry work
more quickly without completing anything useful.

## Know what each source can establish

This lesson uses several kinds of evidence. They support different claims:

| Source | What it establishes |
|---|---|
| [MIT 6.006 Lecture 8](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/40d4851e550507ca14dc778b9b2266cc_MIT6_006S20_lec8.pdf) | The priority-queue interface, heap rule, operations, proofs, and how work grows |
| [Go `container/heap` docs](https://pkg.go.dev/container/heap) | The public Go API contract and documented complexity |
| Pinned product source | What one named release implemented, not what every release must implement |
| Public issue report | What an operator observed and proposed as an explanation in one production setting |
| Merged pull request | What maintainers changed; not proof that every reported symptom had one cause |
| Official design account | The intended model, rollout history, trade-offs, and documented boundaries |

When you summarize a patch, name the behavior it changes and link the complete
patch. Treat a lesson drawn from one incident as a model to test, not a fact
about every scheduler.

## Explain the design to a colleague

One answer to “how would you design this dispatcher?” might be:

> I would separate readiness from priority. A max-heap can select the most
> important of `n` active items with constant-time root access and
> logarithmic updates, while a min-heap can expose the earliest of
> `d` retry deadlines. Failed work should return only after backoff
> or a state change related to its failure. I would keep an ID lookup for
> updates and duplicate handling, but also count activation requests because
> repeated work happens before the queue combines duplicate keys. Tests would
> cover both heap rules, one owner for each submission, readiness, and a clear
> limit on how long ready work may wait or a stated rule for sharing capacity.
> Finally, I would bound the queue and define what happens when new submissions
> exceed final exits or retry attempts consume the available worker
> capacity.

That answer names the operations, structures, costs, correctness rules, and
overload policy. It does not claim that a heap supplies the whole scheduler.

## Questions to review

For each answer, identify the model, implementation, or production evidence
that supports it:

1. Why is a valid heap not a sorted array?
2. Why is bottom-up heap construction linear even though restoring order at
   one node may take logarithmic time?
3. What breaks if `Swap` does not update stored indexes?
4. When does bounded top-k improve on sorting every candidate, and what final
   work may still remain?
5. Which part of issue #81214 concerns ordering, which concerns time, and which
   concerns readiness?
6. Why could scheduler-plugins issue #682 consume quadratic work while its
   destination queue stored each key once?
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
