# Hints for Diagnosis Exercise D02

These hints are about *reading the artifact*, not about the bug. Read
one at a time.

## Hint 1: The shape of a race report

Every race report has the same skeleton:

```
<Read|Write> at <address> by goroutine N:     ← access 1, with stack
Previous write at <address> by goroutine M:   ← access 2, with stack
Goroutine N created at:                       ← who started goroutine N
Goroutine M created at:                       ← who started goroutine M
```

Read it as a sentence: *two goroutines touched the same memory, at
least one of them writing, with nothing ordering the two accesses*.
Whether the newer access reports as a Read or a Write depends on
which instruction the detector caught — later blocks in this same
run show write/write pairs at the same line; any unordered
combination convicts. The stacks tell you where each access
happened; the `created at` stacks tell you where each goroutine was
born. Start with the two access stacks — here they are unusually
easy to compare.

## Hint 2: Same line, twice

Both access stacks point at the same function and line:

```
audit.ConcurrentAudit.func1()
    .../internal/audit/report.go:55 +0x1ec   ← the read
    .../internal/audit/report.go:55 +0x284   ← the previous write
```

Same line, two different instruction offsets — hold onto that detail
for Hint 3. (The `ConcurrentAudit.gowrap1` frame underneath each
stack is the compiler-generated wrapper the `go` statement compiles
into — runtime scaffolding, present in every modern report. The
`func1` frame is your code.)

A line racing against *itself* means: the same function is running in
two goroutines at once, and that line touches something both
instances share. The `created at` stacks confirm the fan-out — both
goroutines were started at `report.go:44`, inside `ConcurrentAudit`,
inside a loop (two goroutines, one creation site, the testing
framework's frames underneath).

So before reading any code you know: `ConcurrentAudit` launches
worker goroutines at line 44; each worker touches shared state at
line 55; nothing orders those accesses. The only open question is
*what* the shared state is.

## Hint 3: The address, the offsets, and the count

`0x00c0001c4018` is the same in both accesses — one memory cell. And
the two accesses are a *read* and a *write* at the same source line,
two instruction offsets apart: the signature of read-modify-write
caught mid-flight. In Go, at a line every worker executes once per
result batch, the classic read-modify-write on one small cell is a
**slice header**: `s = append(s, ...)` *reads* the header (pointer,
len, cap) — the `+0x1ec` access — and *writes* it back — the
`+0x284` access. Two unsynchronized appends race on the header
itself.

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
