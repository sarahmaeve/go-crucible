# W03 Candidate Guide

## Draw logical names and individual records separately

Before opening a packet, write:

- the operation being planned;
- what one graph node must represent for that operation;
- what the service-to-database edge means; and
- why creation order and deletion order differ for that edge.

List at least two possible causes of the unsafe preview. Include one explanation
about ordering and one about how the graph was built. Name the evidence that
would tell the two explanations apart.

## Choose evidence

The [evidence index](./evidence/README.md) describes four packets. Before you
open one, write the question it should answer, the possible results, and why it
is the best next step. You do not need every packet.

## Keep the required behavior

A repair must:

- keep every snapshot record;
- permit a current and pending-deletion record to share one URN;
- reject two current records with one URN;
- connect the service declaration to the intended database record;
- produce dependent-first deletion order; and
- return the same graph when the same records arrive in a different order.

Do not change the test example to remove the older record. Do not reverse an
already incorrect edge only to get the expected list.

## Reproduce the failure

From the `cs-prod-bridge` directory, run:

```bash
go test ./04-build-dependency-graphs/wheel/03-two-resources-one-urn \
  -count=1 -v
```

Then run the symptom tests:

```bash
go test -tags=csbridgewheel10 \
  ./04-build-dependency-graphs/wheel/03-two-resources-one-urn \
  -count=1 -v
```

After you find the cause, change only `snapshot.go`. Run both commands again.
Finish with a short handoff. Name the incomplete identity and the edge that was
attached to the wrong record. Explain why deletion direction matters. Name one
check that makes this kind of invalid graph return an error.

Then read [DEBRIEF.md](./DEBRIEF.md).
