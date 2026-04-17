# Sample Review — PR #173

This is one reasonable review of the `--since` flag PR. Yours will
differ in tone, phrasing, and which process concerns you raise. Compare
the *substance* (did you catch the correctness issues, did you avoid
over-flagging the rename, did you raise the test-plan concern?) rather
than the wording.

## Overall assessment

**Request changes.**

Two correctness issues — a silently-swallowed parse error and an
unclosed file handle — both in the new helpers in
`internal/audit/since.go`. Neither is hard to fix, but both will cause
real problems in the cron-loop workflow this PR is designed to support.
I'd also like to see the unit tests land in this PR rather than as a
follow-up.

## Blockers

### 1. `ParseSince` swallows the parse error silently

**`internal/audit/since.go:14-19`**, severity: **major**.

```go
func ParseSince(raw string) time.Time {
    d, err := time.ParseDuration(raw)
    if err != nil {
        slog.Error("invalid --since value", "value", raw, "error", err)
    }
    return time.Now().Add(-d)
}
```

When `raw` cannot be parsed, `time.ParseDuration` returns `d == 0` and a
non-nil error. The function logs the error and then continues as if
nothing went wrong, returning `time.Now().Add(0) == time.Now()`. The
caller has no way to distinguish "since=0 seconds ago" (a meaningful
value) from "the user typed nonsense." `FilterSince` will then drop
every finding whose `DetectedAt` is not in the future — i.e. all of
them — and the operator running with a typo'd `--since 24hrs` (note:
`"hrs"` does not parse) will silently see an empty report.

**Suggested fix:** change the signature to
`func ParseSince(raw string) (time.Time, error)` and have `main.go`
print the error and exit non-zero. Alternatively, if you want to keep
the single-return shape for ergonomics, have `main.go` call
`log.Fatalf` on invalid input — but the current form, where the
function hides the error from the caller, is not acceptable.

### 2. `LoadLastRun` leaks a file handle

**`internal/audit/since.go:25-37`**, severity: **major**.

```go
func LoadLastRun() time.Time {
    path := stateFilePath()
    f, err := os.Open(path)
    if err != nil {
        return time.Time{}
    }

    var state struct {
        LastRun time.Time `json:"last_run"`
    }
    if err := json.NewDecoder(f).Decode(&state); err != nil {
        slog.Warn("state file unreadable", "path", path, "error", err)
        return time.Time{}
    }
    return state.LastRun
}
```

`f` is opened successfully but there is no `defer f.Close()`. A single
invocation of the binary is fine — the OS reclaims the FD at exit —
but the stated motivation for this PR is cron-loop and daemon use.
A long-running caller that invokes `LoadLastRun` repeatedly leaks one
FD per call and will eventually hit `EMFILE` on the file descriptor
limit.

**Suggested fix:** add `defer f.Close()` immediately after the
error-check for `os.Open`. Either return path below that — successful
decode or failed decode — then triggers the close.

## Suggestions

### Consider returning the `Close` error from `LoadLastRun`

**`internal/audit/since.go:25-37`**, severity: **minor**.

Close-errors on read-only files are almost always benign, but if you
later extend `LoadLastRun` to write state back, a silently-ignored
Close error could mask a lost write. You may want to surface it now
rather than retrofit later. Not a blocker.

### `ParseSince` cutoff semantics are confusing relative to `LoadLastRun`

**`internal/audit/since.go` + `cmd/kube-patrol/main.go:52-57`**,
severity: **minor**.

Both code paths produce a `cutoff time.Time`, but they mean subtly
different things:

- `ParseSince("24h")` returns `now - 24h` — "findings newer than 24
  hours ago."
- `LoadLastRun()` returns the timestamp stored in state — "findings
  newer than the previous successful run."

Those are both reasonable but the naming could make it clearer. A
`FindingsSinceCutoff` helper that wraps both, or a renamed local
(`oldestWanted` instead of `cutoff`?), would help the next person
reading `main.go`.

