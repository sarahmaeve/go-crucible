# W02 Candidate Guide

## What you may change

Change only `OnEvent` in `dispatcher.go`.

The dispatcher keeps a structured record of why each failed repair is blocked.
Do not assume which parts of that record `OnEvent` should use. Infer the rule
from the report and the evidence you choose. Preserve these behaviors:

- active work is ordered by priority descending, sequence ascending, and ID
  ascending;
- a failed attempt moves the repair to blocked state;
- the reported inventory burst does not prevent runnable routine repairs from
  completing;
- a change that makes a blocked repair runnable is not lost; and
- a repair returned to the active heap still has to pass its readiness check.

Do not replace the active priority heap or optimize the scan of blocked items.
This Wheel isolates the rule that selects which blocked items an event returns
to active work. The local function may still inspect every blocked item. An
index by condition would be a separate improvement.

## Choose the next evidence packet

The [evidence index](./evidence/README.md) names available packets without
revealing their contents. Before opening one, write down the question it tests,
the outcomes you expect, and how each outcome would change your explanation.

You do not need every packet.

## Before opening the source

Open `dispatcher.go` only after you can explain:

- the separate questions answered by active ordering, event handling, and the
  readiness check;
- why a correct max-priority order can repeatedly select useless work;
- why backoff—a delay before another attempt—can space failures without showing
  that another attempt is worthwhile; and
- which measurements distinguish scanning blocked work, requesting
  reactivation, attempting work, and completing work.

Run the ordinary behavior tests:

```bash
go test ./03-queues-heaps-scheduling/wheel/02-event-that-woke-everything -v
```

Run the test that reproduces the report's failure pattern:

```bash
go test -tags=csbridgewheel6 \
  ./03-queues-heaps-scheduling/wheel/02-event-that-woke-everything \
  -run TestInventoryBurstPreservesUsefulProgress -count=1 -v
```

Change `OnEvent`, then run both commands again. The tagged test counts
activations, futile attempts, and completions; it has no sleep or timing
threshold.

Write a three-minute handoff that states the blocked condition, the event rule
you adopted, what work is avoided, and how the test proves that a repair is not
lost when it becomes runnable. Then read [DEBRIEF.md](./DEBRIEF.md).
