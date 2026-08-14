# W02: The Timestamp-Only Cursor

**Track:** CS-production bridge | **Unit:** Sequences, sorting, and ordered search  
**Application:** incident-history API | **Time box:** 40–50 minutes

## Incoming report

> During a regional failover, the incident-history UI omitted several alerts.
> Refreshing did not restore them, but users could open each alert by its ID.
> The API returned HTTP 200 for every page, and latency stayed normal. The
> problem appears only during bursts, when many alerts share a millisecond
> timestamp. The client requests 50 results at a time. It uses the last result's
> timestamp as the cursor for the next request.
>
> You can change only sorting and pagination in this endpoint. Do not change
> ingestion or UI rendering. Determine whether the cursor can skip or repeat
> incidents when a page ends within a group that has one timestamp. If it can,
> repair the endpoint.

Before opening the source or tests, write down:

- why HTTP success and normal latency do not prove that every incident was
  returned;
- what “newest first” means when timestamps are equal;
- what the cursor says about unreturned incidents with the same timestamp;
- whether another request or replica can reproduce a stable sort; and
- the smallest test that could show a skipped incident.

After you write your initial explanation, continue with
[CANDIDATE.md](./CANDIDATE.md).
