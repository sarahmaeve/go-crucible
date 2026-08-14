# W03 Candidate Guide

## What you may change

Change only `beginSiblingActivation` in `group.go`.

`RoundState` is state shared during one pass over the member observations.
`SimulateRound` creates a new value for each later pass. Preserve these rules:

- a member observed before the group is ready issues no sibling requests;
- once the group is ready, the pass makes every required sibling eligible;
- the pass does not repeat group-wide work unnecessarily;
- a later round may issue a new group-wide request; and
- the activation queue continues to retain at most one stored entry per ID.

Do not special-case a group size or remove observations from the test data.
Keep the change in the decision to start the group-wide loop.

## Choose evidence

The [evidence index](./evidence/README.md) names packets without showing their
contents. Before you open one, record its question, likely results, and which
explanation each result would weaken.

You do not need every packet.

## Decide when to open the code

Open `group.go` only after you can answer these questions:

- why `g(g-1)` is quadratic even though only `g` IDs exist;
- what work the queue may perform before it rejects a repeated ID;
- which facts need to survive between member observations, and which must reset
  before the next round; and
- what should happen to an unrelated repair under the fixed operation allowance
  described below.

Run the ordinary tests:

```bash
go test ./03-queues-heaps-scheduling/wheel/03-sibling-stampede -v
```

Run the test that reproduces the report:

```bash
go test -tags=csbridgewheel7 \
  ./03-queues-heaps-scheduling/wheel/03-sibling-stampede \
  -run TestOneGroupWideActivationPerRound -count=1 -v
```

The tagged test checks group sizes from 4 through 256. It gives each pass a
fixed operation allowance: one activation request or unrelated dispatch uses
one unit. This makes the work count repeatable. It does not claim that every
production operation takes equal time.

After you diagnose the failure, make the smallest change that satisfies the
tagged test and the rules above. Run both test modes again.

Write a three-minute handoff. State where the repeated calls begin, why storing
each ID once does not erase them, what you changed, and when any new state
resets. Then read [DEBRIEF.md](./DEBRIEF.md).
