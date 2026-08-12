# W03 Candidate Guide

## What you may change

Change only `beginSiblingActivation` in `group.go`.

`RoundState` is synthetic state shared during one pass over the member
observations. The code calls that pass a *reconciliation round*.
`SimulateRound` creates a new state value for each later pass. Preserve these
behaviors:

- a member observed before the group is ready issues no sibling requests;
- the first member observed after the group is ready may request all of its
  `g-1` siblings;
- later members in the same pass do not repeat those group-wide requests;
- a later round may issue a new group-wide request; and
- the activation queue continues to retain at most one stored entry per ID.

Do not special-case a group size or remove observations from the fixture. The
code change belongs where the coordinator decides whether the current member
may start the group-wide loop.

## Choose the next evidence packet

The [evidence index](./evidence/README.md) names available packets without
revealing their contents. Before opening one, record the question it answers,
the results you expect, and which explanation each result would weaken.

You do not need every packet.

## Before opening the source

Open `group.go` only after you can explain:

- why `g(g-1)` is quadratic even though only `g` IDs exist;
- what work the queue may perform before it rejects a repeated ID;
- why the state must cover one round rather than the lifetime of the group;
  and
- what should happen to an unrelated repair under the fixed operation allowance
  described below.

Run the ordinary behavior tests:

```bash
go test ./03-queues-heaps-scheduling/wheel/03-sibling-stampede -v
```

Run the test that reproduces the report's failure pattern:

```bash
go test -tags=csbridgewheel7 \
  ./03-queues-heaps-scheduling/wheel/03-sibling-stampede \
  -run TestOneGroupWideActivationPerRound -count=1 -v
```

The tagged test checks group sizes from 4 through 256. It also gives the pass a
fixed allowance of modeled operations: one activation request or one unrelated
dispatch uses one unit. This makes the amount of work deterministic; it does
not claim that every operation takes the same amount of time.

Write a three-minute handoff that states where the repeated calls originate,
why queue deduplication does not erase them, what state you added, and when
that state resets. Then read [DEBRIEF.md](./DEBRIEF.md).
