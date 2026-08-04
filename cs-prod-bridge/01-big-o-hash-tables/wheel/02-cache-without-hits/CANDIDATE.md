# W02 Candidate Guide

## Boundary

You own how endpoint metadata is keyed and reused inside one resolver process.
The loader contract is that metadata changes are published by replacing the
resolver; time-based refresh and eviction are outside this scenario.

Until localization, do not open `cache.go`, either test file, or `DEBRIEF.md`.

## Evidence loop

Choose from the [evidence index](./evidence/README.md). Before reading a packet,
write the hypothesis it tests, the outcomes you expect, and what action each
outcome would suggest.

## Localization checkpoint

Open source only when you can state:

- the semantic identity of the metadata being cached
- the expected maximum useful key cardinality
- whether observed cardinality follows endpoints or observations
- why expected constant-time lookup can coexist with unbounded total memory

Run the normal behavior tests:

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits -v
```

Then reproduce the production-shaped contract:

```bash
go test -tags=csbridgewheel2 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits \
  -run TestRepeatedObservationsReuseEndpointMetadata -count=1 -v
```

Measure the defective form:

```bash
go test -tags=csbridgewheel2 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits \
  -run '^$' -bench BenchmarkResolveRepeatedEndpoints -benchmem
```

Make the smallest repair supported by the loader and cache-lifetime contracts.
Do not add a timer, arbitrary capacity, or periodic map copy. Verify both test
modes and write a handoff before reading [DEBRIEF.md](./DEBRIEF.md).

