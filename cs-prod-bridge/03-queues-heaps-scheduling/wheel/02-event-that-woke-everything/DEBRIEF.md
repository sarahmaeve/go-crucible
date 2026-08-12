# W02 Debrief

Read this only after completing the handoff.

## Diagnosis

The max-priority order is not broken. Whenever the `gpu-v9` repair is active,
its higher priority correctly puts it before routine work. The problem occurs
before the heap chooses the next repair: unrelated hardware events move it from
the blocked map back to the active heap even though its missing hardware class
is still missing. This move is called *reactivation*.

The blocked record contains two pieces of information:

```text
kind = hardware
key  = gpu-v9
```

`OnEvent` compares only `kind`. An update about CPU metadata therefore moves
the repair from blocked to active. The attempt tests the full condition, fails,
and returns the item to blocked state. A *futile attempt* is one made while the
recorded prerequisite is still unavailable. Across many events, the dispatcher
repeatedly performs work it already has reason to expect will fail.

## Repair

Reactivate only when the concrete event matches the recorded condition:

```go
func (d *Dispatcher) OnEvent(event Event) {
    for id, item := range d.blocked {
        d.stats.EventChecks++
        if item.Needs.Kind != event.Kind || item.Needs.Key != event.Key {
            continue
        }
        delete(d.blocked, id)
        heap.Push(&d.active, item)
        d.stats.Activations++
    }
}
```

This local repair still checks every blocked item for every event. It reduces
activations and attempts, not the number of condition checks. An index from
condition to blocked IDs could reduce lookup work too, but it would add another
structure. Every path that blocks, reactivates, or completes a repair would then
have to keep the map and the index consistent.

Matching the event's `Kind` and `Key` does not prove that the repair is ready.
It says that the event concerns the same prerequisite and that another attempt
might now be useful. `Apply` records separately whether the prerequisite became
available, and `AttemptNext` remains the authority on whether the repair can
run. A matching event can therefore lead to another failed attempt if the
prerequisite is still unavailable.

Backoff—the delay before another attempt—is also a separate tool. It could
reduce how often an incorrectly reactivated repair is tried. It cannot
establish that CPU metadata created a `gpu-v9` device, so it does not replace
the decision about whether the event concerns the blocked prerequisite.

## Verification

Run:

```bash
go test ./03-queues-heaps-scheduling/wheel/02-event-that-woke-everything -v
go test -tags=csbridgewheel6 \
  ./03-queues-heaps-scheduling/wheel/02-event-that-woke-everything \
  -count=1 -v
```

The tagged scenario first blocks a high-priority repair. Twelve irrelevant
hardware events then arrive while twelve routine repairs are active. The fixed
event check causes no reactivation and no new futile attempt, so the routine
work completes. One matching event subsequently reactivates and completes the
blocked repair exactly once because that event also records `gpu-v9` as
available.

## Production inspiration and its limits

[Kubernetes issue #81214](https://github.com/kubernetes/kubernetes/issues/81214)
is a user report from a cluster with 5,000 nodes and more than 100,000 Pods. The
reporter described unschedulable higher-priority Pods returning to active
consideration after broad events, while lower-priority Pods waited. The report
specifically mentions PVC and Service events that blindly moved unschedulable
Pods.

Kubernetes's official
[QueueingHint account](https://kubernetes.io/blog/2024/12/12/scheduler-queueinghint/)
explains the later design: the scheduling queue records which plugin rejected
a Pod, and that plugin can examine a concrete cluster event to say whether it
could make that Pod schedulable. The
[QueueingHint enhancement proposal](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/4247-queueinghint/README.md)
also states an important risk: an incorrect hint can leave work blocked, so
selective reactivation needs tests for both missed and unnecessary retries. The
proposal describes a periodic retry of the unschedulable pool—five minutes by
default—as a fallback when a hint misses a useful event.

The local `Kind`/`Key` equality rule is deliberately smaller than Kubernetes's
plugin callbacks and object-specific checks. It demonstrates the decision
boundary under a local assumption: an event can concern a prerequisite only by
naming the same kind and key. Real systems may need to inspect old and new
object state or account for indirect dependencies. The exercise does not
reproduce Kubernetes source, scale, or scheduling semantics.

## Production follow-up

Measure event checks, activation requests, scheduling attempts, futile
attempts, completions, and how long each repair remains blocked separately. A
falling activation count is not sufficient if newly runnable work stops moving.
For every hint, test both directions: events that cannot help do not reactivate
the item, and events that may remove its recorded reason for failure do
reactivate it.
