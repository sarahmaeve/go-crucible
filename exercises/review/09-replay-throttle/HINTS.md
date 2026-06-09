# Hints for Review Exercise 09

These hints are progressive — read one at a time, try the review again,
and only open the next hint if you're still stuck.

## Hint 1: Count

There are **two** correctness issues that must be fixed before
merging — one in the replay loop, one in the shutdown path — and they
interact. There are **two** suspicious-looking details that are
correct ("Things I Verified"), **one** process concern, and **one**
question.

One of the two bugs matches a numbered exercise; the other does not
match anything you've practised. For that one, the question to ask is
not "which pattern is this?" but "what does this stdlib function
document about its argument?"

## Hint 2: Categories

1. **The replay loop.** The throttle uses `time.NewTicker` —
   correctly, with a deferred `Stop`. But look at the *fourth* select
   case, the idle watchdog. What does `time.After` allocate, how
   often does this loop iterate during a healthy high-volume replay
   (one metric per 10ms), and when does each abandoned timer get
   collected? Multiply by a million-line replay. Then compare how
   `internal/ingest/ticker.go` (already on `main`) handles exactly
   this shape. Exercise 18's lesson, hiding next to a correct ticker.

2. **The shutdown goroutine.** `srv.Shutdown(ctx)` — read that line
   again and ask *which* context is being passed, and what state that
   context is in at the moment the line runs. Then check the
   documented contract of `http.Server.Shutdown`: what does it do
   when its context is already done? What happens to in-flight
   requests — like a half-finished million-line replay?

## Hint 3: Lines

- `internal/ingest/replay.go`, `case <-time.After(rp.idleTimeout):` —
  every loop iteration allocates a fresh 30-second timer; whenever
  any other case wins (which is every iteration of a healthy replay,
  100 times per second), the timer is abandoned but lives in the
  runtime's timer heap until its full 30 seconds elapse. Sustained
  replay ⇒ ~3,000 live timers at steady state per replay stream, plus
  GC pressure — and the idle timeout never actually fires *between*
  iterations the way the author thinks: it's reset by every metric.
  The fix is the pattern `ticker.go` already demonstrates: one
  `time.NewTimer(rp.idleTimeout)` before the loop, `Reset` after each
  received metric, `defer Stop`.

- `cmd/pipeline/main.go`, `srv.Shutdown(ctx)` — the goroutine runs
  this line *after* `<-ctx.Done()`, so the context it hands to
  `Shutdown` is already cancelled. Per `http.Server.Shutdown`'s
  contract, it closes listeners and then waits for in-flight
  connections *only until the context is done* — which it already is.
  So Shutdown returns immediately with `context.Canceled`, the
  function comment's promise ("so in-flight replays finish") is
  false, and a SIGTERM mid-replay truncates the replay with no
  record of how far it got. Fix: give Shutdown its own budget —
  `context.WithTimeout(context.Background(), 30*time.Second)`.

- The interaction, both directions: the throttle bug's memory growth
  is what gets the daemon restarted by ops or the OOM killer — and
  every such restart goes through the shutdown bug, killing whatever
  replay triggered the growth. Conversely the shutdown bug is
  harmless on an idle server; it needs the long-running replays this
  endpoint exists to serve before it has anything to truncate.

- Things that are fine: the
  `err != nil && !errors.Is(err, http.ErrServerClosed)` filter is the
  documented idiom (`ErrServerClosed` is the *success* signal of a
  graceful shutdown); and `http.MaxBytesReader` on the replay body is
  exercise 21's lesson correctly applied — verify both, flag neither.
