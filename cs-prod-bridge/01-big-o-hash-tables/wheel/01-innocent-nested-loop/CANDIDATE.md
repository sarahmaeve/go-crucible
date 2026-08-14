# W01 Candidate Guide

## Boundary

You can change how this component turns a batch of flow events and a metadata
snapshot into enriched events. Do not change snapshot production, admission
limits, deployment size, or downstream storage.

Do not open `service.go`, either test file, or `DEBRIEF.md` until the evidence
points to this component.

## Choose evidence

The [evidence index](./evidence/README.md) lists available packets without
showing their contents. Before you open a packet, write:

1. the possible cause or input quantity that the packet can test;
2. the results that would make each possible cause more or less likely; and
3. why this packet is more useful now than the other packets.

You do not need every packet.

## When to open the code

Open the source only when you can answer these questions:

- Which input quantities affect this component?
- Does latency change with event count, metadata count, or both?
- Does the evidence point more strongly to memory allocation or CPU work?
- What complexity expression do you expect for this component?

Then run the ordinary tests for correct results:

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop -v
```

Run the test that reproduces the growth problem:

```bash
go test -tags=csbridgewheel1 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop \
  -run TestProductionScale -count=1 -v
```

Measure and profile the component:

```bash
go test -tags=csbridgewheel1 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop \
  -run '^$' -bench BenchmarkHandleBatch -benchmem

go test -tags=csbridgewheel1 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop \
  -run '^$' -bench BenchmarkHandleBatch -benchtime=2s \
  -cpuprofile /tmp/cs-bridge-wheel1.pprof
go tool pprof -top /tmp/cs-bridge-wheel1.pprof
```

Make the smallest change that improves the growth you calculated. Preserve the
existing behavior for missing and duplicate keys. Run the ordinary and tagged
tests again. Write a three-minute handoff before you read
[DEBRIEF.md](./DEBRIEF.md).
