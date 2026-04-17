# Hints for Review Exercise 03

Progressive hints — read one at a time, re-read the diff, and only
open the next hint if you're still stuck.

## Hint 1: Count

There are **two** correctness issues in this diff that warrant blocking
the merge. There are **two** things worth *verifying but not flagging*,
and **one** process concern.

A specific warning for this exercise: if your draft review has more
than two blockers, you are probably pattern-matching from earlier
exercises without checking the specifics. Re-read the
README.md — *"Check before you flag"* is there for a reason.

## Hint 2: Categories

Without naming the lines:

1. One correctness issue is about the YAML decoder and a struct field
   that cannot be reached via reflection.
2. One correctness issue is about an error-wrapping site that breaks
   the chain, so `errors.Is(err, ErrInvalidRule)` will return `false`
   at callers. Note the word *site* — this diff has multiple
   `fmt.Errorf` calls. You need to evaluate each one independently,
   not decide on the group.

The two things to verify but not flag:

3. One involves a variable declared with `var` that is then used in an
   append loop. Compare Go's rules for nil slices against Go's rules
   for nil maps before deciding whether to flag.
4. One involves the *majority* of `fmt.Errorf` calls in the diff.
   They look like the bug until you look carefully; most are in fact
   correct.

## Hint 3: Files and lines

- **`internal/transform/rules.go:21`** — look at the `rawRule` struct.
  Three fields, three `yaml:` struct tags. Are all three field names
  spelled the way Go's reflect package requires in order to populate
  them from YAML?

- **`internal/transform/rules.go:38`** — look at the `fmt.Errorf` call
  inside the loop in `LoadRules`. `validateRule` returns an error that
  wraps `types.ErrInvalidRule` with `%w`. What does the wrapping in
  this caller do to that chain? A reviewer downstream calls
  `errors.Is(err, types.ErrInvalidRule)` on the result — does it
  return `true` or `false` after the outer wrap?

- **`internal/transform/rules.go:34`** — `var out []Rule` followed by
  `out = append(out, ...)` in the loop. Does this panic at runtime?
  Compare to R02's nil-map bug. The rules for nil *slices* are different
  from the rules for nil *maps* — `append` on a nil slice is defined
  to return a new slice, not to panic. Verify, note in "Things I
  Verified," and do not flag.

- **`internal/transform/rules.go:31, 51, 55`** — the three other
  `fmt.Errorf` calls. Look at each one individually. What verb does
  each use? Is the wrapping correct at each site? Three of the four
  `fmt.Errorf` calls in this diff are correct.

- **The test plan** — the author argues that "straightforward" code
  doesn't need unit tests. Do you agree? In particular, ask whether a
  silent-failure bug (one where no runtime exception fires, the code
  just does the wrong thing) is more or less likely to be caught by
  manual testing than by a unit test.
