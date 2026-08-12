# W01: The Lost Assignment Loop

**Track:** CS-production bridge | **Unit:** Queues, heaps, and scheduling

**Application:** repair-automation | **Time box:** 45–60 minutes

## Incoming report

> A routine deployment temporarily reduced the capacity of the service that
> assigns fleet-repair jobs to workers. Incoming repair events accumulated
> while the remaining assignment processes were saturated. The team restored
> that service's capacity and reduced new arrivals, but the valid-work backlog
> did not drain.
>
> Worker processes are connected and repeatedly ask for work. Each affected
> worker holds one assignment reference and keeps trying to acquire it. The
> assignment service reports that many of those referenced jobs no longer
> exist. Those workers do not request a different job between attempts.
> Newly started workers accept valid jobs normally.
>
> Your task is limited to the worker's response to an acquire result. Do not
> change queue ordering, assignment creation, capacity, or arrival throttling.
> Preserve retries for failures that can succeed on a later attempt. Decide
> what an ephemeral worker should do when its assigned job is already gone.

Before opening source, tests, or evidence, write down:

- which part of the report describes the trigger and which part describes why
  recovery remained stuck;
- why a connected worker can still provide no useful processing capacity;
- at least two explanations for a backlog that remains after capacity returns;
- which acquire results could change if the same worker tries again; and
- which acquire result means that this worker's assignment cannot be
  completed at all.

Continue with [CANDIDATE.md](./CANDIDATE.md) after recording your first
explanation.
