# Hints for Diagnosis Exercise D03

These hints are about *reading the artifact and structuring the
differential*, not about the bug. Read one at a time.

## Hint 1: Read the frames as a sentence

Bottom-up, the traceback says: `main` called `reprocessBatch`
(main.go:71 is the `Process` call site), which called
`(*Pool).Process` (pool.go:59 is where it invokes `processOne`),
which called `processOne` (pool.go:72 is where it invokes the
processor), which called the closure, which panicked.

Each frame's line number is the **call site within that function** —
so the traceback has already told you exactly which line of
`processOne` hands control to user code, without you opening the
file. The panic entered library code at pool.go:72 and exited the top
of the program without anything stopping it.

One thing the traceback can *never* show: deferred functions that ran
during unwinding and returned normally. If a recovery handler ran and
failed, it left no frame. Its evidence must come from somewhere else.

## Hint 2: Mine the absences

Two absences, two eliminations:

- **No `processor panicked` line — ever, anywhere, with a real
  logger.** The pool's documented recovery logs before converting the
  panic to an error. If recovery had *fired* and something downstream
  re-panicked, the log line would exist. Zero hits across all history
  means `recover()` has never once returned a non-nil value in this
  pool. Not "recovery usually works and failed tonight" — **never
  worked**.
- **The crash is deterministic and immediate** (sample 117, first
  malformed input). Not a race, not load-dependent. Whatever is wrong
  is structural — wrong on every execution, latent until the first
  panic.

Your hypothesis list should now look something like: (a) there is no
recovery on this code path at all (docs lie / wrong method); (b) the
recovery exists but is conditional and the condition excluded this
panic; (c) the recovery exists, runs, and is structurally incapable
of recovering. The absences above already strain (a) — the docs are
emphatic — and (b) — a condition would have matched *some* historical
panic. Now read the source with that list in hand.

## Hint 3: In the source, interrogate the defer

`internal/worker/pool.go`, `processOne`:

```go
defer func() {
	p.recoverPanic(&r)
}()
```

and `recoverPanic` calls `recover()` inside itself. The Go spec's
condition is exact: recover returns nil if it was "not called
directly by a deferred function." The deferred function here is the
anonymous closure; `recoverPanic` is one frame *below* it, so its
`recover()` returns nil on every execution — hypothesis (c). The
recovery handler runs on every panic, recovers nothing, logs nothing
(its `if v := recover(); v != nil` body never executes), and the
panic continues. That is also why the log grep found zero hits ever.

The fix is to make `recoverPanic` itself the deferred function:

```go
defer p.recoverPanic(&r)
```

(`&r` is evaluated at defer time, which is exactly right — it's the
address of the named return value.) This is exercise 22's bug; the
worker package's three tests verify the fix.
