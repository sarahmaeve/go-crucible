# Hints for Review Exercise 05

These hints are progressive — read one at a time, try the review again,
and only open the next hint if you're still stuck.

## Hint 1: Count

There are **two** correctness issues that must be fixed before merging,
**one** suspicious-but-correct pattern for "Things I Verified," **one**
process concern about the test plan, and **one** question about intent.

If your draft flags the accumulation loop inside `Aggregate`, look
again — that one is the red herring.

## Hint 2: Categories

Without naming lines, the two correctness issues are:

1. A helper that annotates summaries does all of its work on copies
   and throws the work away. The feature it powers — a field the PR
   description explicitly promises to the renderer — silently never
   activates. Exercise 17's lesson, in diff form.

2. A goroutine's doc comment promises it can be abandoned via ctx;
   its send statement makes no such promise. When the consumer stops
   reading partway through, where does that goroutine spend the rest
   of the process's life? Exercise 06's lesson, in diff form.

The red herring: not every mutation-inside-a-range-loop is the
Exercise 17 bug. What is the map's *value type* in `Aggregate`'s
accumulation loop, and why does that change everything?

## Hint 3: Lines

- `markTruncated` ranges with `for _, s := range summaries` — `s` is a
  **copy** of the map value (`Summary` is a struct, not a pointer).
  `s.Truncated = ...` mutates the copy, and there is no
  `summaries[key] = s` write-back, so every stored summary keeps
  `Truncated == 0`. The "and N more" feature is a silent no-op. Fix:
  range with the key and write back, or compute `Truncated` at the
  point where `out[team] = *s` is built — the value is already known
  there.

- `Stream`'s goroutine sends with a bare `out <- summaries[t]`. The
  doc comment says "Cancel ctx to abandon delivery," but ctx appears
  only in a debug log line — there is no `select` with a `ctx.Done()`
  case on the send. A consumer that takes one summary and walks away
  (shutdown, error) leaves this goroutine blocked on the send forever,
  and `defer close(out)` never runs. One leaked goroutine per
  evaluation cycle, forever, in a daemon. Fix:

  ```go
  select {
  case out <- summaries[t]:
  case <-ctx.Done():
      return
  }
  ```

- The accumulation loop in `Aggregate` uses `map[string]*Summary` —
  `s.Count++` mutates through the *pointer*, which updates the stored
  value just fine. Pointer-valued maps don't need write-backs;
  value-valued maps do. The later `out[team] = *s` conversion to value
  copies is deliberate (callers can't reach into the aggregator's
  state). Note both in "Things I Verified."

- Process: what fixture would make `Truncated` nonzero? (Four or more
  firing alerts on one team — the included test has two.) What test
  would catch the `Stream` leak? (A consumer that reads one summary,
  cancels ctx, and asserts the channel closes promptly.)
