# W01 Candidate Guide

## Write what the graph means

Before opening any packet, write:

- what one node represents;
- what `A -> B` means in this report;
- where the request is when progress stops; and
- at least two explanations for repeated stage entry.

For each explanation, name one observation that would make it more likely and
one that would make it less likely. Do not start with “the graph code is bad.”
Make each explanation specific enough that the evidence could show it is
wrong.

## Choose evidence

The [evidence index](./evidence/README.md) describes four packets without
showing their contents. Before opening one, write:

1. the question the packet should answer;
2. the results you expect under each remaining explanation; and
3. why this packet is more useful than the others now.

You do not need every packet.

## Define the required behavior

Before source inspection, be able to distinguish these inputs:

- a chain;
- two branches that share one stage;
- a closed loop;
- a stage that names itself; and
- two disconnected components, only one of which contains a loop.

A repair must accept the first two and check every disconnected part of the
graph. It must reject the last three invalid cases with a closed loop that
names the source locations. The report must be the same on every run. Input
order or Go map order must not choose a different error.

## Reproduce the failure

From the `cs-prod-bridge` directory, run the ordinary tests:

```bash
go test ./04-build-dependency-graphs/wheel/01-build-that-loops-back -count=1 -v
```

Then run the symptom tests:

```bash
go test -tags=csbridgewheel8 \
  ./04-build-dependency-graphs/wheel/01-build-that-loops-back \
  -count=1 -v
```

The ordinary tests protect valid chains and shared prerequisites. They also
check that both named stages exist. The tagged tests require a loop with
several edges and a self-dependency to produce useful errors.

After you identify the cause, change only `validator.go`. Run both commands
again. Finish with a three-minute handoff. Explain why the shared stage is
safe, why the reported loop is not, and which source details the error keeps.
Also explain why a recursion limit would only limit the damage, not prove that
the graph is valid.

Then read [DEBRIEF.md](./DEBRIEF.md).
