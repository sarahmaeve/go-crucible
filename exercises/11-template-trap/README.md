# Exercise 11: The Template Trap

**Application:** gh-forge | **Difficulty:** Intermediate

## Symptoms

`BuildAdvancedTemplate` is called with OS targets, Go versions, and a concurrency group. The returned `Template` is used to generate a workflow. The workflow that comes back has no matrix strategy, no OS targets, no Go versions, and no concurrency settings — it is a plain single-job workflow, as if `BaseTemplate.Generate()` was called instead. No error is returned; the output is silently wrong.

## Reproduce

```bash
go test ./internal/generate/ -run TestExercise11 -v
```

## File to Investigate

`internal/generate/template.go` — look at the `BuildAdvancedTemplate` function

Notice what value the `return` statement at the bottom of the function actually returns.

## What You Will Learn

- When a struct embeds another struct, both satisfy the same interface if the embedded type has all the required methods
- Returning `adv.BaseTemplate` instead of `adv` (or `&adv`) assigns only the embedded base struct to the interface — all fields specific to `AdvancedTemplate` are lost
- The compiler does not warn about this because `BaseTemplate` legitimately satisfies the `Template` interface
- Method shadowing with embedded structs: `AdvancedTemplate.Generate()` overrides `BaseTemplate.Generate()`, but only when the concrete type is `AdvancedTemplate`

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/generate/ -run TestExercise11 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
