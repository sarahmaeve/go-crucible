# W01: The Logarithmic Insert

**Track:** CS-production bridge | **Unit:** Sequences, sorting, and ordered search  
**Application:** config-publisher | **Time box:** 40–50 minutes

## Incoming report

> Configuration publishers in the new region take 18 seconds to become ready
> after the first snapshot arrives. One CPU core stays busy during refresh.
> Lookup latency is normal after refresh. The snapshot stays sorted, and binary
> search finds the position for every incoming key. The owner therefore expects
> refresh work to be logarithmic. The new region sends a larger initial batch
> than established regions. No one has reported missing or incorrectly ordered
> records, and no memory alert occurred during refresh.
>
> You can change only the code that builds the new in-process snapshot. Do not
> change update delivery, readiness, or how queries read the completed snapshot.
> Use evidence to determine whether snapshot construction causes the delay.
> Change that code only if it does.

Before opening the source or tests, write down:

- facts in the report and explanations that do not yet have evidence;
- separate variables for current snapshot size and incoming batch size;
- the work done to find a position and the work that can happen afterward;
- at least two possible causes of busy CPU without a memory alert; and
- the first measurement that could distinguish expensive comparisons from
  record movement.

After you write your initial explanation, continue with
[CANDIDATE.md](./CANDIDATE.md).
