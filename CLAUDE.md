# Go Crucible — Development Guide

## What This Repo Is

Go Crucible is a debugging training platform for DevOps/SRE/infrastructure engineers.
It contains 19 exercises across three Go applications, each with an intentionally
planted bug. Learners fix bugs to make tests pass.

## Critical Rule: No Spoilers in Source Code

**Never add comments to `.go` files that reveal the bug mechanism, the fix, or
the exercise number.** This is a training repo — the learner reads the source code
to find the bug. Comments that say "BUG:", "Fix:", "should be X instead of Y",
or reference exercise numbers defeat the purpose.

Acceptable: comments explaining what the code *does* (its intent).
Not acceptable: comments explaining what's *wrong* with the code.

## Where Maintainer Knowledge Lives

- `.crucible/exercises.yaml` — canonical registry of all bugs: location, mechanism,
  fix, and teaching rationale. Read this first when maintaining the repo.
- `solutions/*.patch` — the canonical fix for each exercise. Apply with
  `git apply solutions/NN-*.patch`.
- `.crucible/notes/` — per-exercise maintainer notes (optional, for complex context).
- Exercise `README.md` and `HINTS.md` files in `exercises/` are learner-facing
  and may contain progressive hints, but should not contain the full answer.

## Test Conventions

- Exercise tests are named `TestExerciseNN_ShortName` (e.g., `TestExercise01_SilentFailure`).
- Exercise tests MUST FAIL on the buggy code and PASS after the fix.
- Non-exercise tests (sanity checks, happy paths) MUST PASS on the buggy code.
- Test failure messages should describe WHAT was expected, not WHY it fails or HOW to fix it.
- Exercises 08 and 12 require the `-race` flag. Exercise 08 skips without it.

## Build and Test

```bash
go build ./...              # must compile cleanly
go vet ./...                # one expected warning (exercise 13: WaitGroup.Add in goroutine)
go test ./...               # 19 exercise tests fail, all others pass
go test -race ./...         # also catches exercises 08 and 12
make test-exercise N=01     # run a single exercise
make status                 # pass/fail summary for all 19
```

## Dependencies

- `k8s.io/client-go` (pinned) — used for fake K8s clients in kube-patrol tests.
  Initial `go mod download` is slow due to the large dependency tree.
- `gopkg.in/yaml.v3` — YAML parsing for gh-forge.
- Everything else is stdlib.
