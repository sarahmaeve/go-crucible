# Hints for Review Exercise 10 (Capstone)

These hints are progressive — read one at a time, try the review again,
and only open the next hint if you're still stuck.

## Hint 1: Count

There are **three** correctness issues — one each at beginner,
intermediate, and advanced tier, in that order of how fast you'll
likely spot them. Two of them form a chain: one detonates, the other
fails to contain the blast. There are **three or four**
suspicious-looking things that are correct and belong in "Things I
Verified," **two** process concerns, and **one** question.

Triage tip for a diff this size: read `watch.go` top to bottom once
for structure only, then make three passes — one for state (maps,
fields, who initialises what), one for concurrency (goroutines,
WaitGroups, locks), one for failure paths (errors, panics, what the
doc comments promise).

## Hint 2: Categories

1. **State.** `NewWatcher` initialises one of the two maps it owns.
   Find the other one, find where it is written, and note which flag
   gates that write. A nil map read returns the zero value; a nil map
   write panics. (Exercise 02 — including the part about why some
   paths are safe and others aren't.)

2. **Concurrency.** The per-namespace fan-out: where is `wg.Add(1)`
   relative to the `go` statement? What can `wg.Wait()` do if the
   scheduler hasn't started any of those goroutines yet? Symptom in
   production: cycles that intermittently log
   `total_findings=0` and, with `--diff`, phantom deltas in both
   directions. (Exercise 13 — and `go vet` flags this mechanically.)

3. **Failure paths.** The doc comments promise — twice — that a
   panicking auditor cannot take the daemon down. Interrogate the
   guard on two axes. First, the exercise-22 axis: what exactly is
   deferred, and how many frames below it does `recover()` sit?
   Second, an axis no numbered exercise covered: the auditors run in
   *spawned goroutines*, and a panic can only be recovered by a
   deferred call **in the same goroutine**. Where would the guard
   have to live for the promise to be true?

## Hint 3: Lines and the chain

- `watch.go`, `NewWatcher`: `reports` gets `make(...)`; `lastCounts`
  never does. The only write is `w.lastCounts[ns] = r.Summary.Total`
  inside the `if w.diff` block — so plain watch mode is fine, and the
  **first cycle of any `--diff` run panics** with "assignment to
  entry in nil map." The read two lines earlier
  (`w.lastCounts[ns]`) is legal on a nil map and returns 0 — don't
  let it reassure you.

- `watch.go`, the fan-out: `wg.Add(1)` is the first statement
  *inside* the goroutine. Move it before the `go` statement. Until
  then, `wg.Wait()` can return while zero goroutines are registered,
  and the cycle aggregates whatever subset has finished writing.

- `watch.go`, the guard: `defer func() { w.recoverCyclePanic() }()`
  is exercise 22's exact shape — `recover()` runs one frame too deep
  and always returns nil. But fixing the frame
  (`defer w.recoverCyclePanic()`) only protects panics raised on
  `runCycle`'s own goroutine — which is precisely what the nil-map
  panic in the aggregation loop is. Auditor panics happen in the
  per-namespace goroutines, where nothing recovers, and a panic in
  an unguarded goroutine kills the whole process regardless of any
  recovery elsewhere. The guard must also exist inside the spawned
  goroutine for the doc comment's promise to hold.

- **The chain:** enable `--diff` → first cycle hits the nil-map write
  → the panic unwinds through a guard that cannot fire → daemon dies
  → supervisor restarts it → first cycle panics again. The beginner
  bug is the detonator; the advanced bug is the failed containment;
  the result is a crash loop in cluster monitoring. Say this in your
  overall assessment — it changes the severity of both bugs.

- Things that are fine: the `NewTicker`/`defer Stop`/immediate-first-
  cycle shape (the post-exercise-18 idiom, done right); the nil-map
  *read*; the stale-beats-wrong handling of auditor errors (logged,
  commented, keeps the previous report — defensible; suggest a
  staleness signal, don't block); and main.go's
  `!errors.Is(err, context.Canceled)` filter (clean-shutdown idiom).
