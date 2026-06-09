# Diagnosis Track

A parallel track of exercises focused on **reading diagnostic
artifacts** rather than reading code. Where the numbered exercises
hand you a failing test and the review exercises hand you a diff, a
diagnosis exercise hands you what on-call actually hands you: a
goroutine dump, a race detector report, a crash traceback — captured
from a running system — and asks you to localize the bug from the
artifact *before you open the source*.

## Why a separate track

The crucible's other tracks train you to recognise a trap when you're
looking at the code that contains it. In production, you rarely start
at the code. You start at a symptom plus an artifact, and the first —
often hardest — half of the job is getting from the artifact to the
file and line. The Go toolchain is unusually generous with diagnostics
(`net/http/pprof`, the race detector, panic tracebacks with full
stacks); engineers who can *read* those outputs fluently localize bugs
in minutes that take others days. That reading skill is what this
track trains.

Each diagnosis exercise reuses a numbered exercise's planted bug as
its ground truth. You are not learning a new bug — you are learning a
new entry path to it. If you've already solved the underlying
exercise, this track tests whether you can work the same problem
backwards from the evidence.

## How diagnosis exercises work

1. Each exercise has a `README.md` with an incident-style scenario and
   an `ARTIFACT.txt` — the diagnostic output, captured exactly as a
   real tool emits it.
2. **Read the artifact first.** Do not open any source file yet.
3. Fill in `DIAGNOSIS_TEMPLATE.md`: what in the artifact is the
   signal, your file/line localization, the mechanism you believe is
   at work, and the fix you'd propose — plus how confident you are
   and what would confirm it.
4. *Then* open the source, confirm or revise, apply your fix, and
   verify with the linked numbered exercise's test
   (`make test-exercise N=NN`).
5. Compare your diagnosis against `DIAGNOSIS_NOTES.md` — a canonical
   walkthrough of how an experienced engineer reads that artifact,
   section by section.

The discipline in step 3 is the exercise. Writing the diagnosis down
*before* looking is what turns artifact-reading from vague pattern
matching into a calibrated skill — it's also exactly what a good
incident channel post looks like.

## A note on the artifacts

The artifacts are curated captures: hostnames, binary names, and
memory addresses are illustrative, but every frame that points into
this repository points at the **real file and line at the current
commit**. (Maintainers: that makes artifacts line-number-sensitive —
see the source-of-truth rules in `.crucible/README.md`.) Part of the
skill being trained is triaging frames you own from frames you don't:
runtime internals and other people's binaries appear in the artifacts
just as they do in real dumps.

## When to attempt a diagnosis exercise

Each exercise names the numbered exercise whose bug it captures.
You can attempt it **before** the numbered exercise (hard: localize a
bug you've never met from its artifact alone) or **after** it
(consolidation: recognise a bug you know from a new direction). Both
orders are legitimate; the second is gentler.

## Index

| # | Title | Tier | Artifact | Localizes |
|---|-------|------|----------|-----------|
| [D01](./01-goroutine-pileup/README.md) | The Pile-Up | Intermediate | goroutine profile | 06 |
| [D02](./02-two-stacks/README.md) | Two Stacks, One Line | Intermediate | race detector report | 12 |
| [D03](./03-impossible-crash/README.md) | The Crash That Couldn't Happen | Advanced | panic traceback | 22 |
