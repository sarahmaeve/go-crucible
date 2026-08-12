# W03: The Sibling Stampede

**Track:** CS-production bridge | **Unit:** Queues, heaps, and scheduling

**Application:** maintenance-coordinator | **Time box:** 45–60 minutes

## Incoming report

> A database-maintenance job consists of repairs on several machines and starts
> only when capacity is available for the whole group. When capacity is scarce,
> the group waits as intended. After a later update reports enough capacity to
> try the group again, CPU and log volume spike and unrelated one-machine
> repairs stop advancing.
>
> The coordinator then makes one pass over every group member. The code calls
> this pass a *reconciliation round*. Each member sees that the group is ready
> and asks the activation queue to add every other member. The queue retains at
> most one entry for each repair ID, but it logs every request for an ID that is
> already active.
>
> Your task is limited to deciding which member examined after the group becomes
> ready may initiate the group-wide activation during one pass. Do not remove
> group coordination, suppress the capacity update, or disable the queue's
> duplicate check. At least one member must still make every sibling eligible.

Here, *activation* means asking the queue to reconsider a member; it does not
mean that the member has completed its repair.

Before opening source, tests, or evidence, write down:

- why a queue that retains at most `g` IDs can still receive more than `g`
  requests;
- the number of requests produced when each of `g` members requests its other
  `g-1` siblings;
- what work occurs before the queue discovers that an ID is already active;
- at least two other explanations for high CPU and log volume; and
- how long the coordinator should remember “group-wide activation already
  issued,” and when it must forget that fact.

Continue with [CANDIDATE.md](./CANDIDATE.md) after recording your first
explanation.
