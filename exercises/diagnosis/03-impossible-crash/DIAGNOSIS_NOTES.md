# Canonical Diagnosis — D03: The Crash That Couldn't Happen

This is how an experienced engineer works `ARTIFACT.txt`. D03 is
deliberately a differential-diagnosis exercise: the artifact proves a
contradiction, and the job is to enumerate explanations and eliminate
them against the source. Compare your hypothesis list and elimination
order, not just your final answer.

## What the traceback proves

```
panic: reprocess: malformed sample 117: negative timestamp delta

goroutine 1 [running]:
panic({...})                              runtime/panic.go:792
main.reprocessBatch.func1({...})          /srv/reprocess/main.go:84
worker.(*Pool).processOne(...)            pool.go:72
worker.(*Pool).Process(...)               pool.go:59
main.reprocessBatch(...)                  /srv/reprocess/main.go:71
main.main()                               /srv/reprocess/main.go:38
```

Reading frame line numbers as call sites: the processor closure
(main.go:84) panicked; it had been invoked by `processOne` at
**pool.go:72**; `processOne` was invoked by `Process` at
**pool.go:59**; and the panic travelled the entire chain to the top
of `main` and killed the process (`exit status 2`).

So the artifact alone establishes: the panic *entered* the pool's
per-item function at pool.go:72 and *left it unimpeded*. A traceback
that ends the process looks exactly like this when **no recovery
exists at all** — which is the contradiction, because the package
documentation quoted in the artifact promises recovery, per
invocation, explicitly.

Crucially, a traceback cannot distinguish "no recovery was deferred"
from "a deferred recovery ran and recovered nothing": deferred
functions that run and return during unwinding leave **no frames**.
The traceback's silence about recovery is compatible with both. The
discriminating evidence is elsewhere.

## What the absences prove

**Zero `processor panicked` log lines in all history, with a real
logger attached** (attachment notes 2 and 3). The documented recovery
logs the panic value and stack *when it fires*. Tonight's crash plus
zero historical hits means recovery has never fired — not once, in
any environment, for any panic. That eliminates "works usually,
failed tonight" explanations (races, edge-case panic values) in one
stroke. Whatever is wrong is wrong **structurally, on every
execution**.

**Determinism** corroborates: the crash happened on the first
malformed sample (117 of 2,000), immediately, on goroutine 1. Nothing
schedule-dependent.

## Hypotheses, ranked before opening the source

1. **Recovery exists but is structurally unable to recover** — runs
   on every panic, `recover()` returns nil every time. Fits all
   evidence: docs sincere, no log line ever (the `!= nil` body never
   executes), deterministic. Go has exactly one famous way to build
   this: calling `recover()` somewhere other than directly in the
   deferred function.
2. **The documented recovery is on a different path** than the one
   the traceback shows (e.g., only in some batch-level wrapper, not
   per invocation). Fits the log silence; strains against docs that
   say "per invocation"; killed the moment `processOne` is read.
3. **No recovery in the shipped code at all** (docs describe an
   intended design never implemented). Fits the evidence too — ranked
   last only because maintained packages with emphatic doc contracts
   usually have *something*; one read of `processOne` settles it.

Note what's *not* on the list: anything involving the processor
closure misusing panics, logger misconfiguration (note 3 kills it),
or flakiness (determinism kills it).

## Elimination against the source

`internal/worker/pool.go`, `processOne`:

```go
func (p *Pool) processOne(m types.Metric) (r Result) {
	r.In = m
	defer func() {
		p.recoverPanic(&r)
	}()
	out, err := p.fn(m)
	...
}
```

Recovery exists (hypothesis 3 dead) and is on the per-invocation path
(hypothesis 2 dead). And it has hypothesis 1's exact shape: the
function passed to `defer` is the anonymous closure; `recoverPanic` —
where `recover()` lives — is called *by* the deferred function, one
frame too deep. The spec's condition is precise:

> The return value of recover is nil if … recover was not called
> directly by a deferred function.

So on every panic: the closure runs, `recoverPanic` runs, `recover()`
returns nil, the `if v := recover(); v != nil` body is skipped — no
log, no `r.Err` — the closure returns, and unwinding continues. The
recovery handler executes and accomplishes nothing, every time, which
is precisely what "zero log lines ever + tonight's clean traceback"
predicted.

## The fix

```go
defer p.recoverPanic(&r)
```

Deferring the method itself makes `recoverPanic` the deferred
function, putting `recover()` directly inside it. `&r` is evaluated
at defer time — the address of the named return value — so the
recovered error lands on the right Result. The body of `recoverPanic`
needs no changes.

This is exercise 22's planted bug. Verify with the whole package —
`go test ./internal/worker/ -v` — exercise test plus both companions.

**Field confirmation for the reprocess team:** on the next malformed
sample they should see a `processor panicked` log line carrying the
panic value and stack, the batch should complete all 2,000 samples,
and exactly the malformed ones should come back with `Result.Err`
set. If they see the log line but still crash, the diagnosis was
wrong — that combination would have pointed at a re-panic downstream
instead.

## What this artifact teaches beyond the bug

- Frame line numbers are call sites: a traceback tells you the exact
  hand-off line in every function on the path before you open
  anything.
- Recovery that fails by construction is invisible in a traceback —
  deferred functions that ran leave no frames. Diagnose it from
  *absence*: the log/side-effect the recovery should have produced.
- "Never worked" and "failed this time" are different bug classes
  with different hypothesis lists. A zero-hit historical grep is one
  of the strongest pieces of evidence an incident can produce — it
  collapses the second class entirely.
- Write the hypothesis list *before* reading the source, ranked, each
  with its kill-condition. The source read then takes one minute and
  can't seduce you into the first plausible story.
