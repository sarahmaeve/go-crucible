# Hints for Diagnosis Exercise D01

These hints are about *reading the artifact*, not about the bug. Read
one at a time.

## Hint 1: Where to look

The `debug=1` view buckets identical stacks and prefixes each bucket
with a count. Read the counts before you read any frames: 3,848 of
the process's 3,871 goroutines share **one** stack. Everything else —
the HTTP server's 8, the GC's 6, the scheduler's own handful — is a
normal-looking baseline. One bucket *is* the incident.

## Hint 2: What the state line tells you

In the `debug=2` view, the header of each goroutine reads:

```
goroutine 81442 [chan send, 1129 minutes]:
```

Three facts in one line: the goroutine exists, it is blocked sending
on a channel (`chan send` — not receiving, not selecting), and it has
been blocked for **1,129 minutes**. A goroutine eighteen hours deep
into a single channel send is not slow — its receiver is gone. Note
also the *spread* of durations in the snip note (2 minutes to 9,841
minutes): goroutines have been entering this state steadily for the
whole migration week, one per churn event. That spread is what
distinguishes a leak from a one-off stall.

## Hint 3: Walking the frames

Each leaked goroutine has exactly one frame of its own plus a
`created by` line:

```
github.com/go-crucible/go-crucible/internal/ingest.ReadMetrics.func1()
	/build/go-crucible/internal/ingest/reader.go:18 +0x84
created by github.com/go-crucible/go-crucible/internal/ingest.ReadMetrics in goroutine 412
	/build/go-crucible/internal/ingest/reader.go:12 +0x6a
```

`ReadMetrics.func1` is an anonymous function inside `ReadMetrics` —
launched by the `go` statement at `reader.go:12`, blocked at the send
on `reader.go:18`. The scheduler's frames (`/srv/metrics-scheduler/...`)
appear only as the *creator's* creator — the leak is wholly inside
the library code at those two lines.

Before opening the file, you can already write the diagnosis: the
goroutine started at `reader.go:12` sends on a channel at
`reader.go:18` with nothing forcing it to give up when its context is
cancelled. Now open `internal/ingest/reader.go:18` and check: is that
send wrapped in a `select` with a `ctx.Done()` case? What *should* it
look like? (This is exercise 06's bug — the fix is the same.)
