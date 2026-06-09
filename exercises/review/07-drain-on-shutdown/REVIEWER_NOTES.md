# Sample Review — PR #243

This is one reasonable review of the drain-on-shutdown PR. Yours will
differ in tone, phrasing, and which process concerns you raise.
Compare the *substance*: did you find all three independent issues,
walk the shutdown end to end, and avoid flagging the `len(in)` log?

## Overall assessment

**Request changes.**

The feature design is right — spill the buffer, replay later, bounded
and simple. But the shutdown path has three independent defects, one
per leg: the signal refactor deregisters the handlers immediately (so
in production the drain can never run — the first SIGTERM is an
instant default kill), the drain loop spins forever on a closed
channel despite its doc comment promising otherwise, and the error
path closes `drainDone` twice, turning any drain failure into a panic.
Each is a small fix. Together they mean the feature, as merged, would
ship dead code that panics in its only reachable failure mode.

## Blockers

### 1. `setupSignals` deregisters the handlers before they can ever fire

**`cmd/pipeline/main.go`, `setupSignals`**, severity: **critical**.

```go
func setupSignals() context.Context {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return ctx
}
```

`defer stop()` runs when **`setupSignals` returns** — that is,
immediately. `NotifyContext`'s stop function deregisters the signal
handlers and restores the default disposition. Net effect: the
returned context can never be cancelled by a signal, and the first
SIGTERM kills the process on the spot — no drain, no log line, just
gone. The feature this PR adds is unreachable in production.

The pre-refactor code had `defer stop()` in `main`'s frame, where it
correctly meant "deregister when the *program* is done." Moving the
two lines into a helper silently moved the defer's frame. (Same
lesson as a recovery helper that calls `recover()` one frame too
deep: `defer` is frame-scoped, and extracting code across a function
boundary changes its meaning.)

**Suggested fix:** return the stop function to the caller and defer
it there:

```go
func setupSignals() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}
// in main:
ctx, stop := setupSignals()
defer stop()
```

(Or simply keep the two lines inline in `main` — they were fine.)

### 2. `Drain` spins forever on a closed channel — `break` exits the select, not the loop

**`internal/ingest/drain.go`, the receive case**, severity: **major**.

```go
case m, ok := <-in:
	if !ok {
		break
	}
```

In Go, an unlabeled `break` inside a `select` terminates the
**select statement**, not the enclosing `for`. On a closed channel
the receive is always immediately ready with `ok == false`, so each
iteration re-enters the select, the closed channel wins again (the
`default` branch is never chosen when another case is ready), and the
loop spins at 100% CPU forever. The doc comment explicitly promises
"Drain returns once the buffer is empty or in is closed" — the code
cannot keep that promise.

Today's only call site never closes `out`, so this is latent — but
the doc comment invites exactly the usage that hangs, and the
replay follow-up or any test that closes the channel will hit it.

**Suggested fix:** make the exit explicit:

```go
case m, ok := <-in:
	if !ok {
		return n, nil
	}
```

(or a labeled `break drainLoop` if you prefer to fall out of the
loop). Either way, please add the closed-channel test that the doc
comment implies — it currently would hang.

### 3. The drain error path closes `drainDone` twice

**`cmd/pipeline/main.go`, the shutdown block in `RunPipeline`**,
severity: **major**.

```go
if err != nil {
	slog.Warn("drain failed; buffered metrics lost", "err", err)
	close(drainDone)
}
slog.Info("drain complete", "spilled", n)
close(drainDone)
```

There is no `return` after the warn, so on any drain error execution
falls through: `drainDone` is closed, "drain complete" is logged
(directly contradicting the warn one line earlier), and `close` runs
a second time — `panic: close of closed channel`, during shutdown,
which is the worst possible time. The only reachable failure mode of
this feature ends in a panic.

**Suggested fix:** make the close unconditional and single:

```go
n, err := drainer.Drain(out)
close(drainDone)
if err != nil {
	slog.Warn("drain failed; buffered metrics lost", "err", err)
	return nil
}
slog.Info("drain complete", "spilled", n)
```

(If `drainDone` later gains more close sites, guard it with
`sync.Once` — this file has been burned by double-close before.)

## Suggestions

### Consider crash-safety for the spill file

**`internal/ingest/drain.go`**, severity: **minor**.

`O_APPEND` plus a plain `Close` means a crash mid-drain leaves a
torn last line, and the replay side will have to tolerate it. Either
note that contract in the replay PR, or write to a temp file and
rename. Not a blocker for v1.

## Questions

### What happens to metrics that arrive *during* the drain?

`Drain`'s `default:` branch returns the moment the buffer is empty —
but the readers feeding `out` aren't stopped first, so a metric
published a microsecond after the buffer empties is lost, same as
today. If that's accepted for v1 (the window is tiny), say so in the
doc comment. Related: when replay lands, spilled metrics will be
re-published — does the dedup layer's `ErrDuplicate` handling cover
replayed metrics, or do we need an idempotency key in the spill
format?

## Nits

None worth the author's time.

## Things I Verified

### `len(in)` in the log line is fine

**`internal/ingest/drain.go`.** Channel length is approximate the
instant it's read, which makes it wrong for control flow — but this
is a log message, where an approximate count is exactly the right
tool. Verified, not flagged.

### The per-call `defer f.Close()` is correct

One spill file opened once per `Drain` call, closed when `Drain`
returns — this is the function-scoped defer working as intended, not
the defer-in-loop trap. (The loop here iterates channel receives, not
file opens.)

## Process

### Each planted failure mode maps to a missing test — please add all three

**PR test plan**, severity: **major process concern**.

The included test drains an open channel holding two buffered
metrics — the one scenario in which none of the three bugs can fire.
Specifically:

1. The signal path is explicitly deferred ("hard to automate"), but
   blocker 1 means the feature does not work via signals *at all*. A
   subprocess test that sends itself SIGTERM, or even a documented
   manual checklist run, must gate this PR — it is the feature.
2. The doc comment promises closed-channel behaviour that has no
   test. (Writing it today would hang — which is the point.)
3. The drain-error path (unwritable spill path) has no test; today it
   panics.

---

## What this sample review is trying to model

- **Walking a path end to end instead of hunting a bug.** The three
  questions — can it trigger, does it terminate, does the aftermath
  run once — partition the shutdown so that finding one bug doesn't
  end the search. Compound paths deserve systematic coverage, not
  first-hit-wins.
- **Transferring a lesson across surfaces.** The frame-scoped-defer
  trap was learned on `recover()`; here it bites `signal.NotifyContext`.
  Reviewing by *mechanism* (what frame does this defer belong to?)
  catches what reviewing by memorized pattern misses.
- **Severity through reachability.** Blocker 1 makes the feature
  unreachable; blocker 3 fires only on errors; blocker 2 is latent
  until someone closes the channel. The review says which is which —
  "all three are blockers" is true, but *why each one matters* is
  what the author needs.
- **Doc comments as contracts.** Two of the three bugs contradict
  prose in the same diff ("returns once … closed", "drain complete"
  after "drain failed"). Reading the comments against the code is
  half the review.
