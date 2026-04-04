# Go Crucible — Future Stages

## Learning Progression

The Crucible is designed as stage 1 of a three-stage progression:

1. **Stage 1: Debug and Fix** (current Crucible) — Learn Go's footguns by encountering
   them in realistic infrastructure code. 19 exercises across three apps. The learner
   reads code they didn't write, finds planted bugs, and fixes them.

2. **Stage 2: Extend and Design** — Add features to existing code, write tests, make
   architectural choices. Assumes the learner can already spot common Go gotchas.

3. **Stage 3: Build from Scratch** — Design a controller, implement a CLI tool, structure
   a project. Requires both debugging fluency and design judgment.

Stages 2 and 3 should not be attempted before the learner can reliably complete the
Crucible exercises. The pattern recognition built in stage 1 is the vocabulary that
makes later stages possible.

---

## Stage 2: Extend and Design (not yet built)

### High-Priority Exercises

#### HTTP Client Patterns (pipeline app)
These are the most common source of production incidents in Go infrastructure code.

- **HTTP response body leak** — `http.Response.Body` can be non-nil even when `err != nil`.
  The exercise would give learners a working HTTP client that leaks under error conditions.
  Different from exercise 09 because the gotcha is specific to the `net/http` contract.

- **Missing client timeout** — `http.Client{}` has no default timeout. Connections hang
  forever when a downstream service stops responding. Exercise: a health checker that
  hangs when a dependency is slow, because the client has no timeout configured.

- **Connection pool exhaustion** — Not reading/draining response bodies before closing
  prevents connection reuse. Exercise: a metrics scraper that works for 100 targets but
  fails at 500 because the connection pool is exhausted.

- **Retry with backoff** — Every SRE writes retry logic. Exercise: implement a retry
  wrapper with exponential backoff, jitter, and context cancellation. This is a "write
  from scratch" exercise, not a bug fix.

#### Kubernetes Controller Patterns (kube-patrol app)
For SREs working with operators and custom controllers.

- **Stale informer cache** — A reconciler reads from the informer cache after an update
  but gets stale data because the cache hasn't synced yet. Exercise: fix a controller
  that intermittently processes outdated resource versions.

- **Status update conflict** — Optimistic concurrency with `resourceVersion`. Exercise:
  a controller that crashes with "the object has been modified" because it doesn't
  retry on conflict.

- **Requeue semantics** — Returning an error vs explicitly requeueing. Exercise: a
  controller that retries forever on a permanent error because it doesn't distinguish
  transient from permanent failures.

#### CLI Tool Patterns (gh-forge app)

- **Exit code handling** — A CLI tool that returns exit 0 on partial failure, masking
  errors in CI pipelines. Exercise: fix the exit code logic.

- **Stdin/stdout piping** — A tool that breaks when piped (`tool | head`) because it
  doesn't handle SIGPIPE or broken pipe errors.

### Medium-Priority Exercises

#### Testing Skills
The Crucible teaches debugging existing tests. Stage 2 should teach writing them.

- **Write a table-driven test** — Given a function with no tests, write comprehensive
  test coverage including edge cases.

- **Write a fake for an external dependency** — Given an interface, implement a test
  fake that simulates realistic behavior (latency, errors, partial results).

- **Integration test with build tags** — Separate unit tests from integration tests
  using `//go:build integration`.

#### Observability

- **Metric label cardinality explosion** — A Prometheus metric with an unbounded label
  (e.g., user ID) that causes memory exhaustion. Exercise: fix the label strategy.

- **Structured logging pitfalls** �� A service that logs sensitive data (tokens, PII) in
  error messages. Exercise: sanitize the logging.

---

## Stage 3: Build from Scratch (not yet built)

### Design Exercises

These are open-ended and don't have a single correct answer.

- **Design a concurrent pipeline** — Given requirements (ingest from N sources, transform,
  fan out to M sinks), design the goroutine topology, channel structure, and shutdown
  mechanism.

- **Design an interface boundary** — Given a concrete implementation, extract an interface
  that enables testing and future extension. Evaluate tradeoffs (narrow vs wide interface).

- **Structure a Go project** — Given a set of requirements, decide on package layout,
  dependency direction, and public API surface.

### Operational Exercises

- **Graceful connection draining** — Not just shutdown (exercise 19) but draining
  in-flight requests with a deadline before forced termination.

- **Leader election** — Implement leader election using a shared resource (e.g., a
  Kubernetes lease or a file lock).

- **Circuit breaker** — Implement a circuit breaker that opens after N failures,
  half-opens after a timeout, and closes on success.

---

## Interview Preparation Gaps

The Crucible covers "What's wrong with this code?" well. These interview patterns
are NOT covered and would need separate materials:

- **"Explain the output"** — Given a goroutine ordering problem, predict what prints.
  Conceptual, not hands-on.
- **"How would you improve this?"** — Refactoring for clarity/performance, not bug fixing.
- **"Design a system"** — Whiteboard-style architecture using Go concurrency primitives.
- **"Code review"** — Evaluate a PR for correctness, style, and performance. Different
  from finding a planted bug because the code may be correct but suboptimal.
