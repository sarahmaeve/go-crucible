# W02: The Timestamp-Only Cursor

**Track:** CS-production bridge | **Unit:** Sequences, sorting, and ordered search  
**Application:** incident-history API | **Time box:** 40–50 minutes

## Incoming report

> During a regional failover, the incident-history UI omitted several alerts.
> Refreshing did not restore them, but opening an alert by its ID worked. The
> API returned HTTP 200 for every page and latency stayed normal. The omission
> appears only during bursts, when many alerts share the same millisecond
> timestamp. The client requests 50 results at a time and uses the timestamp
> of the last result as the cursor for its next request.
>
> Your task is limited to sorting and pagination in this endpoint. Do not
> change ingestion or UI rendering. Determine whether the current cursor can
> skip or repeat incidents when a page ends inside a group with the same
> timestamp. If it can, repair both the ordering and cursor.

Before opening the source or tests, write down:

- why HTTP success and normal latency do not show that pagination returned
  every incident;
- the exact meaning of “newest first” when timestamps are equal;
- what a timestamp-only cursor can say about unreturned incidents with the same
  timestamp;
- whether a stable sort supplies an order that another request or replica can
  reproduce; and
- the smallest test case that would demonstrate a skipped incident.

Continue with [CANDIDATE.md](./CANDIDATE.md) after recording your initial
explanation.
