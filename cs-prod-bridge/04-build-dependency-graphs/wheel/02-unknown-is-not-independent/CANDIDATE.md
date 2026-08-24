# W02 Candidate Guide

## An empty list can mean different things

Before opening a packet, distinguish the two analyzer outcomes in the report
that both return no dependency IDs: analysis completed and found none, and
analysis did not reach a conclusion. For each outcome, answer:

- what evidence would support it;
- whether the evidence supports running the work at the same time; and
- what message, if any, an operator should receive.

Then state what one node and one edge represent in this local planner. Also
explain why a missing target is a separate error rather than a third meaning
for an empty dependency slice. Do not conclude “there is no edge” until you
can name the evidence that would show it.

## Choose evidence

The [evidence index](./evidence/README.md) describes the available packets.
Before opening one, write the question it should answer and the different
results expected under your remaining explanations. You do not need every
packet.

## Keep the three results separate

A repair must keep these outcomes distinct:

- analysis completed and found no dependencies: ready;
- analysis completed and found an unfinished dependency: blocked; and
- analysis did not determine the relationship: blocked with an explanation.
  Here, **undetermined** means that the analyzer did not reach a conclusion.

The output order must stay the same on every run. Repeated dependency names
must still produce one edge. Do not disable all work that can safely run at the
same time, and do not add a dependency that the analyzer did not find.

## Reproduce the failure

From the `cs-prod-bridge` directory, run:

```bash
go test ./04-build-dependency-graphs/wheel/02-unknown-is-not-independent \
  -count=1 -v
```

Then enable the symptom tests:

```bash
go test -tags=csbridgewheel9 \
  ./04-build-dependency-graphs/wheel/02-unknown-is-not-independent \
  -count=1 -v
```

After you diagnose the failure, change only `planner.go`. Run both commands
again. Finish with a short handoff. Explain which state lost information, why
a known empty result is safe here, and why an undetermined result is not. Also
explain how an operator can tell that the planner held the work back on
purpose.

Then read [DEBRIEF.md](./DEBRIEF.md).
