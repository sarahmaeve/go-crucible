# Unit 05: Bloom Filters and Approximate Membership

Read the lesson source here, or run `make bridge-serve` from the repository root
for the fully rendered site:

**[Open the Bloom filters guide](../content/05-bloom-filters/_index.md)**

The lesson begins with Cassandra SSTable lookups, transfers the same negative
lookup contract to VictoriaLogs, and uses Git and Go Ethereum to examine
persistence and saturation. The Go section defines a narrow `MayContain`
contract and explains why an ephemeral `hash/maphash` seed is not a persistent
filter format.

The runnable [Skip the Cold Segment lab](./lab/) places one process-local Bloom
filter in front of each immutable segment's exact sorted index. It exposes
filter probes, exact checks avoided, false positives, density, planned
capacity, modeled and observed false-positive rates, and over-capacity behavior
before adding lookup and rebuild benchmarks. Its executable examples, fuzz
properties, concurrent-read test, worksheet, and repeatable experiment runner
compare workload mix, sizing targets, capacity load, exact-check cost, and
uniform versus hot absent keys.

The report-first [Filter from Yesterday Wheel](./wheel/) turns the generation
ownership rule into a deterministic incident investigation. Its ordinary tests
preserve healthy skipping and exact fallback; the `csbridgewheel12` build tag
reproduces a newly published key being skipped through the previous
generation's filter.
