# W02: The Event That Woke Everything

**Track:** CS-production bridge | **Unit:** Queues, heaps, and scheduling

**Application:** fleet-repair | **Time box:** 45–60 minutes

## Incoming report

> Routine repairs wait even though workers are available. CPU and scheduling
> attempts rise whenever the inventory watcher publishes hardware metadata
> changes. The highest-priority repair requires hardware class `gpu-v9`, which
> the fleet does not currently contain.
>
> After its first failed attempt, that repair is recorded as blocked on
> `hardware/gpu-v9`. An update to CPU metadata causes it to appear in
> the active priority heap again. The priority order then selects it before the
> routine repairs; it fails for the same reason and returns to blocked state.
>
> Your task is limited to the rule that decides whether an inventory event
> moves a blocked repair back to the active heap. Do not lower the repair's
> priority, change the active ordering rule, invent the missing hardware, or
> suppress an event that actually creates `gpu-v9` capacity.

Before opening source, tests, or evidence, write down:

- what the priority rule is doing correctly;
- the condition that must change before the blocked repair can succeed;
- at least two possible explanations for attempts rising while completions do
  not;
- what evidence would distinguish duplicate queue entries, an incorrect
  readiness check, and a blocked repair returning to active work too often; and
- one rule that would reduce retries but could incorrectly leave a repair
  blocked after it becomes runnable.

Continue with [CANDIDATE.md](./CANDIDATE.md) after recording your first
explanation.
