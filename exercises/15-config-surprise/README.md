# Exercise 15: The Config Surprise

**Application:** gh-forge | **Difficulty:** Advanced

## Symptoms

A workflow is configured with `concurrency.cancel-in-progress: false` — a deliberate choice meaning "queue runs in order, do not cancel the in-progress one". After a round-trip through `RoundTripWorkflow` (parse → re-serialize), the output YAML no longer contains the `cancel-in-progress` field at all. Downstream consumers treat the absent field as the default (which may be `true`), silently changing the workflow's behaviour.

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
- This is a common footgun when the zero value of a type carries a distinct meaning (disabled, off, queue-mode)
- The fix: remove `omitempty` from fields where the zero value is a valid, meaningful configuration choice

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/parser/ -run TestExercise15 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
