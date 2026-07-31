# W01 Facilitator Truth Table — Replay Regression

This file contains scenario spoilers. Candidate-facing material lives under
`exercises/wheel/01-replay-regression/`.

## Ground truth

`Deduplicator.Ingest` classifies duplicate errors by searching `err.Error()`
for the legacy phrase `already recorded`. Both stores conform to the interface
contract and wrap `types.ErrDuplicate`, but Atlas uses `conflict on write` in
its explanatory text. The local classifier therefore absorbs the legacy replay
and leaks the Atlas replay to its caller.

The restrained repair is to classify the error chain with
`errors.Is(err, types.ErrDuplicate)`. Neither store, the key function, nor the
retry policy needs to change.

## Why it reached production

First writes work through both stores, and replay behavior works through the
legacy implementation whose wording accidentally satisfies the classifier.
The latent coupling activates only when a retry reaches a conforming store with
different diagnostic prose. A happy-path test or a replay test using only the
legacy fake cannot expose it.

## Initially plausible hypotheses

1. Atlas does not return the shared duplicate sentinel.
2. The deduplication key is unstable across attempts.
3. Upstream retries represent distinct operations incorrectly treated as one.
4. Atlas returns a conforming result which the deduplicator classifies using an
   undocumented property.
5. The deduplicator absorbs the replay but a downstream component later fails.

## Question map

| Candidate intent | Answer or packet |
|---|---|
| Establish impact, frequency, first attempt versus replay | Evidence 01 |
| Compare versions, configuration, rollout, or recent changes | Evidence 02 |
| Compare operation identifiers, metric fields, or dedup keys | Evidence 03 |
| Ask what either interface promises | Evidence 04 |
| Compare raw and translated errors across stores | Evidence 05 |
| Request the smallest local reproduction | Evidence 06 |
| Ask whether Atlas writes duplicates downstream | No evidence of duplicate publication; the observed failure is an error returned before publication. |
| Ask whether rollback is authorized | The migration team owns mitigation. The candidate may recommend it but should continue the bounded diagnosis. |
| Ask for logs from unrelated ingestion stages | Available logs show the same returned error but add no discriminating information. |

Answer equivalent questions equivalently; exact phrasing is never required.
Do not volunteer the next packet.

## Optional injects

- If the learner commits to “Atlas is broken” before comparing contracts, say
  that the adapter team has produced a conformance test showing its error
  matches the shared sentinel, then ask what local evidence would challenge or
  preserve the learner's theory.
- If the learner patches the Atlas message, reveal that a third conforming
  store uses `unique constraint violation` and ask whether the repair preserves
  the interface boundary.
- If the learner repairs by swallowing every store error, run a neighboring
  case in which `Put` returns an availability error; it must still propagate.

## Completion criteria

- Separates rollout correlation from proof of adapter fault
- Establishes identity and boundary contract before repair
- Locates the first divergence in the deduplicator
- Uses semantic error-chain classification without swallowing unrelated errors
- Runs the focused reproduction and package regression scope
- Reports that Atlas conforms to the stated contract
- Recommends contract-level multi-implementation testing as prevention
