# Diagnosis Exercise D03: The Crash That Couldn't Happen

**Track:** Diagnosis | **Tier:** Advanced | **Artifact:** panic traceback | **Localizes:** Exercise 22

## Scenario

A teammate's one-off batch tool, `reprocess`, replays archived metric
samples through this repo's `worker.Pool` to backfill a corrected
aggregation. The pool's package documentation is explicit: a
processor function that panics on a malformed input is **caught,
logged, and surfaced as an error on that input's Result**; the rest
of the batch continues. The team relies on that contract — archived
data is exactly where malformed samples live.

Last night `reprocess` crashed 117 samples into a 2,000-sample batch.
The job runner captured stderr: a few normal progress lines, then a
panic traceback. That capture is [`ARTIFACT.txt`](./ARTIFACT.txt).

Two details from the team, both verified:

- The processor closure intentionally panics on malformed samples —
  that is the documented, supported way to reject one input.
- The pool's recovery is supposed to log `processor panicked` when it
  fires. The team grepped **all** historical logs, every environment,
  since the pool was adopted: zero hits. Ever.

This tier is harder than D01/D02 on purpose: the artifact alone will
not hand you the line. It will hand you a *contradiction* — and a
short list of hypotheses only one of which survives contact with the
source.

## Your task

1. Read [`ARTIFACT.txt`](./ARTIFACT.txt). **Do not open any source
   file yet.**
2. Fill in [`DIAGNOSIS_TEMPLATE.md`](./DIAGNOSIS_TEMPLATE.md). For
   this exercise the localization section asks for something subtler:
   state what the traceback *proves*, then enumerate the hypotheses
   that could explain it — ranked, with the evidence for each.
3. Open `internal/worker/pool.go` and eliminate hypotheses until one
   survives.
4. Apply your fix, then verify:

   ```bash
   go test ./internal/worker/ -v
   ```

   (All three tests in the package must pass — exercise 22's test and
   the two companion tests.)

5. Compare with [`DIAGNOSIS_NOTES.md`](./DIAGNOSIS_NOTES.md).

If you get stuck, [`HINTS.md`](./HINTS.md) escalates progressively.

## What you will learn

- How to read a Go panic traceback: the panic value, the
  `goroutine N [running]` header, the runtime `panic` frame, and the
  call chain beneath it — including which line each frame charges
  (the *call site* in that function, not its first line).
- What a traceback does **not** show: deferred functions that ran
  during unwinding and returned. Recovery that fails leaves no frames
  — its evidence is pure absence.
- Differential diagnosis from absence: "no `processor panicked` log
  line, ever" eliminates more hypotheses than anything visibly present
  in the artifact.
- The Go spec's precise condition on `recover()` — it returns nil
  unless called **directly by a deferred function** — and the
  refactoring pattern that silently violates it.
