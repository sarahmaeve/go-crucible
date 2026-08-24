# Maintainer note: `testing/synctest` coverage for exercises 06 and 10

Added 2026-06-25. These are an opt-in teaching layer, not graded exercises.

## What exists

- `internal/ingest/reader_synctest_test.go` — `TestExercise06_Synctest`
- `internal/health/checker_test.go` — canonical `TestExercise10_HangingHealthCheck`
- `exercises/06-stuck-pipeline/EXTENSION.md`, `exercises/10-hanging-health-check/EXTENSION.md`
- `docs/synctest.md` — concept page + fit map, linked from both EXTENSION files

Exercise 06 retains an opt-in synctest extension that turns a goroutine leak
into a located deadlock failure. Exercise 10 now uses synctest in its canonical
test because the repository requires Go 1.27; its fake clock asserts
`context.DeadlineExceeded` exactly and instantly.

## Why exercise 06 still uses a build tag

The exercise 06 extension carries `//go:build synctest`. That tag is the
isolation mechanism — it keeps that extension out of every default-tag command,
so it does **not**
participate in the graded suite:

- `make test` / `make status` / `go test ./...` — run without `-tags`, so the
  files are excluded from the build and never execute.
- `make verify-failures` — runs `go test ./... -run "^TestExerciseNN"` without
  `-tags`; the extensions can't affect the pass/fail verdict for 06 or 10.
- `make verify-vet` — `go vet ./...` runs without `-tags`, so the extension
  files are not vetted and cannot perturb the "exactly one WaitGroup warning"
  invariant.
- `make verify-quick` (`tools/verify`) — the spoiler lint scans production and
  test `.go` comments. Test comments may describe the observable lifecycle
  contract but must not disclose the blocked operation or repair. The tree
  check requires `README.md`+`HINTS.md` to *exist* and permits extra files such
  as `EXTENSION.md`; Markdown is outside the code-comment lint.

The function is deliberately named `TestExercise06_Synctest` for learner
discoverability. `tools/verify.testFunctions` scans all `_test.go` files ignoring
build tags, which is harmless because 06 also has its canonical test.

Run it with: `go test -tags synctest ./internal/ingest -run TestExercise06_Synctest -v`

## Behaviour against the tree state

- 06 is buggy on `main`: the extension FAILS via synctest deadlock detection
  (names `reader.go:18`), and PASSES once the bug is fixed.
- 10 is pre-solved on `main`: its canonical test PASSES instantly; reverse-apply
  `solutions/10-hanging-health-check.patch` to see it FAIL at fake t=10s.

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
