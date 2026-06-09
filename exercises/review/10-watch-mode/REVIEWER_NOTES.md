# Sample Review — PR #189

This is one reasonable review of the watch-mode PR. Yours will differ
in tone, order, and emphasis. Compare the *substance*: did you find
all three bugs across the tiers, name the detonation chain in your
top-line assessment, recognise that the panic guard fails on two
independent axes, and clear the things that deserved clearing?

## Overall assessment

**Request changes.**

The structure is right — one Watcher type, the existing auditors
reused unchanged, state under a mutex, the post-ticker idioms done
properly. But three issues block it, and two of them chain: the
`--diff` path writes to a map that is never initialised, so the first
delta cycle panics — and the panic guard that the doc comments
promise twice cannot fire, both because the `recover()` sits one
frame too deep and because auditor panics happen in goroutines the
guard doesn't live in. Net effect: `--watch --diff` is a crash loop
on first use, in the daemon that watches everything else. The third
issue, `wg.Add` inside the spawned goroutine, makes cycle totals
intermittently lie in the meantime. All three fixes are small.

## Blockers

### 1. `lastCounts` is never initialised — the first `--diff` cycle panics

**`internal/audit/watch.go`, `NewWatcher` + the diff block in
`runCycle`**, severity: **critical** (as the detonator of the chain
below).

`NewWatcher` makes `reports` but not `lastCounts`. The only write —

```go
w.lastCounts[ns] = r.Summary.Total
```

— is gated behind `if w.diff`, so plain watch mode never touches it
and works fine; the first cycle of any `--diff` run dies with
`assignment to entry in nil map`. Note the read two lines earlier
(`w.lastCounts[ns]`) is *legal* — nil maps read as empty — which is
exactly why this survives a skim: the panic is on the write, four
lines into the feature's selling point.

**Suggested fix:** `lastCounts: make(map[string]int),` in the
constructor. (And see blocker 3 for why this panic is fatal rather
than logged.)

### 2. `wg.Add(1)` inside the goroutine — cycles intermittently report a subset

**`internal/audit/watch.go`, the fan-out in `runCycle`**, severity:
**major**.

```go
go func(ns string) {
	wg.Add(1)
	defer wg.Done()
```

`Wait` is only required to wait for goroutines that have already
called `Add`. With `Add` as the goroutine's first statement, the main
goroutine can reach `wg.Wait()` before the scheduler has started any
worker — `Wait` sees a zero counter and returns. The cycle then
aggregates whatever subset of namespaces happens to have finished:
`total_findings=0` lines on a healthy cluster, and with `--diff`,
phantom negative deltas followed by phantom positive ones as reports
flap in and out of the aggregate. Operators will chase "what changed
in namespace X?" when the answer is "your monitor raced itself."

