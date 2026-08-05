# W01: The Logarithmic Insert

**Track:** CS-production bridge | **Unit:** Sequences, sorting, and ordered search  
**Application:** config-publisher | **Time box:** 40–50 minutes

## Incoming report

> Configuration publishers in the new region take 18 seconds to become ready
> after receiving their first snapshot. During refresh, one CPU core is busy;
> afterward, lookup latency is normal. The snapshot is kept sorted and every
> incoming key is located with binary search, so the owner expects refresh
> work to be logarithmic. The new region sends a larger initial delta than
> established regions. No missing or incorrectly ordered records have been
> reported, and no memory alert fired during the refresh.
>
> Your task is limited to the code that builds the new in-process snapshot. Do
> not change how updates arrive, when the publisher reports ready, or how
> queries read the completed snapshot. Use the evidence to decide whether
> snapshot construction causes the delay, and change that code only if it does.

Before opening the source or tests, write down:

- which statements are observations and which are explanations offered by the
  report;
- separate variables for current snapshot size and incoming batch size;
- the work binary search performs and the work insertion may still perform
  afterward;
- at least two explanations consistent with busy CPU and no reported memory
  alert;
  and
- the first measurement you would use to distinguish expensive comparisons
  from copying records.

Continue with [CANDIDATE.md](./CANDIDATE.md) after recording your initial
explanation.
