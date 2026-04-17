# `.crucible/` — Maintainer Notes (contains spoilers)

This directory is the canonical source of truth for the go-crucible exercise
design. It is **tracked in git and ships publicly**, so any learner who
goes looking will find it.

That is a deliberate trade-off:

- **Pro:** maintainers and contributors get one authoritative place to look
  when extending the repo. The registry stays in sync with the source because
  edits are part of the same commit.
- **Con:** a motivated learner can short-circuit the exercises by reading the
  `mechanism` and `fix` fields here.

If you are a learner: please do not read these files until you have tried the
exercise. The learning value of the Crucible is in the struggle, not in the
answer. The hint files under `exercises/NN-*/HINTS.md` are the learner-safe
escalation path.

## Contents

- **`exercises.yaml`** — registry of all 20 intentional bugs, what each one
  teaches, and which patch file fixes it. Edit this file whenever the source
  code around an exercise changes (e.g., a log-package migration that shifts
  the snippet shown in a hint).

- **`notes/NN-*.md`** — optional long-form notes for individual exercises.
  Use these for context that is too long or too discursive for
  `exercises.yaml` (idiom choices, why a particular fix was picked over a
  defensible alternative, cross-references between exercises).

- **`notes/sessions/`** *(optional)* — per-session retrospectives for large
  refactors that cut across multiple exercises. Useful when an outside
  contributor asks "why is it like this?" months later.

## Source-of-truth rules

1. If the source code around an exercise changes, update the matching entry
   in `exercises.yaml` in the **same commit**. The `mechanism` and `fix`
   fields quote code; they rot quickly if treated as separate docs.

2. Solution patches under `solutions/*.patch` must apply cleanly against the
   current source tree and produce a passing test. If a refactor shifts
   line numbers or changes surrounding idiom, regenerate the patch.

3. When an exercise is "solved in main" (the fix is applied on the default
   branch rather than left as a puzzle for the learner), add
   `solved_in_main: <YYYY-MM-DD>` to that exercise's entry. Learners who
   want the buggy version can `git apply -R solutions/NN-*.patch` to
   reintroduce it.
