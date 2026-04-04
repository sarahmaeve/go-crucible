# Hints for Exercise 16: The Leaking Linter

## Hint 1: Direction

The test uses a directory with many files and a low file descriptor limit. The function fails with "too many open files" or the test verifies that file handles are closed promptly. Look for resource management inside the loop — specifically, what runs when does a `defer` inside a `for` loop actually execute?

## Hint 2: Narrower

Open `internal/lint/linter.go` and find the `defer f.Close()` line inside the `for` loop in `LintWorkflows`. In Go, `defer` is bound to the surrounding function, not to the surrounding block. Every iteration of the loop defers a close — but none of them run until `LintWorkflows` itself returns. With many files, all handles accumulate.

## Hint 3: Almost There

There are two clean fixes:

**Option A — explicit close (no defer):** Replace `defer f.Close()` with `f.Close()` at the end of the loop body (after the YAML is decoded and linted).

**Option B — extract a helper function:** Move the open-decode-lint logic into a helper:

```go
func lintFile(path string, rules []lintRule) ([]types.LintFinding, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close() // now deferred to the helper's return, i.e., end of this file's processing
    // ... decode and lint
}
```

Call the helper from the loop. Each call returns and the deferred close fires before the next iteration opens the next file.
