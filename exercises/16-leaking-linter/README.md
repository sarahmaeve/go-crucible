# Exercise 16: The Leaking Linter

**Application:** gh-forge | **Difficulty:** Advanced

## Symptoms

`LintWorkflows` is called on a directory containing many YAML files. On a system with a tight open-file-descriptor limit, the function fails midway through with "too many open files". On systems with generous limits, the function completes but holds all file handles open until it returns — which is observable with `lsof`. Each `defer f.Close()` runs only when `LintWorkflows` itself returns, not at the end of each loop iteration.

## Reproduce

```bash
go test ./internal/lint/ -run TestExercise16 -v
```

## File to Investigate

`internal/lint/linter.go` — look at the `LintWorkflows` function

Find the `defer f.Close()` statement. Consider the scope of `defer` in Go.

## What You Will Learn

- `defer` is bound to the enclosing function's return, not to the enclosing block or loop iteration
- `defer` inside a `for` loop accumulates deferred calls for the entire duration of the function
- The fix: extract the per-file logic into a helper function so that each file's `defer f.Close()` runs when the helper returns, which is at the end of each iteration
- Alternatively, close explicitly (`f.Close()`) at the end of the loop body without using `defer`

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/lint/ -run TestExercise16 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
