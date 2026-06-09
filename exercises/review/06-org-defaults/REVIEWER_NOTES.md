# Sample Review — PR #236

This is one reasonable review of the org-defaults PR. Yours will differ
in tone, phrasing, and which process concerns you raise. Compare the
*substance* (did you catch the wrong return and the shared-map write,
did you separate the benign twins from the bugs?) rather than the
wording.

## Overall assessment

**Request changes.**

The design intent is good — one `OrgDefaults` value as the single
source of org policy, layered onto the existing `BaseTemplate` by
embedding. But the two delivery paths each have a defeating bug:
`BuildOrgTemplate` returns the embedded base instead of the org
template (so its workflows carry no org policy at all), and
`OrgTemplate.Generate` writes the per-repo `REPO` entry into the
*shared* defaults map (so bulk-generated workflows all alias one map
and end up with the last repo's value). The included test is
structurally unable to see either. All fixable with small diffs.

## Blockers

### 1. `BuildOrgTemplate` returns the embedded `BaseTemplate` — org policy silently absent

**`internal/generate/org.go:42-49`**, severity: **major**.

```go
func BuildOrgTemplate(repo string, defaults OrgDefaults) Template {
	t := OrgTemplate{ ... }
	return t.BaseTemplate
}
```

`BaseTemplate` also satisfies `Template`, so this compiles — but the
value that crosses the interface boundary is the embedded base, and
method dispatch follows the value. Callers of
`BuildOrgTemplate(...).Generate()` run `BaseTemplate.Generate()`: a
valid workflow with **no** org env block, **no** `REPO` entry, and
**no** permissions baseline. For a PR whose purpose is "the security
team edits one struct and every repo gets the policy," this is the
worst failure shape: the output looks plausible and lints clean, it
just silently lacks the policy. Nobody notices until an audit does.

**Suggested fix:** `return t`.

### 2. `Generate` writes into the shared defaults map — bulk generation cross-contaminates

**`internal/generate/org.go:33-35`**, severity: **major**.

```go
env := o.Defaults.Env
env["REPO"] = o.Repo
wf.Env = env
```

`env := o.Defaults.Env` copies the map *reference*, not the map. The
next line writes `REPO` into the caller's shared `OrgDefaults.Env`,
and `wf.Env = env` aliases that same map into the returned workflow.
Trace `GenerateAll([]string{"repo-a", "repo-b"}, defaults)`:

1. Iteration one: shared map gets `REPO=repo-a`; workflow one's `Env`
   *is* the shared map.
2. Iteration two: shared map gets `REPO=repo-b` — which retroactively
   changes workflow one's `Env` too, since both alias one map.

Every generated workflow reports the **last** repo's `REPO`, and the
caller's `defaults` value is mutated as a side effect — anything else
generated from it afterwards inherits a stray `REPO`. There is also a
latent panic here: if a caller constructs `OrgDefaults` without an
`Env` map, `env["REPO"] = ...` is a write to a nil map.

**Suggested fix:** build a fresh map per workflow; this fixes the
aliasing, the caller-visible mutation, and the nil case in one move:

```go
env := make(map[string]string, len(o.Defaults.Env)+1)
for k, v := range o.Defaults.Env {
	env[k] = v
}
env["REPO"] = o.Repo
wf.Env = env
```

(`maps.Clone` is fine too, but note it returns nil for a nil source,
so you'd still need the make-then-set shape for the `REPO` write.)

## Suggestions

### Defensively copy `Permissions` as well

**`internal/generate/org.go:36`**, severity: **minor**.

`wf.Permissions = o.Defaults.Permissions` aliases the shared map the
same way the `Env` line does. Nothing in this diff writes through it,
so it is not corrupting anything **today** — which is why this is a
suggestion and not blocker 3. But every generated workflow holding a
reference into the shared defaults is a mutation-at-a-distance trap
for the next feature (the first "per-repo permissions override" will
recreate blocker 2 here). Cloning it costs three lines.

### `BuildOrgTemplate` and `GenerateAll` duplicate construction

**`internal/generate/org.go:42-49, 53-60`**, severity: **minor**.

The `OrgTemplate{...}` literal appears in both. Having `GenerateAll`
call `BuildOrgTemplate` (post-fix) keeps the two paths from drifting —
right now a future field added in one literal and not the other would
be exactly the kind of silent divergence this review is full of.

## Questions

### What is the intended precedence when env keys collide?

`BaseTemplate.Generate` currently produces no `Env`, so there is no
collision today — but the obvious next step is repo-level or
template-level env. When a repo defines `CACHE_BUCKET` itself, who
wins: the org default or the repo? Nothing in the diff or description
decides. Whichever the answer, it deserves a sentence on `OrgDefaults`
and a test, before someone encodes the opposite assumption. Asking,
not blocking.

## Nits

None worth the author's time.

## Things I Verified

### `o.BaseTemplate.Generate()` inside `Generate` is the correct pattern, not the exercise-11 trap

**`internal/generate/org.go:28`.** The same `x.BaseTemplate` expression
shape as blocker 1 — but here it is exactly how embedding is meant to
be used: explicitly invoke the embedded type's method to produce the
base output, then layer the org defaults on top of the result. The
difference is what the expression is *for*: delegating to the base
inside the outer method (fine) versus handing the base to the caller
*as* the whole (blocker 1). The expression is not the bug; the value
crossing the interface boundary is.

### Value receiver on `OrgTemplate.Generate` is consistent

**`internal/generate/org.go:27`.** `BaseTemplate.Generate` and
`AdvancedTemplate.Generate` both use value receivers; `OrgTemplate`
matching them keeps the method-set story simple (both `OrgTemplate`
and `*OrgTemplate` satisfy `Template`). Consistent with the package.

### `make([]types.Workflow, 0, len(repos))` + append

**`internal/generate/org.go:54`.** Pre-sized, appended in order,
returned once — no aliasing concerns on the slice side. Fine.

## Process

### The test fixture is structurally unable to catch either blocker

**PR test plan + `internal/generate/org_test.go`**, severity:
**major process concern**.

Two observations, two asks:

1. The test calls `GenerateAll` with **one** repo. At fixture size
   one, "all workflows share one mutated map" and "each workflow has
   its own map" produce identical output — the last writer is the
   only writer. The manual three-repo test *would* have shown three
   identical `REPO` values, but nobody diffed the outputs. **Ask:**
   a two-repo test asserting `wfs[0].Env["REPO"] == "repo-a"` and
   `wfs[1].Env["REPO"] == "repo-b"`, plus an assertion that
   `defaults.Env` does not contain `"REPO"` afterwards (catches the
   caller-visible mutation).
2. `BuildOrgTemplate` — the public constructor this PR adds — is never
   invoked by any test, so blocker 1 never executes under test.
   **Ask:** a test that builds via `BuildOrgTemplate`, calls
   `Generate()`, and asserts the org env and permissions are present.
   On the current code it fails with an empty `Env` — which is the
   bug, made visible.

---

## What this sample review is trying to model

- **Telling twins apart.** Two `x.BaseTemplate` expressions, two
  map-aliasing assignments — in each pair, one is a bug and one is
  not, and the difference is articulated, not vibed. "Things I
  Verified" is where the benign twin goes, with the reason.
- **Severity calibration on the same pattern.** The `Env` alias is a
  blocker (live write, cross-contamination); the `Permissions` alias
  is a suggestion (no write today, trap tomorrow). Same shape,
  different consequence, different severity — flagging both as
  blockers would be over-flagging.
- **Fixture-size reasoning.** The review doesn't just say "add more
  tests"; it explains why size-one input *cannot* express the failure
  and names the exact assertions that make both bugs visible.
- **Naming the worst failure shape.** Output that looks right and
  silently lacks the policy is called out as worse than a crash —
  that judgement, not the language rule, is what convinces an author
  the fix is urgent.
