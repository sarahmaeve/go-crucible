# Hints for Review Exercise 07

These hints are progressive — read one at a time, try the review again,
and only open the next hint if you're still stuck.

## Hint 1: Count

There are **three** independent correctness issues — one in each leg
of the shutdown: triggering it, draining, and finishing up. There is
**one** suspicious-looking detail that is fine ("Things I Verified"),
**one** process concern about the test plan, and **one** question
about intent.

If you found two bugs and stopped, re-read the README's closing note:
three questions, three answers.

## Hint 2: Categories

1. **Can shutdown be triggered at all?** Look at what `setupSignals`
   does with `stop` — and *when* its `defer` runs. Compare with where
   the `defer stop()` lived before the refactor. Exercise 22 taught
   you that moving code across a function boundary changes which
   frame a `defer` belongs to; that lesson is not specific to
   `recover()`.

2. **Does the drain terminate?** The author handled the comma-ok
   flag — `if !ok { break }`. In Go, what does `break` break out of
   when you are inside a `select` inside a `for`? Now trace what the
   loop does on a closed channel, forever. (Exercise 14's territory,
   one mutation away.)

3. **Does the aftermath run exactly once?** Trace the error path
   through `RunPipeline`'s new shutdown block, statement by
   statement, and count how many times `close(drainDone)` executes.
   What does Go do on the second `close` of a closed channel?

## Hint 3: Lines

- `cmd/pipeline/main.go`, `setupSignals`: `defer stop()` fires when
  **setupSignals returns** — immediately. `signal.NotifyContext`'s
  stop function deregisters the handlers and restores default signal
  disposition. So the returned ctx can never be cancelled by a
  signal, and the first SIGTERM kills the process instantly — the
  drain this PR exists to add can never run in production. (The old
  code deferred `stop()` in `main`'s frame, where it belonged.)

- `internal/ingest/drain.go`, the receive case: `break` inside a
  `select` exits the **select**, not the `for`. On a closed channel,
  the receive is always immediately ready with `ok == false`, so the
  loop spins at full CPU forever — and the `default` branch can never
  be chosen, because a closed channel always wins the select. The doc
  comment explicitly promises "returns once … in is closed"; the code
  cannot. Fix: a labeled break, a `return`, or restructure.

- `cmd/pipeline/main.go`, the drain error path: `close(drainDone)` is
  followed by fall-through to `slog.Info(...)` and a second
  `close(drainDone)` — there is no `return` after the warn. Any drain
  error turns into a `panic: close of closed channel` during
  shutdown (and logs "drain complete" right after "drain failed").

- The fine detail: `len(in)` in the log line. Channel length is
  approximate under concurrency, but for a log message it's exactly
  right. Don't flag it; verify it.

- The process gap: the included test drains an **open** channel with
  buffered items. The doc comment's closed-channel promise is
  untested (that test would currently hang), the signal path is
  untested (bug 1 invisible), and the drain-error path is untested
  (bug 3 invisible). Each planted bug maps to a missing test.
