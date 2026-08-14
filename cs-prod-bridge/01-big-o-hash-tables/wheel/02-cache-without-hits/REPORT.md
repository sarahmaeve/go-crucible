# W02: The Cache Without Hits

**Track:** CS-production bridge | **Unit:** Big-O and hash tables  
**Application:** endpoint-resolver | **Time box:** 35–45 minutes

## Incoming report

> The endpoint resolver's heap grows throughout the day. The metadata store
> receives almost one read for every observation. The resolver has a map-backed
> cache, and its lookups have expected `O(1)` cost. Each replica has only about
> 600 active endpoints. The incident lead thinks the Go 1.24 map implementation
> keeps deleted buckets. They recommend copying the map at regular intervals.
> No one has reported stale metadata.
>
> You are responsible for cache keys and lookups. Another team is responsible
> for expiration and metadata publication. Find out why this cache does not reduce
> reads from the metadata store.

Do not inspect source or tests yet. Record:

- facts about work and facts about memory
- what `O(1)` does not tell you about a cache
- at least two possible causes of both heap growth and store-read growth
- measurements that would show whether requests reuse keys

After you write your initial explanation, continue with
[CANDIDATE.md](./CANDIDATE.md).
