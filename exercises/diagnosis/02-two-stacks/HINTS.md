# Hints for Diagnosis Exercise D02

These hints are about *reading the artifact*, not about the bug. Read
one at a time.

## Hint 1: The shape of a race report

Every race report has the same skeleton:

```
Write at <address> by goroutine N:    ← access 1, with stack
Previous write at <address> by goroutine M:   ← access 2, with stack
Goroutine N created at:               ← who started goroutine N
Goroutine M created at:               ← who started goroutine M
```

Read it as a sentence: *two goroutines wrote the same memory without
synchronization between the writes*. The stacks tell you where each
write happened; the `created at` stacks tell you where each goroutine
was born. Start with the two access stacks — here they are unusually
easy to compare.

## Hint 2: Same line, twice

Both access stacks are identical:

```
audit.ConcurrentAudit.func1()
    .../internal/audit/report.go:55
```

A line racing against *itself* means: the same function is running in
two goroutines at once, and that line touches something both
instances share. The `created at` stacks confirm the fan-out — both
goroutines were started at `report.go:44`, inside `ConcurrentAudit`,
inside a loop (two goroutines, one creation site).

So before reading any code you know: `ConcurrentAudit` launches
worker goroutines at line 44; each worker writes shared state at line
55; nothing orders those writes. The only open question is *what* the
shared state is.

## Hint 3: The address, and the count

`0x00c0001c4018` is the same in both accesses — one memory cell. A
write/write race on a single small cell, at a line every worker
executes once per result batch, in Go, is very characteristically a
**slice header**: `s = append(s, ...)` *reads* the header (pointer,
len, cap) and *writes* it back, so two unsynchronized appends race on
the header itself.

That also explains the assertion failure: `got 35` instead of 50.
Two appends that interleave can both read the old header and write
back competing versions — one batch's worth of findings simply
vanishes. The CI annotation's failing counts (35, 40, 45) are all
50 minus a multiple of 5: whole 5-finding batches lost, varying by
how the interleaving fell. Intermittent, count-varying loss is the
fingerprint of an append race, not of a logic bug.

Now open `internal/audit/report.go:55` and confirm. While you're
there: compare with `ParallelAudit` later in the same file — it locks
its append (and has a different exercise's bug instead). The fix for
line 55 is the pattern you'll see there.
