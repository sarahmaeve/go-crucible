# W01: Replay Regression

**Track:** Wheel of Misfortune | **Tier:** Intermediate  
**Application:** pipeline | **Time box:** 35–45 minutes

## Incoming report

> Metric ingestion began returning errors during this morning's upstream retry
> burst. The first reports arrived shortly after the Atlas cache-store rollout.
> The storage on-call says the new adapter rejects duplicate keys differently
> and probably does not support our replay behavior. The migration team is
> evaluating a rollback.
>
> The larger ingestion incident is being coordinated elsewhere. You own the
> pipeline deduplication boundary. Determine whether that component can explain
> the reported replay failures, repair any confirmed local defect, and report
> what you know.

Do not inspect source or tests yet.

Start a copy of the shared
[worksheet](../WORKSHEET.template.md), then record:

- What the report actually establishes
- Which statement is a causal theory rather than an observation
- At least two plausible explanations
- The two or three missing facts with the highest diagnostic value

Continue with [CANDIDATE.md](./CANDIDATE.md) when your initial model is written.
