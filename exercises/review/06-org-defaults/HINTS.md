# Hints for Review Exercise 06

These hints are progressive — read one at a time, try the review again,
and only open the next hint if you're still stuck.

## Hint 1: Count

There are **two** correctness issues that must be fixed before merging,
**two** near-twins of those issues that are fine (or merely worth a
suggestion) — both belong in "Things I Verified" — plus **one** process
concern about the included test and **one** question about intent.

This exercise is deliberately built from doubles: each bug has a
look-alike in the same diff that is not a bug. Severity calibration is
the skill being tested.

## Hint 2: Categories

Without naming lines, the two correctness issues are:

1. A function builds the right value and returns the wrong *part* of
   it. The compiler is satisfied because the part also implements the
   interface — but the part's method runs instead of the whole's, and
   everything this PR exists to add is silently absent from the
   output. Exercise 11's lesson, in diff form.

2. A "copy" of a shared map that isn't a copy. Mutating it writes
   into the shared defaults, and every workflow generated from those
   defaults aliases the same map — so they all end up with the *last*
   writer's value. The corruption only manifests when more than one
   workflow is generated... which is the one thing the included test
   never does. Exercise 07's lesson, in diff form.

## Hint 3: Lines

- `BuildOrgTemplate`: `return t.BaseTemplate`. The method set that
  satisfies `Template` here is `BaseTemplate`'s — so callers get
  `BaseTemplate.Generate()`, which knows nothing about env defaults,
  permissions, or `REPO`. Every workflow built through this
  constructor silently lacks the org policy this PR exists to roll
  out. Fix: `return t`.

- `OrgTemplate.Generate`: `env := o.Defaults.Env` copies a map
  *reference*. `env["REPO"] = o.Repo` then writes into the shared
  `OrgDefaults`. Trace `GenerateAll(["repo-a", "repo-b"], defaults)`:
  iteration one sets `REPO=repo-a` in the shared map and aliases it
  into workflow one's `Env`; iteration two overwrites the same map
  with `REPO=repo-b`. Both workflows now read `repo-b`, and the
  caller's `defaults.Env` has been polluted as a bonus. Fix: build a
  fresh map and copy entries (or `maps.Clone` plus a nil-check) —
  which also fixes the latent panic when `Defaults.Env` is nil.

- `o.BaseTemplate.Generate()` inside `OrgTemplate.Generate` is the
  *correct* embedding usage: explicitly invoking the embedded type's
  method to produce the base output, then layering on top. The bug is
  what `BuildOrgTemplate` *returns*, not how `Generate` *delegates*.

- `wf.Permissions = o.Defaults.Permissions` aliases the shared map
  exactly like the `Env` line — but nothing in this diff ever writes
  through it, so it corrupts nothing today. Suggest a defensive copy;
  don't give it the same severity as the `Env` site.

- The test: one repo. At fixture size one, "every workflow shares one
  map" and "every workflow has its own map" are indistinguishable —
  the last writer is the only writer. And `BuildOrgTemplate` is never
  called at all, so the wrong-return bug never executes under test.