**Suggested fix:** move `wg.Add(1)` above the `go` statement. Worth
knowing: `go vet` flags this exact pattern ("WaitGroup.Add called
from inside new goroutine") — see the process section.

### 3. The panic guard cannot fire — for two independent reasons

**`internal/audit/watch.go`, the deferred guard in `runCycle` +
`recoverCyclePanic`**, severity: **critical** (it converts blocker 1
from a logged error into a crash loop).

```go
defer func() {
	w.recoverCyclePanic()
}()
```

First: `recover()` only returns the panic value when called
*directly* by a deferred function. Here the deferred function is the
anonymous closure; `recoverCyclePanic` — where `recover()` lives — is
one frame below it, so `recover()` returns nil on every execution.
The guard runs, recovers nothing, logs nothing.

Second, and independently: the auditors execute in the per-namespace
goroutines, and a panic can only be recovered by a deferred call **in
the goroutine that is panicking**. Even with the frame fixed —
`defer w.recoverCyclePanic()` — a panicking auditor still kills the
process, because nothing inside the spawned goroutine recovers. The
doc comments promise "a panicking auditor must not take the daemon
down" twice; as written, the guard can keep that promise for neither
auditor panics (wrong goroutine) nor its own aggregation code (wrong
frame).

**Suggested fix:** both halves:

```go
// in runCycle, guarding the aggregation on this goroutine:
defer w.recoverCyclePanic()

// and inside the per-namespace goroutine, first line:
defer w.recoverCyclePanic()
```

(One method, deferred directly, in each goroutine that needs
protection. If per-namespace recovery should also mark that
namespace's report as failed, thread that through — but that's a
design extension, not the fix.)

### The chain — say it in the merge decision

Enable `--diff` → first cycle hits blocker 1's nil-map write → the
panic unwinds through blocker 3's non-functional guard → the daemon
dies → the supervisor restarts it → the first cycle panics again.
A crash loop in the tool that monitors everything else, triggered by
its headline flag, contained by nothing. Individually these are a
one-line init bug and a misplaced defer; together they are the
feature failing closed on first contact. Blockers 1 and 3 should land
in the same commit, with a test that proves the combination (a
`--diff` cycle with a panicking auditor) survives.

## Suggestions

### Consider a per-cycle timeout

**`internal/audit/watch.go`, `runCycle`**, severity: **minor**.

A hung API call in one namespace stalls the whole cycle indefinitely
(`wg.Wait` has no deadline), and the ticker quietly coalesces missed
ticks — the daemon looks alive while auditing nothing. A
`context.WithTimeout(ctx, w.interval)` around each cycle bounds the
damage and makes the stall visible as an auditor error.

### Surface staleness when keeping the previous report

The stale-beats-wrong choice (below) is sound, but nothing records
*how* stale a kept report is. A `LastUpdated time.Time` per namespace
(logged on each cycle, or in `TotalFindings`'s successor) lets the
alert-routing integration distinguish "clean" from "unable to audit
for three days." Non-blocking.

## Questions

### What should `--diff` deltas mean across a failed cycle?

When an auditor fails and the previous report is kept, the next
successful cycle's delta is computed against the *kept* count — so a
finding that appeared and resolved during the gap never shows up in
any delta. For triage logs that's probably fine; for the planned
alert-routing it may not be. Is delta-across-gaps the intended
semantics? A sentence in the doc comment either way.

## Nits

None worth the author's time.

## Things I Verified

### The ticker idiom is correct

`time.NewTicker` + `defer ticker.Stop()` + an immediate first cycle
before the loop — the right shape for a daemon interval (and the one
`internal/ingest/ticker.go` uses). No per-iteration `time.After`,
nothing leaks. Clear.

### The nil-map *read* is legal — the bug is only the write

`w.lastCounts[ns]` on the read side returns zero for a nil map; first
-cycle deltas would correctly compute against 0 if the write below it
didn't panic first. Worth saying explicitly because it's why the diff
block *looks* symmetrical and safe.

### Stale-beats-wrong on auditor error is a defensible call

The error path logs and returns without touching `w.reports[ns]`,
keeping the previous cycle's report, with a comment saying exactly
that. For paging decisions, a stale count beats a partial one that
reads as "findings dropped." Verified as deliberate; see the
staleness suggestion.

### `!errors.Is(err, context.Canceled)` in main is the clean-shutdown idiom

`Watch` returns `ctx.Err()` when the signal context cancels, so
filtering `context.Canceled` before `fatalf` is correct — a clean
Ctrl-C should exit 0, not log "watch failed."

### Locking is consistent

Every access to `reports` (worker writes, aggregation loop,
`TotalFindings`) holds `w.mu`; the snapshot copies the map rather
than returning it. (`lastCounts` is only touched under the same lock
in the aggregation loop, so once it's initialised it needs no extra
synchronisation.)

## Process

### The included test is blind to all three bugs — and itself racy

**PR test plan + `internal/audit/watch_test.go`**, severity: **major
process concern**.

One namespace, `diff=false`, one cycle, no panicking auditor: bug 1's
path is never taken, bug 3's guard is never exercised, and bug 2 is
present but will only flake occasionally — meaning this test will
intermittently fail in CI *because of* blocker 2, and the first
person to see it will call the test flaky rather than the code racy.
Asks:

1. A `--diff` test running two cycles and asserting deltas — fails
   immediately today (blocker 1).
2. A panicking-auditor test asserting the daemon survives the cycle
   and logs the panic — fails today (blocker 3), and is the
   regression guard for the doc comment's core promise.
3. After the `wg.Add` fix, run the package tests with `-count=20`
   once to confirm the subset-aggregation flake is gone.

### `go vet` would have caught blocker 2 mechanically — wire it in

**Process**, severity: **minor but cheap**.

The misplaced `WaitGroup.Add` is one of the few concurrency bugs
with a first-class vet check. The PR checklist runs tests but not
`vet`; one line in the pre-merge script turns this whole class into
a non-event.

---

## What this sample review is trying to model

- **Structure-first triage at feature-branch scale.** One structural
  read, then passes by concern — state, concurrency, failure paths.
  On 200 lines, hunting line-by-line for "the bug" misses the chain;
  passes by concern are how each tier's bug surfaced.
- **Naming the chain, not just the links.** Blocker 1 is a one-line
  init fix and blocker 3 a misplaced defer — individually minor-
  sounding. The review's job is to price the *combination*: headline
  flag → panic → failed containment → crash loop. Severity lives in
  the system, not the line.
- **Pushing a known lesson one step further.** Exercise 22 taught the
  frame rule; this guard also fails on the goroutine axis, which no
  numbered exercise covered. The reasoning tool — "where exactly must
  `recover()` sit for this promise to hold?" — generalises; the
  memorised pattern alone would have found half the bug.
- **Verification as half the review.** Five cleared items, each with
  its reason — the ticker idiom, the legal nil-map read, the
  deliberate staleness choice, the shutdown filter, the locking. On a
  capstone-sized diff, the cleared list is what proves the review
  actually covered the branch.
- **Tests and tools as process findings.** The included test isn't
  just thin — it's *racy because of the code under review*, and vet
  already knew about blocker 2. Connecting findings back to the
  workflow that should have caught them is what turns one review into
  fewer future reviews.
