# W02 Debrief

Read this only after completing the handoff.

## Cause

`cacheKey` contains the observation time. That time changes on almost every
event, so repeated observations of one endpoint usually create different keys.

The cache stores endpoint metadata. During one resolver lifetime, the endpoint
IP identifies that metadata. Expected constant-time map operations say nothing
about the cache hit ratio or the total number of keys in memory.

Let `o` be the number of observations and `p` be the number of active endpoint
IPs. The number of distinct keys is `Theta(o)`, not `Theta(p)`. Each lookup can
still have expected `O(1)` cost while loader work and memory grow with `o`.
Retries of the same observation cause the few cache hits.

The evidence does not support the theory about deleted Go map buckets. This
code does not delete entries. The map contains live entries that the resolver
can still reach.

## Smallest repair

Key the cache only by IP:

```go
cache map[string]Metadata
```

Use `observation.IP` for lookup and assignment. Do not add time-based expiration
inside this component. The loader promises that metadata stays unchanged during
the resolver's lifetime. Publication replaces the complete resolver.

After the repair, all `o` observations take expected `Theta(o)` lookup time in
total. The resolver makes at most `p` loader calls and keeps `Theta(p)` cache
entries.

## Check the repair

Run:

```bash
go test ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits -v
go test -tags=csbridgewheel2 \
  ./cs-prod-bridge/01-big-o-hash-tables/wheel/02-cache-without-hits \
  -count=1 -v
```

The ordinary tests check error handling and enriched results. The tagged test
uses repeated observations of stable endpoints. It checks loader calls and map
entries without using a machine-specific memory or latency limit.

## What to measure in production

Measure cache entries, requests, hits, misses, and active endpoints together.
Map length alone shows growth, but it does not show whether that growth is
correct.
