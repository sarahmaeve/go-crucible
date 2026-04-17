# Sample Review — PR #214

This is one reasonable review. Yours may legitimately differ in tone,
severity, or which process concerns you raise. Compare the *substance*.

## Overall assessment

**Request changes.**

Two correctness issues. The first silently drops all tag data — every
rule's `Replace` map ends up nil regardless of what the YAML contains.
The second silently breaks the error chain so `errors.Is(err,
ErrInvalidRule)` returns `false` at callers that rely on it. Both
bugs pass the PR's manual test plan, which is a strong argument for
the unit tests you proposed to skip.

## Blockers

### 1. `rawRule.replace` is unexported — tag replacements are silently dropped

**`internal/transform/rules.go:23`**, severity: **major**.

```go
type rawRule struct {
    Name    string            `yaml:"name"`
    Match   string            `yaml:"match"`
    replace map[string]string `yaml:"replace"`
}
```

Go's reflect-based YAML decoder only populates **exported** struct
fields. A `yaml:"replace"` struct tag on a lowercase field is silently
ignored — the decoder cannot reach the field, so it does nothing. At
line 44 (`Replace: r.replace`) we copy the zero value of that field
into the returned `Rule`. Every `Rule.Replace` in the returned slice
is nil, regardless of what the YAML config contained.

Concretely: the alert-routing PR this feature unblocks would receive
rules with `Replace == nil` and attach no `team` or `runbook` tags to
any metric. Routing would silently fan out alerts with no routing
metadata.

**Suggested fix:** rename `replace` to `Replace` and update the copy
at line 44. Please also add a unit test that asserts
`Rule.Replace["team"] == "platform"` on the cpu-host-routing fixture
from the PR description — the current manual test plan does not
inspect the returned tags beyond "they appeared in the output
stream," which is very easy to read as passing when it actually isn't.

### 2. `fmt.Errorf("rule %q: %v", ...)` breaks the `ErrInvalidRule` chain

**`internal/transform/rules.go:38`**, severity: **major**.

```go
if err := validateRule(r); err != nil {
    return nil, fmt.Errorf("rule %q: %v", r.Name, err)
}
```

`validateRule` wraps the sentinel correctly — `fmt.Errorf("name is
required: %w", types.ErrInvalidRule)`. The caller here then re-wraps
with `%v`, which formats the inner error as a string and discards the
chain. A caller of `LoadRules` that does `errors.Is(err,
types.ErrInvalidRule)` will get `false`.

That matters for exactly the use case the PR cites: the alert-routing
PR wants to distinguish "invalid rule config, tell the operator" from
"YAML decode error, different recovery." With `%v` here, both kinds
of errors look identical to `errors.Is`.

**Suggested fix:** change `%v` to `%w`. The rest of the diff already
uses `%w` correctly — this looks like a typo rather than a conscious
choice.

## Suggestions

None worth the author's time on this PR; the blockers and the process
concern are the substantive items.

## Things I Verified

### `var out []Rule` followed by `append` is not a nil-map bug

**`internal/transform/rules.go:34-45`**.

I briefly paused on this because a nil *map* panics on write — that
was the bug in a recent exercise. But Go's rules for nil slices are
different: `append` on a nil slice is defined to allocate and return a
new slice. This is correct, idiomatic, and worth keeping. Not
flagging.

### Three of the four `fmt.Errorf` calls use `%w` correctly

**`internal/transform/rules.go:31`**, **`internal/transform/rules.go:51`**,
**`internal/transform/rules.go:55`**.

I checked each `fmt.Errorf` site individually because a single
`%v`/`%w` error is easy to miss in a diff that has several of them.

- Line 31: `fmt.Errorf("load rules: %w", err)` around `yaml.Unmarshal`
  — correct. The underlying YAML error is wrappable, and the caller
  may want to inspect it.
- Line 51: `fmt.Errorf("name is required: %w", types.ErrInvalidRule)`
  — correct. The sentinel is in the chain.
- Line 55: `fmt.Errorf("match is required: %w", types.ErrInvalidRule)`
  — correct. Same.

Only the one on line 38 is buggy (see Blocker 2). Three of four
`fmt.Errorf` calls being right is not the same as the diff being
right; a single silent chain break is enough to defeat
`errors.Is`-based routing.

## Questions

### Is `Match` a literal prefix or a pattern?

**`internal/transform/rules.go:15`**, severity: **question**.

The field is named `Match` and the docstring says "when a metric's
name starts with Match." That is clear — it's a literal prefix, not a
regex or glob. I wanted to confirm this is the deliberate choice
(rather than `strings.HasPrefix` being a stand-in for something
richer later). If the answer is "yes, literal prefix forever," great.
If the answer is "we expect to swap this for a pattern later," please
leave a `// TODO: pattern support` on the struct so the next person
knows the shape may change.

Not a blocker either way.

## Process

### Please add unit tests in this PR, not as an optional follow-up

**PR test plan**, severity: **major process concern**.

The test plan argues that "config loading is straightforward" and so
no unit tests are included. I'd push back on that framing in two
ways:

First, both of the blockers above pass the manual test plan as
written. "Loaded three rules, pipeline produced output with tags" is
compatible with `Rule.Replace == nil everywhere` (you'd see *some*
tags from other code paths but not the ones the rules define) and
with `ErrInvalidRule` being unreachable by `errors.Is` (a manual test
that checks `err != nil` rather than `errors.Is(err, ErrInvalidRule)`
would see no difference between the bug and the fix). The exact
failure modes this PR introduces are invisible to the exact tests
this PR performed.

Second, straightforward code is where unit tests pay off *most*, not
least. Two tests would have caught both bugs in this PR:

1. A test that asserts `Rule.Replace["team"] == "platform"` on a
   fixture with a `replace:` block. About 15 lines.
2. A test that invokes `LoadRules` with a rule missing `name:` and
   asserts `errors.Is(err, types.ErrInvalidRule) == true`. About 10
   lines.

I am not asking for exhaustive coverage. I am asking for the two
cases the PR explicitly claims to handle correctly. Please add them
to this PR — it is faster than the follow-up would be, and it
catches this round's bugs right now.

---

## What this sample review is trying to model

- **Severity sized to the blast radius, not the fix cost.** Both
  blockers are one-line fixes, but the first causes silent data loss
  and the second defeats `errors.Is`-based routing in a dependent PR.
  One-line fixes can still be major findings; don't under-severity
  them because they're "easy to address."
- **Two verification items instead of one.** R03 has two items in
  "Things I Verified" rather than one. The first tests whether the
  learner over-generalized from R02 (nil map → nil slice). The second
  tests whether the learner checked each `fmt.Errorf` site
  individually. Both are the same skill — checking every specific
  instance rather than treating "pattern is present in the diff" as
  dispositive.
- **Silent failures are the hardest review shape.** Neither planted
  bug in this diff produces a visible runtime symptom. No panic, no
  resource leak, no loud error — just wrong behaviour that a thin
  test cannot distinguish from right behaviour. The process concern
  above makes that point explicit: a manual test plan that checks
  "it ran without error" and "I saw tags in the output" is
  compatible with both bugs being present. That is the whole
  argument for unit tests in code like this.
- **The red-herring shape for this exercise is pattern distinction,
  not visual suspicion.** R01's red herring was a rename (visible,
  structural). R02's was a type choice (visible, the PR description
  addressed it). R03's red herrings are two things the learner is
  *primed to flag* from earlier exercises. The review skill being
  trained: "knowing a pattern isn't enough; verify it applies here."
