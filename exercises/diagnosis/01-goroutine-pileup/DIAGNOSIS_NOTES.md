# Canonical Diagnosis — D01: The Pile-Up

This is how an experienced engineer reads `ARTIFACT.txt`, in the order
the information actually lands. Compare your written diagnosis against
the substance here — your phrasing and confidence calibration may
legitimately differ.

## Reading the artifact

### 1. The debug=1 counts (ten seconds in)

```
goroutine profile: total 3871
3848 @ ...
#	... ingest.ReadMetrics+0x84	.../internal/ingest/reader.go:18
#	... main.(*scheduler).runTarget.func1+0x90	.../scheduler.go:219
```

The aggregated view exists precisely for this moment: 3,848 of 3,871
goroutines share a single stack, and that stack is one frame deep into
*our* library. Every other bucket is recognizable baseline — 8 HTTP
connections in `netpoll` (the pprof server itself and friends), 6 GC
background workers, a handful of scheduler-owned loops. You don't need
to understand the scheduler's code to dismiss its buckets: 4 + 2 + 1
goroutines are not a 3,900-goroutine incident.

**Leak signature: many goroutines, one stack.** A burst of legitimate
work also produces many goroutines, which is why the count alone isn't
proof — the next two observations are.

### 2. The debug=2 state headers

```
goroutine 81442 [chan send, 1129 minutes]:
```

- `chan send` — parked trying to *send*. The question to ask of any
  blocked sender: who is supposed to be receiving, and where are they?
  There is no corresponding population of receivers anywhere in the
  profile.
- `1129 minutes` — nineteen hours in a single send. And the snip note
  says durations range from 2 to 9,841 minutes: a steady accrual, one
  goroutine at a time, for exactly the week the config migration was
  churning targets. Arrival pattern matches the incident timeline —
  that is the correlation that upgrades "suspicious" to "cause."

### 3. The frames

```
github.com/go-crucible/go-crucible/internal/ingest.ReadMetrics(...)
	/build/go-crucible/internal/ingest/reader.go:18 +0x84
main.(*scheduler).runTarget.func1()
	/srv/metrics-scheduler/scheduler.go:219 +0x91
created by main.(*scheduler).runTarget in goroutine 412
	/srv/metrics-scheduler/scheduler.go:212 +0x190
```

The scheduler deliberately owns the goroutine and invokes the synchronous
`ReadMetrics` API from it. The goroutine is parked inside our library at
`reader.go:18`; the scheduler frame and creator identify ownership but do not
change where the blocking operation needs its cancellation path.

The SRE's claim — "we cancel every removed target's context" — now
reads differently: cancellation is being *delivered* but nothing at
`reader.go:18` is *listening* for it.

## The diagnosis

**Localization:** `internal/ingest/reader.go:18`, the channel send inside
`ReadMetrics`, running in a goroutine owned by the embedding scheduler.

**Mechanism:** the goroutine's send `out <- m` is bare. When the
consumer stops reading — which is exactly what happens when the
scheduler cancels a removed target's context and abandons the channel —
the send blocks forever. The goroutine holds its metric, its channel
reference, and its stack, permanently. One removed target equals one
immortal goroutine: the dashboard's monotonic climb.

**Fix:** give the send an exit path —

```go
select {
case out <- m:
case <-ctx.Done():
	return ctx.Err()
}
```

This is exercise 06's planted bug; `make test-exercise N=06` verifies
the fix.

**Confidence:** postable as a finding, not just a hypothesis — count,
state, duration spread, and frame all agree, and the incident timeline
matches the wait-duration spread. The one confirming read in the
source (is the send wrapped in a `select`?) takes ten seconds.

## What this artifact teaches beyond the bug

- `debug=1` answers "what is most of my process doing?"; `debug=2`
  answers "how long, and in what state?". Capture both; read them in
  that order.
- Wait *durations* are the cheapest leak/burst discriminator: a burst
  has uniform young durations, a leak has a spread as old as the
  trigger.
- Caller and `created by` frames distinguish ownership from mechanism: the
  scheduler owns the goroutine, while the library owns the uncancellable send.
- Frames you don't own (runtime, the embedding binary) are context,
  not noise: here they proved the *caller* was healthy and the library
  was not, before a single line of source was read.
