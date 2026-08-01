# Hints for Exercise 15: The Config Surprise

## Hint 1: Direction

A workflow round-trips through `RoundTripWorkflow` and comes back missing an
explicitly false field. Compare how the public workflow type and the internal
round-trip type represent presence.

## Hint 2: Narrower

Open `internal/parser/workflow.go` and find the `concurrencyIntermediate` struct. Look at the JSON tag on `CancelInProgress`:

```go
CancelInProgress bool `json:"cancel-in-progress,omitempty"`
```

`omitempty` tells the JSON encoder to skip this field when it equals the zero value for its type. For `bool`, zero is `false`. A `false` value — even a deliberate one — is silently dropped from the JSON output.

The public `WorkflowConcurrency` uses `*bool`: nil means absent, while pointers
to false and true preserve explicit choices. Flattening that pointer into the
plain intermediate boolean discards information before marshaling even begins.

## Hint 3: Almost There

Carry the pointer through the intermediate representation:

```go
CancelInProgress *bool `json:"cancel-in-progress,omitempty"`
```

Map the public pointer directly into that field. With `omitempty`, nil remains
absent, while pointers to false and true are both serialized. The exercise test
covers all three states.
