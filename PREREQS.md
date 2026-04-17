# Prerequisites

Before starting Go Crucible, make sure you have the following set up and are
comfortable with the basics below. You don't need to be a Go expert — the
exercises will teach you the tricky parts. But you do need to be able to read
Go code and run commands in a terminal.

## What You Need Installed

- **Go** (1.25 or later) — [install instructions](https://go.dev/doc/install).
  Verify with `go version`. The `k8s.io/client-go` dependency pins the floor at
  Go 1.25; earlier toolchains will fail at `go mod download`.
- **Git** — for cloning the repo and optionally applying solution patches.
- **A text editor** you're comfortable navigating code in. VS Code with the Go
  extension is a good default, but anything that lets you jump to definitions
  and search across files will work.
- **A terminal** — you'll be running `go test` commands and reading their output.

## Go Syntax You Should Recognize

You don't need to write Go from memory, but you should be able to read and
follow code that uses:

- **Functions** — including multiple return values (`func Foo() (string, error)`)
- **Error handling** — the `if err != nil { return err }` pattern shows up everywhere
- **Structs** — type definitions and field access (`pod.Name`, `config.Threshold`)
- **Maps and slices** — `map[string]int`, `[]string`, basic reads and writes
- **For loops** — especially `for _, item := range items`
- **Imports and packages** — `import "fmt"`, `package audit`, dotted access like `types.Finding`

If any of these look unfamiliar, spend an hour with the
[Tour of Go](https://go.dev/tour/) first. The first few sections (up through
"Methods and interfaces") cover everything you need.

## Concepts You Should Understand

- **Functions return values**, and callers are responsible for checking them —
  especially errors.
- **Variables have types**, and every type has a zero value (`0` for numbers,
  `""` for strings, `nil` for pointers/maps/slices).
- **`nil` means "no value"** — you'll see it a lot in error checks and
  optional fields.
- **Packages organize code** — files in the same directory share a package
  name, and `internal/` means the package is private to the project.
- **`go test` runs tests** — that's your primary tool in every exercise.

## What You Do NOT Need

Don't worry if you haven't used these yet. The exercises introduce them
gradually, and each one comes with a README explaining the concept:

- Goroutines and channels (starts at exercise 06)
- Interfaces and type assertions (exercise 05)
- Mutexes and synchronization (exercise 08)
- Build tags, profiling, or Go toolchain configuration
- Any experience writing Go from scratch — reading comprehension is enough

## Quick Self-Check

Can you read this function and explain what it does?

```go
func FindExpired(items []Item) ([]Item, error) {
    var expired []Item
    for _, item := range items {
        if item.ExpiresAt.Before(time.Now()) {
            expired = append(expired, item)
        }
    }
    if len(expired) == 0 {
        return nil, nil
    }
    return expired, nil
}
```

If that makes sense to you — it iterates over a slice, checks a timestamp,
collects matches, and returns them — you're ready to start. Head to
[exercises/README.md](exercises/README.md) and begin with Exercise 01.
