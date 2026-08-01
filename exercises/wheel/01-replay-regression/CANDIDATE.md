# W01 Candidate Guide

## Boundary

You own the behavior between `Deduplicator` and its `CacheStore`. Other teams
own upstream retry policy, the store rollout, and incident coordination. You
may identify evidence outside your boundary, but do not redesign those systems.

Until the localization stage, do not open:

- `exercises/20-brittle-match/`
- `internal/ingest/dedup.go` or `internal/ingest/dedup_test.go`
- `solutions/20-brittle-match.patch`
- `.crucible/`
- The W01 debrief under `solutions/wheel/`

## Clarification and evidence

In facilitated mode, ask the facilitator questions naturally. In self-study
mode, use the [evidence index](./evidence/README.md). Before opening a packet,
write down:

1. The question or hypothesis you are investigating
2. Why the evidence could distinguish the current explanations
3. What different answers would imply

Open one packet at a time. You do not need every packet, and packet number is
not a recommended order.

## Localization checkpoint

Move to source only after you can state:

- The expected behavior at the deduplication boundary
- Whether the same logical metric is involved on first attempt and replay
- Whether the raw store result or the deduplicator's translation first violates
  the contract
- Which competing explanation is now least likely, and why

Then inspect `internal/ingest/dedup.go`. You may inspect the focused test after
you have written the expected behavior in your own words.

## Reproduce and repair

Run:

```bash
go test ./internal/ingest/ -run TestExercise20 -count=1 -v
```

Make the smallest repair justified by the evidence. Do not change either store
implementation or couple the deduplicator to Atlas-specific wording.
Your focused verification must also demonstrate that an unrelated store
failure still reaches the caller; duplicate handling is not permission to
discard every error.

Verify the focused behavior, then the package:

```bash
go test ./internal/ingest/ -run TestExercise20 -count=1 -v
go test ./internal/ingest/ -skip '^TestExercise' -count=1
```

## Handoff

Give a three-minute summary covering:

- The reported impact and confirmed scope
- The evidence locating the first contract violation
- The repair and verification
- Whether the store adapter violated its contract
- One delivery or test improvement that would prevent recurrence

Only then compare your work with the
[W01 debrief](../../../solutions/wheel/01-replay-regression.md).
