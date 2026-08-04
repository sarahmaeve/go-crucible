# W02: The Cache Without Hits

**Track:** CS-production bridge | **Unit:** Big-O and hash tables  
**Application:** endpoint-resolver | **Time box:** 35–45 minutes

## Incoming report

> The endpoint resolver's heap grows throughout the day and the metadata store
> is receiving nearly one read per observation. The resolver has a map-backed
> cache, lookups are expected `O(1)`, and there are only about 600 active
> endpoints per replica. The incident lead suspects the Go 1.24 map
> implementation retains deleted buckets and recommends periodically copying
> the map. No stale metadata has been reported.
>
> You own cache keying and lookup. Expiry policy and metadata publication are
> owned by another team. Determine why this cache is not reducing store load.

Do not inspect source or tests yet. Record:

- which facts concern time complexity and which concern space
- what `O(1)` does not tell you about a cache
- at least two explanations for simultaneous heap and store-read growth
- which measurements would reveal whether keys are being reused

Continue with [CANDIDATE.md](./CANDIDATE.md) after writing an initial model.

