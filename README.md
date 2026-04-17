# go-crucible

A Go debugging training repository. Each exercise contains a real, runnable bug in a realistic application context. Your job is to find the bug, understand why it exists, fix it, and confirm the fix with the provided test.

## Who This Is For

Go developers who want to sharpen their debugging instincts across the full spectrum of common Go pitfalls — from beginner mistakes like nil map writes and swallowed errors, through intermediate concurrency bugs and context misuse, up to advanced issues like timer leaks and compound shutdown failures.

No Kubernetes cluster is required. All tests run locally against in-process fakes.

## Prerequisites

- Go 1.25 or later (`go version`) — required by the pinned `k8s.io/client-go`
- Git

## Getting Started

Clone the repository and download dependencies:

```bash
git clone https://github.com/go-crucible/go-crucible.git
cd go-crucible
go mod download
```

> **Note:** `go mod download` pulls `k8s.io/client-go` and its transitive dependencies, which is a large module graph (~200 MB). This is a one-time download.

To see every failing test at once:

```bash
go test ./...
```

All tests fail out of the box — that is the point.

## How to Work Through the Exercises

1. Open [exercises/README.md](./exercises/README.md) and pick an exercise.
2. Read the exercise's own `README.md` for context, symptoms, and the file to investigate.
3. Open the buggy source file and find the bug.
4. Apply your fix.
5. Run the exercise's test command to confirm it passes.
6. If you get stuck, open `HINTS.md` in the same exercise directory for progressive hints.

## Race-Condition Exercises

Exercises 08 and 12 require the race detector:

```bash
go test -race ./internal/transform/ -run TestExercise08 -v
go test -race ./internal/audit/     -run TestExercise12 -v
```

Exercise 13 is best observed with repeated runs:

```bash
go test -race ./internal/audit/ -run TestExercise13 -v -count=10
```

## Exercise Index

See [exercises/README.md](./exercises/README.md) for the full checklist of all 19 exercises organized by difficulty.

## Repository Layout

```
cmd/            Entry points (pipeline daemon, gh-forge CLI, kube-patrol CLI)
internal/       All buggy application packages
  alert/        Alert evaluation (pipeline)
  audit/        Kubernetes resource auditing (kube-patrol)
  client/       Kubernetes client wrapper (kube-patrol)
  generate/     Workflow template generation (gh-forge)
  health/       Health-check primitives (pipeline)
  ingest/       Metric ingestion (pipeline)
  lint/         Workflow linter (gh-forge)
  parser/       GitHub Actions YAML parsing (gh-forge)
  transform/    Metric transformation (pipeline)
exercises/      One subdirectory per exercise — README.md and HINTS.md
solutions/      Reference solutions (consult only after you have tried)
testdata/       Sample YAML files used by tests
.crucible/      Maintainer registry — contains spoilers; do not read until
                after attempting an exercise
```

## A note on pre-solved exercises

Some exercises are pre-solved on `main` as reference implementations (currently
10, 18, 19 — see `.crucible/exercises.yaml` for the authoritative list). If
you want to practise on those, reintroduce the buggy form with:

```bash
git apply -R solutions/NN-<name>.patch
```

Re-apply the solution to check your work:

```bash
git apply solutions/NN-<name>.patch
```

Every solution patch round-trips cleanly in both directions.
