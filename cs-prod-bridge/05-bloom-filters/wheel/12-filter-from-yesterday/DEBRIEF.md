# W12 Debrief

Read this only after completing the handoff.

## Cause

In the runnable exercise, `prepareSnapshot` built a matching replacement: exact
data generation 2 and a filter built from generation 2's keys. `Reload` then
copied the active snapshot and replaced only its `data` field:

```go
next := *current
next.data = prepared.data
c.current.Store(&next)
```

The pointer update happened atomically, but the stored snapshot was internally
mismatched. It combined new exact data with the previous filter. The old filter
could therefore report a newly added key as definitely absent, causing the
read path to skip the exact index that contained it.

This is not an ordinary result from a Bloom filter. The guarantee that there
are no false negatives applies only when every live key was added to the
filter being queried. Here, the read path queried a filter built for the
previous generation.

## Smallest repair

Publish the complete prepared snapshot instead of reconstructing it as a
partial update:

```go
c.current.Store(prepared)
```

Both components are immutable after preparation. Storing their shared snapshot
with one atomic pointer update means each reader sees either the complete old
snapshot or the complete new one. The reload mutex ensures that only one reload
chooses and publishes a new generation at a time. If record validation or
filter construction fails, `prepareSnapshot` returns before the store, so the
previous pair remains active.

A larger production system could instead reject mismatched metadata or ignore
an untrusted filter and perform exact checks. It must not trust a “definitely
absent” answer from a filter that may not include every live key.

## Check the repair

Run:

```bash
go test ./05-bloom-filters/wheel/12-filter-from-yesterday -count=1 -v
go test -tags=csbridgewheel12 \
  ./05-bloom-filters/wheel/12-filter-from-yesterday \
  -count=1 -v
```

The ordinary tests verify that a possible match leads to an exact check, a
healthy negative can skip that check, duplicate keys are rejected, and a
failed reload leaves the old snapshot active. The tagged tests require matching
generation numbers and make a new-generation key discoverable. Their
controlled membership implementation removes false positives from a test of
whether the filter and data stay paired.

## What the production source shows

The [Cassandra storage-engine documentation](https://cassandra.apache.org/doc/latest/cassandra/architecture/storage-engine.html)
describes immutable SSTables, their component files, and the creation of new
SSTables during compaction. Each SSTable has a Bloom filter for that SSTable's
partition keys. That supports treating the exact data and its filter as one
unit whose parts must stay together.

It does not show the local bug, data types, snapshot update, route records, or
generation trace. Those details are invented to create a small, runnable
example of the requirement that a filter stay with its data.

## What to measure in production

Record the data generation, filter generation, format version, and build
result together. Count cases in which the system ignores a filter because its
generation label is missing or does not match. You can also sample “definitely
absent” answers and check those keys directly against the exact index.
Ordinary reads cannot detect this problem because they skip the exact index
after such an answer.

Alert on any generation mismatch or any sampled key that the filter rejects
but the exact index contains. Do not count these correctness failures as part
of the ordinary false-positive rate.
