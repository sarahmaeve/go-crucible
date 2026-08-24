# W02 Candidate Guide

## What you may change

Change only `OnEvent` in `dispatcher.go`.

The dispatcher records why each failed repair is blocked. Use the report and
evidence to decide which parts of that record `OnEvent` needs. Preserve these
rules:

- active work is ordered by priority descending, sequence ascending, and ID
  ascending;
- a failed attempt moves the repair to blocked state;
- the reported inventory burst does not prevent runnable routine repairs from
  completing;
- a change that can make a blocked repair runnable is not lost; and
- a repair returned to the active heap still has to pass its readiness check.

Do not replace the priority heap or optimize the scan of blocked items. This
Wheel covers only the rule that selects which items an event returns to active
work. The function may still inspect every blocked item. An index would be a
separate change.

## Choose evidence

The [evidence index](./evidence/README.md) names packets without showing their
contents. Before you open one, write the question it tests, the likely results,
and how each result would change your explanation.

You do not need every packet.

## Decide when to open the code

Open `dispatcher.go` only after you can answer these questions:

- the separate questions answered by active ordering, event handling, and the
  readiness check;
- why a correct max-priority order can repeatedly select useless work;
- why backoff—a delay before another attempt—can space failures without proving
  that another attempt can help; and
- which measurements distinguish scanning blocked work, returning it to the
  active heap, attempting it, and completing it.

Run the ordinary tests:

```bash
go test ./03-queues-heaps-scheduling/wheel/02-event-that-woke-everything -v
```

Run the test that reproduces the report:

```bash
go test -tags=csbridgewheel6 \
  ./03-queues-heaps-scheduling/wheel/02-event-that-woke-everything \
  -run TestInventoryBurstPreservesUsefulProgress -count=1 -v
```

After you diagnose the failure, make the smallest change that passes the tagged
test while preserving useful events. Run both commands again. The tagged test
counts activations, attempts that cannot succeed, and completions. It does not
use sleeps or a timing limit.

Write a three-minute handoff. State the blocked condition, your event rule, the
work it avoids, and how the test checks newly runnable work. Then read
[DEBRIEF.md](./DEBRIEF.md).
