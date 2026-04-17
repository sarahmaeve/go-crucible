# Exercise 01 — Maintainer Notes

## Keep the bug alive across refactors

Exercise 01 is the simplest bug in the repo (a missing `return nil, err` after
a logged error). Precisely because it's so small, it is easy to accidentally
"fix" during an unrelated refactor of `internal/audit/pods.go`. When
`exercises.yaml` shows `number: "01"`, the source file **must** contain:

```go
pods, err := c.ListPods(ctx, namespace)
if err != nil {
    slog.Error("AuditPodLimits: failed to list pods", "err", err)
    // no return — this is the bug
}
```

If you find yourself adding a `return nil, err` during a cleanup pass,
stop: you are about to break the exercise.

## History: stale solution patch

From the initial commit through 2026-04-16, `solutions/01-silent-failure.patch`
was a stub that said:

> No diff: internal/audit/pods.go already contained the correct fix
> (return nil, err) in the initial commit. The bug described in
> exercises.yaml (log-and-fall-through without returning) is not present
> in the source tree.

That note was wrong — the bug **was** present. The patch became stale during
the "finish fixes of polluted files" cleanup (commit b2187fb), which restored
several exercise bugs but did not regenerate their solution patches. As a
result, `make verify-solution N=01` reported success against buggy source for
several months.

Regenerated 2026-04-16 with a real diff against the current `slog`-based
source. `make verify-solution N=01` now genuinely verifies that the patch
converts the buggy tree into a passing tree.

## Logger migration

The error path was migrated from `log.Printf` to `slog.Error` as part of the
2026-04-16 logger-unification sweep. This does not affect the lesson (log-and-
continue is still the antipattern), but the `HINTS.md` code snippet had to be
updated to match the new call. If you migrate loggers again, remember to
update both:

- `internal/audit/pods.go` (the buggy source)
- `exercises/01-silent-failure/HINTS.md` (the snippet the learner reads)
- `solutions/01-silent-failure.patch` (regenerate)
