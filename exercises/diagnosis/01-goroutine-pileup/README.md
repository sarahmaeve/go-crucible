# Diagnosis Exercise D01: The Pile-Up

**Track:** Diagnosis | **Tier:** Intermediate | **Artifact:** goroutine profile | **Localizes:** Exercise 06

## Scenario

The metrics-scheduler team runs a service that embeds this repo's
`ingest` package: it starts one metrics reader per scrape target and
cancels that target's context when the target is removed from the
config. After a fleet-wide config migration that churned targets for
a week, their dashboards show the scheduler's goroutine count at
~3,900 and climbing — it has never once gone down. Memory tracks the
goroutine count. Nothing unusual in the logs.

An SRE captured the Go runtime's goroutine profile from the service's
`net/http/pprof` endpoint — first the aggregated form (`?debug=1`),
then a full-stack sample (`?debug=2`) — and attached both to the
incident. That capture is [`ARTIFACT.txt`](./ARTIFACT.txt).

The scheduler's own code is not in this repository, and the SRE swears
the scheduler cancels every removed target's context. The suspicion is
in the library — your library.

## Your task

1. Read [`ARTIFACT.txt`](./ARTIFACT.txt). **Do not open any source
   file yet.**
2. Fill in [`DIAGNOSIS_TEMPLATE.md`](./DIAGNOSIS_TEMPLATE.md): the
   signal in the artifact, your file/line localization, the mechanism,
   and the fix you'd propose — before looking at the code.
3. Open the source and confirm or revise your diagnosis.
4. Apply your fix, then verify:

   ```bash
   make test-exercise N=06
   ```

5. Compare with [`DIAGNOSIS_NOTES.md`](./DIAGNOSIS_NOTES.md).

If you get stuck reading the artifact, [`HINTS.md`](./HINTS.md) has
progressive hints.

## What you will learn

- How to read the two forms of the pprof goroutine profile: the
  aggregated `debug=1` view (identical stacks bucketed with counts)
  and the full `debug=2` view (per-goroutine state, wait reason, and
  wait duration).
- The single most useful leak signature in Go: **many goroutines, one
  stack** — and how the bucket count, the wait reason, and the wait
  duration together tell you it's a leak rather than a burst.
- How `created by` frames let you walk from a parked goroutine back to
  the code that started it.
- Why "the caller cancels the context" is not enough — the goroutine
  has to be *listening*.
