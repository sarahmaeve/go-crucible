# W03: The Sibling Stampede

**Track:** CS-production bridge | **Unit:** Queues, heaps, and scheduling

**Application:** maintenance-coordinator | **Time box:** 45–60 minutes

## Incoming report

> A database-maintenance job contains repairs for several machines. It starts
> only when the whole group has enough capacity. The group waits correctly when
> capacity is scarce. After an update reports enough capacity, CPU use and log
> volume spike. Unrelated one-machine repairs stop advancing.
>
> The coordinator then visits every group member once. The code calls this pass
> a **reconciliation round**. Each member sees that the group is ready and asks
> the activation queue to add every other member. The queue stores each repair
> ID at most once, but logs every repeated request.
>
> Change only the decision to begin group-wide activation during one pass. Do
> not remove group coordination, ignore the capacity update, or disable the
> queue's duplicate check. Every required sibling must still become eligible.

Here, *activation* means asking the queue to reconsider a member; it does not
mean that the member has completed its repair.

Before you open the source, tests, or evidence, write:

- why a queue that retains at most `g` IDs can still receive more than `g`
  requests;
- the number of requests produced when each of `g` members requests its other
  `g-1` siblings;
- which work happens before the queue discovers that an ID is already active;
- at least two other explanations for high CPU and log volume; and
- which facts must last for one round and which must reset before a later round.

After you record your first explanation, continue with
[CANDIDATE.md](./CANDIDATE.md).
