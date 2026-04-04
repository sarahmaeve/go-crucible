# Exercise 04: The Missing Workflow

**Application:** gh-forge | **Difficulty:** Beginner

## Symptoms

`ParseWorkflow` successfully parses a YAML workflow file that contains `on:` trigger definitions and `env:` variables. No error is returned. But the `Workflow` struct that comes back has `On == nil` and `Env == nil` — the trigger and environment data simply do not appear. The YAML is valid and the fields exist in the file; they are just invisible to the parser.

## Reproduce

```bash
go test ./internal/parser/ -run TestExercise04 -v
```

## File to Investigate

`internal/parser/workflow.go` — look at the `rawWorkflow` struct definition

Compare the struct field names for `on` and `env` against the other fields like `Name` and `Jobs`.

## What You Will Learn

- Go's visibility rules: unexported (lowercase) struct fields are invisible to reflection-based packages such as `encoding/json` and `gopkg.in/yaml.v3`
- The YAML decoder uses reflection to set fields; if a field is unexported, the decoder silently skips it without any error
- The fix is to export the fields (capitalize them) and add the appropriate `yaml:` struct tags

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/parser/ -run TestExercise04 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
