# W01: The Lost Assignment Loop

**Track:** CS-production bridge | **Unit:** Queues, heaps, and scheduling

**Application:** repair-automation | **Time box:** 45–60 minutes

## Incoming report

> A routine deployment temporarily reduced the capacity of the service that
> assigns fleet-repair jobs. New repair events accumulated while the remaining
> assignment processes were full. The team restored capacity and reduced new
> arrivals, but the valid backlog did not drain.
>
> Workers are connected and repeatedly ask for work. Each affected worker holds
> one assignment reference and keeps trying to acquire it. Many referenced jobs
> no longer exist. These workers do not request a different job between
> attempts. New workers accept valid jobs normally.
>
> Your task is limited to the worker's response to an acquire result. Do not
> change queue ordering, assignment creation, capacity, or the rate of new work.
> Preserve retries for failures that can succeed later. Decide what a one-job,
> or **ephemeral**, worker should do when its assigned job is already gone.

Before you open the source, tests, or evidence, write:

- which event started the backlog and which later condition kept it stuck;
- why a connected worker can still provide no useful processing capacity;
- at least two explanations for a backlog that remains after capacity returns;
- which acquire results could change if the same worker tries again; and
- which acquire result means this worker can never complete its named
  assignment.

After you record your first explanation, continue with
[CANDIDATE.md](./CANDIDATE.md).
