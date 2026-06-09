# Diagnosis Exercise D02: Two Stacks, One Line

**Track:** Diagnosis | **Tier:** Intermediate | **Artifact:** race detector report | **Localizes:** Exercise 12

## Scenario

Nightly CI runs this repo's test suite under the race detector
(`go test -race -count=5 ./...`). For the past month a job has been
failing intermittently — two nights out of five — and the team has
been re-running it until it goes green. ("It's flaky," says the
channel topic.) Separately, an operator reports that concurrent audits
sometimes return fewer findings than the same audit run serially.

Tonight someone finally copied the failing job's output into the
incident tracker instead of clicking re-run. That excerpt is
[`ARTIFACT.txt`](./ARTIFACT.txt).

## Your task

1. Read [`ARTIFACT.txt`](./ARTIFACT.txt). **Do not open any source
   file yet.**
2. Fill in [`DIAGNOSIS_TEMPLATE.md`](./DIAGNOSIS_TEMPLATE.md): the
   signal, your file/line localization, the mechanism, and your
   proposed fix — before looking at the code.
3. Open the source and confirm or revise your diagnosis.
4. Apply your fix, then verify:

   ```bash
   go test -race -count=5 ./internal/audit/ -run TestExercise12 -v
   ```

5. Compare with [`DIAGNOSIS_NOTES.md`](./DIAGNOSIS_NOTES.md).

If you get stuck reading the artifact, [`HINTS.md`](./HINTS.md) has
progressive hints.

## What you will learn

- The anatomy of a race detector report: the two access stacks
  ("Write at … by goroutine N" / "Previous write at … by goroutine
  M"), what the shared address refers to, and the `created at` stacks
  that tell you where both goroutines came from.
- What it means when **both stacks point at the same line** — a
  function racing against another instance of itself — and why that
  is the most common shape for fan-out bugs.
- Why a data race produces *intermittent, wrong-count* symptoms
  rather than crashes, and why "the test is flaky" is so often
  spelled "the code is racy."
- How `created at` stacks let you distinguish *which* concurrent
  call-site is responsible when the racing function has several.
