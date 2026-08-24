# W02 Candidate Guide

## What must stay true

Change files in this Wheel only. Preserve these rules:

- incidents are presented newest first;
- following cursors through a fixed snapshot returns every incident once;
- no page exceeds the requested size;
- non-positive page sizes return an error; and
- `ListPage` does not modify the caller's slice.

Do not replace cursors with offsets. Do not increase the page size or timestamp
precision as a fix.

## Read the evidence before the code

1. In [Packet 1](./evidence/01.md), mark where the first page stops and identify
   the missing records.
2. In [Packet 2](./evidence/02.md), find how often timestamps tie in this
   workload.
3. In [Packet 3](./evidence/03.md), check whether two replicas get the same
   order from a stable sort.
4. In [Packet 4](./evidence/04.md), list the API rules and the fields already
   available in `Cursor`.
5. Then open `page.go` and its ordinary tests.

From the `cs-prod-bridge` module directory, run the ordinary tests:

```bash
go test ./02-sequences-sorting-search/wheel/02-timestamp-only-cursor -count=1
```

Then run the opt-in test for the timestamp tie:

```bash
go test -tags csbridgewheel4 \
  ./02-sequences-sorting-search/wheel/02-timestamp-only-cursor \
  -run TestPaginationIsCompleteAtTimestampBoundary -count=1
```

## Make and test the repair

After you inspect the evidence and code, write your diagnosis. Make the smallest
change that satisfies the tagged test while preserving the rules above. Keep
the public `Cursor` type unchanged. Add or strengthen a test for the failure.

Write a short handoff containing:

- how the old sort and cursor could skip records;
- why the ordinary tests did not expose the problem;
- the fields and directions used by the repaired sort;
- what the implementation assumes about snapshot changes between requests; and
- one metric or automated page-sequence test that would find the problem again.

After your tests pass, read [DEBRIEF.md](./DEBRIEF.md).
