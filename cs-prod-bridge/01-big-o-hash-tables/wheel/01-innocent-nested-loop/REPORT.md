# W01: The Innocent Nested Loop

**Track:** CS-production bridge | **Unit:** Big-O and hash tables  
**Application:** flow-enricher | **Time box:** 35–45 minutes

## Incoming report

> Flow enrichment is saturating one core and exceeding its 500 ms batch SLO in
> the new shared-services region. This region has many more registered endpoints
> than the original regions. Its event count is still below the configured batch
> limit. The service owner thinks recent Go map changes increased garbage
> collection work and proposes doubling the current memory limit. No one has
> reported errors or missing enrichments.
>
> You are responsible for in-process enrichment. Find out whether this component
> explains the change at scale. If the evidence supports a fix within this
> component, make the smallest such fix.

Do not inspect source or tests yet. Record:

- facts reported by the team
- claims that do not yet have evidence
- at least two possible causes that fit the report
- input quantities that you must measure separately
- the first evidence that could help you choose between the possible causes

After you write your initial explanation, continue with
[CANDIDATE.md](./CANDIDATE.md).
