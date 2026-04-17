# Review Track

A parallel track of exercises focused on **reading change** rather than
reading code. Where the numbered exercises ask you to find a bug in an
isolated file, review exercises ask you to review a pull request — a
diff spanning several files — and produce a set of review comments.

## Why a separate track

Debugging and code review are the same skill in different tenses:

- Debugging is reading code you already own, looking for traps that are
  already there.
- Review is reading code someone else proposes, looking for traps before
  they land.

The crucible's numbered exercises train the first. This track trains the
second. Both matter. Review is the skill that keeps bad code out of your
codebase — whether that code was written by a hurried colleague, by you
last week, or by an LLM that doesn't understand the invariants your
service relies on.

## How review exercises work

- Each exercise presents a simulated pull request: a description, a
  unified diff, and a prompt.
- Your deliverable is a review — structured comments with severity and
  line references. There is no test to run, no patch to apply. Success
  is writing a review that catches the real issues without over-flagging
  the benign ones.
- Each exercise has a `REVIEWER_NOTES.md` with a canonical sample review.
  Compare yours to it after you've tried. Yours may legitimately differ
  in tone, severity thresholds, or which process concerns you raise.

## The false-positive discipline

A common flaw in new reviewers is over-flagging: treating "different
from how I would have written it" as a problem. Every review exercise
includes at least one **red herring** — something that looks suspicious
but is actually fine. The `REVIEW_TEMPLATE.md` has a "Things I Verified"
section; a good review fills it in. Naming what you checked and found
to be OK is as important as naming what you found to be wrong.

## When to attempt a review exercise

Each review exercise states a tier — Basic, Intermediate, or Advanced —
and names the numbered exercises whose reflexes it draws on. Attempt a
review exercise only after you've completed the numbered exercises it
references. The review exercise assumes you can already spot the
underlying patterns in isolation; what you're learning is how to spot
them inside a diff.

## Index

| # | Title | Tier | Draws on |
|---|-------|------|----------|
| [01](./01-first-review/README.md) | First Review: The `--since` Flag PR | Basic | 01, 09 |
| [02](./02-annotations-feature/README.md) | The Annotations Feature PR | Basic | 02, 04 |
