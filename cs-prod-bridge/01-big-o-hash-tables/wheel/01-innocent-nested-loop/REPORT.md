# W01: The Innocent Nested Loop

**Track:** CS-production bridge | **Unit:** Big-O and hash tables  
**Application:** flow-enricher | **Time box:** 35–45 minutes

## Incoming report

> Flow enrichment is saturating one core and exceeding its 500 ms batch SLO in
> the new shared-services region. The region has many more registered endpoints
> than the original regions, but batch event count is within its configured
> limit. The service owner thinks recent Go map changes increased GC cost and
> proposes doubling the memory limit. No errors or missing enrichments have
> been reported.
>
> You own the in-process enrichment boundary. Determine whether it explains the
> scaling behavior and make a bounded repair if the evidence supports one.

Do not inspect source or tests yet. Record:

- what is observed versus claimed
- at least two explanations consistent with the report
- the workload dimensions you need separately
- the first evidence that would distinguish your explanations

Continue with [CANDIDATE.md](./CANDIDATE.md) after writing your initial model.

