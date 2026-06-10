# Canonical Diagnosis — D02: Two Stacks, One Line

This is how an experienced engineer reads `ARTIFACT.txt`. Compare your
written diagnosis against the substance, not the phrasing.

## Reading the artifact

### 1. Reframe "flaky" immediately

The header facts: fails two nights in five, passes on re-run, *and*
the failing assertion's number changes between failures (35, 40, 45).
A deterministic logic bug fails the same way every time. A test that
fails intermittently with varying numbers under `-race` — and the
detector printed a WARNING — is not flaky; it is correctly reporting
a race each time the schedule happens to interleave. The month of
re-runs was a month of ignored evidence.

### 2. The two access stacks

```
Read at 0x00c0001c4018 by goroutine 27:
  audit.ConcurrentAudit.func1()  report.go:55 +0x1ec
Previous write at 0x00c0001c4018 by goroutine 25:
  audit.ConcurrentAudit.func1()  report.go:55 +0x284
```

Three observations, in descending order of importance:

- **The same line in both stacks** — `report.go:55` is racing against
  itself. The function isn't fighting some other subsystem; it is
  running concurrently with its own copies and sharing something it
  shouldn't.
- **A read and a write at the same line, two instruction offsets
  apart** — one goroutine was loading the cell while another had
  stored to it, unordered. A read/write pair at a single source line
  is read-modify-write caught mid-flight. (Later blocks in the same
  run show write/write pairs at the same line; which shape gets
  reported first is scheduling luck, and any unordered combination
  convicts.)
- **The same address in both** — one memory cell, not two elements of
  an array. Every worker is hitting one shared variable.

One frame below each access, `ConcurrentAudit.gowrap1` is the
compiler-generated wrapper that a `go` statement compiles into —
runtime scaffolding you will see in every modern race report and
traceback. Your code is the `func1` frame above it.

### 3. The created-at stacks

```
Goroutine 27 (running) created at:
  audit.ConcurrentAudit()        report.go:44
  audit_test.TestExercise12_RaceReport()  report_test.go:75
```

Both goroutines were born at the *same* creation site, `report.go:44`
— a `go` statement inside `ConcurrentAudit`, evidently in a loop
(distinct goroutines, one line). The test frame below it tells you
which entry point drove it, useful when a racy function has several
callers; here there's just the one. `(running)` vs `(finished)` is
incidental — a goroutine ending doesn't un-race its writes.

### 4. The corroborating count

```
report_test.go:81: expected 50 total findings ... got 35
```

The shared cell is written by every worker at line 55, and the test
loses findings in multiples of 5 — i.e., whole per-auditor batches.
The candidate that fits a single shared cell + whole-batch loss +
one write per worker is a slice header under unsynchronized
`append`: each append reads (ptr, len, cap) and writes them back, so
two interleaved appends can both extend the *old* slice and the
last writer wins, discarding the other's batch entirely. This also
explains the operator's "concurrent audits return fewer findings than
serial" report — same bug, production phrasing.

## The diagnosis

**Localization:** `internal/audit/report.go:55` — an unsynchronized
`findings = append(findings, result...)` executed by worker goroutines
launched at `report.go:44` inside `ConcurrentAudit`.

**Mechanism:** slice append is read-modify-write on the shared slice
header. N goroutines appending without a lock race on the header;
interleaved appends silently drop whole result batches (hence counts
of 35/40/45 against an expected 50, varying with the schedule), and
in unlucky interleavings can corrupt the slice outright.

**Fix:** guard the append with a mutex:

```go
mu.Lock()
findings = append(findings, result...)
mu.Unlock()
```

(Reading the file confirms the diagnosis and adds an irony worth
noting in review culture: the function already owns an `errMu` that
correctly serializes error recording, two lines above the unguarded
append — and its sibling `ParallelAudit` locks its append correctly
while having a different bug entirely. Synchronization discipline is
per-access-site, not per-file.)

This is exercise 12's planted bug;
`go test -race -count=5 ./internal/audit/ -run TestExercise12` verifies
the fix.

**Confidence:** postable as a finding. The detector is not heuristic —
a reported race is a real race; the only judgement call is the
mechanism story, and the count arithmetic locks that in.

## What this artifact teaches beyond the bug

- A race report is four stacks and an address; read them as a
  sentence: *who touched it, who touched it before, who started
  each*. Read vs write tells you whether you caught the load or the
  store of a read-modify-write.
- **Same line in both stacks** = a function racing its own instances =
  look for the fan-out that launched it (the `created at` line hands
  it to you).
- Varying-by-run wrong *counts* are a concurrency fingerprint.
  Deterministic bugs produce deterministic wrongness.
- `-race` failures are never "flaky tests." The race detector has no
  false positives — only schedules that didn't happen to expose the
  bug tonight.
