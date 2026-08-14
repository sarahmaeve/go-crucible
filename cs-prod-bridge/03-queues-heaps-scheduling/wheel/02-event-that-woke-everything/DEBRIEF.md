# W02 Debrief

Read this only after completing the handoff.

## Cause

The priority order is correct. When the `gpu-v9` repair is active, its higher
priority puts it before routine work. The problem occurs earlier. Unrelated
hardware events move it from blocked state to the active heap even though
`gpu-v9` is still missing. This move is called **reactivation**.

The blocked record contains two pieces of information:

```text
kind = hardware
key  = gpu-v9
```

`OnEvent` compares only `kind`. A CPU metadata update therefore returns the
repair to active work. The attempt tests the full condition, fails, and blocks
the item again. An attempt made while its required resource is still missing
cannot succeed. Many unrelated events make the dispatcher repeat this work.

## Smallest repair

Return the repair to active work only when the event matches its recorded
condition:

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

This repair still checks every blocked item for every event. It reduces
activations and attempts, not condition checks. An index from conditions to
blocked IDs could reduce lookup work, but would add another structure. Every
block, reactivation, and completion would have to update both structures.

Matching `Kind` and `Key` does not prove that the repair is ready. It says only
that the event concerns the required resource and another attempt may help.
`Apply` records whether the resource became available. `AttemptNext` still
decides whether the repair can run. A matching event can lead to another
failure if the resource remains unavailable.

Backoff is also separate. It can reduce unnecessary retries, but cannot show
that CPU metadata created a `gpu-v9` device. It does not replace the event
check.

## Check the repair

Run:

```bash
go test ./03-queues-heaps-scheduling/wheel/02-event-that-woke-everything -v
go test -tags=csbridgewheel6 \
  ./03-queues-heaps-scheduling/wheel/02-event-that-woke-everything \
  -count=1 -v
```

The tagged scenario blocks one high-priority repair. Twelve unrelated events
arrive while twelve routine repairs are active. The repaired check does not
reactivate the blocked repair, so routine work completes. A later matching
event records `gpu-v9` as available, reactivates the repair, and completes it
once.

## What the production sources establish

[Kubernetes issue #81214](https://github.com/kubernetes/kubernetes/issues/81214)
is a user report from a cluster with 5,000 nodes and more than 100,000 Pods. The
reporter described blocked higher-priority Pods returning to active work after
broad events while lower-priority Pods waited. The report specifically names
PVC and Service events that moved all blocked Pods.

Kubernetes's official
[QueueingHint account](https://kubernetes.io/blog/2024/12/12/scheduler-queueinghint/)
explains the later design. The queue records which plugin rejected a Pod. That
plugin can inspect one cluster event and decide whether it might make the Pod
runnable. The
[QueueingHint enhancement proposal](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/4247-queueinghint/README.md)
also states a risk: a wrong hint can leave work blocked. Selective reactivation
therefore needs tests for both missed and unnecessary retries. The proposal
describes retrying the complete blocked pool every five minutes by default as a
fallback for a missed event.

The local `Kind`/`Key` rule is much smaller than Kubernetes's plugin callbacks
and object checks. It assumes an event can concern a required resource only by
naming the same kind and key. Real systems may need old and new object state or
indirect dependencies. The exercise does not reproduce Kubernetes code, scale,
or complete scheduling behavior.

## What to measure in production

Measure event checks, activation requests, scheduling attempts, attempts that
cannot succeed, completions, and time spent blocked. Fewer activations are not
an improvement if runnable work stays blocked. Test both directions for every
rule: unrelated events do nothing, and events that may resolve the recorded
failure return the item to active work.
