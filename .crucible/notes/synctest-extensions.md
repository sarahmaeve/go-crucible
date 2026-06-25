# Maintainer note: `testing/synctest` extensions for exercises 06 and 10

Added 2026-06-25. These are an opt-in teaching layer, not graded exercises.

## What exists

- `internal/ingest/reader_synctest_test.go` — `TestExercise06_Synctest`
- `internal/health/checker_synctest_test.go` — `TestExercise10_Synctest`
- `exercises/06-stuck-pipeline/EXTENSION.md`, `exercises/10-hanging-health-check/EXTENSION.md`
- `docs/synctest.md` — concept page + fit map, linked from both EXTENSION files

Each extension test is a `testing/synctest` rewrite of the matching canonical
exercise test. They demonstrate testing *technique*, not the fix: 06 shows
synctest turning a goroutine leak into a located deadlock failure (vs the
canonical `NumGoroutine` + `time.Sleep` heuristic); 10 shows the fake clock
asserting `context.DeadlineExceeded` exactly and instantly (vs the canonical
"returned within a second" wall-clock proxy).

## Why a build tag, and why these don't touch the canonical contract

Both files carry `//go:build synctest`. That tag is the isolation mechanism — it
keeps the extensions out of every default-tag command, so they do **not**
participate in the graded suite:

- `make test` / `make status` / `go test ./...` — run without `-tags`, so the
  files are excluded from the build and never execute.
- `make verify-failures` — runs `go test ./... -run "^TestExerciseNN"` without
  `-tags`; the extensions can't affect the pass/fail verdict for 06 or 10.
- `make verify-vet` — `go vet ./...` runs without `-tags`, so the extension
  files are not vetted and cannot perturb the "exactly one WaitGroup warning"
  invariant.
- `make verify-quick` (`tools/verify`) — the spoiler lint scans non-test `.go`
  files only, and the tree check requires `README.md`+`HINTS.md` to *exist* (it
  does not forbid extra files like `EXTENSION.md`). `EXTENSION.md` and
  `docs/synctest.md` are Markdown and are not scanned at all.

The functions are deliberately named `TestExerciseNN_Synctest` for learner
discoverability (`-run TestExercise06`). `tools/verify.testFunctions` regex-scans
all `_test.go` files ignoring build tags, so it will "see" these and count 06/10
as having a test — harmless, since they already do via the canonical tests.

Run them with: `go test -tags synctest ./internal/<pkg>/ -run TestExerciseNN_Synctest -v`

## Behaviour against the tree state

- 06 is buggy on `main`: the extension FAILS via synctest deadlock detection
  (names `reader.go:18`), and PASSES once the bug is fixed.
- 10 is pre-solved on `main`: the extension PASSES instantly; reverse-apply
  `solutions/10-hanging-health-check.patch` to see it FAIL.

There is no solution patch and no registry entry for these — they are not
exercises in the `make status` sense. If the underlying `reader.go` / `checker.go`
signatures change, update the extension tests alongside the canonical ones.

## Why only 06 and 10

These two are the cleanest demonstrations of synctest's two superpowers
(durable-block/leak detection and the fake clock). The fit map in
`docs/synctest.md` records the full assessment, including the non-fits — 14
(busy-spin, never durably blocks), 18 (allocation growth, not observable via
synctest), 08/12 (data races — `-race`'s job) — and the partial fit, 19 (only the
ctx-ignoring-goroutine leg). Kept deliberately narrow so the extensions teach
judgement, not just enthusiasm.
