# Sample Review — PR #260

This is one reasonable review of the replay-throttle PR. Yours will
differ in tone and emphasis. Compare the *substance*: did you catch
the watchdog's timer churn next to a correct ticker, derive the
Shutdown bug from the API contract rather than a memorised pattern,
and connect the two in your overall assessment?

## Overall assessment

**Request changes.**

The replay design is solid — decoder goroutine feeding a throttled
loop, correct ticker for the rate limit, body cap on the endpoint.
Two issues block it, and they compound each other: the idle watchdog
allocates an abandoned 30-second timer on every iteration (so a
healthy high-volume replay steadily grows the timer heap — and the
watchdog never actually measures idleness, since every metric resets
it), and `srv.Shutdown(ctx)` is called with the already-cancelled
daemon context, so "graceful" shutdown returns immediately and kills
in-flight replays. Together: the replay load creates the memory
growth that gets the daemon restarted, and every restart truncates
the replay that caused it. Fixes are small; both have correct
siblings to copy from.

## Blockers

### 1. The idle watchdog is `time.After` in a hot loop — timer churn under exactly the load replay exists for

**`internal/ingest/replay.go`, the fourth select case**, severity:
**major**.

```go
case <-time.After(rp.idleTimeout):
	return n, fmt.Errorf("replay: input stalled for %s", rp.idleTimeout)
```

`time.After` allocates a fresh timer every time the `select` is
entered — every loop iteration. During a healthy replay at the
configured rate (one metric per 10ms), another case wins ~100 times
per second, and each loser timer sits in the runtime's timer heap
until its full 30 seconds elapse: roughly **3,000 live timers per
replay stream at steady state**, times however many concurrent
replays, for the entire duration of a multi-million-line replay.
That's sustained allocation pressure in the very scenario this
endpoint was built for.

Note the rate limiter three lines up does it right —
`time.NewTicker` with a deferred `Stop`. So does
`internal/ingest/ticker.go`, already on `main`, for this exact
loop shape. The watchdog needs the same treatment:

```go
idle := time.NewTimer(rp.idleTimeout)
defer idle.Stop()
for {
	select {
	...
	case m, ok := <-lines:
		...
		idle.Reset(rp.idleTimeout)
	case <-idle.C:
		return n, fmt.Errorf("replay: input stalled for %s", rp.idleTimeout)
	}
}
```

This also fixes a semantic wrinkle: with `time.After`, the watchdog
restarts from zero on *every* select entry, so it measures "time
since the last loop iteration," which is only accidentally the same
as "input stalled." The explicit `Reset` on receipt makes the
intended meaning real.

### 2. `srv.Shutdown(ctx)` receives an already-cancelled context — graceful shutdown isn't

**`cmd/pipeline/main.go`, the shutdown goroutine in `serveAdmin`**,
severity: **major**.

```go
go func() {
	<-ctx.Done()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Warn("admin server shutdown", "err", err)
	}
}()
```

This line runs *after* `<-ctx.Done()`, so the context handed to
`Shutdown` is, by construction, already cancelled.
`http.Server.Shutdown`'s contract: it closes listeners, then waits
for in-flight connections to finish *only until its context is
done* — and this one already is. Shutdown returns immediately with
`context.Canceled`; in-flight connections are abandoned to be closed
when the process exits. The function's own comment — "shuts it down
gracefully so in-flight replays finish" — is exactly what does not
happen. A SIGTERM forty minutes into a fifty-minute replay throws
the whole thing away, with only a `Warn` line (which will log on
*every* shutdown, busy or idle, since `context.Canceled` is returned
either way — so the one signal you have is also a false alarm).

**Suggested fix:** the shutdown deserves its own budget, detached
from the context whose cancellation *triggered* it:

```go
<-ctx.Done()
shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := srv.Shutdown(shutCtx); err != nil {
	slog.Warn("admin server shutdown incomplete", "err", err)
}
```

### The interaction belongs in the merge decision

