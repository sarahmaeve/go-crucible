# W03 Debrief

Read this only after completing the handoff.

## Diagnosis

For a group of `g` members, each member requests activation of its other
`g-1` siblings:

```text
g members * (g - 1) requests per member = g(g - 1)
```

The activation queue stores at most `g` distinct IDs, but deduplication occurs
inside each request. The caller has already entered the queue code, looked up
the ID, and possibly acquired a lock or formatted a log message. A bound on
retained entries therefore does not bound request-processing work.

The sibling loop is necessary once: one member examined after the group becomes
ready must make the other members eligible. The defect is allowing every such
member in the same pass to initiate the whole loop again.

## Repair

Record whether the action has already been issued during this pass:

```go
func beginSiblingActivation(state *RoundState) bool {
    if state.activationIssued {
        return false
    }
    state.activationIssued = true
    return true
}
```

The first member examined after the group becomes ready makes `g-1` requests.
Later members see the state and make none. `SimulateRound` creates a fresh
`RoundState`, so a later pass can issue a new activation if the group still
requires it. Remembering the bit forever would suppress legitimate future
work.

The fixed operation allowance in the tagged test makes the consequence
explicit. With one group-wide activation, `g-1` requests leave room for an
unrelated dispatch within `2g` modeled operations. When every member starts the
loop, activation requests consume that allowance first. This counts operations;
it does not claim that every production request has the same CPU cost.

## Verification

Run:

```bash
go test ./03-queues-heaps-scheduling/wheel/03-sibling-stampede -v
go test -tags=csbridgewheel7 \
  ./03-queues-heaps-scheduling/wheel/03-sibling-stampede \
  -count=1 -v
```

The ordinary tests preserve the behavior when only one member is observed,
confirm that no activation occurs before the group is ready, and confirm that a
later pass may activate the group again. The tagged test checks the number of
requests, the set of activated siblings, the absence of repeated requests, and
progress for unrelated work.

## Production inspiration and its limits

[Scheduler-plugins issue #682](https://github.com/kubernetes-sigs/scheduler-plugins/issues/682)
reports a production machine-learning workload using the Kubernetes
coscheduling plugin. Under quota and resource restrictions, Pod groups could
remain pending; the reporter observed millions of activation-related log lines,
much higher CPU, and no useful scheduling progress for some unrelated Pods
whose resources were available.

The merged
[scheduler-plugins PR #700](https://github.com/kubernetes-sigs/scheduler-plugins/pull/700)
stores an `Activate` flag in the state for one Pod's scheduling attempt. In the
plugin's `Permit` step, it sets that flag only for an attempt that sees zero
already assigned group members. `ActivateSiblings` later reads the same
attempt's state and returns without waking the group unless the flag is present.
In plain terms, the patch identifies the one scheduling attempt that may wake
the group instead of letting every attempt do so.

The exercise uses a different mechanism. Its `RoundState` is shared by all
member observations in one synthetic pass; Kubernetes's scheduling-attempt
state is not. The local Boolean directly records “already issued during this
pass,” while the production patch selects one attempt using the assigned-member
count and carries permission only through that attempt. Both limit repeated
group-wide work, but their state ownership and lifetimes are different.

The exact `g(g-1)` counts, fixed operation allowance, maintenance domain, and
logs are synthetic. The public issue supplies the reported production symptoms;
the merged patch supplies an inspectable code change intended to eliminate the
repeated wake-ups.

## Production follow-up

Measure activation requests, distinct activated IDs, duplicate requests, log
volume, and how long unrelated work waits separately. Include the group size
and the number of passes. Even `g-1` requests per pass can become excessive if
the system starts passes without limit, so also count how often a member is
allowed to initiate the group-wide action.
