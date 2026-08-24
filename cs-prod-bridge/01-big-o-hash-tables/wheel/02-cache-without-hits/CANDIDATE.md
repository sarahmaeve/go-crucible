# W02 Candidate Guide

## Boundary

You can change how one resolver process keys and reuses endpoint metadata. The
loader promises that a metadata update replaces the resolver. Do not add
time-based refresh or eviction in this scenario.

Do not open `cache.go`, either test file, or `DEBRIEF.md` until the evidence
points to this component.

## Choose evidence

Choose from the [evidence index](./evidence/README.md). Before you read a packet,
write the possible cause it can test. Also write the results you expect and
what you would do after each result.

## When to open the code

Open the source only when you can answer these questions:

- What information identifies one cached metadata item?
- What is the largest useful number of cache keys?
- Does the number of observed keys follow endpoint count or observation count?
- How can each lookup have expected constant cost while total memory continues
  to grow?

Run the ordinary tests for correct behavior:

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits -v
```

Then run the test that reproduces the production behavior:

```bash
go test -tags=csbridgewheel2 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits \
  -run TestRepeatedObservationsReuseEndpointMetadata -count=1 -v
```

Measure the current implementation:

```bash
go test -tags=csbridgewheel2 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits \
  -run '^$' -bench BenchmarkResolveRepeatedEndpoints -benchmem
```

Make the smallest repair permitted by the loader and cache-lifetime rules. Do
not add a timer, an arbitrary capacity, or a periodic map copy. Run both test
modes again. Write a handoff before you read [DEBRIEF.md](./DEBRIEF.md).