Neither bug is exotic alone. Together they form a loop: blocker 1's
timer growth is the kind of slow memory climb that gets a daemon
restarted (by ops or the OOM killer), and every restart goes through
blocker 2, which truncates the in-flight replay — whose re-run then
rebuilds the timer pressure. The system's failure mode isn't either
bug; it's the cycle. Worth stating in the PR so the fixes land
together rather than one-at-a-time.

## Suggestions

### Report replay progress on abort

**`cmd/pipeline/main.go`, `newReplayHandler`**, severity: **minor**.

The 500 body includes the count ("replay aborted after N metrics"),
which is good — but on a connection torn down by shutdown the client
may never see it. Consider logging `n` server-side on the error path
too, so ops can resume a truncated replay from a known offset
(pairs with the dedup question below).

## Questions

### Are replayed metrics idempotent against live ingestion?

A truncated replay will be re-POSTed from the top. Does the dedup
layer (`Deduplicator` / `types.ErrDuplicate`) sit downstream of the
replay sink, so the first N metrics of a re-run are treated as
idempotent duplicates rather than double-counted? If yes, a sentence
in the handler doc saying so; if no, that's a prerequisite for safe
re-runs and worth a linked issue before this merges.

## Nits

None worth the author's time.

## Things I Verified

### The `ErrServerClosed` filter is the documented idiom

**`cmd/pipeline/main.go`.**
`err != nil && !errors.Is(err, http.ErrServerClosed)` looks like
error-swallowing to a fresh eye, but `ListenAndServe` returns
`ErrServerClosed` as its *success* signal after `Shutdown` — treating
it as an error would log a failure on every clean exit. Correct as
written.

### The body cap is exercise-21's lesson, correctly applied

**`cmd/pipeline/main.go`.** `r.Body = http.MaxBytesReader(w, r.Body,
replayMaxBytes)` before the decoder, assigned back to `r.Body`, with
a cap sized for the artifact being uploaded. Nothing to flag —
pleasant to see the ingest handler's convention carried over.

### The decoder goroutine terminates on all paths

**`internal/ingest/replay.go`.** The producer selects on
`lines <- m` vs `ctx.Done()` and `defer close(lines)` runs on every
exit, so the consumer's `ok == false` path is reachable and the
goroutine cannot leak on cancellation — the exercise-06 shape, done
right. (The buffered `decErr` means the producer never blocks on the
error send either.)

## Process

### Both blockers are invisible to the test plan — and one of them needs a different kind of test

**PR test plan**, severity: **major process concern**.

A 50-line unit replay and a 1,000-line manual replay cannot surface
timer accumulation (the timers expire harmlessly within 30 seconds of
so short a run), and "Ctrl-C during idle" is precisely the shutdown
case where blocker 2 has nothing to truncate. Two asks:

1. For the watchdog: after the `NewTimer`/`Reset` fix, a unit test
   that the watchdog fires on a stalled reader *after* metrics have
   flowed (today's stall test starts stalled, which passes either
   way). Timer-heap growth itself is better caught by review than by
   test — which is what just happened — but a comment in the loop
   citing `ticker.go` will keep it from regressing.
2. For shutdown: a test that starts the admin server, begins a slow
   replay (blocking sink), cancels the daemon context, and asserts
   the replay request completes rather than aborting. This fails on
   the current code and is the regression guard for blocker 2.

---

## What this sample review is trying to model

- **Reviewing past the pattern library.** Blocker 2 matches no
  numbered exercise. It falls out of one discipline: when a function
  takes a context, ask *which* context, in *what state*, and what the
  callee documents about it. API contracts are the review tool that
  generalises.
- **Correct siblings as evidence.** The same diff contains the right
  ticker three lines above the wrong `time.After`, and the same
  package contains `ticker.go`. Pointing at the in-tree correct form
  makes the fix cheap to accept and hard to argue with.
- **Naming the interaction.** Two medium bugs that feed each other
  are a severe system, and the merge decision should hear that story,
  not two disconnected findings.
- **Positive verification with specifics.** The `ErrServerClosed`
  idiom, the body cap, the clean producer-goroutine — each verified
  with the reason it's right. On an advanced diff, what you *cleared*
  is as load-bearing as what you flagged.
