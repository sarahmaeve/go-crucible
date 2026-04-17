# Hints for Review Exercise 01

These hints are progressive — read one at a time, try the review again,
and only open the next hint if you're still stuck.

## Hint 1: Count

There are **two** correctness issues in this diff that rise to the level
of "please change before merging." There is also **one** change you
should *verify but not flag* (the review template has a "Things I
Verified" section — that is where it goes), and **one** process concern
worth raising in the PR comments.

If your draft review has one or zero correctness issues, look again.
If your draft has four or five correctness issues, you are probably
over-flagging — one of the things that looks wrong is actually fine.

## Hint 2: Categories

Without naming the specific lines, the two correctness issues are:

1. An error-handling issue in a new helper function — the helper notices
   a problem, writes about it, and then continues as if nothing
   happened. Callers have no way to know they got a bogus result.
2. A resource-lifecycle issue in a new helper function — a resource is
   opened but never closed. A single run is fine; a cron loop invoking
   the binary repeatedly, or a long-running daemon, accumulates open
   handles.

If those descriptions sound familiar from the numbered exercises, that
is intentional. Both patterns are in the basic tier.

## Hint 3: Files

- The error-handling issue is in `internal/audit/since.go`. Look at the
  first function defined in the new file. What does it do when the
  input cannot be parsed? What does the *caller* see?

- The resource-lifecycle issue is also in `internal/audit/since.go`.
  Look at the second function defined in the new file. It opens
  something with `os.Open`. What is conspicuously absent on the next
  line?

- The change you should verify but not flag is in
  `internal/audit/report.go`. It is a rename. Ask yourself: is the
  rename consistent within the diff? Are all usages that the diff
  shows also updated? If yes, this is a benign cleanup — say so in
  "Things I Verified" and move on.

- The process concern is in the PR description itself, not the code.
  Re-read the test plan.
