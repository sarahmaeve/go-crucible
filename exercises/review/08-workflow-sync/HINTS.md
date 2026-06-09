# Hints for Review Exercise 08

These hints are progressive — read one at a time, try the review again,
and only open the next hint if you're still stuck.

## Hint 1: Count

There are **two** correctness issues that must be fixed before
merging. There are **two** near-twins of those issues that are fine —
one of them sits two lines below its buggy sibling — and both belong
in "Things I Verified." There is **one** process concern about the
tests, and **one** question about partial-failure semantics.

Also: one thing you might want to flag has already been addressed in
the PR description. Re-read it before you write that comment.

## Hint 2: Categories

1. A struct for the JSON hop **flattens a pointer field into a plain
   value** and tags it `omitempty`. The original type used a pointer
   for a reason. Ask: in this domain, is the zero value of that field
   a meaningful configuration or just "unset"? Compare
   `types.Strategy`'s declaration of the same field. (Exercise 15's
   lesson — and the twin two lines down has the same tag with a
   different answer.)

2. The file-walking loop opens each file and defers the close. Where
   does that defer run — per iteration, or when the whole sweep
   returns? Now run it against two thousand files. (Exercise 16's
   lesson — and the PR description tells you exactly where the loop
   came from, which is worth a comment of its own.)

## Hint 3: Lines

- `syncStrategy.FailFast bool \`json:"fail-fast,omitempty"\`` —
  GitHub's default is `fail-fast: true`, so `fail-fast: false` is a
  deliberate, meaningful choice. `types.Strategy` models it as
  `*bool` precisely so absent ≠ false. The flatten at
  `FailFast: j.Strategy.FailFast != nil && *j.Strategy.FailFast`
  destroys that distinction, and `omitempty` then deletes every
  explicit `false` during sync. Org-wide effect: every workflow that
  opted out of fail-fast silently opts back in on the next cron run.
  The fix is to keep `*bool` through the intermediate (exactly as
  `types.Strategy` does).

- `MaxParallel int \`json:"max-parallel,omitempty"\`` — same keyword,
  two lines down, **fine**: `max-parallel: 0` is not a meaningful
  GitHub configuration (unset means unlimited), so dropping the zero
  is correct. The verdict on `omitempty` is domain semantics, not
  syntax. Verify, don't flag.

- `SyncWorkflows`'s loop: `f, err := os.Open(path)` … `defer f.Close()`
  inside `for _, entry := range entries`. Every descriptor stays open
  until the function returns; at fleet scale the sweep dies with
  `too many open files` partway through — after having already
  rewritten half the tree. `lint.LintWorkflows` has the identical
  latent loop (the PR adapted it); worth a note to fix the original
  too. Simplest fix here: `data, err := os.ReadFile(path)` — no
  handle to manage at all (and note `os.WriteFile` two lines later
  needs no Close either; don't flag it).

- The tests: every fixture either omits `fail-fast` or the test
  doesn't assert on it, and three files can't exhaust descriptors.
  Ask for a `fail-fast: false` round-trip assertion (it fails today)
  and either the ReadFile refactor or a many-files test.
