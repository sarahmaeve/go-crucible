# Sample Review — PR #251

This is one reasonable review of the workflow-sync PR. Yours will
differ in tone and emphasis. Compare the *substance*: did you catch
both fleet-scale bugs, separate the two `omitempty` verdicts, and
respect the PR description's pre-answered concern?

## Overall assessment

**Request changes.**

The sync design is right, and reusing the linter's file-matching
logic keeps the tools consistent. But both real problems here only
show up at the scale this command exists for: the flattened
`fail-fast` field means every workflow that deliberately set
`fail-fast: false` gets silently flipped back to GitHub's default on
the first org-wide run, and the defer-in-loop file handling means the
sweep itself dies with `too many open files` partway through a
fleet-sized directory — after rewriting half of it. Desk testing
cannot see either; that's exactly why they need to be fixed before
the cron job exists.

## Blockers

### 1. `fail-fast: false` is silently deleted — and the org default flips it to `true`

**`internal/parser/sync.go`, `syncStrategy` + the flatten in
`normalizeWorkflow`**, severity: **critical**.

```go
FailFast bool `json:"fail-fast,omitempty"`
...
FailFast: j.Strategy.FailFast != nil && *j.Strategy.FailFast,
```

`types.Strategy` models this field as `*bool` deliberately: GitHub's
default is `fail-fast: true`, so an explicit `fail-fast: false` is a
meaningful opt-out (teams set it so one flaky matrix cell doesn't
cancel the others). The flatten collapses "absent" and "explicitly
false" into the same `false`, and `omitempty` then deletes that
`false` during serialisation. After one sync, every opted-out
workflow in the org has no `fail-fast` key — which GitHub reads as
`true`. The change is invisible in the diff of any single repo unless
someone knows to look, and the sync is idempotent, so it will
re-apply itself from cron forever.

**Suggested fix:** keep the pointer through the intermediate, exactly
as `types.Strategy` does:

```go
FailFast *bool `json:"fail-fast,omitempty"`
...
FailFast: j.Strategy.FailFast,
```

With a pointer, `omitempty` drops only nil (truly absent), and both
`true` and `false` survive the round-trip. The "pointer fields make
edits awkward" comment is solving the wrong problem — the awkwardness
is the feature.

### 2. The sweep leaks one file descriptor per file — and dies mid-fleet

**`internal/parser/sync.go`, `SyncWorkflows` loop**, severity:
**major**.

```go
f, err := os.Open(path)
...
defer f.Close()
data, err := io.ReadAll(f)
```

`defer` is function-scoped: none of these closes run until
`SyncWorkflows` returns, so the sweep holds every descriptor it has
ever opened. At eleven files (the manual test) that's invisible; at
the couple of thousand files this command is being built for, the
process hits the descriptor limit and aborts partway — *after* having
already rewritten the files it got through, leaving the fleet
half-synced.

**Suggested fix:** there's no reason to hold a handle at all:

```go
data, err := os.ReadFile(path)
```

reads and closes internally. (If you keep the streaming form for
symmetry with the linter, close per iteration by extracting the body
into a helper function.)

**Related:** the PR description says this loop was adapted from
`lint.LintWorkflows` — which has the identical latent defer-in-loop
(`internal/lint/linter.go`). Worth a follow-up ticket to fix the
original; copied code carries its copied bugs, and the linter is one
big-directory run away from the same failure.

## Suggestions

### Make the rewrite atomic per file

**`internal/parser/sync.go`**, severity: **minor**.

`os.WriteFile` truncates in place; a crash mid-write leaves a
corrupt workflow file in a repo we don't own the recovery story for.
Write to a temp file in the same directory and `os.Rename`. Cheap
insurance for a tool that edits two thousand files from cron.

## Questions

### What are the partial-failure semantics?

Any parse or write error aborts the whole sweep with files 1..k
already rewritten and the rest untouched — and one malformed workflow
in one repo means the org never gets a complete sync. Options:
skip-and-report (like the linter skips invalid YAML), or
validate-everything-then-write. Which is intended? Whichever it is,
the doc comment should say so, because cron will exercise it weekly.

## Nits

None worth the author's time.

## Things I Verified

### `max-parallel` with `omitempty` is correct — same keyword, different verdict

**`internal/parser/sync.go`, `syncStrategy.MaxParallel`.** Two lines
below the `fail-fast` bug, the same `omitempty` tag is *right*:
`max-parallel: 0` is not a meaningful GitHub configuration (unset
means unlimited; zero is not a valid limit), so dropping the zero
value loses nothing. The verdict on `omitempty` is never syntactic —
it's "is this zero a configuration?" — and the answer differs per
field even within one struct.

### Comment and key-order loss — pre-answered in the PR description

The round-trip discards YAML comments and ordering, which would
normally be a major concern for files humans edit. The description
declares synced files machine-owned and accepts the loss explicitly.
Verified against the description; not flagged. (R02's lesson: read
the description before writing the comment.)

### `os.WriteFile` needs no Close

The write side has no handle to leak — `WriteFile` opens, writes, and
closes internally. Only the read side has the bug.

## Process

### The fixtures are structurally blind to both blockers

**PR test plan + `internal/parser/sync_test.go`**, severity: **major
process concern**.

Three fixture files cannot exhaust descriptors, and none of the
fixtures asserts on `fail-fast` — `matrix.yml`'s test checks that
matrix *dimensions* survive, not strategy *settings*. Two asks:

1. A fixture with `fail-fast: false` and an assertion that the synced
   output still contains it. This test fails on the current code and
   is the regression guard for blocker 1.
2. Either the `os.ReadFile` refactor (making the descriptor question
   moot) or a test that syncs a few hundred generated files under a
   lowered `RLIMIT_NOFILE`. If neither is practical, at minimum a
   comment in the loop acknowledging the constraint.

---

## What this sample review is trying to model

- **Reviewing at deployment scale, not diff scale.** Both bugs are
  invisible in the hunks and obvious in the fleet. The reviewer's
  question is never "does this code look right?" but "what happens
  when this runs where it's actually going to run?"
- **Per-field verdicts on the same pattern.** `omitempty` appears
  twice in one struct: one critical bug, one correct choice, decided
  entirely by domain semantics. Pattern-matching the keyword in
  either direction — flag both, pass both — gets one of them wrong.
- **Following provenance.** "Adapted from LintWorkflows" is a gift to
  the reviewer: it says where else the same bug lives. Reviews that
  chase copied code to its source fix two bugs for the price of one.
- **Honouring the description.** The comment-loss concern was
  answered before it was asked. Flagging it anyway tells the author
  you didn't read their writeup — and costs credibility the real
  blockers need.
