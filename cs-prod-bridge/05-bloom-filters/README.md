# Unit 05: Bloom Filters and Approximate Membership

Open the HTML guide for the foundations lesson, production examples, formulas,
and source notes:

**[Open the Bloom filters guide](../public/05-bloom-filters/index.html)**

The lesson begins with Cassandra SSTable lookups, transfers the same negative
lookup contract to VictoriaLogs, and uses Git and Go Ethereum to examine
persistence and saturation. The Go section defines a narrow `MayContain`
contract and explains why an ephemeral `hash/maphash` seed is not a persistent
filter format.

The exploration lab and Wheel scenarios are designed in the
[Unit 05 research record](../../docs/cs-prod-bridge-unit-05.md) but are not yet
implemented. This directory will hold their runnable Go code when they are
added.
