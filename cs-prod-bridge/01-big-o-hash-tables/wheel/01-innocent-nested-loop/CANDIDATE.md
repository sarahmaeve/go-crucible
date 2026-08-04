# W01 Candidate Guide

## Boundary

You own conversion of a batch of flow events plus a metadata snapshot into
enriched events. Snapshot production, admission limits, deployment sizing, and
downstream storage are outside your repair boundary.

Until localization, do not open `service.go`, either test file, or `DEBRIEF.md`.

## Evidence loop

The [evidence index](./evidence/README.md) lists available packets without
giving their contents. Before opening one, write:

1. which hypothesis or workload variable it tests
2. which outcomes would strengthen or weaken the current explanations
3. why it is more useful now than the other packets

You do not need every packet.

## Localization checkpoint

Open source only when you can state:

- the relevant workload variables
- whether latency tracks event count, metadata count, or both
- whether allocation or CPU work is the stronger current explanation
- an expected complexity expression for the boundary

Then run the ordinary correctness checks:

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop -v
```

Reproduce the scaling contract:

```bash
go test -tags=csbridgewheel1 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/01-innocent-nested-loop \
  -run TestProductionScale -count=1 -v
```

For measurement and profiling:

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

Make the smallest change that alters the derived growth while preserving
missing-key and duplicate-key behavior. Verify ordinary and tagged tests, then
write a three-minute handoff before reading [DEBRIEF.md](./DEBRIEF.md).

