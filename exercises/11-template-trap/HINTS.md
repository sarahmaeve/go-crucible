# Hints for Exercise 11: The Template Trap

## Hint 1: Direction

`BuildAdvancedTemplate` returns a `Template` interface. The returned value is supposed to produce an advanced workflow with a matrix strategy. Instead it produces a plain workflow. No error is returned. The wrong `Generate()` method is being called — trace which concrete type is inside the interface.

## Hint 2: Narrower

Open `internal/generate/template.go` and look at the `return` statement at the bottom of `BuildAdvancedTemplate`. It returns `adv.BaseTemplate` — the embedded struct, not `adv` itself. `BaseTemplate` satisfies the `Template` interface because it has its own `Name()` and `Generate()` methods. The compiler accepts this silently.

## Hint 3: Almost There

Change:

```go
return adv.BaseTemplate
```

to:

```go
return adv
```

Now the interface holds an `AdvancedTemplate` value, and when `Generate()` is called, Go's method dispatch selects `AdvancedTemplate.Generate()` — which includes the matrix strategy, OS targets, Go versions, and concurrency settings.
