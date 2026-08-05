# W02 Candidate Workspace

## What must stay true

You may change files in this Wheel only. Preserve the following behavior:

- incidents are presented newest first;
- following cursors through a fixed snapshot returns every incident once;
- no page exceeds the requested size;
- non-positive page sizes return an error; and
- `ListPage` does not modify the caller's slice.

Do not replace cursor pagination with offsets. Do not solve the incident by
increasing the page size or timestamp precision.

## Read the evidence in this order

1. Use [Packet 1](./evidence/01.md) to mark where the first page stops and
   identify the missing records.
2. Use [Packet 2](./evidence/02.md) to determine how common tied timestamps are
   in this workload.
3. Use [Packet 3](./evidence/03.md) to check whether stable sorting gives two
   replicas the same order.
4. Use [Packet 4](./evidence/04.md) to list the API requirements and the fields
   already available in `Cursor`.
5. Then inspect `page.go` and its ordinary tests.

From the `cs-prod-bridge` module directory, confirm that the ordinary tests
pass:

```bash
go test ./02-sequences-sorting-search/wheel/02-timestamp-only-cursor -count=1
```

Then run the opt-in test that covers the timestamp tie:

```bash
go test -tags csbridgewheel4 \
  ./02-sequences-sorting-search/wheel/02-timestamp-only-cursor \
  -run TestPaginationIsCompleteAtTimestampBoundary -count=1
```

## Repair the ordering and cursor

Choose enough fields to give every incident a unique position in the sorted
result. Use the same field order when sorting incidents, deciding which
incident comes after a cursor, and constructing the next cursor. Keep the
public `Cursor` type unchanged, and add or strengthen a regression test.

Write a short handoff containing:

- how the old sort and cursor skipped records;
- why the ordinary tests did not expose the problem;
- the fields and directions used by the repaired sort;
- what the implementation assumes about changes to the snapshot between page
  requests; and
- one metric or automated traversal test that would catch the problem again.

Read [DEBRIEF.md](./DEBRIEF.md) only after your tests pass.