## Questions

### How does first-run behaviour interact with `FilterSince`?

**`internal/audit/since.go:34`**, `LoadLastRun` returns `time.Time{}`
(the zero time, January 1, Year 1) when the state file does not exist.
`FilterSince` then keeps every finding whose `DetectedAt.After(zero)`
is true — which is every finding. Good: first-run behaviour is
"report everything," which matches the PR description.

Not a change request — I just want to confirm this was the intent, and
that you have a manual test that covers it. If yes, please add a line
in the test plan to document the first-run case.

## Nits

None worth the author's time.

## Things I Verified

### The `FindingCount` → `NumFindings` rename is consistent

**`internal/audit/report.go:10-17`**.

The rename shows up in three places within the diff:

1. The struct field on line 11 (was `FindingCount`, now `NumFindings`).
2. The field reference inside `Summary()` on line 17.
3. (Implied by the PR description: "all call sites updated" — not
   shown in this diff but asserted by the author.)

The rename is internal to the `audit` package (the struct type is
exported but the field change is consistent within the diff), and the
PR description asserts all call sites outside the diff are updated. I
checked every reference to `FindingCount` and `NumFindings` *within*
the diff and they are consistent. No functional concern.

A process-style observation: you might consider splitting the rename
into its own PR — it is unrelated to the `--since` feature and
inflates the review surface. This is a tone call, not a requirement.
If the project convention is "small PRs get merged together," ignore
me.

### `FilterSince` uses `After` rather than `!Before`

**`internal/audit/since.go:41`**.

`f.DetectedAt.After(cutoff)` is correct and idiomatic. I briefly
wondered whether the boundary case (a finding with `DetectedAt ==
cutoff`) should be included or excluded, but `After` returns `false`
for equality and "findings strictly newer than the last run" is the
right semantic for avoiding duplicate reports. Good.

## Process

### Please land unit tests in this PR, not as a follow-up

**PR test plan**, severity: **major process concern**.

The test plan lists "Unit tests for `ParseSince` and `LoadLastRun` —
follow-up." Given that both helpers have correctness issues that
a unit test would have caught (a test that asserts `ParseSince`
returns an error on malformed input; a test that invokes `LoadLastRun`
many times and asserts no FD leak via `testing.AllocsPerRun` or an
equivalent), I'd like to see the tests land alongside the code. A
follow-up PR for "tests for last week's feature" rarely lands with the
same priority as the feature itself.

If time is genuinely tight, a single table-driven test for
`ParseSince` covering parseable, unparseable, and empty inputs is
probably 20 lines and unblocks me. `LoadLastRun` is harder to test
(touches the filesystem) but even an integration-style test that
creates a state file, reads it, and checks the returned timestamp
would add a lot of confidence.

---

## What this sample review is trying to model

- **Two blockers cited with file/line references and a suggested fix.**
  A review that only says "this is wrong" without proposing a direction
  costs the author another round trip. Proposing a fix is not
  mandatory, but on a PR where the fix is obvious, withholding it
  wastes time.
- **A Questions section separate from Blockers.** Not every uncertainty
  is a problem. Sometimes you want clarification; that goes in
  Questions. Don't block merges on your own uncertainty.
- **A Things I Verified section.** The rename would have been an easy
  over-flag. A careful reviewer checks, confirms consistency, and
  *says so* — both to help the author and to demonstrate that the
  review was thorough.
- **A process concern separated from correctness.** "Please add tests"
  is not the same kind of comment as "this leaks a file descriptor."
  Labelling them differently helps the author prioritise what to
  address first.
- **Tone.** The review is direct about the bugs and generous about the
  author's intent. "This will silently drop every finding" is stronger
  than "maybe consider handling this differently" — on a correctness
  issue, be clear. But on the test-plan note, the reviewer offers a
  small concession ("a single table-driven test is probably 20 lines")
  that acknowledges the author's time constraint.
