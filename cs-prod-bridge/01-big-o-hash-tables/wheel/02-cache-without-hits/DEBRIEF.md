# W02 Debrief

Read this only after completing the handoff.

## Diagnosis

Expected constant-time map operations say nothing about hit ratio or the number
of keys retained. The semantic object being cached is endpoint metadata, whose
identity is the endpoint IP during one resolver lifetime. `cacheKey` also
contains observation time, which changes on almost every event.

As a result, distinct key count is `Theta(o)` for observations rather than
`Theta(p)` for active endpoint IPs. Each operation can remain expected `O(1)`
while total loader work and retained space grow linearly with observations.
Retries of exactly the same observation explain the few cache hits.

The Go 1.24 implementation theory does not fit the evidence: no deletion occurs,
and the map retains live entries reachable from the resolver.

## Bounded repair

Key the cache only by IP:

```go
cache map[string]Metadata
```

Look up and assign with `observation.IP`. Do not add timestamp expiry inside
this boundary; the loader contract says metadata is stable for the resolver's
lifetime and publication replaces the resolver.

The corrected workload has expected `Theta(o)` lookup time, at most `p` loader
calls, and `Theta(p)` retained cache space.

## Verification

Run:

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits -v
go test -tags=csbridgewheel2 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits \
  -count=1 -v
```

The ordinary tests retain error and enrichment semantics. The tagged check
uses repeated observations of stable endpoints and asserts both loader calls
and retained entries, avoiding a machine-specific heap or latency threshold.

## Production follow-up

Measure cache entries, requests, hits, misses, and meaningful domain
cardinality together. A map-length metric without endpoint cardinality would
show growth but not explain whether that growth was legitimate.
