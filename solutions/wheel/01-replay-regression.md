# W01 Debrief: Replay Regression

Read this only after completing the scenario.

## What the report established

The report established a retry-related error increase correlated with the
Atlas rollout. It did not establish that Atlas violated the `CacheStore`
contract. “The adapter rejects duplicate keys differently” described visible
variation; “therefore it does not support replay behavior” was the reporter's
theory.

## High-value evidence

The most discriminating path was:

1. Confirm failures were limited to replays while first writes succeeded.
2. Confirm the same logical metric produced the same deduplication key.
3. Establish that store implementations may use different prose but must return
   an error matching `types.ErrDuplicate`.
4. Compare raw store results with caller-visible deduplicator results.

That comparison located the first contract violation in the translation layer,
not in Atlas, the key function, or downstream publication.

## Root cause and repair

`Deduplicator.Ingest` inspected the rendered error text for the legacy phrase
`already recorded`. Atlas wrapped the same sentinel with different diagnostic
wording, so the classifier leaked a conforming duplicate result to its caller.

The narrow repair is:

```go
if errors.Is(err, types.ErrDuplicate) {
	return nil
}
```

This preserves unrelated errors and honors the existing interface contract.
Changing Atlas's message would merely move the undocumented coupling into the
adapter and fail again when another implementation chose different prose.

## Verification

The focused table must pass for both stores, followed by the non-exercise
package tests:

```bash
go test ./internal/ingest/ -run TestExercise20 -count=1 -v
go test ./internal/ingest/ -skip '^TestExercise' -count=1
```

The existing canonical patch is `solutions/20-brittle-match.patch`.

## Example handoff

> Replay errors increased when Atlas traffic encountered the retry burst. The
> adapter is conforming: both stores return errors matching ErrDuplicate, and
> the same metric produces the same key on both attempts. The first divergence
> is in Deduplicator.Ingest, which recognized only the legacy store's message
> wording. I changed classification to use the shared error identity, verified
> both store cases, and ran the package regression tests. We should retain a
> contract test that runs every CacheStore implementation through the same
> first-write and replay cases; rollback is not required for this local defect.

## Other valid routes

A learner might begin with key identity, interface conformance, or a direct
legacy-versus-Atlas reproduction. Any route is sound if it distinguishes the
alternatives before committing to the repair. Reading the source first and
guessing the intended API can find the same line, but it does not demonstrate
the investigation skill this track is designed to train.
