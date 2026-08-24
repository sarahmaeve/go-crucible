# W02: The Event That Woke Everything

**Track:** CS-production bridge | **Unit:** Queues, heaps, and scheduling

**Application:** fleet-repair | **Time box:** 45–60 minutes

## Incoming report

> Routine repairs wait even though workers are available. CPU use and
> scheduling attempts rise whenever the inventory watcher reports hardware
> metadata changes. The highest-priority repair needs hardware class `gpu-v9`,
> which the fleet does not have.
>
> After its first failure, the repair is blocked on `hardware/gpu-v9`. A CPU
> metadata update returns it to the active priority heap. The heap selects it
> before routine repairs. It fails for the same reason and becomes blocked
> again.
>
> Change only the rule that decides whether an inventory event returns a
> blocked repair to the active heap. Do not lower its priority, change the heap
> order, invent missing hardware, or ignore an event that creates `gpu-v9`
> capacity.

Before you open the source, tests, or evidence, write:

- what the priority rule is doing correctly;
- the condition that must change before the blocked repair can succeed;
- at least two possible explanations for attempts rising while completions do
  not;
- what evidence would distinguish duplicate queue entries, an incorrect
  readiness check, and a blocked repair returning to active work too often; and
- one rule that would reduce retries but might leave a repair blocked after it
  can run.

After you record your first explanation, continue with
[CANDIDATE.md](./CANDIDATE.md).
