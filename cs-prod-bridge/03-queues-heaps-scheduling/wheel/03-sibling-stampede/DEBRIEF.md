# W03 Debrief

Read this only after completing the handoff.

## Cause

For a group of `g` members, each member requests activation of its other
`g-1` siblings:

```text
g members * (g - 1) requests per member = g(g - 1)
```

The activation queue stores at most `g` IDs, but it rejects repeats only after a
request enters the queue code. By then, the request may have looked up the ID,
taken a lock, and formatted a log. A limit on stored entries does not limit the
work spent processing requests.

The sibling loop is needed once after the group becomes ready. The defect is
letting every later member in the same pass start the whole loop again.

## Smallest repair

Record whether this pass has already issued the action:

```go
func beginSiblingActivation(state *RoundState) bool {
    if state.activationIssued {
        return false
    }
    state.activationIssued = true
    return true
}
```

The first member examined after readiness makes `g-1` requests. Later members
make none. `SimulateRound` creates a new `RoundState`, so a later pass can act
again if needed. Keeping the Boolean for the group's lifetime would incorrectly
block future work.

The tagged test gives each pass a fixed operation allowance. One group-wide
action uses `g-1` requests and leaves room for unrelated work within `2g`
operations. If every member starts the loop, activation requests use the whole
allowance first. This is an operation count, not a claim that every production
request has equal CPU cost.

## Check the repair

Run:

```bash
go test ./03-queues-heaps-scheduling/wheel/03-sibling-stampede -v
go test -tags=csbridgewheel7 \
  ./03-queues-heaps-scheduling/wheel/03-sibling-stampede \
  -count=1 -v
```

The ordinary tests cover one observed member, no activation before readiness,
and a later pass acting again. The tagged test checks request count, activated
siblings, repeated requests, and progress for unrelated work.

## What the production sources establish

[Scheduler-plugins issue #682](https://github.com/kubernetes-sigs/scheduler-plugins/issues/682)
reports a production machine-learning workload using the Kubernetes
coscheduling plugin. Quota and resource limits left Pod groups waiting. The
reporter observed millions of activation logs, much higher CPU use, and no
progress for some unrelated Pods that could run.

The merged
[scheduler-plugins PR #700](https://github.com/kubernetes-sigs/scheduler-plugins/pull/700)
stores an `Activate` flag for one Pod's scheduling attempt. In `Permit`, it sets
the flag only when no group member is already assigned. `ActivateSiblings`
later reads the same state and returns unless the flag is present. The patch
therefore selects one attempt that may wake the group.

The exercise uses a different mechanism. All member observations share one
`RoundState` during an invented pass; Kubernetes's attempt state is not shared
that way. The local Boolean records “already issued this pass.” The production
patch selects one attempt using the assigned-member count. Both reduce repeated
group-wide work, but the state has a different owner and lifetime.

The exact `g(g-1)` counts, operation allowance, maintenance domain, and logs are
invented. The public issue supplies the production symptoms. The merged patch
supplies a code change intended to remove repeated wakeups.

## What to measure in production

Measure activation requests, distinct IDs, repeated requests, log volume, and
unrelated wait time separately. Include group size and number of passes. Even
`g-1` requests per pass become excessive if passes repeat without limit. Count
how often the system permits a group-wide action.
