# Sample Review — PR #198

This is one reasonable review. Yours may legitimately differ in tone,
severity, or which process concerns you raise. Compare the *substance*.

## Overall assessment

**Request changes.**

Two correctness issues: the intermediate decoding struct has an
unexported field that the YAML decoder will silently skip, and
`BuildOwnerIndex` writes to a nil map and will panic on first invocation.
Neither issue is covered by the included test — which is itself worth
discussing.

## Blockers

### 1. `rawAnnotation.tags` is unexported — tags are silently dropped

**`internal/parser/annotations.go:22-26`**, severity: **major**.

```go
type rawAnnotation struct {
    Name  string            `yaml:"name"`
    Owner string            `yaml:"owner"`
    tags  map[string]string `yaml:"tags"`
}
```

Go's `yaml.v3` decoder (like `encoding/json`, and every other
reflect-based decoder in the standard ecosystem) only populates
**exported** struct fields. The `yaml:"tags"` struct tag on a lowercase
field is silently ignored — the decoder cannot reach the field via
reflection, so it does nothing. Every `Annotations.Tags` in the
returned map will be `nil`, regardless of what the input YAML
contained.

Concretely: with the PR's example YAML (`tags: {compliance: sox, runbook: ...}`)
the returned `Annotations{Owner: "platform", Tags: nil}` silently drops
`compliance` and `runbook`. The alert-routing feature the PR cites as
motivation will fire alerts with no compliance context attached.

**Suggested fix:** rename `tags` to `Tags` and update the assignment at
line 41 (`Tags: a.Tags`). Then add a test (see the test-coverage
concern below) that asserts `Tags["compliance"] == "sox"` so this
regression cannot recur.

### 2. `BuildOwnerIndex` panics on first write — nil map

**`internal/parser/annotations.go:48-58`**, severity: **major**.

```go
func BuildOwnerIndex(jobs []Job, annotations map[string]Annotations) map[string][]string {
    var index map[string][]string
    for _, job := range jobs {
        ann, ok := annotations[job.Name]
        if !ok {
            slog.Warn("job has no annotations", "job", job.Name)
            continue
        }
        index[ann.Owner] = append(index[ann.Owner], job.Name)
    }
    return index
}
```

`var index map[string][]string` declares `index` with its zero value,
which for a map type is `nil`. Reads from a nil map return the zero
value of the element type (so the `append(index[ann.Owner], ...)` on
the right-hand side produces `[]string{job.Name}` without panicking).
But the **assignment** `index[ann.Owner] = ...` on a nil map panics at
runtime with `assignment to entry in nil map`.

The first call to `BuildOwnerIndex` with a non-empty annotated job
panics. The function is reachable via the alert-routing feature this
PR is setting up, so this will manifest in production the first time
routing runs against a real workflow.

**Suggested fix:** initialise the map with `make`:

```go
index := make(map[string][]string, len(annotations))
```

The capacity hint is optional but cheap and documents intent.

## Suggestions

### Document the "no annotations" case in the API contract

**`internal/parser/annotations.go:48`**, severity: **minor**.

`BuildOwnerIndex` logs a warning for jobs with no annotations and
skips them. That is probably the right behaviour but it isn't stated
in the function's docstring. A caller reading the docstring shouldn't
have to trace into the implementation to learn "jobs without
annotations are silently dropped from the index."

Consider adding one line to the doc: "Jobs without a matching
annotation are logged and omitted from the index."

## Questions

### What is the intended behaviour for duplicate annotation names?

**`internal/parser/annotations.go:37-41`**, severity: **question**.

Two `rawAnnotation` entries with the same `Name` will silently
overwrite each other in the returned map (`result[a.Name] = ...`).
That may be the right behaviour (annotations are user-authored and
duplicates are a configuration error) or the wrong one (fail loudly
so the author notices). I don't have strong feelings, but whichever
you choose, please document it in the function docstring and add a
test for the chosen behaviour.

## Things I Verified

### `Tags map[string]string` (string values only)

**`internal/parser/annotations.go:17`**.

At first I wondered whether `map[string]string` was too restrictive —
what about numeric values like priorities? The PR description has an
explicit **Scope note** that tags are string-to-string and that a
richer `Value` union is deferred to a follow-up. That is a reasonable
scope call, and the type matches the decision. Not flagging.

## Process

### The test exercises the happy path only — and would pass even with the two bugs above

**`internal/parser/annotations_test.go`**, severity: **major process concern**.

I appreciate that the test is in this PR rather than deferred. But
the test as written would not catch either of the bugs above:

- The fixture at lines 11-14 has no `tags:` block on either annotation.
  Even with `rawAnnotation.tags` renamed correctly, the test would
  still pass — it never asserts anything about `Tags`.
- `BuildOwnerIndex` is never called, so the nil-map panic never fires
  under test.

Please extend this test (same PR — shouldn't be more than 20-30
lines) to cover:

1. An annotation with a non-empty `tags:` block, asserting the
   resulting `Annotations.Tags` contains the expected entries. This
   would have caught bug 1.
2. A call to `BuildOwnerIndex` with at least one annotated job,
   asserting the result is a non-empty map with the expected keys.
   This would have caught bug 2.
3. Bonus: a call to `BuildOwnerIndex` with a job that has no matching
   annotation, asserting it is omitted from the result.

A reviewer should feel able to merge a PR when the tests demonstrate
the feature works under the scenarios the PR description claims to
support. The current test demonstrates exactly one of those scenarios
(owner parsing); the other two (tags, index-building) are entirely
unasserted.

---

## What this sample review is trying to model

- **Two blockers, each with a suggested fix.** Same shape as exercise 01.
- **Test criticism, not test absence.** Exercise 01's process concern
  was about tests being punted to follow-up. This exercise's process
  concern is more nuanced: the tests exist but don't exercise the
  risky code paths. A "tests exist ✓" checkbox is not a review; reading
  the tests critically is.
- **A question that stays a question.** "What do you want to happen
  on duplicate names?" is not a blocker — the behaviour is well-
  defined (last-write-wins via map assignment). The reviewer wants the
  intent documented, not the code changed.
- **A Scope-note reference in the Things I Verified section.** The PR
  description explicitly pre-empts the Tags-type question; a good
  reviewer reads the description before raising concerns the author
  has already addressed.
- **Severity calibration.** The unexported field is a **major** bug
  because it silently drops data. The undocumented behaviour in
  `BuildOwnerIndex` is a **minor** suggestion because the behaviour
  itself is fine. Two bugs that both crash the build are different
  from two bugs where one silently corrupts data and the other causes
  a loud panic — both fixable, but the silent-corruption one is the
  one you lose sleep over.
