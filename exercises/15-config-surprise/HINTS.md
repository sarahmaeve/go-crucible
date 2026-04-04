# Hints for Exercise 15: The Config Surprise

## Hint 1: Direction

A workflow round-trips through `RoundTripWorkflow` and comes back missing a field. The missing field is a boolean set to `false`. In Go, `false` is the zero value for `bool`. Think about what `omitempty` does with zero values during marshaling.

## Hint 2: Narrower

Open `internal/parser/workflow.go` and find the `concurrencyIntermediate` struct. Look at the JSON tag on `CancelInProgress`:

```go
CancelInProgress bool `json:"cancel-in-progress,omitempty"`
```

`omitempty` tells the JSON encoder to skip this field when it equals the zero value for its type. For `bool`, zero is `false`. A `false` value — even a deliberate one — is silently dropped from the JSON output.

## Hint 3: Almost There

Remove `,omitempty` from the `CancelInProgress` JSON tag:

```go
CancelInProgress bool `json:"cancel-in-progress"`
```

Now `false` is always written to the JSON output, preserved through the round-trip, and marshaled back into the output YAML. The field is only absent when the entire `Concurrency` block is absent (which is still handled by `omitempty` on the outer `Concurrency` pointer field).
