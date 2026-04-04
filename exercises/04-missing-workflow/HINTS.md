# Hints for Exercise 04: The Missing Workflow

## Hint 1: Direction

The YAML is valid and parses without error, but specific fields in the result are always nil. The parser uses a struct to receive the decoded YAML. Look at the struct definition carefully — there is something different about the fields that are always nil compared to the fields that work.

## Hint 2: Narrower

Open `internal/parser/workflow.go` and look at the `rawWorkflow` struct. Compare the field `Name` (which works) with the fields `on` and `env` (which are always nil). Notice the capitalization. The `yaml` decoder uses reflection to set fields; it can only set exported fields.

## Hint 3: Almost There

Change the unexported fields to exported ones and add the correct `yaml:` tags:

```go
// Before (unexported — yaml decoder cannot set these):
on  map[string]any
env map[string]string

// After (exported with yaml tags):
On  map[string]any    `yaml:"on"`
Env map[string]string `yaml:"env,omitempty"`
```

Then update the two lines in `ParseWorkflow` that read `raw.on` and `raw.env` to use `raw.On` and `raw.Env`.
