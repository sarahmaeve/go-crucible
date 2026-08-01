# Exercise 15: The Config Surprise

**Application:** gh-forge | **Difficulty:** Advanced

## Symptoms

The public workflow model distinguishes an absent
`concurrency.cancel-in-progress` setting from explicit `false` and explicit
`true`. After a round trip through `RoundTripWorkflow` (parse → re-serialize),
the intermediate representation collapses absence and false into the same Go
zero value, so an explicit choice disappears. That loss matters to tools that
audit, diff, or enforce the written form.

## Reproduce

```bash
go test ./internal/parser/ -run TestExercise15 -v
```

## File to Investigate

`internal/parser/workflow.go` — look at the `concurrencyIntermediate` struct and its `CancelInProgress` field

Examine the JSON struct tag on that field carefully.

## What You Will Learn

- `omitempty` on a plain boolean collapses absence and false during serialization
- A pointer boolean can represent nil, false, and true when presence matters
- This is a common footgun when explicit presence matters to configuration
  provenance, or when another schema gives absence and the zero value different meanings
- Presence information must survive every intermediate model in a round-trip pipeline
- The fix: keep `*bool` rather than flattening the setting to `bool` in the JSON intermediate

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/parser/ -run TestExercise15 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
