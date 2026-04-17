# Hints for Review Exercise 02

Progressive hints — read one at a time, re-read the diff, and only
open the next hint if you're still stuck.

## Hint 1: Count

There are **two** correctness issues in this diff that warrant blocking
the merge. There is **one** thing worth *verifying but not flagging*
(the review template's "Things I Verified" section), and **one**
process concern related to the PR's test coverage.

If your draft review has zero correctness issues, or four-plus, take
another pass — you're either missing something or over-flagging.

Note: the PR *does* include a unit test. Resist the temptation to check
the "tests exist" box and move on. Read the test critically — what
does it assert, and what does it not exercise?

## Hint 2: Categories

Without naming the specific lines:

1. One issue is about **what the YAML decoder actually populates**.
   A field is declared, looks decorated with a `yaml:` tag — and is
   nonetheless silently ignored by the decoder. Callers get a zero
   value on that field and have no way to know the input was lost.
2. The other issue is about **a map variable** that is declared but
   never initialised before writes are attempted. The first write at
   runtime panics.

Both patterns are in the basic tier. One is in a helper function that
inverts a map; the other is in the intermediate decoding struct.

## Hint 3: Files and lines

- **`internal/parser/annotations.go:22`** — look at the `rawAnnotation`
  struct. Three fields are declared. Are all three spelled the way Go
  requires in order for the YAML decoder to populate them? The `yaml`
  struct tag on each field is the decoder's *label*; the decoder only
  populates fields it can access via reflection. What access rule
  applies to struct fields?

- **`internal/parser/annotations.go:48`** — look at `BuildOwnerIndex`.
  The first statement inside the function declares `index` with `var`.
  Three lines later, the code writes to `index[ann.Owner]`. What is
  the runtime state of a `var m map[...]...` that has never had
  `make` called on it?

- **Test coverage (`internal/parser/annotations_test.go`)** — the test
  asserts `Owner` on two annotations. What field does it never
  inspect? What function does it never call? A good review flags
  these gaps, because the bugs above would pass this test even after
  a half-fix.

- **The red herring** is about the *type* chosen for one of the new
  fields. The PR description addresses it explicitly — re-read the
  "Scope note" in the PR summary before flagging this as an issue.
