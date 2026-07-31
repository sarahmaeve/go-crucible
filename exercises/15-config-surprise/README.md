# Exercise 15: The Config Surprise

**Application:** gh-forge | **Difficulty:** Advanced

## Symptoms

A workflow explicitly contains `concurrency.cancel-in-progress: false`. After
a round-trip through `RoundTripWorkflow` (parse → re-serialize), the output
YAML no longer contains the field. [GitHub Actions currently requires an
explicit `true` to cancel an in-progress
run](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency),
so omission has the same runtime behavior as `false`; nevertheless, the
serializer has failed its stated job because an explicit configuration choice
disappeared. That loss matters to tools that audit, diff, or enforce the
written form.

## Reproduce

```bash
go test ./internal/parser/ -run TestExercise15 -v
```

## File to Investigate

`internal/parser/workflow.go` — look at the `concurrencyIntermediate` struct and its `CancelInProgress` field

Examine the JSON struct tag on that field carefully.

## What You Will Learn

- `omitempty` in JSON/YAML tags omits the field when it equals the Go zero value for its type
- For `bool`, the zero value is `false` — so `omitempty` silently drops `false` values even when they are semantically meaningful
- This is a common footgun when explicit presence matters to configuration
  provenance, or when another schema gives absence and the zero value different meanings
- The fix: remove `omitempty` from fields where the zero value is a valid, meaningful configuration choice
- Removing `omitempty` makes this simplified model emit an explicit value whenever a concurrency block exists; a genuinely tri-state model requires a pointer or another presence-tracking representation throughout the parse pipeline

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/parser/ -run TestExercise15 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
